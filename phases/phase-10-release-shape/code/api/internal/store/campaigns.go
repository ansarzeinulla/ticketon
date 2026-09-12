package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Errors returned when a promo code cannot be applied. Each is distinct because
// "invalid code" tells an attendee nothing about what to do next.
var (
	ErrPromoCodeTaken     = errors.New("that promo code is already in use")
	ErrPromoNotFound      = errors.New("no campaign matches that code")
	ErrPromoNotActive     = errors.New("this code is not currently active")
	ErrPromoNotStarted    = errors.New("this code is not valid yet")
	ErrPromoExpired       = errors.New("this code has expired")
	ErrPromoExhausted     = errors.New("this code has reached its redemption limit")
	ErrPromoNotApplicable = errors.New("this code does not apply to the selected tickets")
	ErrPromoWrongEvent    = errors.New("this code is for a different event")
)

// Discount types, matching the discount_type enum.
const (
	DiscountPercentage = "percentage"
	DiscountFixed      = "fixed_amount"
)

// Campaign statuses, matching the campaign_status enum.
const (
	CampaignDraft     = "draft"
	CampaignActive    = "active"
	CampaignDisabled  = "disabled"
	CampaignExhausted = "exhausted"
	CampaignExpired   = "expired"
)

// Campaign is a promotional campaign with its code and QR token.
type Campaign struct {
	ID              uuid.UUID  `json:"id"`
	EventID         uuid.UUID  `json:"event_id"`
	Name            string     `json:"name"`
	DiscountType    string     `json:"discount_type"`
	DiscountValue   string     `json:"discount_value"`
	StartsAt        *time.Time `json:"starts_at,omitempty"`
	EndsAt          *time.Time `json:"ends_at,omitempty"`
	MaxRedemptions  *int       `json:"max_redemptions,omitempty"`
	RedemptionCount int        `json:"redemption_count"`
	Status          string     `json:"status"`
	// QRToken is opaque and always prefixed CMP_ (SRS 4.14). It never encodes
	// the discount itself: the server is the only thing that decides value.
	QRToken      string    `json:"qr_token"`
	Code         string    `json:"code"`
	CodeID       uuid.UUID `json:"promo_code_id"`
	CodeIsActive bool      `json:"code_is_active"`
	CreatedAt    time.Time `json:"created_at"`

	// TicketTypeIDs restricts the campaign; empty means it applies to all.
	TicketTypeIDs []uuid.UUID `json:"ticket_type_ids"`

	// Reporting figures (SRS 4.14: "record exact redemptions, orders, tickets
	// sold, gross revenue, discount amount, and net revenue for each campaign").
	OrdersCount   int    `json:"orders_count"`
	TicketsSold   int    `json:"tickets_sold"`
	GrossRevenue  string `json:"gross_revenue_kzt"`
	DiscountGiven string `json:"discount_given_kzt"`
}

