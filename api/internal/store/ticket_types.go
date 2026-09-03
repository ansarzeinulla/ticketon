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

// ErrTicketTypeNameTaken is returned when an event already has a ticket type
// with that name (the schema enforces UNIQUE (event_id, name)).
var ErrTicketTypeNameTaken = errors.New("a ticket type with this name already exists for the event")

// TicketType mirrors a row of the ticket_types table.
//
// Remaining is not a column: it is derived so clients never have to reproduce
// the inventory arithmetic themselves.
type TicketType struct {
	ID                uuid.UUID `json:"id"`
	EventID           uuid.UUID `json:"event_id"`
	Name              string    `json:"name"`
	Description       *string   `json:"description,omitempty"`
	PriceKZT          string    `json:"price_kzt"`
	QuantityTotal     int       `json:"quantity_total"`
	QuantitySold      int       `json:"quantity_sold"`
	QuantityReserved  int       `json:"quantity_reserved"`
	QuantityRefunded  int       `json:"quantity_refunded"`
	QuantityRemaining int       `json:"quantity_remaining"`
	// QuantityCheckedIn is how many of this tier walked through the door
	// (SRS 4.3). Counted from tickets, not stored, so it cannot drift.
	QuantityCheckedIn int        `json:"quantity_checked_in"`
	MaxPerOrder       int        `json:"max_per_order"`
	SalesStartAt      *time.Time `json:"sales_start_at,omitempty"`
	SalesEndAt        *time.Time `json:"sales_end_at,omitempty"`
	IsHidden          bool       `json:"is_hidden"`
	IsFree            bool       `json:"is_free"`
	// PriceCategory ties this tier to a venue section of the same category, so
	// a seat on the map knows what it costs (SRS 4.3.1).
	PriceCategory *string   `json:"price_category,omitempty"`
	DisplayOrder  int       `json:"display_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// OnSaleAt reports whether the type may be bought at the given moment.
func (t TicketType) OnSaleAt(now time.Time) bool {
	if t.IsHidden {
		return false
	}
	if t.SalesStartAt != nil && now.Before(*t.SalesStartAt) {
		return false
	}
	if t.SalesEndAt != nil && !now.Before(*t.SalesEndAt) {
		return false
	}
	return true
}

// TicketTypeStore reads and writes ticket types.
type TicketTypeStore struct {
	pool *pgxpool.Pool
}

// NewTicketTypeStore builds a TicketTypeStore.
func NewTicketTypeStore(pool *pgxpool.Pool) *TicketTypeStore {
	return &TicketTypeStore{pool: pool}
}

// price_kzt is scanned as text so the numeric(14,2) value survives the trip
// without a float rounding it. Money never becomes a float64 here.
const ticketTypeColumns = `id, event_id, name, description, price_kzt::text,
	quantity_total, quantity_sold, quantity_reserved, quantity_refunded,
	quantity_total - quantity_sold - quantity_reserved AS quantity_remaining,
	-- SRS 4.3 asks organizers to see available, reserved, sold, refunded AND
	-- checked-in per ticket type. The first four are counters on this row; the
	-- fifth is only knowable from the tickets themselves, so it is counted
	-- here rather than denormalised into a column that could drift.
	(SELECT count(*) FROM tickets t
	  WHERE t.ticket_type_id = ticket_types.id
	    AND t.status = 'checked_in') AS quantity_checked_in,
	max_per_order, sales_start_at, sales_end_at, is_hidden, is_free,
	price_category, display_order, created_at, updated_at`

func scanTicketType(row pgx.Row) (TicketType, error) {
	var t TicketType
	err := row.Scan(&t.ID, &t.EventID, &t.Name, &t.Description, &t.PriceKZT,
		&t.QuantityTotal, &t.QuantitySold, &t.QuantityReserved, &t.QuantityRefunded,
		&t.QuantityRemaining, &t.QuantityCheckedIn,
		&t.MaxPerOrder, &t.SalesStartAt, &t.SalesEndAt,
		&t.IsHidden, &t.IsFree, &t.PriceCategory, &t.DisplayOrder,
		&t.CreatedAt, &t.UpdatedAt)
	return t, err
}

// CreateTicketTypeParams describes a new ticket type.
type CreateTicketTypeParams struct {
	EventID       uuid.UUID
	Name          string
	Description   *string
	PriceKZT      string
	QuantityTotal int
	MaxPerOrder   int
	SalesStartAt  *time.Time
	SalesEndAt    *time.Time
	IsHidden      bool
	PriceCategory *string
	DisplayOrder  int
}

// Create inserts a ticket type.
func (s *TicketTypeStore) Create(ctx context.Context, p CreateTicketTypeParams) (TicketType, error) {
	t, err := scanTicketType(s.pool.QueryRow(ctx, `
		INSERT INTO ticket_types (
			event_id, name, description, price_kzt, quantity_total,
			max_per_order, sales_start_at, sales_end_at, is_hidden,
			price_category, display_order)
		VALUES ($1, $2, $3, $4::numeric, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+ticketTypeColumns,
		p.EventID, p.Name, p.Description, p.PriceKZT, p.QuantityTotal,
		p.MaxPerOrder, p.SalesStartAt, p.SalesEndAt, p.IsHidden,
		p.PriceCategory, p.DisplayOrder))
	if err != nil {
		return TicketType{}, mapTicketTypeError(err)
	}
	return t, nil
}

