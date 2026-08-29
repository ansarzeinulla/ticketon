package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TicketDetail is one ticket joined with everything a printed ticket or a
// check-in screen needs, so the PDF handler makes a single query.
type TicketDetail struct {
	ID         uuid.UUID
	TicketCode string
	QRToken    string
	Status     string
	IssuedAt   time.Time

	EventID      uuid.UUID
	EventTitle   string
	EventSlug    string
	StartsAt     time.Time
	EndsAt       time.Time
	Timezone     string
	VenueName    string
	VenueAddress string
	OrganizerID  uuid.UUID

	TicketTypeName string

	OrderID     uuid.UUID
	OrderNumber string
	BuyerUserID *uuid.UUID

	AttendeeName  string
	AttendeeEmail string

	SeatSection string
	SeatRow     string
	SeatNumber  string
}

// TicketStore reads issued tickets.
type TicketStore struct {
	pool *pgxpool.Pool
}

// NewTicketStore builds a TicketStore.
func NewTicketStore(pool *pgxpool.Pool) *TicketStore { return &TicketStore{pool: pool} }

// GetDetail returns one ticket with its event, order and attendee context.
//
// The attendee falls back to the buyer on the order: a ticket may be issued
// without a named attendee, and a printed ticket still needs a name on it.
func (s *TicketStore) GetDetail(ctx context.Context, id uuid.UUID) (TicketDetail, error) {
	var d TicketDetail
	var venueName, venueAddress, seatSection, seatRow, seatNumber *string

	err := s.pool.QueryRow(ctx, `
		SELECT t.id, t.ticket_code, t.qr_token, t.status::text, t.issued_at,
		       e.id, e.title, e.slug, e.starts_at, e.ends_at, e.timezone,
		       e.venue_name, e.venue_address, e.organizer_id,
		       tt.name,
		       o.id, o.order_number, o.buyer_user_id,
		       COALESCE(a.full_name, o.buyer_name),
		       COALESCE(a.email::text, o.buyer_email::text),
		       t.seat_section, t.seat_row, t.seat_number
		  FROM tickets t
		  JOIN events e       ON e.id = t.event_id
		  JOIN ticket_types tt ON tt.id = t.ticket_type_id
		  JOIN orders o       ON o.id = t.order_id
		  LEFT JOIN attendees a ON a.id = t.attendee_id
		 WHERE t.id = $1`, id,
	).Scan(&d.ID, &d.TicketCode, &d.QRToken, &d.Status, &d.IssuedAt,
		&d.EventID, &d.EventTitle, &d.EventSlug, &d.StartsAt, &d.EndsAt, &d.Timezone,
		&venueName, &venueAddress, &d.OrganizerID,
		&d.TicketTypeName,
		&d.OrderID, &d.OrderNumber, &d.BuyerUserID,
		&d.AttendeeName, &d.AttendeeEmail,
		&seatSection, &seatRow, &seatNumber)

	if errors.Is(err, pgx.ErrNoRows) {
		return TicketDetail{}, ErrNotFound
	}
	if err != nil {
		return TicketDetail{}, mapError(err)
	}

	d.VenueName = derefOr(venueName, "")
	d.VenueAddress = derefOr(venueAddress, "")
	d.SeatSection = derefOr(seatSection, "")
	d.SeatRow = derefOr(seatRow, "")
	d.SeatNumber = derefOr(seatNumber, "")

	return d, nil
}

func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}
