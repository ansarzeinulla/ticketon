package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventOrder is one row of the organizer's order list.
type EventOrder struct {
	ID          uuid.UUID `json:"id"`
	OrderNumber string    `json:"order_number"`
	BuyerName   string    `json:"buyer_name"`
	BuyerEmail  string    `json:"buyer_email"`
	Status      string    `json:"status"`
	TotalKZT    string    `json:"total_kzt"`
	TicketCount int       `json:"ticket_count"`
	// LiveTickets still admit somebody; CheckedIn already have.
	LiveTickets int        `json:"live_tickets"`
	CheckedIn   int        `json:"checked_in"`
	PlacedAt    *time.Time `json:"placed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Guest is one issued ticket with the person it was issued to.
type Guest struct {
	TicketID       uuid.UUID `json:"ticket_id"`
	TicketCode     string    `json:"ticket_code"`
	FullName       string    `json:"full_name"`
	Email          string    `json:"email"`
	TicketTypeName string    `json:"ticket_type_name"`
	Status         string    `json:"status"`
	OrderNumber    string    `json:"order_number"`
}

// GuestStore reads who is coming to an event.
type GuestStore struct {
	pool *pgxpool.Pool
}

// NewGuestStore builds a GuestStore.
func NewGuestStore(pool *pgxpool.Pool) *GuestStore { return &GuestStore{pool: pool} }

// guestListLimit keeps one page to what a dashboard table can show.
const guestListLimit = 200

// ListEventOrders returns an event's orders, newest first.
func (s *GuestStore) ListEventOrders(ctx context.Context, eventID uuid.UUID) ([]EventOrder, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.order_number, o.buyer_name, o.buyer_email, o.status::text,
		       o.total_kzt::text,
		       count(t.id)                                                   AS ticket_count,
		       count(t.id) FILTER (WHERE t.status IN ('valid', 'checked_in')) AS live_tickets,
		       count(t.id) FILTER (WHERE t.status = 'checked_in')             AS checked_in,
		       o.placed_at, o.created_at
		  FROM orders o
		  LEFT JOIN tickets t ON t.order_id = o.id
		 WHERE o.event_id = $1
		 GROUP BY o.id
		 ORDER BY o.created_at DESC
		 LIMIT $2`, eventID, guestListLimit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	orders := []EventOrder{}
	for rows.Next() {
		var o EventOrder
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.BuyerName, &o.BuyerEmail, &o.Status,
			&o.TotalKZT, &o.TicketCount, &o.LiveTickets, &o.CheckedIn,
			&o.PlacedAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// ListGuests returns the tickets issued for an event, optionally narrowed by a
// name, email, ticket code or order number.
func (s *GuestStore) ListGuests(ctx context.Context, eventID uuid.UUID, query string) ([]Guest, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.ticket_code, a.full_name, a.email::text, tt.name,
		       t.status::text, o.order_number
		  FROM tickets t
		  JOIN attendees a     ON a.id = t.attendee_id
		  JOIN ticket_types tt ON tt.id = t.ticket_type_id
		  JOIN orders o        ON o.id = t.order_id
		 WHERE t.event_id = $1
		   AND ($2 = '' OR a.full_name ILIKE '%' || $2 || '%'
		                OR a.email::text ILIKE '%' || $2 || '%'
		                OR t.ticket_code ILIKE '%' || $2 || '%'
		                OR o.order_number ILIKE '%' || $2 || '%')
		 ORDER BY a.full_name, t.ticket_code
		 LIMIT $3`, eventID, query, guestListLimit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	guests := []Guest{}
	for rows.Next() {
		var g Guest
		if err := rows.Scan(&g.TicketID, &g.TicketCode, &g.FullName, &g.Email,
			&g.TicketTypeName, &g.Status, &g.OrderNumber); err != nil {
			return nil, err
		}
		guests = append(guests, g)
	}
	return guests, rows.Err()
}
