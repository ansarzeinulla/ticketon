package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// Scanner result codes. The mobile app switches on these to decide whether to
// show the green or the red screen, so they are part of the contract.
const (
	CodeAlreadyCheckedIn = "already_checked_in"
	CodeCampaignToken    = "campaign_token"
	CodeWrongEvent       = "wrong_event"
	CodeTicketNotValid   = "ticket_not_valid"
	CodeUnknownTicket    = "unknown_ticket"
)

type checkInRequest struct {
	QRToken string `json:"qr_token"`
	Device  string `json:"device_label"`
}

type checkInResponse struct {
	Result  string              `json:"result"` // always "valid" here
	CheckIn store.CheckInResult `json:"check_in"`
}

// handleCheckIn admits the holder of a scanned QR token.
//
// Only the organizer, an assigned Event Admin, or a platform admin may scan:
// SRS 4.8 requires that staff see and act on their assigned events only.
func (s *Server) handleCheckIn(w http.ResponseWriter, r *http.Request) {
	eventID, ok := s.eventIDForScanning(w, r)
	if !ok {
		return
	}

	var req checkInRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	if blank(req.QRToken) {
		httpx.WriteValidationError(w, fieldErrors{"qr_token": "A scanned code is required."})
		return
	}

	result, err := s.checkIns.CheckIn(r.Context(), eventID, req.QRToken,
		mustUserID(r.Context()), req.Device)
	if err != nil {
		s.writeCheckInError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, checkInResponse{Result: "valid", CheckIn: result})
}

// scannerErrorBody extends the error envelope with what the scanner screen
// needs to render a useful red result rather than a bare message.
type scannerErrorBody struct {
	Error struct {
		Code         string              `json:"code"`
		Message      string              `json:"message"`
		AttendeeName string              `json:"attendee_name,omitempty"`
		CheckedInAt  *time.Time          `json:"checked_in_at,omitempty"`
		Stats        *store.CheckInStats `json:"stats,omitempty"`
	} `json:"error"`
}

func (s *Server) writeCheckInError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		already  *store.AlreadyCheckedInError
		campaign *store.CampaignTokenError
		wrong    *store.WrongEventError
		notValid *store.TicketNotAdmissibleError
	)

	switch {
	case errors.As(err, &already):
		var body scannerErrorBody
		body.Error.Code = CodeAlreadyCheckedIn
		body.Error.Message = "This ticket has already been used to enter."
		body.Error.AttendeeName = already.AttendeeName
		body.Error.CheckedInAt = &already.CheckedInAt
		body.Error.Stats = &already.Stats
		httpx.WriteJSON(w, http.StatusConflict, body)

	case errors.As(err, &campaign):
		// SRS 4.14: a campaign QR is never admission, and the scanner should
		// say exactly that so staff are not left guessing.
		httpx.WriteError(w, http.StatusBadRequest, CodeCampaignToken,
			"This is a promotional campaign code, not an admission ticket.")

	case errors.As(err, &wrong):
		var body scannerErrorBody
		body.Error.Code = CodeWrongEvent
		body.Error.Message = "This ticket is for " + wrong.EventTitle + ", not this event."
		httpx.WriteJSON(w, http.StatusConflict, body)

	case errors.As(err, &notValid):
		var body scannerErrorBody
		body.Error.Code = CodeTicketNotValid
		body.Error.Message = "This ticket is " + notValid.Status + " and cannot be used for entry."
		body.Error.AttendeeName = notValid.AttendeeName
		httpx.WriteJSON(w, http.StatusConflict, body)

	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeUnknownTicket,
			"This code does not match any ticket.")

	default:
		httpx.WriteInternalError(w, r, err)
	}
}

type reverseCheckInRequest struct {
	Reason string `json:"reason"`
}

type statsResponse struct {
	Stats store.CheckInStats `json:"stats"`
}

// handleReverseCheckIn undoes an accidental scan (SRS 4.8).
func (s *Server) handleReverseCheckIn(w http.ResponseWriter, r *http.Request) {
	detail, ok := s.loadTicket(w, r)
	if !ok {
		return
	}

	if !s.mayScan(w, r, detail.EventID) {
		return
	}

	var req reverseCheckInRequest
	if r.ContentLength > 0 {
		if err := httpx.DecodeJSON(w, r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
			return
		}
	}

	stats, err := s.checkIns.Reverse(r.Context(), detail.ID, mustUserID(r.Context()), req.Reason)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"This ticket is not currently checked in.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, statsResponse{Stats: stats})
}

// handleCheckInStats returns the live registered/checked-in counts.
func (s *Server) handleCheckInStats(w http.ResponseWriter, r *http.Request) {
	eventID, ok := s.eventIDForScanning(w, r)
	if !ok {
		return
	}

	stats, err := s.checkIns.Stats(r.Context(), eventID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, statsResponse{Stats: stats})
}

type scannableEventsResponse struct {
	Events []store.ScannableEvent `json:"events"`
}

// handleListScannableEvents powers the scanner app's event selector: the events
// this account may check attendees in for, and nothing else.
func (s *Server) handleListScannableEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.staff.ListScannable(r.Context(), mustUserID(r.Context()))
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, scannableEventsResponse{Events: events})
}

// eventIDForScanning parses {id} and checks the caller may scan for it.
func (s *Server) eventIDForScanning(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	eventID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The event id must be a UUID.")
		return uuid.Nil, false
	}

	if _, err := s.events.GetByID(r.Context(), eventID); errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return uuid.Nil, false
	} else if err != nil {
		httpx.WriteInternalError(w, r, err)
		return uuid.Nil, false
	}

	if !s.mayScan(w, r, eventID) {
		return uuid.Nil, false
	}
	return eventID, true
}

// mayScan writes a 403 and returns false when the caller has no authority over
// the event's gate.
func (s *Server) mayScan(w http.ResponseWriter, r *http.Request, eventID uuid.UUID) bool {
	claims, _ := claimsFromContext(r.Context())
	if claims != nil && claims.HasRole(store.RolePlatformAdmin) {
		return true
	}

	allowed, err := s.staff.CanScan(r.Context(), eventID, mustUserID(r.Context()))
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return false
	}
	if !allowed {
		httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
			"You are not assigned to check attendees in for this event.")
		return false
	}
	return true
}
