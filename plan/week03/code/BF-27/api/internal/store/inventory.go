package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InventoryLine is the stock position of one ticket type.
type InventoryLine struct {
	TicketTypeID uuid.UUID `json:"ticket_type_id"`
	Name         string    `json:"name"`
	Total        int       `json:"quantity_total"`
	Sold         int       `json:"quantity_sold"`
	Remaining    int       `json:"quantity_remaining"`
}

// InventoryStore answers "how many are left" without loading whole ticket
// types, so a page can poll it cheaply while an attendee is choosing.
type InventoryStore struct {
	pool *pgxpool.Pool
}

// NewInventoryStore builds an InventoryStore.
func NewInventoryStore(pool *pgxpool.Pool) *InventoryStore { return &InventoryStore{pool: pool} }

// ForEvent returns the stock of every visible ticket type of an event.
func (s *InventoryStore) ForEvent(ctx context.Context, eventID uuid.UUID) ([]InventoryLine, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, quantity_total, quantity_sold,
		       quantity_total - quantity_sold - quantity_reserved
		  FROM ticket_types
		 WHERE event_id = $1 AND is_hidden = false
		 ORDER BY display_order, created_at`, eventID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	lines := []InventoryLine{}
	for rows.Next() {
		var l InventoryLine
		if err := rows.Scan(&l.TicketTypeID, &l.Name, &l.Total, &l.Sold, &l.Remaining); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}
