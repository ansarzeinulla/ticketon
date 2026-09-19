package api

import (
	"net/http"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

type inventoryResponse struct {
	TicketTypes []store.InventoryLine `json:"ticket_types"`
	SoldOut     bool                  `json:"sold_out"`
}

// handleEventInventory reports what is left of each ticket type. Public, like
// the event page it refreshes: the numbers are shown to attendees anyway.
func (s *Server) handleEventInventory(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadEvent(w, r)
	if !ok {
		return
	}
	if !s.canView(r, event) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return
	}

	lines, err := s.inventory.ForEvent(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	remaining := 0
	for _, l := range lines {
		remaining += l.Remaining
	}
	httpx.WriteJSON(w, http.StatusOK, inventoryResponse{
		TicketTypes: lines,
		SoldOut:     len(lines) > 0 && remaining == 0,
	})
}
