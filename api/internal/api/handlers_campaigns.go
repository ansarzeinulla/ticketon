package api

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
	"github.com/biletflow/api/internal/ticketpdf"
)

// promoCodePattern matches the promo_codes_code_shape_chk constraint: a code
// has to survive being printed on a poster and typed back in, so it stays
// URL-safe and free of punctuation.
var promoCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)

// Scanner and checkout error codes for promotional problems.
const (
	CodePromoNotFound      = "promo_not_found"
	CodePromoNotActive     = "promo_not_active"
	CodePromoExpired       = "promo_expired"
	CodePromoNotStarted    = "promo_not_started"
	CodePromoExhausted     = "promo_exhausted"
	CodePromoNotApplicable = "promo_not_applicable"
	CodePromoWrongEvent    = "promo_wrong_event"
)

type campaignRequest struct {
	Name           string     `json:"name"`
	Code           string     `json:"code"`
	DiscountType   string     `json:"discount_type"`
	DiscountValue  string     `json:"discount_value"`
	MaxRedemptions *int       `json:"max_redemptions"`
	StartsAt       *time.Time `json:"starts_at"`
	EndsAt         *time.Time `json:"ends_at"`
	TicketTypeIDs  []string   `json:"ticket_type_ids"`
	Active         *bool      `json:"active"`
}

// campaignView adds the derived links a client needs to the stored campaign.
type campaignView struct {
	store.Campaign
	// CampaignURL is what the QR encodes: a trackable event link carrying only
	// the opaque token (SRS 4.14).
	CampaignURL string `json:"campaign_url"`
	QRImageURL  string `json:"qr_image_url"`
	Remaining   int    `json:"remaining"`
}

type campaignResponse struct {
	Campaign campaignView `json:"campaign"`
}

type campaignListResponse struct {
	Campaigns []campaignView `json:"campaigns"`
}

func (s *Server) viewCampaign(c store.Campaign, eventSlug string) campaignView {
	return campaignView{
		Campaign:    c,
		CampaignURL: store.CampaignLink(s.cfg.WebBaseURL, eventSlug, c.QRToken),
		QRImageURL:  "/api/v1/campaigns/" + c.ID.String() + "/qr.png",
		Remaining:   c.Remaining(),
	}
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	var req campaignRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}

	if blank(req.Name) {
		errs.add("name", "Give the campaign a name.")
	}

	if blank(req.Code) {
		errs.add("code", "A promo code is required.")
	} else if !promoCodePattern.MatchString(req.Code) {
		errs.add("code", "Use 3-32 letters, digits, hyphens or underscores - no spaces.")
	}

	switch req.DiscountType {
	case store.DiscountPercentage, store.DiscountFixed:
	default:
		errs.add("discount_type", "Discount type must be percentage or fixed_amount.")
	}

	if !moneyPattern.MatchString(req.DiscountValue) {
		errs.add("discount_value", "Discount must be a positive amount, such as 20 or 1500.")
	} else {
		value, err := strconv.ParseFloat(req.DiscountValue, 64)
		switch {
		case err != nil || value <= 0:
			errs.add("discount_value", "Discount must be greater than zero.")
		case req.DiscountType == store.DiscountPercentage && value > 100:
			errs.add("discount_value", "A percentage discount cannot exceed 100.")
		}
	}

	if req.MaxRedemptions != nil && *req.MaxRedemptions <= 0 {
		errs.add("max_redemptions", "The redemption limit must be at least 1.")
	}
	if req.StartsAt != nil && req.EndsAt != nil && !req.EndsAt.After(*req.StartsAt) {
		errs.add("ends_at", "The campaign must end after it starts.")
	}

	ticketTypeIDs := make([]uuid.UUID, 0, len(req.TicketTypeIDs))
	for _, raw := range req.TicketTypeIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			errs.add("ticket_type_ids", "Each ticket type must be a UUID.")
			break
		}
		ticketTypeIDs = append(ticketTypeIDs, id)
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	// A campaign is created active by default: an organizer making a promo code
	// almost always wants it to work straight away.
	status := store.CampaignActive
	if req.Active != nil && !*req.Active {
		status = store.CampaignDisabled
	}

	campaign, err := s.campaigns.Create(r.Context(), store.CreateCampaignParams{
		EventID:        event.ID,
		Name:           req.Name,
		Code:           req.Code,
		DiscountType:   req.DiscountType,
		DiscountValue:  req.DiscountValue,
		StartsAt:       req.StartsAt,
		EndsAt:         req.EndsAt,
		MaxRedemptions: req.MaxRedemptions,
		Status:         status,
		TicketTypeIDs:  ticketTypeIDs,
		CreatedBy:      mustUserID(r.Context()),
	})
	if errors.Is(err, store.ErrPromoCodeTaken) {
		httpx.WriteValidationError(w, fieldErrors{
			"code": "That promo code is already in use. Choose another.",
		})
		return
	}
	if err != nil {
		var constraintErr *store.ConstraintError
		if errors.As(err, &constraintErr) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeValidationFailed,
				"The campaign violates a database rule: "+constraintErr.Constraint+".")
			return
		}
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, event.ID, mustUserID(r.Context()), "campaign.created", "campaign",
		campaign.ID.String(), "Created campaign "+campaign.Name+" ("+campaign.Code+")")

	httpx.WriteJSON(w, http.StatusCreated, campaignResponse{
		Campaign: s.viewCampaign(campaign, event.Slug),
	})
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	campaigns, err := s.campaigns.ListForEvent(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	views := make([]campaignView, 0, len(campaigns))
	for _, c := range campaigns {
		views = append(views, s.viewCampaign(c, event.Slug))
	}
	httpx.WriteJSON(w, http.StatusOK, campaignListResponse{Campaigns: views})
}

