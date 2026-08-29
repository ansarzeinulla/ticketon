package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventAnalytics is the organizer dashboard's figures for one event.
//
// Every number here is computed from the authoritative order, ticket, campaign
// and check-in rows (SRS 4.15), never from a running counter that could drift.
// Money stays a decimal string all the way out.
type EventAnalytics struct {
	EventID uuid.UUID `json:"event_id"`

	// Capacity and sales.
	TotalCapacity    int     `json:"total_capacity"`
	TicketsSold      int     `json:"tickets_sold"`
	TicketsRemaining int     `json:"tickets_remaining"`
	TicketsRefunded  int     `json:"tickets_refunded"`
	PercentageSold   float64 `json:"percentage_sold"`

	// Money, in KZT.
	GrossRevenueKZT string `json:"gross_revenue_kzt"`
	DiscountsKZT    string `json:"discounts_kzt"`
	RefundsKZT      string `json:"refunds_kzt"`
	NetRevenueKZT   string `json:"net_revenue_kzt"`
	OrdersCount     int    `json:"orders_count"`

	// Attendance.
	CheckedIn         int     `json:"checked_in"`
	Absent            int     `json:"absent"`
	CheckInPercentage float64 `json:"check_in_percentage"`

	ByTicketType  []TicketTypeSales `json:"by_ticket_type"`
	ByCampaign    []CampaignSales   `json:"by_campaign"`
	SalesOverTime []SalesPoint      `json:"sales_over_time"`
}

// TicketTypeSales breaks the figures down by ticket type (SRS 4.15).
type TicketTypeSales struct {
	TicketTypeID  uuid.UUID `json:"ticket_type_id"`
	Name          string    `json:"name"`
	PriceKZT      string    `json:"price_kzt"`
	QuantityTotal int       `json:"quantity_total"`
	Sold          int       `json:"sold"`
	Remaining     int       `json:"remaining"`
	CheckedIn     int       `json:"checked_in"`
	RevenueKZT    string    `json:"revenue_kzt"`
}

