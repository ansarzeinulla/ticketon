package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// Cancellation error codes (SRS 4.9).
const (
	CodeAlreadyCancelled = "already_cancelled"
	CodeNotCancellable   = "not_cancellable"
	// CodeUseRefund is returned when somebody tries to cancel an order that
	// took money. It names the action they wanted rather than only refusing
	// the one they asked for.
	CodeUseRefund = "use_refund"
)

type cancelRequest struct {
	Reason string `json:"reason"`
}

type cancelResponse struct {
	Order store.Order `json:"order"`
	// CancelledTickets is what the confirmation shows: "1 ticket is now void".
	CancelledTickets int `json:"cancelled_tickets"`
}

// handleCancelOrder withdraws a free registration (SRS 4.9: "Organizers shall
// be able to cancel free registrations").
//
// Authorisation matches the refund endpoint exactly: the event's organizer, or
// a platform admin for moderation. An attendee cannot cancel their own place -
// SRS 4.9 gives the power to the organizer, and self-service withdrawal is a
// policy decision this MVP does not make.
func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The order id must be a UUID.")
		return
	}

	var req cancelRequest
	// An empty body is fine: a reason is useful, not required.
	if r.ContentLength > 0 {
		if err := httpx.DecodeJSON(w, r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
			return
		}
	}
	if runeLen(req.Reason) > maxReasonLength {
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

	if _, ok := s.requireEventOwner(w, r, order.Order.EventID); !ok {
		return
	}

	result, err := s.refunds.Cancel(r.Context(), store.CancelParams{
		OrderID: orderID,
		ActorID: mustUserID(r.Context()),
		Reason:  req.Reason,
	})
	switch {
	case errors.Is(err, store.ErrAlreadyCancelled):
		httpx.WriteError(w, http.StatusConflict, CodeAlreadyCancelled,
			"This registration has already been cancelled.")
		return
	case errors.Is(err, store.ErrPaidOrderNeedsRefund):
		httpx.WriteError(w, http.StatusConflict, CodeUseRefund,
			"This order was paid for. Refund it instead, so the money goes back.")
		return
	case errors.Is(err, store.ErrOrderNotCancellable):
		httpx.WriteError(w, http.StatusConflict, CodeNotCancellable,
			"This order is not in a state that can be cancelled.")
		return
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No order with this id.")
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}

	// SRS 4.10: the attendee is told. Sent after the cancellation has
	// committed, so nothing is announced that did not happen.
	s.sendCancellationConfirmation(result, req.Reason)

	httpx.WriteJSON(w, http.StatusOK, cancelResponse{
		Order:            result.Order,
		CancelledTickets: result.CancelledTickets,
	})
}
