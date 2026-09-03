package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// Hold error codes, so the UI can say what happened rather than "error".
const (
	CodeHoldExpired    = "hold_expired"
	CodeHoldNotPending = "hold_not_pending"
	CodeSeatTaken      = "seat_taken"
)

type holdRequest struct {
	Items []checkoutItemRequest `json:"items"`
	// SeatIDs are the specific seats an assigned-seating basket is claiming
	// (SRS 4.3.1).
	SeatIDs []string `json:"seat_ids"`
	// Buyer details are optional here - somebody picking seats has not filled
	// in the form yet - and required to confirm.
	BuyerName  string `json:"buyer_name"`
	BuyerEmail string `json:"buyer_email"`
}

type holdResponse struct {
	Hold store.Hold `json:"hold"`
}

// handleCreateHold reserves inventory for a basket (SRS 4.6).
//
// Anonymous, like checkout: an attendee picks seats before signing in, and
// demanding an account to look at a seat map would lose the sale.
func (s *Server) handleCreateHold(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadSellableEvent(w, r)
	if !ok {
		return
	}

	var req holdRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	seatIDs, seatErr := parseUUIDs(req.SeatIDs)

	errs := fieldErrors{}
	if seatErr != nil {
		errs.add("seat_ids", "Each seat must be identified by a UUID.")
	}

	// An assigned-seating basket names seats, not tiers: what a seat costs
	// follows from where it is, and the server works that out. Only a general
	// admission basket has to say which tier it wants.
	var items []store.CheckoutItem
	if len(seatIDs) == 0 {
		merged, itemErrs := mergeCheckoutItems(req.Items)
		for field, message := range itemErrs {
			errs.add(field, message)
		}
		items = merged
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	var buyerUserID *uuid.UUID
	if claims, ok := claimsFromContext(r.Context()); ok {
		if id, err := claims.UserID(); err == nil {
			buyerUserID = &id
		}
	}

	held, err := s.checkout.Hold(r.Context(), store.HoldParams{
		EventID:     event.ID,
		BuyerUserID: buyerUserID,
		BuyerName:   req.BuyerName,
		BuyerEmail:  normalizeEmail(req.BuyerEmail),
		Items:       items,
		SeatIDs:     seatIDs,
	})
	if err != nil {
		s.writeCheckoutError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/orders/"+held.OrderID.String()+"/hold")
	httpx.WriteJSON(w, http.StatusCreated, holdResponse{Hold: held})
}

// handleGetHold reads a basket back, so a reloaded page resumes it.
func (s *Server) handleGetHold(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The order id must be a UUID.")
		return
	}

	held, err := s.checkout.GetHold(r.Context(), orderID)
	if err != nil {
		s.writeCheckoutError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, holdResponse{Hold: held})
}

// handleReleaseHold cancels a basket and returns its inventory immediately.
func (s *Server) handleReleaseHold(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The order id must be a UUID.")
		return
	}

	if err := s.checkout.Release(r.Context(), orderID); err != nil {
		s.writeCheckoutError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "released",
		"message": "The reservation was cancelled and the tickets are back on sale.",
	})
}

type confirmRequest struct {
	BuyerName  string  `json:"buyer_name"`
	BuyerEmail string  `json:"buyer_email"`
	BuyerPhone *string `json:"buyer_phone"`

	PromoCode     string `json:"promo_code"`
	CampaignToken string `json:"campaign_token"`
}

// handleConfirmHold pays for a held basket and issues its tickets (SRS 4.6).
func (s *Server) handleConfirmHold(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The order id must be a UUID.")
		return
	}

	var req confirmRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	// The basket is read first so the promo can be resolved against its event.
	held, err := s.checkout.GetHold(r.Context(), orderID)
	if err != nil {
		s.writeCheckoutError(w, r, err)
		return
	}

	errs := fieldErrors{}
	buyerEmail := normalizeEmail(req.BuyerEmail)
	if msg := validateEmail(buyerEmail); msg != "" {
		errs.add("buyer_email", msg)
	}
	if msg := validateLine("Your name", req.BuyerName, minNameLength, maxNameLength); msg != "" {
		errs.add("buyer_name", msg)
	}
	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	var buyerUserID *uuid.UUID
	if claims, ok := claimsFromContext(r.Context()); ok {
		if id, idErr := claims.UserID(); idErr == nil {
			buyerUserID = &id
		}
	}

	promo, ok := s.resolvePromoForBasket(w, r, held.EventID, held.Items,
		req.PromoCode, req.CampaignToken)
	if !ok {
		return
	}

	if req.BuyerPhone != nil && blank(*req.BuyerPhone) {
		req.BuyerPhone = nil
	}

	result, err := s.checkout.Confirm(r.Context(), store.ConfirmParams{
		OrderID:     orderID,
		BuyerUserID: buyerUserID,
		BuyerName:   req.BuyerName,
		BuyerEmail:  buyerEmail,
		BuyerPhone:  req.BuyerPhone,
		Promo:       promo,
	})
	if err != nil {
		s.writeCheckoutError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)

	// SRS 4.10: purchase confirmation and ticket delivery, after the response.
	event, eventErr := s.events.GetByID(r.Context(), held.EventID)
	if eventErr == nil {
		s.sendOrderConfirmation(result, event)
	}
}

// parseUUIDs converts a list of ids, rejecting the whole list if any is
// malformed - a basket half-understood is worse than one refused.
func parseUUIDs(raw []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(raw))
	for _, value := range raw {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// loadSellableEvent resolves the event and applies the gates that decide
// whether it may take an order at all.
func (s *Server) loadSellableEvent(w http.ResponseWriter, r *http.Request) (store.Event, bool) {
	eventID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The event id must be a UUID.")
		return store.Event{}, false
	}

	event, err := s.events.GetByID(r.Context(), eventID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return store.Event{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.Event{}, false
	}

	if !s.eventIsOpenForSales(w, r, event) {
		return store.Event{}, false
	}
	return event, true
}
