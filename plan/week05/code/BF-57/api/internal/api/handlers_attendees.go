package api

import (
	"net/http"
	"strings"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

type eventOrdersResponse struct {
	Orders []store.EventOrder `json:"orders"`
}

type guestListResponse struct {
	Attendees []store.Guest `json:"attendees"`
	Total     int           `json:"total"`
	Query     string        `json:"query"`
}

// handleListEventOrders is the organizer's view of who bought what.
func (s *Server) handleListEventOrders(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	orders, err := s.guests.ListEventOrders(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, eventOrdersResponse{Orders: orders})
}

// handleListAttendees lists the people holding tickets, searchable by name,
// email, ticket code or order number. It is also the scanner app's manual
// lookup (SRS 4.8), so it is authorised the way scanning is: the organizer or
// somebody assigned to work this event's door. An attendee list must not be
// readable by anyone who happens to know an event id.
func (s *Server) handleListAttendees(w http.ResponseWriter, r *http.Request) {
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

	guests, err := s.guests.ListGuests(r.Context(), eventID, query)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, guestListResponse{
		Attendees: guests, Total: len(guests), Query: query,
	})
}
