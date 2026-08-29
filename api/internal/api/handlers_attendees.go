package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

type attendeeSearchResponse struct {
	Attendees []store.AttendeeTicket `json:"attendees"`
	Total     int                    `json:"total"`
	Query     string                 `json:"query"`
}

// handleSearchAttendees backs the scanner app's manual lookup (SRS 4.8).
//
// Authorised the same way scanning is: the organizer or somebody assigned to
// work this event's door. An attendee list is exactly the kind of thing that
// must not be readable by anyone who happens to know an event id.
func (s *Server) handleSearchAttendees(w http.ResponseWriter, r *http.Request) {
	// The same gate the scanner passes through: the event must exist and this
	// account must be authorised to work its door.
	eventID, ok := s.eventIDForScanning(w, r)
	if !ok {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 120 {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"That search term is too long.")
		return
	}

	attendees, err := s.attendees.Search(r.Context(), eventID, query)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, attendeeSearchResponse{
		Attendees: attendees, Total: len(attendees), Query: query,
	})
}

type manualCheckInRequest struct {
	TicketID string `json:"ticket_id"`
	Device   string `json:"device_label"`
}

// handleManualCheckIn admits a ticket found by search rather than by camera
// (SRS 4.8).
//
// It is a separate route from the scanner's, because the two are addressed by
// different things: a scan carries a QR token, this carries a ticket id that
// only an authorised device could have obtained from the search above. What
// happens in the database is identical.
func (s *Server) handleManualCheckIn(w http.ResponseWriter, r *http.Request) {
	eventID, ok := s.eventIDForScanning(w, r)
	if !ok {
		return
	}

	var req manualCheckInRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	ticketID, err := uuid.Parse(strings.TrimSpace(req.TicketID))
	if err != nil {
		errs := fieldErrors{}
		errs.add("ticket_id", "A ticket id in UUID form is required.")
		httpx.WriteValidationError(w, errs)
		return
	}

	device := strings.TrimSpace(req.Device)
	if device == "" {
		// Recorded so the check-in history distinguishes a door scan from a
		// name typed in by hand - the second is worth being able to audit.
		device = "manual search"
	}

	result, err := s.checkIns.CheckInByTicketID(
		r.Context(), eventID, ticketID, mustUserID(r.Context()), device)
	if err != nil {
		s.writeCheckInError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)
}
