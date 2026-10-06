package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BuyerOrder is one of the signed-in attendee's own orders, with just enough
// event context to be listed on its own (SRS 4.9: "Attendees shall be able to
// view their orders"; SRS 4.7: tickets reachable "through the attendee's
// account", not only through the emailed link).
type BuyerOrder struct {
	ID            uuid.UUID  `json:"id"`
	OrderNumber   string     `json:"order_number"`
	Status        string     `json:"status"`
	TotalKZT      string     `json:"total_kzt"`
	PlacedAt      *time.Time `json:"placed_at"`
	EventID       uuid.UUID  `json:"event_id"`
	EventTitle    string     `json:"event_title"`
	EventSlug     string     `json:"event_slug"`
	EventStartsAt time.Time  `json:"event_starts_at"`
	Timezone      string     `json:"timezone"`
	TicketCount   int        `json:"ticket_count"`
	LiveTickets   int        `json:"live_tickets"`
}

// BuyerOrderStore lists a user's own orders.
type BuyerOrderStore struct {
	pool *pgxpool.Pool
}

// NewBuyerOrderStore builds a BuyerOrderStore.
func NewBuyerOrderStore(pool *pgxpool.Pool) *BuyerOrderStore { return &BuyerOrderStore{pool: pool} }

// ListForUser returns the orders belonging to one user, newest first.
//
// Ownership is the same rule support uses: the order was placed while signed in
// (buyer_user_id), or it was a guest checkout whose email now belongs to this
// account. That is what lets someone who bought as a guest register with the
// same address later and still find their tickets.
func (s *BuyerOrderStore) ListForUser(
	ctx context.Context, userID uuid.UUID, limit int,
) ([]BuyerOrder, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.order_number, o.status::text, o.total_kzt::text, o.placed_at,
		       e.id, e.title, e.slug, e.starts_at, e.timezone,
		       (SELECT count(*) FROM tickets t WHERE t.order_id = o.id),
		       (SELECT count(*) FROM tickets t
		         WHERE t.order_id = o.id AND t.status IN ('valid', 'checked_in'))
		  FROM orders o
		  JOIN events e ON e.id = o.event_id
		 WHERE o.buyer_user_id = $1
		    OR o.buyer_email = (SELECT email FROM users WHERE id = $1)
		 ORDER BY COALESCE(o.placed_at, o.created_at) DESC
		 LIMIT $2`, userID, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	orders := []BuyerOrder{}
	for rows.Next() {
		var o BuyerOrder
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.Status, &o.TotalKZT, &o.PlacedAt,
			&o.EventID, &o.EventTitle, &o.EventSlug, &o.EventStartsAt, &o.Timezone,
			&o.TicketCount, &o.LiveTickets); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, mapError(rows.Err())
}
