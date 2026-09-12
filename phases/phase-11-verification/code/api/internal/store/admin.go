package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminUserRow is one account in the platform admin's search results.
type AdminUserRow struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	Status      string     `json:"status"`
	Verified    bool       `json:"email_verified"`
	Roles       []string   `json:"roles"`
	EventCount  int        `json:"event_count"`
	OrderCount  int        `json:"order_count"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// AdminEventRow is one event in the results.
type AdminEventRow struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Slug           string    `json:"slug"`
	Status         string    `json:"status"`
	Lifecycle      string    `json:"lifecycle"`
	OrganizerEmail string    `json:"organizer_email"`
	OrganizerName  string    `json:"organizer_name"`
	StartsAt       time.Time `json:"starts_at"`
	TicketsSold    int       `json:"tickets_sold"`
	RevenueKZT     string    `json:"revenue_kzt"`
	// ActivationStatus is the paid-sales activation state, which SRS 4.12
	// requires an admin to be able to inspect.
	ActivationStatus string `json:"activation_status"`
}

// AdminOrderRow is one order in the results.
type AdminOrderRow struct {
	ID          uuid.UUID `json:"id"`
	OrderNumber string    `json:"order_number"`
	BuyerName   string    `json:"buyer_name"`
	BuyerEmail  string    `json:"buyer_email"`
	EventTitle  string    `json:"event_title"`
	EventID     uuid.UUID `json:"event_id"`
	Status      string    `json:"status"`
	TotalKZT    string    `json:"total_kzt"`
	RefundedKZT string    `json:"refunded_kzt"`
	TicketCount int       `json:"ticket_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// AdminPaymentRow is one payment or refund in the results. SRS 4.12 asks
// admins to monitor refunds and payment failures, so failed payments are as
// interesting here as successful ones.
type AdminPaymentRow struct {
	ID          uuid.UUID `json:"id"`
	Purpose     string    `json:"purpose"`
	Status      string    `json:"status"`
	AmountKZT   string    `json:"amount_kzt"`
	OrderNumber *string   `json:"order_number,omitempty"`
	EventTitle  *string   `json:"event_title,omitempty"`
	IsSimulated bool      `json:"is_simulated"`
	CreatedAt   time.Time `json:"created_at"`
}

// AdminSearchResult is everything one query found.
type AdminSearchResult struct {
	Query    string            `json:"query"`
	Users    []AdminUserRow    `json:"users"`
	Events   []AdminEventRow   `json:"events"`
	Orders   []AdminOrderRow   `json:"orders"`
	Payments []AdminPaymentRow `json:"payments"`
}

// AdminStore backs the platform administration portal (SRS 2.1, 4.12).
type AdminStore struct {
	pool *pgxpool.Pool
}

// NewAdminStore builds an AdminStore.
func NewAdminStore(pool *pgxpool.Pool) *AdminStore { return &AdminStore{pool: pool} }

// adminSearchLimit caps each section of the results. An admin looking for one
// account does not need a thousand rows, and an empty query should not try to
// render the whole platform.
const adminSearchLimit = 25

// Search looks for a term across users, events, orders and payments (SRS 4.12).
//
// An empty query is legitimate and returns the most recent rows of each kind:
// the portal opens on something useful rather than four empty tables.
func (s *AdminStore) Search(ctx context.Context, query string) (AdminSearchResult, error) {
	result := AdminSearchResult{
		Query:    query,
		Users:    []AdminUserRow{},
		Events:   []AdminEventRow{},
		Orders:   []AdminOrderRow{},
		Payments: []AdminPaymentRow{},
	}

	var err error
	if result.Users, err = s.searchUsers(ctx, query); err != nil {
		return AdminSearchResult{}, err
	}
	if result.Events, err = s.searchEvents(ctx, query); err != nil {
		return AdminSearchResult{}, err
	}
	if result.Orders, err = s.searchOrders(ctx, query); err != nil {
		return AdminSearchResult{}, err
	}
	if result.Payments, err = s.searchPayments(ctx, query); err != nil {
		return AdminSearchResult{}, err
	}
	return result, nil
}

// The searches below all follow the same shape: an empty query matches
// everything, and a non-empty one is matched case-insensitively against the
// handful of fields an admin would actually type. `$1 = ''` short-circuits the
// ILIKE so the empty case does not scan with a '%%' pattern.