// Remaining redemptions, or -1 when the campaign is unlimited.
func (c Campaign) Remaining() int {
	if c.MaxRedemptions == nil {
		return -1
	}
	remaining := *c.MaxRedemptions - c.RedemptionCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CampaignStore manages promotional campaigns.
type CampaignStore struct {
	pool *pgxpool.Pool
}

// NewCampaignStore builds a CampaignStore.
func NewCampaignStore(pool *pgxpool.Pool) *CampaignStore { return &CampaignStore{pool: pool} }

// NewCampaignToken returns an opaque CMP_ token for a campaign QR.
//
// The CMP_ prefix is enforced by campaigns_qr_token_prefix_chk and is what
// keeps a promotional QR disjoint from an admission one, so a discount code can
// never be presented at the gate (SRS 4.14).
func NewCampaignToken() string {
	return CampaignTokenPrefix + uuid.NewString()
}

const campaignColumns = `c.id, c.event_id, c.name, c.discount_type::text, c.discount_value::text,
	c.starts_at, c.ends_at, c.max_redemptions, c.redemption_count, c.status::text,
	c.qr_token, c.created_at, p.code::text, p.id, p.is_active`

func scanCampaign(row pgx.Row) (Campaign, error) {
	var c Campaign
	err := row.Scan(&c.ID, &c.EventID, &c.Name, &c.DiscountType, &c.DiscountValue,
		&c.StartsAt, &c.EndsAt, &c.MaxRedemptions, &c.RedemptionCount, &c.Status,
		&c.QRToken, &c.CreatedAt, &c.Code, &c.CodeID, &c.CodeIsActive)
	c.TicketTypeIDs = []uuid.UUID{}
	return c, err
}

// CreateCampaignParams describes a new campaign.
type CreateCampaignParams struct {
	EventID        uuid.UUID
	Name           string
	Code           string
	DiscountType   string
	DiscountValue  string
	StartsAt       *time.Time
	EndsAt         *time.Time
	MaxRedemptions *int
	Status         string
	TicketTypeIDs  []uuid.UUID
	CreatedBy      uuid.UUID
}

// Create inserts a campaign, its promo code and any ticket-type restriction in
// one transaction, so a half-built campaign can never be reachable.
func (s *CampaignStore) Create(ctx context.Context, p CreateCampaignParams) (Campaign, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var campaignID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO campaigns (event_id, name, discount_type, discount_value, starts_at,
		                       ends_at, max_redemptions, status, qr_token, created_by)
		VALUES ($1, $2, $3::discount_type, $4::numeric, $5, $6, $7, $8::campaign_status, $9, $10)
		RETURNING id`,
		p.EventID, p.Name, p.DiscountType, p.DiscountValue, p.StartsAt, p.EndsAt,
		p.MaxRedemptions, p.Status, NewCampaignToken(), p.CreatedBy).Scan(&campaignID)
	if err != nil {
		return Campaign{}, mapError(err)
	}

	var codeID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO promo_codes (campaign_id, code) VALUES ($1, $2) RETURNING id`,
		campaignID, p.Code).Scan(&codeID)
	if isUniqueViolation(err, "promo_codes_code_key") {
		return Campaign{}, ErrPromoCodeTaken
	}
	if err != nil {
		return Campaign{}, mapError(err)
	}

	for _, ticketTypeID := range p.TicketTypeIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO campaign_ticket_types (campaign_id, ticket_type_id) VALUES ($1, $2)`,
			campaignID, ticketTypeID); err != nil {
			return Campaign{}, mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Campaign{}, mapError(err)
	}
	return s.GetByID(ctx, campaignID)
}

// GetByID returns one campaign with its restriction list and reporting figures.
func (s *CampaignStore) GetByID(ctx context.Context, id uuid.UUID) (Campaign, error) {
	c, err := scanCampaign(s.pool.QueryRow(ctx, `
		SELECT `+campaignColumns+`
		  FROM campaigns c JOIN promo_codes p ON p.campaign_id = c.id
		 WHERE c.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	}
	if err != nil {
		return Campaign{}, mapError(err)
	}
	if err := s.hydrate(ctx, &c); err != nil {
		return Campaign{}, err
	}
	return c, nil
}

// ListForEvent returns an event's campaigns, newest first.
func (s *CampaignStore) ListForEvent(ctx context.Context, eventID uuid.UUID) ([]Campaign, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+campaignColumns+`
		  FROM campaigns c JOIN promo_codes p ON p.campaign_id = c.id
		 WHERE c.event_id = $1
		 ORDER BY c.created_at DESC`, eventID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	campaigns := []Campaign{}
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}

	for i := range campaigns {
		if err := s.hydrate(ctx, &campaigns[i]); err != nil {
			return nil, err
		}
	}
	return campaigns, nil
}

// hydrate loads the ticket-type restriction and the campaign's sales figures.
func (s *CampaignStore) hydrate(ctx context.Context, c *Campaign) error {
	rows, err := s.pool.Query(ctx,
		`SELECT ticket_type_id FROM campaign_ticket_types WHERE campaign_id = $1`, c.ID)
	if err != nil {
		return mapError(err)
	}
	c.TicketTypeIDs = []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		c.TicketTypeIDs = append(c.TicketTypeIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return mapError(err)
	}

	// Figures come from the authoritative order records, not a counter.
	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(count(DISTINCT o.id), 0),
		       COALESCE(count(t.id), 0),
		       COALESCE(sum(DISTINCT o.total_kzt), 0)::numeric(14,2)::text,
		       COALESCE(sum(DISTINCT o.discount_kzt), 0)::numeric(14,2)::text
		  FROM orders o
		  LEFT JOIN tickets t ON t.order_id = o.id
		 WHERE o.campaign_id = $1
		   AND o.status IN ('paid', 'completed', 'refunded', 'partially_refunded')`,
		c.ID).Scan(&c.OrdersCount, &c.TicketsSold, &c.GrossRevenue, &c.DiscountGiven)
	return mapError(err)
}

// SetStatus enables or disables a campaign.
func (s *CampaignStore) SetStatus(ctx context.Context, id uuid.UUID, status string) (Campaign, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE campaigns SET status = $2::campaign_status WHERE id = $1`, id, status)
	if err != nil {
		return Campaign{}, mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return Campaign{}, ErrNotFound
	}
	return s.GetByID(ctx, id)
}

