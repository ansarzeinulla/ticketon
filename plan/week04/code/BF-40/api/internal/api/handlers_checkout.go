package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// maxItemsPerOrder caps how many distinct ticket types one order may contain.
const maxItemsPerOrder = 20

// CodeInsufficientInventory is returned when stock cannot cover the request.
// It is its own code so the UI can react specifically - refresh the remaining
// counts and let the attendee pick again - rather than showing a generic error.
const (
	CodeInsufficientInventory = "insufficient_inventory"
	CodeNotOnSale             = "not_on_sale"
	CodeSalesClosed           = "sales_closed"
)

type checkoutItemRequest struct {
	TicketTypeID string `json:"ticket_type_id"`
	Quantity     int    `json:"quantity"`
}

type checkoutRequest struct {
	BuyerName  string                `json:"buyer_name"`
	BuyerEmail string                `json:"buyer_email"`
	BuyerPhone *string               `json:"buyer_phone"`
	Items      []checkoutItemRequest `json:"items"`

	// SeatIDs buys specific seats, for an assigned-seating event (SRS 4.3.1).
	// When present the tiers are derived from the seats, so `items` is not
	// needed.
	SeatIDs []string `json:"seat_ids"`
}

// handleCheckout runs the simulated purchase.
//
// Authentication is optional: an attendee may buy as a guest, which SRS 12
// leaves open as a product decision. When a token is present the order is
// linked to that account so it shows up in their order history later.
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The event id must be a UUID.")
		return
	}

	var req checkoutRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	event, err := s.events.GetByID(r.Context(), eventID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	if !s.eventIsOpenForSales(w, r, event) {
		return
	}

	// --- the request itself --------------------------------------------------
	errs := fieldErrors{}

	buyerEmail := normalizeEmail(req.BuyerEmail)
	if msg := validateEmail(buyerEmail); msg != "" {
		errs.add("buyer_email", msg)
	}
	if msg := validateLine("Your name", req.BuyerName, minNameLength, maxNameLength); msg != "" {
		errs.add("buyer_name", msg)
	}

	seatIDs, seatErr := parseUUIDs(req.SeatIDs)
	if seatErr != nil {
		errs.add("seat_ids", "Each seat must be identified by a UUID.")
	}

	// Seats price themselves; only a general admission basket names tiers.
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
		if id, idErr := claims.UserID(); idErr == nil {
			buyerUserID = &id
		}
	}

	if req.BuyerPhone != nil && blank(*req.BuyerPhone) {
		req.BuyerPhone = nil
	}

	// --- the atomic part -----------------------------------------------------
	result, err := s.checkout.Checkout(r.Context(), store.CheckoutParams{
		EventID:     event.ID,
		BuyerUserID: buyerUserID,
		BuyerName:   req.BuyerName,
		BuyerEmail:  buyerEmail,
		BuyerPhone:  req.BuyerPhone,
		Items:       items,
		SeatIDs:     seatIDs,
	})
	if err != nil {
		s.writeCheckoutError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/orders/"+result.Order.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, result)
}

// checkoutErrorBody adds the inventory numbers to the standard error envelope,
// so a client can show "only 3 left" without a second request.
type checkoutErrorBody struct {
	Error struct {
		Code         string `json:"code"`
		Message      string `json:"message"`
		TicketTypeID string `json:"ticket_type_id,omitempty"`
		Requested    int    `json:"requested,omitempty"`
		Remaining    int    `json:"remaining"`
	} `json:"error"`
}

