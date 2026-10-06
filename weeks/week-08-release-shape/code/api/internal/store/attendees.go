package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AttendeeTicket is one ticket in the manual-search results (SRS 4.8,
// "search for attendees manually").
//
// It deliberately does NOT carry the QR token. Staff searching by name are
// standing next to the person; handing every device a working admission
// credential for every attendee would make the QR pointless. Check-in is done
// by ticket id instead, which only works for staff already authorised on the
// event.
type AttendeeTicket struct {
	TicketID       uuid.UUID  `json:"ticket_id"`
	TicketCode     string     `json:"ticket_code"`
	AttendeeName   string     `json:"attendee_name"`
	AttendeeEmail  string     `json:"attendee_email"`
	TicketTypeName string     `json:"ticket_type_name"`
	Status         string     `json:"status"`
	OrderNumber    string     `json:"order_number"`
	CheckedInAt    *time.Time `json:"checked_in_at,omitempty"`
	// Admissible mirrors the rule the gate enforces, so the app can show a
	// disabled row instead of a button that is certain to be refused.
	Admissible bool `json:"admissible"`
}

// AttendeeStore searches the people holding tickets for an event.
type AttendeeStore struct {
	pool *pgxpool.Pool
}

// NewAttendeeStore builds an AttendeeStore.
func NewAttendeeStore(pool *pgxpool.Pool) *AttendeeStore { return &AttendeeStore{pool: pool} }

// attendeeSearchLimit keeps a door-side lookup to one screenful. Someone typing
// "a" wants to be told to type more, not handed nine hundred rows.
const attendeeSearchLimit = 25

// Search finds tickets for an event by attendee name, email, ticket code or
// order number.
//
// An empty query returns the most recent tickets, so the screen opens with
// something rather than nothing while staff are still typing.
func (s *AttendeeStore) Search(
	ctx context.Context, eventID uuid.UUID, query string,
) ([]AttendeeTicket, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.ticket_code, a.full_name, a.email::text, tt.name,
		       t.status::text, o.order_number, t.checked_in_at
		  FROM tickets t
		  JOIN attendees a    ON a.id = t.attendee_id
		  JOIN ticket_types tt ON tt.id = t.ticket_type_id
		  JOIN orders o       ON o.id = t.order_id
		 WHERE t.event_id = $1
		   AND ($2 = '' OR a.full_name ILIKE '%' || $2 || '%'
		                OR a.email::text ILIKE '%' || $2 || '%'
		                OR t.ticket_code ILIKE '%' || $2 || '%'
		                OR o.order_number ILIKE '%' || $2 || '%')
		 ORDER BY a.full_name, t.ticket_code
		 LIMIT $3`, eventID, query, attendeeSearchLimit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []AttendeeTicket{}
	for rows.Next() {
		var t AttendeeTicket
		if err := rows.Scan(&t.TicketID, &t.TicketCode, &t.AttendeeName, &t.AttendeeEmail,
			&t.TicketTypeName, &t.Status, &t.OrderNumber, &t.CheckedInAt); err != nil {
			return nil, err
		}
		// Only a valid ticket can still be admitted. checked_in, cancelled and
		// refunded all cannot - the same rule the gate applies.
		t.Admissible = t.Status == "valid"
		out = append(out, t)
	}
	return out, rows.Err()
}