// Delete removes a campaign that has never been redeemed.
func (s *CampaignStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM campaigns WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// HasRedemptions reports whether a campaign has ever been used.
func (s *CampaignStore) HasRedemptions(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM promo_redemptions WHERE campaign_id = $1)`, id).Scan(&exists)
	return exists, mapError(err)
}

// ResolveParams identifies a campaign by typed code or scanned QR token.
type ResolveParams struct {
	EventID       uuid.UUID
	Code          string
	CampaignToken string
}

// Resolve finds the campaign behind a typed code or a scanned campaign token.
//
// It deliberately does not check validity: callers decide whether they want a
// preview (which explains why a code cannot be used) or a purchase.
func (s *CampaignStore) Resolve(ctx context.Context, p ResolveParams) (Campaign, error) {
	var (
		row   pgx.Row
		query = `SELECT ` + campaignColumns + `
		           FROM campaigns c JOIN promo_codes p ON p.campaign_id = c.id
		          WHERE `
	)

	switch {
	case strings.TrimSpace(p.CampaignToken) != "":
		row = s.pool.QueryRow(ctx, query+`c.qr_token = $1`, strings.TrimSpace(p.CampaignToken))
	case strings.TrimSpace(p.Code) != "":
		row = s.pool.QueryRow(ctx, query+`p.code = $1`, strings.TrimSpace(p.Code))
	default:
		return Campaign{}, ErrPromoNotFound
	}

	c, err := scanCampaign(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrPromoNotFound
	}
	if err != nil {
		return Campaign{}, mapError(err)
	}

	if p.EventID != uuid.Nil && c.EventID != p.EventID {
		return Campaign{}, ErrPromoWrongEvent
	}

	if err := s.hydrate(ctx, &c); err != nil {
		return Campaign{}, err
	}
	return c, nil
}

// CheckUsable reports why a campaign cannot be redeemed right now, or nil.
func CheckUsable(c Campaign, now time.Time) error {
	if !c.CodeIsActive {
		return ErrPromoNotActive
	}

	switch c.Status {
	case CampaignActive:
		// usable, subject to the checks below
	case CampaignExhausted:
		return ErrPromoExhausted
	case CampaignExpired:
		return ErrPromoExpired
	default:
		return ErrPromoNotActive
	}

	if c.StartsAt != nil && now.Before(*c.StartsAt) {
		return ErrPromoNotStarted
	}
	if c.EndsAt != nil && !now.Before(*c.EndsAt) {
		return ErrPromoExpired
	}
	if c.MaxRedemptions != nil && c.RedemptionCount >= *c.MaxRedemptions {
		return ErrPromoExhausted
	}
	return nil
}

// AppliesTo reports whether the campaign covers a ticket type. A campaign with
// no restriction applies to everything.
func (c Campaign) AppliesTo(ticketTypeID uuid.UUID) bool {
	if len(c.TicketTypeIDs) == 0 {
		return true
	}
	for _, id := range c.TicketTypeIDs {
		if id == ticketTypeID {
			return true
		}
	}
	return false
}

// CampaignLink is the trackable HTTPS URL a campaign QR encodes.
//
// SRS 4.14 requires the link to carry an opaque token rather than a discount
// the client could tamper with, so only qr_token travels in it - the server
// looks up what that token is worth.
func CampaignLink(webBaseURL, eventSlug, qrToken string) string {
	base := strings.TrimRight(webBaseURL, "/")
	return fmt.Sprintf("%s/events/%s?c=%s", base, eventSlug, qrToken)
}