func (s *Server) writeCheckoutError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		inventoryErr *store.InsufficientInventoryError
		notOnSaleErr *store.NotOnSaleError
		maxErr       *store.ExceedsMaxPerOrderError
		seatGone     *store.SeatUnavailableError
	)

	switch {
	case errors.As(err, &seatGone):
		httpx.WriteError(w, http.StatusConflict, CodeSeatUnavailable,
			"One of those seats is no longer available. Refresh the map and choose again.")

	case errors.As(err, &inventoryErr):
		var body checkoutErrorBody
		body.Error.Code = CodeInsufficientInventory
		body.Error.Message = capitalise(inventoryErr.Error()) + "."
		body.Error.TicketTypeID = inventoryErr.TicketTypeID.String()
		body.Error.Requested = inventoryErr.Requested
		body.Error.Remaining = inventoryErr.Remaining
		httpx.WriteJSON(w, http.StatusConflict, body)

	case errors.As(err, &maxErr):
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict, capitalise(maxErr.Error())+".")

	case errors.As(err, &notOnSaleErr):
		httpx.WriteError(w, http.StatusConflict, CodeNotOnSale, capitalise(notOnSaleErr.Error())+".")

	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound,
			"One of the selected ticket types does not belong to this event.")

	default:
		httpx.WriteInternalError(w, r, err)
	}
}

// handleGetOrder returns a placed order. The id is an unguessable UUID, which
// is what the confirmation link relies on; a signed-in buyer or the event's
// organizer may also read it.
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The order id must be a UUID.")
		return
	}

	result, err := s.checkout.GetOrder(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No order with this id.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)
}

// capitalise upper-cases the first letter of a sentence built from an error.
func capitalise(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	if runes[0] >= 'a' && runes[0] <= 'z' {
		runes[0] -= 'a' - 'A'
	}
	return string(runes)
}

// eventIsOpenForSales applies the gates that decide whether an event may take
// an order at all, writing the refusal itself when it may not.
func (s *Server) eventIsOpenForSales(
	w http.ResponseWriter, r *http.Request, event store.Event,
) bool {
	switch {
	case event.Status == store.EventStatusCancelled:
		httpx.WriteError(w, http.StatusConflict, CodeSalesClosed,
			"This event has been cancelled and is no longer selling tickets.")
		return false
	case event.Status != store.EventStatusPublished:
		httpx.WriteError(w, http.StatusConflict, CodeSalesClosed,
			"This event is not on sale.")
		return false
	}

	now := s.now()
	if event.RegistrationOpensAt != nil && now.Before(*event.RegistrationOpensAt) {
		httpx.WriteError(w, http.StatusConflict, CodeSalesClosed,
			"Registration for this event has not opened yet.")
		return false
	}
	if event.RegistrationClosesAt != nil && !now.Before(*event.RegistrationClosesAt) {
		httpx.WriteError(w, http.StatusConflict, CodeSalesClosed,
			"Registration for this event has closed.")
		return false
	}
	return true
}

// mergeCheckoutItems validates the requested lines and folds duplicates
// together, so two entries for the same tier are one purchase and the
// per-order limit cannot be dodged by splitting the request.
func mergeCheckoutItems(requested []checkoutItemRequest) ([]store.CheckoutItem, fieldErrors) {
	errs := fieldErrors{}

	if len(requested) == 0 {
		errs.add("items", "Select at least one ticket.")
		return nil, errs
	}
	if len(requested) > maxItemsPerOrder {
		errs.add("items", "Too many different ticket types in one order.")
		return nil, errs
	}

	merged := map[uuid.UUID]int{}
	order := []uuid.UUID{}
	for _, item := range requested {
		id, err := uuid.Parse(item.TicketTypeID)
		if err != nil {
			errs.add("items", "Each item needs a ticket_type_id in UUID form.")
			break
		}
		if item.Quantity <= 0 {
			errs.add("items", "Each selected ticket type needs a quantity of at least 1.")
			break
		}
		if _, seen := merged[id]; !seen {
			order = append(order, id)
		}
		merged[id] += item.Quantity
	}
	if errs.any() {
		return nil, errs
	}

	items := make([]store.CheckoutItem, 0, len(order))
	for _, id := range order {
		items = append(items, store.CheckoutItem{TicketTypeID: id, Quantity: merged[id]})
	}
	return items, errs
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
