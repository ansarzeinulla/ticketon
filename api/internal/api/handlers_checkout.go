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
	// CodePaymentFailed reports a declined simulated payment (SRS 4.6, 4.10).
	CodePaymentFailed = "payment_failed"
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

	// Either a typed promo code or the opaque token from a scanned campaign QR.
	// The discount itself is never accepted from the client (SRS 4.14).
	PromoCode     string `json:"promo_code"`
	CampaignToken string `json:"campaign_token"`
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

	// --- the event has to be open for business ------------------------------
	switch {
	case event.Status == store.EventStatusSuspended:
		// SRS 4.12: a suspended event stops selling immediately. Checked before
		// any inventory is touched, so a suspension takes effect on the very
		// next request rather than after whatever is in flight.
		httpx.WriteError(w, http.StatusForbidden, CodeEventSuspended,
			"Ticket sales for this event have been suspended by BiletFlow pending review.")
		return
	case event.Status == store.EventStatusCancelled:
		httpx.WriteError(w, http.StatusConflict, CodeSalesClosed,
			"This event has been cancelled and is no longer selling tickets.")
		return
	case event.Status != store.EventStatusPublished:
		httpx.WriteError(w, http.StatusConflict, CodeSalesClosed,
			"This event is not on sale.")
		return
	}

	// SRS 4.12: suspending a user has to stop their events taking money, not
	// merely block their own login. Checked here, alongside the event's own
	// status and before any inventory is touched, so a suspension takes effect
	// on the very next request. Tickets already sold stay valid - stranding
	// paying attendees is not the remedy for an organizer's misconduct.
	organizer, err := s.users.GetByID(r.Context(), event.OrganizerID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		httpx.WriteInternalError(w, r, err)
		return
	}
	if organizer.Status == store.StatusSuspended {
		httpx.WriteError(w, http.StatusForbidden, CodeOrganizerSuspended,
			"Ticket sales for this event are paused while BiletFlow reviews the organizer's account.")
		return
	}

	now := s.now()
	if event.RegistrationOpensAt != nil && now.Before(*event.RegistrationOpensAt) {
		httpx.WriteError(w, http.StatusConflict, CodeSalesClosed,
			"Registration for this event has not opened yet.")
		return
	}
	if event.RegistrationClosesAt != nil && !now.Before(*event.RegistrationClosesAt) {
		httpx.WriteError(w, http.StatusConflict, CodeSalesClosed,
			"Registration for this event has closed.")
		return
	}

	// --- the request itself --------------------------------------------------
	errs := fieldErrors{}

	buyerEmail := normalizeEmail(req.BuyerEmail)
	if msg := validateEmail(buyerEmail); msg != "" {
		errs.add("buyer_email", msg)
	}
	if blank(req.BuyerName) {
		errs.add("buyer_name", "Your name is required.")
	} else if len(req.BuyerName) > maxNameLength {
		errs.add("buyer_name", "Name is too long.")
	}

	if len(req.Items) == 0 {
		errs.add("items", "Select at least one ticket.")
	} else if len(req.Items) > maxItemsPerOrder {
		errs.add("items", "Too many different ticket types in one order.")
	}

	// Merge duplicate lines so two entries for the same type are one purchase,
	// and so the per-order limit cannot be dodged by splitting the request.
	merged := map[uuid.UUID]int{}
	order := []uuid.UUID{}
	for i, item := range req.Items {
		id, parseErr := uuid.Parse(item.TicketTypeID)
		if parseErr != nil {
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
		_ = i
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	items := make([]store.CheckoutItem, 0, len(order))
	for _, id := range order {
		items = append(items, store.CheckoutItem{TicketTypeID: id, Quantity: merged[id]})
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

	// --- resolve the promo, if the attendee brought one ---------------------
	var promo *store.Campaign
	if !blank(req.PromoCode) || !blank(req.CampaignToken) {
		campaign, resolveErr := s.campaigns.Resolve(r.Context(), store.ResolveParams{
			EventID:       event.ID,
			Code:          req.PromoCode,
			CampaignToken: req.CampaignToken,
		})
		if resolveErr != nil {
			s.writePromoError(w, r, resolveErr)
			return
		}
		if usableErr := store.CheckUsable(campaign, now); usableErr != nil {
			s.writePromoError(w, r, usableErr)
			return
		}

		// A campaign restricted to ticket types the buyer did not pick would
		// otherwise silently discount nothing.
		covers := false
		for _, item := range items {
			if campaign.AppliesTo(item.TicketTypeID) {
				covers = true
				break
			}
		}
		if !covers {
			s.writePromoError(w, r, store.ErrPromoNotApplicable)
			return
		}

		promo = &campaign
	}

	// --- the atomic part -----------------------------------------------------
	result, err := s.checkout.Checkout(r.Context(), store.CheckoutParams{
		EventID:     event.ID,
		BuyerUserID: buyerUserID,
		BuyerName:   req.BuyerName,
		BuyerEmail:  buyerEmail,
		BuyerPhone:  req.BuyerPhone,
		Items:       items,
		Promo:       promo,
	})
	if err != nil {
		s.writeCheckoutError(w, r, err)

		// SRS 4.10: the buyer is told their payment failed. Sent after the
		// response, like every other notification, and only for a decline -
		// an out-of-stock basket is not a payment failure.
		var declined *store.PaymentDeclinedError
		if errors.As(err, &declined) {
			s.sendPaymentFailure(event, req.BuyerName, buyerEmail, "", declined.Reason)
		}
		return
	}

	w.Header().Set("Location", "/api/v1/orders/"+result.Order.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, result)

	// SRS 4.10: purchase confirmation and ticket delivery. The order is
	// already committed and the response already written, so a notification
	// problem cannot cost the attendee their tickets.
	s.sendOrderConfirmation(result, event)
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
		inventoryErr  *store.InsufficientInventoryError
		notOnSaleErr  *store.NotOnSaleError
		maxErr        *store.ExceedsMaxPerOrderError
		activationErr *store.PaidSalesNotActiveError
		declinedErr   *store.PaymentDeclinedError
	)

	switch {
	case errors.As(err, &declinedErr):
		// 402 Payment Required: the request was well formed and the event was
		// willing to sell, but the (simulated) money did not arrive. SRS 4.6:
		// "Failed or abandoned transactions shall not create valid tickets" -
		// the decline happens before any inventory moves, so there is nothing
		// to unwind and nothing was issued.
		httpx.WriteError(w, http.StatusPaymentRequired, CodePaymentFailed, declinedErr.Reason)

	case errors.As(err, &activationErr):
		// 403 rather than 409: this is not a race the attendee can retry out
		// of, it is the event not being cleared to take money yet (SRS 4.5).
		httpx.WriteError(w, http.StatusForbidden, CodePaidSalesNotActive,
			capitalise(activationErr.Error())+".")

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
		// A promo can run out between the preview and the payment, so the
		// checkout path has to report that as precisely as the preview does.
		if status, code, message := promoErrorCode(err); code != "" {
			httpx.WriteError(w, status, code, message)
			return
		}
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
