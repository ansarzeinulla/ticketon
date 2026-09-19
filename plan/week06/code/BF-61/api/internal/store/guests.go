package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