func (s *AdminStore) searchUsers(ctx context.Context, q string) ([]AdminUserRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.email::text, u.full_name, u.status::text,
		       u.email_verified_at IS NOT NULL,
		       COALESCE(array_agg(DISTINCT r.role::text) FILTER (WHERE r.role IS NOT NULL), '{}'),
		       (SELECT count(*) FROM events e WHERE e.organizer_id = u.id),
		       (SELECT count(*) FROM orders o WHERE o.buyer_user_id = u.id),
		       u.last_login_at, u.created_at
		  FROM users u
		  LEFT JOIN user_roles r ON r.user_id = u.id
		 WHERE $1 = '' OR u.email::text ILIKE '%' || $1 || '%'
		                OR u.full_name ILIKE '%' || $1 || '%'
		 GROUP BY u.id
		 ORDER BY u.created_at DESC
		 LIMIT $2`, q, adminSearchLimit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []AdminUserRow{}
	for rows.Next() {
		var u AdminUserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Status, &u.Verified,
			&u.Roles, &u.EventCount, &u.OrderCount, &u.LastLoginAt, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *AdminStore) searchEvents(ctx context.Context, q string) ([]AdminEventRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.id, e.title, e.slug::text, e.status::text, u.email::text, u.full_name,
		       e.starts_at, e.ends_at, e.cancelled_at,
		       (SELECT count(*) FROM tickets t
		         WHERE t.event_id = e.id AND t.status IN ('valid','checked_in')),
		       (SELECT COALESCE(sum(o.total_kzt), 0)::numeric(14,2)::text FROM orders o
		         WHERE o.event_id = e.id
		           AND o.status IN ('paid','completed','refunded','partially_refunded')),
		       COALESCE((SELECT a.status::text FROM paid_sales_activations a
		                  WHERE a.event_id = e.id), 'not_started')
		  FROM events e
		  JOIN users u ON u.id = e.organizer_id
		 WHERE $1 = '' OR e.title ILIKE '%' || $1 || '%'
		                OR e.slug::text ILIKE '%' || $1 || '%'
		                OR u.email::text ILIKE '%' || $1 || '%'
		 ORDER BY e.created_at DESC
		 LIMIT $2`, q, adminSearchLimit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	now := time.Now()
	out := []AdminEventRow{}
	for rows.Next() {
		var (
			e           AdminEventRow
			endsAt      time.Time
			cancelledAt *time.Time
		)
		if err := rows.Scan(&e.ID, &e.Title, &e.Slug, &e.Status, &e.OrganizerEmail,
			&e.OrganizerName, &e.StartsAt, &endsAt, &cancelledAt,
			&e.TicketsSold, &e.RevenueKZT, &e.ActivationStatus); err != nil {
			return nil, err
		}
		// Reuse the one definition of lifecycle rather than writing a second.
		e.Lifecycle = Lifecycle(Event{
			Status: e.Status, StartsAt: e.StartsAt, EndsAt: endsAt, CancelledAt: cancelledAt,
		}, now)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *AdminStore) searchOrders(ctx context.Context, q string) ([]AdminOrderRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.order_number, o.buyer_name, o.buyer_email::text,
		       e.title, e.id, o.status::text, o.total_kzt::text, o.refunded_kzt::text,
		       (SELECT count(*) FROM tickets t WHERE t.order_id = o.id),
		       o.created_at
		  FROM orders o
		  JOIN events e ON e.id = o.event_id
		 WHERE $1 = '' OR o.order_number ILIKE '%' || $1 || '%'
		                OR o.buyer_email::text ILIKE '%' || $1 || '%'
		                OR o.buyer_name ILIKE '%' || $1 || '%'
		 ORDER BY o.created_at DESC
		 LIMIT $2`, q, adminSearchLimit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []AdminOrderRow{}
	for rows.Next() {
		var o AdminOrderRow
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.BuyerName, &o.BuyerEmail,
			&o.EventTitle, &o.EventID, &o.Status, &o.TotalKZT, &o.RefundedKZT,
			&o.TicketCount, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *AdminStore) searchPayments(ctx context.Context, q string) ([]AdminPaymentRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.purpose::text, p.status::text, p.amount_kzt::text,
		       o.order_number, e.title, p.is_simulated, p.created_at
		  FROM payments p
		  LEFT JOIN orders o ON o.id = p.order_id
		  LEFT JOIN events e ON e.id = COALESCE(p.event_id, o.event_id)
		 WHERE $1 = '' OR o.order_number ILIKE '%' || $1 || '%'
		                OR e.title ILIKE '%' || $1 || '%'
		                OR p.status::text ILIKE '%' || $1 || '%'
		 ORDER BY p.created_at DESC
		 LIMIT $2`, q, adminSearchLimit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []AdminPaymentRow{}
	for rows.Next() {
		var p AdminPaymentRow
		if err := rows.Scan(&p.ID, &p.Purpose, &p.Status, &p.AmountKZT,
			&p.OrderNumber, &p.EventTitle, &p.IsSimulated, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PlatformStats is the summary the portal opens with.
type PlatformStats struct {
	Users            int    `json:"users"`
	Organizers       int    `json:"organizers"`
	Events           int    `json:"events"`
	PublishedEvents  int    `json:"published_events"`
	SuspendedEvents  int    `json:"suspended_events"`
	Orders           int    `json:"orders"`
	TicketsSold      int    `json:"tickets_sold"`
	GrossRevenueKZT  string `json:"gross_revenue_kzt"`
	RefundedKZT      string `json:"refunded_kzt"`
	FailedPayments   int    `json:"failed_payments"`
	OpenSupportCases int    `json:"open_support_cases"`
}

// Stats counts the platform, for the admin dashboard header.
func (s *AdminStore) Stats(ctx context.Context) (PlatformStats, error) {
	var p PlatformStats
	err := s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM users),
		       (SELECT count(DISTINCT organizer_id) FROM events),
		       (SELECT count(*) FROM events),
		       (SELECT count(*) FROM events WHERE status = 'published'),
		       (SELECT count(*) FROM events WHERE status = 'suspended'),
		       (SELECT count(*) FROM orders),
		       (SELECT count(*) FROM tickets WHERE status IN ('valid','checked_in')),
		       (SELECT COALESCE(sum(total_kzt), 0)::numeric(14,2)::text FROM orders
		         WHERE status IN ('paid','completed','refunded','partially_refunded')),
		       (SELECT COALESCE(sum(refunded_kzt), 0)::numeric(14,2)::text FROM orders),
		       (SELECT count(*) FROM payments WHERE status = 'failed'),
		       (SELECT count(*) FROM support_cases WHERE status <> 'resolved')`,
	).Scan(&p.Users, &p.Organizers, &p.Events, &p.PublishedEvents, &p.SuspendedEvents,
		&p.Orders, &p.TicketsSold, &p.GrossRevenueKZT, &p.RefundedKZT,
		&p.FailedPayments, &p.OpenSupportCases)
	if err != nil {
		return PlatformStats{}, mapError(err)
	}
	return p, nil
}

// ReportRow is one line of the operational report (SRS 4.12, "export basic
// operational reports").
type ReportRow struct {
	EventID        uuid.UUID
	Title          string
	Status         string
	Lifecycle      string
	OrganizerEmail string
	StartsAt       time.Time
	Timezone       string
	Capacity       *int
	TicketsSold    int
	CheckedIn      int
	Orders         int
	GrossKZT       string
	DiscountsKZT   string
	RefundedKZT    string
	NetKZT         string
	Activation     string
}

// Report returns one row per event, ordered oldest first so a spreadsheet
// reads chronologically.
func (s *AdminStore) Report(ctx context.Context) ([]ReportRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.id, e.title, e.status::text, e.starts_at, e.ends_at, e.cancelled_at,
		       u.email::text, e.timezone, e.capacity,
		       (SELECT count(*) FROM tickets t
		         WHERE t.event_id = e.id AND t.status IN ('valid','checked_in')),
		       (SELECT count(*) FROM tickets t
		         WHERE t.event_id = e.id AND t.status = 'checked_in'),
		       (SELECT count(*) FROM orders o WHERE o.event_id = e.id
		           AND o.status IN ('paid','completed','refunded','partially_refunded')),
		       (SELECT COALESCE(sum(o.total_kzt), 0)::numeric(14,2)::text FROM orders o
		         WHERE o.event_id = e.id
		           AND o.status IN ('paid','completed','refunded','partially_refunded')),
		       (SELECT COALESCE(sum(o.discount_kzt), 0)::numeric(14,2)::text FROM orders o
		         WHERE o.event_id = e.id
		           AND o.status IN ('paid','completed','refunded','partially_refunded')),
		       (SELECT COALESCE(sum(o.refunded_kzt), 0)::numeric(14,2)::text FROM orders o
		         WHERE o.event_id = e.id),
		       (SELECT (COALESCE(sum(o.total_kzt), 0) - COALESCE(sum(o.refunded_kzt), 0))
		                 ::numeric(14,2)::text FROM orders o
		         WHERE o.event_id = e.id
		           AND o.status IN ('paid','completed','refunded','partially_refunded')),
		       COALESCE((SELECT a.status::text FROM paid_sales_activations a
		                  WHERE a.event_id = e.id), 'not_started')
		  FROM events e
		  JOIN users u ON u.id = e.organizer_id
		 ORDER BY e.starts_at`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	now := time.Now()
	out := []ReportRow{}
	for rows.Next() {
		var (
			r           ReportRow
			endsAt      time.Time
			cancelledAt *time.Time
		)
		if err := rows.Scan(&r.EventID, &r.Title, &r.Status, &r.StartsAt, &endsAt,
			&cancelledAt, &r.OrganizerEmail, &r.Timezone, &r.Capacity,
			&r.TicketsSold, &r.CheckedIn, &r.Orders, &r.GrossKZT, &r.DiscountsKZT,
			&r.RefundedKZT, &r.NetKZT, &r.Activation); err != nil {
			return nil, err
		}
		r.Lifecycle = Lifecycle(Event{
			Status: r.Status, StartsAt: r.StartsAt, EndsAt: endsAt, CancelledAt: cancelledAt,
		}, now)
		out = append(out, r)
	}
	return out, rows.Err()
}