// ListForEvent returns an event's ticket types in display order.
// When onlyVisible is set, hidden types are left out - the public view.
func (s *TicketTypeStore) ListForEvent(ctx context.Context, eventID uuid.UUID, onlyVisible bool) ([]TicketType, error) {
	query := `SELECT ` + ticketTypeColumns + ` FROM ticket_types WHERE event_id = $1`
	if onlyVisible {
		query += ` AND is_hidden = false`
	}
	query += ` ORDER BY display_order ASC, created_at ASC`

	rows, err := s.pool.Query(ctx, query, eventID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	types := []TicketType{}
	for rows.Next() {
		t, err := scanTicketType(rows)
		if err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, rows.Err()
}

// GetByID returns one ticket type.
func (s *TicketTypeStore) GetByID(ctx context.Context, id uuid.UUID) (TicketType, error) {
	t, err := scanTicketType(s.pool.QueryRow(ctx,
		`SELECT `+ticketTypeColumns+` FROM ticket_types WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return TicketType{}, ErrNotFound
	}
	if err != nil {
		return TicketType{}, mapError(err)
	}
	return t, nil
}

// UpdateTicketTypeParams carries the fields a PATCH may change.
type UpdateTicketTypeParams struct {
	Name          Optional[string]
	Description   Optional[string]
	PriceKZT      Optional[string]
	QuantityTotal Optional[int]
	MaxPerOrder   Optional[int]
	PriceCategory Optional[string]
	SalesStartAt  Optional[time.Time]
	SalesEndAt    Optional[time.Time]
	IsHidden      Optional[bool]
	DisplayOrder  Optional[int]
}

// Update applies the supplied fields.
//
// Lowering quantity_total below what is already sold is rejected by the
// ticket_types_inventory_chk constraint, which surfaces as a ConstraintError.
func (s *TicketTypeStore) Update(ctx context.Context, id uuid.UUID, p UpdateTicketTypeParams) (TicketType, error) {
	var (
		sets []string
		args []any
	)
	set := func(column string, present bool, value any) {
		if !present {
			return
		}
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	set("name", p.Name.Set, p.Name.Ptr())
	set("description", p.Description.Set, p.Description.Ptr())
	set("quantity_total", p.QuantityTotal.Set, p.QuantityTotal.Ptr())
	set("max_per_order", p.MaxPerOrder.Set, p.MaxPerOrder.Ptr())
	set("price_category", p.PriceCategory.Set, p.PriceCategory.Ptr())
	set("sales_start_at", p.SalesStartAt.Set, p.SalesStartAt.Ptr())
	set("sales_end_at", p.SalesEndAt.Set, p.SalesEndAt.Ptr())
	set("is_hidden", p.IsHidden.Set, p.IsHidden.Ptr())
	set("display_order", p.DisplayOrder.Set, p.DisplayOrder.Ptr())

	if p.PriceKZT.Set && p.PriceKZT.Valid {
		args = append(args, p.PriceKZT.Value)
		sets = append(sets, fmt.Sprintf("price_kzt = $%d::numeric", len(args)))
	}

	if len(sets) == 0 {
		return s.GetByID(ctx, id)
	}

	args = append(args, id)
	query := `UPDATE ticket_types SET ` + strings.Join(sets, ", ") +
		fmt.Sprintf(" WHERE id = $%d RETURNING ", len(args)) + ticketTypeColumns

	t, err := scanTicketType(s.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return TicketType{}, ErrNotFound
	}
	if err != nil {
		return TicketType{}, mapTicketTypeError(err)
	}
	return t, nil
}

// Delete removes a ticket type. It fails while order items reference it,
// because order_items.ticket_type_id is ON DELETE RESTRICT - sold tickets keep
// their history.
func (s *TicketTypeStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM ticket_types WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// HasSales reports whether anything has been sold from this ticket type.
func (s *TicketTypeStore) HasSales(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM order_items WHERE ticket_type_id = $1)`, id).Scan(&exists)
	return exists, mapError(err)
}

// mapTicketTypeError adds the ticket-type-specific unique violation on top of
// the shared mapping.
func mapTicketTypeError(err error) error {
	mapped := mapError(err)

	var constraintErr *ConstraintError
	if errors.As(mapped, &constraintErr) {
		return mapped
	}
	if isUniqueViolation(err, "ticket_types_event_id_name_key") {
		return ErrTicketTypeNameTaken
	}
	return mapped
}
