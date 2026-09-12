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
	// Suspended tells the page to show the moderation banner instead of a
	// ticket selector (SRS 4.12).
	Suspended bool `json:"suspended"`
	// PaidSalesActive reports whether this event may take money yet (SRS 4.5).
	// When it is false and paid tickets exist, the page explains that those
	// tickets are not on sale rather than letting an attendee fill in a form
	// the checkout is certain to refuse.
	PaidSalesActive bool `json:"paid_sales_active"`
	// PaidSalesRequired is whether activation gates anything here at all. A
	// free event never needs it.
	PaidSalesRequired bool `json:"paid_sales_required"`
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

	// A suspended event stays visible, unlike a draft: people already hold
	// links to it, and telling them sales are paused is more use than a 404.
	suspended := event.Status == store.EventStatusSuspended
	visible := event.Status == store.EventStatusPublished || suspended

	if !visible || event.Visibility == store.VisibilityPrivate {
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
	if suspended {
		onSale = false
	}

	// Paid sales need an activated event (SRS 4.5). This mirrors the gate the
	// checkout enforces inside its transaction; the checkout remains the
	// authority, and this only decides what the page offers.
	activation, err := s.activations.ForEvent(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	if activation.RequiredForSales && !activation.IsActive {
		freeRemaining := 0
		for _, t := range types {
			if t.IsFree && t.OnSaleAt(now) {
				freeRemaining += t.QuantityRemaining
			}
		}
		// A free tier on the same event stays buyable: activation gates money,
		// not registration.
		onSale = onSale && freeRemaining > 0
	}

	httpx.WriteJSON(w, http.StatusOK, publicEventResponse{
		Event:             event,
		TicketTypes:       types,
		OnSale:            onSale && len(types) > 0,
		SoldOut:           len(types) > 0 && remaining == 0,
		Suspended:         suspended,
		PaidSalesActive:   activation.IsActive,
		PaidSalesRequired: activation.RequiredForSales,
	})
}
