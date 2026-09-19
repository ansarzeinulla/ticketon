package api

import (
	"errors"
	"net/http"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// publicEventResponse is what an attendee sees: the event plus the ticket types
// that are actually offered, each with a remaining count.
type publicEventResponse struct {
	Event       store.Event        `json:"event"`
	TicketTypes []store.TicketType `json:"ticket_types"`
	OnSale      bool               `json:"on_sale"`
	SoldOut     bool               `json:"sold_out"`
}

// handleGetPublicEvent serves the attendee-facing event page, addressed by slug
// because that is what appears in a shareable URL.
//
// Private and unpublished events are 404 here even for their organizer: this is
// the public view, and the organizer has their own authenticated endpoints.
// Unlisted events resolve, which is the point of an unlisted link.
func (s *Server) handleGetPublicEvent(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if blank(slug) {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"An event slug is required.")
		return
	}

	event, err := s.events.GetBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this slug.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	if event.Status != store.EventStatusPublished || event.Visibility == store.VisibilityPrivate {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this slug.")
		return
	}

	types, err := s.ticketTypes.ListForEvent(r.Context(), event.ID, true)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	now := s.now()
	onSale := false
	remaining := 0
	for _, t := range types {
		if t.OnSaleAt(now) {
			onSale = true
			remaining += t.QuantityRemaining
		}
	}

	// Registration windows gate the whole event, on top of per-type windows.
	if event.RegistrationOpensAt != nil && now.Before(*event.RegistrationOpensAt) {
		onSale = false
	}
	if event.RegistrationClosesAt != nil && !now.Before(*event.RegistrationClosesAt) {
		onSale = false
	}

	httpx.WriteJSON(w, http.StatusOK, publicEventResponse{
		Event:       event,
		TicketTypes: types,
		OnSale:      onSale && len(types) > 0,
		SoldOut:     len(types) > 0 && remaining == 0,
	})
}
