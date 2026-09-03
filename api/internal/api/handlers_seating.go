package api

import (
	"errors"
	"net/http"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// CodeNoSeatingPlan is returned for an event sold by tier rather than by seat.
const (
	CodeNoSeatingPlan   = "no_seating_plan"
	CodeSeatUnavailable = "seat_unavailable"
)

type seatMapResponse struct {
	SeatMap store.SeatMap `json:"seat_map"`
}

// handleEventSeatMap serves the interactive seat map (SRS 4.3.1).
//
// Public, like the event page it is drawn on: an attendee must be able to see
// what is left before deciding to buy, and before signing in.
//
// The response is deliberately live rather than cached. A seat map whose held
// seats are a minute stale is a map that invites somebody to click a seat they
// cannot have.
func (s *Server) handleEventSeatMap(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadEvent(w, r)
	if !ok {
		return
	}
	if !s.canView(r, event) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return
	}

	plan, err := s.seating.MapForEvent(r.Context(), event.ID)
	if errors.Is(err, store.ErrNoSeatingPlan) {
		httpx.WriteError(w, http.StatusConflict, CodeNoSeatingPlan,
			"This event sells general admission tickets, not assigned seats.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, seatMapResponse{SeatMap: plan})
}
