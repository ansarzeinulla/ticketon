package api

import (
	"context"
	"errors"
	"math/big"
	"net/http"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

type promoPreviewRequest struct {
	Code          string                `json:"code"`
	CampaignToken string                `json:"campaign_token"`
	Items         []checkoutItemRequest `json:"items"`
}

// promoPreviewResponse is what the checkout screen shows before payment. The
// figures are computed by the server; the client never decides a discount.
type promoPreviewResponse struct {
	Code          string `json:"code"`
	CampaignID    string `json:"campaign_id"`
	CampaignName  string `json:"campaign_name"`
	DiscountType  string `json:"discount_type"`
	DiscountValue string `json:"discount_value"`
	SubtotalKZT   string `json:"subtotal_kzt"`
	DiscountKZT   string `json:"discount_kzt"`
	TotalKZT      string `json:"total_kzt"`
	Remaining     int    `json:"remaining"`
	// AppliesToAll is false when the campaign covers only some ticket types.
	AppliesToAll bool `json:"applies_to_all"`
}

// handlePromoPreview validates a promo code against a basket and returns what
// the discount would be.
//
// This is a preview, not a reservation: nothing is held and no counter moves.
// The authoritative check happens inside the checkout transaction, because the
// last redemption may be taken between previewing and paying.
func (s *Server) handlePromoPreview(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The event id must be a UUID.")
		return
	}

	var req promoPreviewRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	if blank(req.Code) && blank(req.CampaignToken) {
		httpx.WriteValidationError(w, fieldErrors{"code": "Enter a promo code."})
		return
	}

	campaign, err := s.campaigns.Resolve(r.Context(), store.ResolveParams{
		EventID:       eventID,
		Code:          req.Code,
		CampaignToken: req.CampaignToken,
	})
	if err != nil {
		s.writePromoError(w, r, err)
		return
	}

	if err := store.CheckUsable(campaign, s.now()); err != nil {
		s.writePromoError(w, r, err)
		return
	}

	// Price the basket from the stored ticket types, never from the client.
	subtotal, eligible, err := s.priceBasket(r.Context(), eventID, req.Items, campaign)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound,
				"One of the selected ticket types does not belong to this event.")
			return
		}
		httpx.WriteInternalError(w, r, err)
		return
	}

	if eligible.Sign() == 0 && subtotal.Sign() > 0 {
		s.writePromoError(w, r, store.ErrPromoNotApplicable)
		return
	}

	discount := discountFor(campaignPricingAdapter{campaign}, eligible)
	total := new(big.Rat).Sub(subtotal, discount)

	httpx.WriteJSON(w, http.StatusOK, promoPreviewResponse{
		Code:          campaign.Code,
		CampaignID:    campaign.ID.String(),
		CampaignName:  campaign.Name,
		DiscountType:  campaign.DiscountType,
		DiscountValue: campaign.DiscountValue,
		SubtotalKZT:   formatMoney(subtotal),
		DiscountKZT:   formatMoney(discount),
		TotalKZT:      formatMoney(total),
		Remaining:     campaign.Remaining(),
		AppliesToAll:  len(campaign.TicketTypeIDs) == 0,
	})
}

// priceBasket returns the basket subtotal and the part of it the campaign
// covers, both priced from the stored ticket types.
//
// The client sends only ticket type ids and quantities; prices come from the
// database, so a tampered basket cannot inflate a percentage discount.
func (s *Server) priceBasket(
	ctx context.Context, eventID uuid.UUID,
	items []checkoutItemRequest, campaign store.Campaign,
) (subtotal, eligible *big.Rat, err error) {
	subtotal, eligible = new(big.Rat), new(big.Rat)

	for _, item := range items {
		if item.Quantity <= 0 {
			continue
		}

		ticketTypeID, parseErr := uuid.Parse(item.TicketTypeID)
		if parseErr != nil {
			return nil, nil, store.ErrNotFound
		}

		ticketType, getErr := s.ticketTypes.GetByID(ctx, ticketTypeID)
		if getErr != nil {
			return nil, nil, getErr
		}
		if ticketType.EventID != eventID {
			return nil, nil, store.ErrNotFound
		}

		price, ok := parseMoney(ticketType.PriceKZT)
		if !ok {
			return nil, nil, store.ErrNotFound
		}

		line := new(big.Rat).Mul(price, new(big.Rat).SetInt64(int64(item.Quantity)))
		subtotal.Add(subtotal, line)

		if campaign.AppliesTo(ticketTypeID) {
			eligible.Add(eligible, line)
		}
	}

	return subtotal, eligible, nil
}

// campaignPricingAdapter exposes a stored campaign to the discount maths.
type campaignPricingAdapter struct{ c store.Campaign }

func (a campaignPricingAdapter) IsPercentage() bool {
	return a.c.DiscountType == store.DiscountPercentage
}

func (a campaignPricingAdapter) Value() string { return a.c.DiscountValue }

// writePromoError translates a promo failure, falling back to a 500 for
// anything unexpected.
func (s *Server) writePromoError(w http.ResponseWriter, r *http.Request, err error) {
	if status, code, message := promoErrorCode(err); code != "" {
		httpx.WriteError(w, status, code, message)
		return
	}
	httpx.WriteInternalError(w, r, err)
}
