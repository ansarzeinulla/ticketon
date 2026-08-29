package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// Refund error codes. Each failure has its own so the dashboard can say what
// actually happened rather than showing one generic message.
const (
	CodeAlreadyRefunded = "already_refunded"
	CodeNotRefundable   = "not_refundable"
)

type refundRequest struct {
	Reason string `json:"reason"`
}

type refundResponse struct {
	Refund store.Refund `json:"refund"`
	Order  store.Order  `json:"order"`
	// VoidedTickets is what the confirmation shows: "3 tickets are now void".
	VoidedTickets int `json:"voided_tickets"`
}

// handleRefundOrder reverses a paid order (SRS 4.9).
//
// Only the event's organizer - or a platform admin, for moderation - may
// refund. An attendee cannot refund their own order: SRS 4.9 gives the power
// to "authorized organizers", and a self-service refund would be a policy
// decision this MVP does not make.
func (s *Server) handleRefundOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The order id must be a UUID.")
		return
	}

	var req refundRequest
	// An empty body is fine: a reason is useful, not required.
	if r.ContentLength > 0 {
		if err := httpx.DecodeJSON(w, r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
			return
		}
	}
	if len(req.Reason) > maxReasonLength {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The reason is too long.")
		return
	}

	// Authorisation is decided by the order's event, so the order is read
	// first - but only its event id is used before the check.
	order, err := s.checkout.GetOrder(r.Context(), orderID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No order with this id.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	event, ok := s.requireEventOwner(w, r, order.Order.EventID)
	if !ok {
		return
	}

	actorID := mustUserID(r.Context())
	result, err := s.refunds.Refund(r.Context(), store.RefundParams{
		OrderID: orderID,
		ActorID: actorID,
		Reason:  req.Reason,
	})
	switch {
	case errors.Is(err, store.ErrAlreadyRefunded):
		httpx.WriteError(w, http.StatusConflict, CodeAlreadyRefunded,
			"This order has already been refunded.")
		return
	case errors.Is(err, store.ErrFreeOrderNotRefundable):
		// A free registration has nothing to refund. Name the action that does
		// work rather than leaving the organizer to guess (SRS 4.9).
		httpx.WriteError(w, http.StatusConflict, CodeNotRefundable,
			"This registration was free, so there is nothing to refund. Cancel it instead.")
		return
	case errors.Is(err, store.ErrOrderNotRefundable):
		httpx.WriteError(w, http.StatusConflict, CodeNotRefundable,
			"Only a paid order can be refunded.")
		return
	case errors.Is(err, store.ErrNoSucceededPayment):
		httpx.WriteError(w, http.StatusConflict, CodeNotRefundable,
			"This order has no completed payment to refund.")
		return
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No order with this id.")
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}

	// SRS 4.10: the attendee is told their refund completed. Sent after the
	// refund has committed, so nothing is announced that did not happen.
	s.sendRefundConfirmation(result, req.Reason)
	_ = event

	httpx.WriteJSON(w, http.StatusOK, refundResponse{
		Refund:        result.Refund,
		Order:         result.Order,
		VoidedTickets: result.VoidedTickets,
	})
}

type eventOrdersResponse struct {
	Orders []store.EventOrder `json:"orders"`
	Total  int                `json:"total"`
}

// handleListEventOrders is the organizer's attendee view: who bought what, and
// which orders can still be refunded.
func (s *Server) handleListEventOrders(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	orders, err := s.refunds.ListEventOrders(r.Context(), event.ID, 0)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, eventOrdersResponse{Orders: orders, Total: len(orders)})
}

// requireEventOwner authorises an action against an event the caller did not
// name in the path - a refund is addressed by order id, but permission still
// belongs to the event.
func (s *Server) requireEventOwner(
	w http.ResponseWriter, r *http.Request, eventID uuid.UUID,
) (store.Event, bool) {
	event, err := s.events.GetByID(r.Context(), eventID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return store.Event{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.Event{}, false
	}

	claims, _ := claimsFromContext(r.Context())
	userID := mustUserID(r.Context())
	if event.OrganizerID != userID && !(claims != nil && claims.HasRole(store.RolePlatformAdmin)) {
		httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
			"Only the organizer of this event can perform this action.")
		return store.Event{}, false
	}
	return event, true
}