type campaignStatusRequest struct {
	Active bool `json:"active"`
}

func (s *Server) handleSetCampaignStatus(w http.ResponseWriter, r *http.Request) {
	campaign, event, ok := s.loadOwnedCampaign(w, r)
	if !ok {
		return
	}

	var req campaignStatusRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	status := store.CampaignDisabled
	if req.Active {
		// Re-enabling a campaign that has already run out would be a lie, so
		// its exhausted state is preserved.
		if campaign.MaxRedemptions != nil && campaign.RedemptionCount >= *campaign.MaxRedemptions {
			httpx.WriteError(w, http.StatusConflict, CodePromoExhausted,
				"This campaign has used all of its redemptions and cannot be re-enabled.")
			return
		}
		status = store.CampaignActive
	}

	updated, err := s.campaigns.SetStatus(r.Context(), campaign.ID, status)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, event.ID, mustUserID(r.Context()), "campaign."+status, "campaign",
		updated.ID.String(), "Campaign "+updated.Code+" set to "+status)

	httpx.WriteJSON(w, http.StatusOK, campaignResponse{
		Campaign: s.viewCampaign(updated, event.Slug),
	})
}

func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	campaign, event, ok := s.loadOwnedCampaign(w, r)
	if !ok {
		return
	}

	// A redeemed campaign is part of the sales record, so it is disabled rather
	// than deleted (SRS 4.14 requires exact campaign reporting).
	redeemed, err := s.campaigns.HasRedemptions(r.Context(), campaign.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	if redeemed {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"This campaign has redemptions and cannot be deleted. Disable it instead.")
		return
	}

	if err := s.campaigns.Delete(r.Context(), campaign.ID); err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, event.ID, mustUserID(r.Context()), "campaign.deleted", "campaign",
		campaign.ID.String(), "Deleted campaign "+campaign.Code)

	w.WriteHeader(http.StatusNoContent)
}

// handleCampaignQR renders the campaign QR as a PNG.
//
// It encodes the trackable event link, never the discount: SRS 4.14 is explicit
// that the client must not be handed a value it could tamper with.
//
// No authentication: this image is meant to be printed on a poster, so there is
// nothing in it to protect - and an <img> tag cannot send a bearer header
// anyway, which is what made the organizer dashboard show a broken image.
// Creating, disabling and reporting on campaigns all still require the token.
func (s *Server) handleCampaignQR(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The campaign id must be a UUID.")
		return
	}

	campaign, err := s.campaigns.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No campaign with this id.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	event, err := s.events.GetByID(r.Context(), campaign.EventID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	png, err := ticketpdf.QRPNG(store.CampaignLink(s.cfg.WebBaseURL, event.Slug, campaign.QRToken))
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	filename := "biletflow-campaign-" + safeFilename(campaign.Code) + ".png"

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(png)))
	// The organizer downloads this to put on a poster.
	w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// loadOwnedCampaign resolves {id} and checks the caller owns its event.
func (s *Server) loadOwnedCampaign(w http.ResponseWriter, r *http.Request) (store.Campaign, store.Event, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The campaign id must be a UUID.")
		return store.Campaign{}, store.Event{}, false
	}

	campaign, err := s.campaigns.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No campaign with this id.")
		return store.Campaign{}, store.Event{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.Campaign{}, store.Event{}, false
	}

	event, err := s.events.GetByID(r.Context(), campaign.EventID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.Campaign{}, store.Event{}, false
	}

	claims, _ := claimsFromContext(r.Context())
	if event.OrganizerID != mustUserID(r.Context()) &&
		!(claims != nil && claims.HasRole(store.RolePlatformAdmin)) {
		httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
			"Only the organizer of this event can manage its campaigns.")
		return store.Campaign{}, store.Event{}, false
	}

	return campaign, event, true
}

// promoErrorCode maps a promo failure to the code a client switches on.
func promoErrorCode(err error) (int, string, string) {
	switch {
	case errors.Is(err, store.ErrPromoNotFound):
		return http.StatusNotFound, CodePromoNotFound, "That promo code was not recognised."
	case errors.Is(err, store.ErrPromoWrongEvent):
		return http.StatusConflict, CodePromoWrongEvent, "That promo code is for a different event."
	case errors.Is(err, store.ErrPromoNotActive):
		return http.StatusConflict, CodePromoNotActive, "That promo code is not active."
	case errors.Is(err, store.ErrPromoNotStarted):
		return http.StatusConflict, CodePromoNotStarted, "That promo code is not valid yet."
	case errors.Is(err, store.ErrPromoExpired):
		return http.StatusConflict, CodePromoExpired, "That promo code has expired."
	case errors.Is(err, store.ErrPromoExhausted):
		return http.StatusConflict, CodePromoExhausted,
			"That promo code has been fully redeemed."
	case errors.Is(err, store.ErrPromoNotApplicable):
		return http.StatusConflict, CodePromoNotApplicable,
			"That promo code does not apply to the tickets you selected."
	default:
		return 0, "", ""
	}
}