// CampaignSales attributes sales to a promotional campaign.
type CampaignSales struct {
	CampaignID  uuid.UUID `json:"campaign_id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Redemptions int       `json:"redemptions"`
	TicketsSold int       `json:"tickets_sold"`
	RevenueKZT  string    `json:"revenue_kzt"`
	DiscountKZT string    `json:"discount_kzt"`
}

// SalesPoint is one day of sales, for the sales-over-time chart. Derived from
// order timestamps, as SRS 4.15 requires.
type SalesPoint struct {
	Day        string `json:"day"`
	Orders     int    `json:"orders"`
	Tickets    int    `json:"tickets"`
	RevenueKZT string `json:"revenue_kzt"`
}

// AnalyticsFilter narrows the figures (SRS 4.15: event, date range, ticket type).
type AnalyticsFilter struct {
	From         *time.Time
	To           *time.Time
	TicketTypeID *uuid.UUID
}

// AnalyticsStore computes organizer analytics.
type AnalyticsStore struct {
	pool *pgxpool.Pool
}

// NewAnalyticsStore builds an AnalyticsStore.
func NewAnalyticsStore(pool *pgxpool.Pool) *AnalyticsStore { return &AnalyticsStore{pool: pool} }

// countedOrderStatuses are the orders that represent real money taken. A
// pending or failed basket is not a sale.
const countedOrderStatuses = `('paid', 'completed', 'refunded', 'partially_refunded')`

// ForEvent computes the dashboard figures for one event.
func (s *AnalyticsStore) ForEvent(
	ctx context.Context, eventID uuid.UUID, f AnalyticsFilter,
) (EventAnalytics, error) {
	a := EventAnalytics{
		EventID:       eventID,
		ByTicketType:  []TicketTypeSales{},
		ByCampaign:    []CampaignSales{},
		SalesOverTime: []SalesPoint{},
	}

	// Nil-safe bounds: the filter is optional and the SQL below treats a NULL
	// bound as "no limit", so the same statement serves a filtered and an
	// unfiltered request.
	var from, to any
	if f.From != nil {
		from = *f.From
	}
	if f.To != nil {
		to = *f.To
	}
	var ticketTypeID any
	if f.TicketTypeID != nil {
		ticketTypeID = *f.TicketTypeID
	}

	// --- capacity and attendance --------------------------------------------
	err := s.pool.QueryRow(ctx, `
		WITH types AS (
			SELECT id, quantity_total, quantity_sold, quantity_reserved
			  FROM ticket_types
			 WHERE event_id = $1
			   AND ($2::uuid IS NULL OR id = $2)
		),
		issued AS (
			SELECT t.status
			  FROM tickets t
			 WHERE t.event_id = $1
			   AND ($2::uuid IS NULL OR t.ticket_type_id = $2)
		)
		SELECT
			COALESCE((SELECT sum(quantity_total) FROM types), 0),
			COALESCE((SELECT sum(quantity_reserved) FROM types), 0),
			(SELECT count(*) FROM issued WHERE status IN ('valid', 'checked_in')),
			(SELECT count(*) FROM issued WHERE status = 'refunded'),
			(SELECT count(*) FROM issued WHERE status = 'checked_in')`,
		eventID, ticketTypeID,
	).Scan(&a.TotalCapacity, new(int), &a.TicketsSold, &a.TicketsRefunded, &a.CheckedIn)
	if err != nil {
		return EventAnalytics{}, mapError(err)
	}

	a.TicketsRemaining = a.TotalCapacity - a.TicketsSold
	if a.TicketsRemaining < 0 {
		a.TicketsRemaining = 0
	}
	if a.TotalCapacity > 0 {
		a.PercentageSold = round1(float64(a.TicketsSold) / float64(a.TotalCapacity) * 100)
	}

	// Absent holders are people who bought and did not turn up (SRS 4.15).
	a.Absent = a.TicketsSold - a.CheckedIn
	if a.Absent < 0 {
		a.Absent = 0
	}
	if a.TicketsSold > 0 {
		a.CheckInPercentage = round1(float64(a.CheckedIn) / float64(a.TicketsSold) * 100)
	}

	// --- money ---------------------------------------------------------------
	// Summed in SQL so numeric(14,2) never becomes a float.
	err = s.pool.QueryRow(ctx, `
		SELECT count(*),
		       COALESCE(sum(total_kzt), 0)::numeric(14,2)::text,
		       COALESCE(sum(discount_kzt), 0)::numeric(14,2)::text,
		       COALESCE(sum(refunded_kzt), 0)::numeric(14,2)::text,
		       COALESCE(sum(total_kzt) - sum(refunded_kzt), 0)::numeric(14,2)::text
		  FROM orders
		 WHERE event_id = $1
		   AND status::text IN `+countedOrderStatuses+`
		   AND ($2::timestamptz IS NULL OR placed_at >= $2)
		   AND ($3::timestamptz IS NULL OR placed_at < $3)`,
		eventID, from, to,
	).Scan(&a.OrdersCount, &a.GrossRevenueKZT, &a.DiscountsKZT, &a.RefundsKZT, &a.NetRevenueKZT)
	if err != nil {
		return EventAnalytics{}, mapError(err)
	}

	if a.ByTicketType, err = s.byTicketType(ctx, eventID, from, to, ticketTypeID); err != nil {
		return EventAnalytics{}, err
	}
	if a.ByCampaign, err = s.byCampaign(ctx, eventID, from, to); err != nil {
		return EventAnalytics{}, err
	}
	if a.SalesOverTime, err = s.salesOverTime(ctx, eventID, from, to); err != nil {
		return EventAnalytics{}, err
	}

	return a, nil
}

func (s *AnalyticsStore) byTicketType(
	ctx context.Context, eventID uuid.UUID, from, to, ticketTypeID any,
) ([]TicketTypeSales, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT tt.id, tt.name, tt.price_kzt::text, tt.quantity_total,
		       COALESCE(live.sold, 0),
		       COALESCE(live.checked_in, 0),
		       COALESCE(money.revenue, 0)::numeric(14,2)::text
		  FROM ticket_types tt
		  LEFT JOIN LATERAL (
		      SELECT count(*) FILTER (WHERE t.status IN ('valid', 'checked_in')) AS sold,
		             count(*) FILTER (WHERE t.status = 'checked_in')             AS checked_in
		        FROM tickets t WHERE t.ticket_type_id = tt.id
		  ) live ON true
		  LEFT JOIN LATERAL (
		      SELECT sum(oi.line_total_kzt) AS revenue
		        FROM order_items oi
		        JOIN orders o ON o.id = oi.order_id
		       WHERE oi.ticket_type_id = tt.id
		         AND o.status::text IN `+countedOrderStatuses+`
		         AND ($3::timestamptz IS NULL OR o.placed_at >= $3)
		         AND ($4::timestamptz IS NULL OR o.placed_at < $4)
		  ) money ON true
		 WHERE tt.event_id = $1
		   AND ($2::uuid IS NULL OR tt.id = $2)
		 ORDER BY tt.display_order, tt.created_at`,
		eventID, ticketTypeID, from, to)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	sales := []TicketTypeSales{}
	for rows.Next() {
		var t TicketTypeSales
		if err := rows.Scan(&t.TicketTypeID, &t.Name, &t.PriceKZT, &t.QuantityTotal,
			&t.Sold, &t.CheckedIn, &t.RevenueKZT); err != nil {
			return nil, err
		}
		t.Remaining = t.QuantityTotal - t.Sold
		if t.Remaining < 0 {
			t.Remaining = 0
		}
		sales = append(sales, t)
	}
	return sales, rows.Err()
}

