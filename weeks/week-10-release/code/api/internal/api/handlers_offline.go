package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// maxSyncBatch caps one reconciliation. A door that has been offline for an
// evening has hundreds of admissions, not hundreds of thousands, and an
// unbounded batch is an unbounded transaction.
const maxSyncBatch = 500

type rosterResponse struct {
	Roster store.Roster `json:"roster"`
}

// handleEventRoster hands a scanner the ticket list to work offline from
// (SRS 4.8).
//
// Authorised exactly like scanning: this is the guest list plus the means to
// validate it, and it has no business being readable by anyone who is not
// working that door.
func (s *Server) handleEventRoster(w http.ResponseWriter, r *http.Request) {
	eventID, ok := s.eventIDForScanning(w, r)
	if !ok {
		return
	}

	roster, err := s.offline.Roster(r.Context(), eventID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	// Never cached: a roster is a snapshot, and a stale one held by an
	// intermediary is worse than a fresh download.
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, rosterResponse{Roster: roster})
}

type syncRequest struct {
	CheckIns []struct {
		TicketID  string `json:"ticket_id"`
		ScannedAt string `json:"scanned_at"`
		Device    string `json:"device_label"`
	} `json:"check_ins"`
}

type syncResponse struct {
	Results []store.SyncResult `json:"results"`
	// Counts save the app tallying the results itself just to show a summary.
	Recorded         int `json:"recorded"`
	AlreadyCheckedIn int `json:"already_checked_in"`
	Rejected         int `json:"rejected"`
}

// handleSyncCheckIns reconciles admissions made while the device was offline
// (SRS 4.8, "synchronize check-in records with the central platform").
//
// Every entry is reported on individually. One ticket refunded while the door
// was offline must not discard the good admissions queued behind it, and the
// staff who let that person in are entitled to know it happened.
func (s *Server) handleSyncCheckIns(w http.ResponseWriter, r *http.Request) {
	eventID, ok := s.eventIDForScanning(w, r)
	if !ok {
		return
	}

	var req syncRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	if len(req.CheckIns) == 0 {
		httpx.WriteJSON(w, http.StatusOK, syncResponse{Results: []store.SyncResult{}})
		return
	}
	if len(req.CheckIns) > maxSyncBatch {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeValidationFailed,
			"Too many queued check-ins in one request. Sync them in smaller batches.")
		return
	}

	entries := make([]store.SyncedCheckIn, 0, len(req.CheckIns))
	for _, entry := range req.CheckIns {
		ticketID, err := uuid.Parse(entry.TicketID)
		if err != nil {
			httpx.WriteValidationError(w, fieldErrors{
				"check_ins": "Each queued check-in needs a ticket_id in UUID form.",
			})
			return
		}

		// A malformed timestamp is not worth refusing the whole batch over:
		// the store clamps it to now, which is the safest reading of "we know
		// this happened, we are unsure exactly when".
		scannedAt, _ := time.Parse(time.RFC3339, entry.ScannedAt)

		entries = append(entries, store.SyncedCheckIn{
			TicketID: ticketID, ScannedAt: scannedAt, Device: entry.Device,
		})
	}

	results, err := s.offline.SyncCheckIns(r.Context(), eventID, mustUserID(r.Context()), entries)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	response := syncResponse{Results: results}
	for _, result := range results {
		switch result.Outcome {
		case "recorded":
			response.Recorded++
		case "already_checked_in":
			response.AlreadyCheckedIn++
		default:
			response.Rejected++
		}
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}