func (s *AnalyticsStore) byCampaign(
	ctx context.Context, eventID uuid.UUID, from, to any,
) ([]CampaignSales, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.name, p.code::text,
		       count(DISTINCT o.id),
		       count(t.id),
		       COALESCE(sum(DISTINCT o.total_kzt), 0)::numeric(14,2)::text,
		       COALESCE(sum(DISTINCT o.discount_kzt), 0)::numeric(14,2)::text
		  FROM campaigns c
		  JOIN promo_codes p ON p.campaign_id = c.id
		  LEFT JOIN orders o ON o.campaign_id = c.id
		       AND o.status::text IN `+countedOrderStatuses+`
		       AND ($2::timestamptz IS NULL OR o.placed_at >= $2)
		       AND ($3::timestamptz IS NULL OR o.placed_at < $3)
		  LEFT JOIN tickets t ON t.order_id = o.id
		 WHERE c.event_id = $1
		 GROUP BY c.id, c.name, p.code
		 ORDER BY c.created_at DESC`,
		eventID, from, to)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	sales := []CampaignSales{}
	for rows.Next() {
		var c CampaignSales
		if err := rows.Scan(&c.CampaignID, &c.Name, &c.Code, &c.Redemptions,
			&c.TicketsSold, &c.RevenueKZT, &c.DiscountKZT); err != nil {
			return nil, err
		}
		sales = append(sales, c)
	}
	return sales, rows.Err()
}

// salesOverTime buckets orders by day (SRS 4.15: "sales over time using order
// timestamps").
func (s *AnalyticsStore) salesOverTime(
	ctx context.Context, eventID uuid.UUID, from, to any,
) ([]SalesPoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT to_char(date_trunc('day', o.placed_at), 'YYYY-MM-DD') AS day,
		       count(DISTINCT o.id),
		       COALESCE(sum(items.quantity), 0),
		       COALESCE(sum(DISTINCT o.total_kzt), 0)::numeric(14,2)::text
		  FROM orders o
		  LEFT JOIN LATERAL (
		      SELECT sum(oi.quantity) AS quantity
		        FROM order_items oi WHERE oi.order_id = o.id
		  ) items ON true
		 WHERE o.event_id = $1
		   AND o.status::text IN `+countedOrderStatuses+`
		   AND o.placed_at IS NOT NULL
		   AND ($2::timestamptz IS NULL OR o.placed_at >= $2)
		   AND ($3::timestamptz IS NULL OR o.placed_at < $3)
		 GROUP BY day
		 ORDER BY day`,
		eventID, from, to)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	points := []SalesPoint{}
	for rows.Next() {
		var p SalesPoint
		if err := rows.Scan(&p.Day, &p.Orders, &p.Tickets, &p.RevenueKZT); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// round1 keeps percentages to one decimal place.
func round1(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}
