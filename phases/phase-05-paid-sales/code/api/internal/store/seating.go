package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Seat states the map distinguishes (SRS 4.3.1).
//
// The SRS asks for available, selected, temporarily held, sold, unavailable and
// accessible to be told apart. "Selected" is the client's own business - it is
// what this browser has clicked and nobody else can know it - so the server
// reports the other five.
const (
	SeatAvailable   = "available"
	SeatHeld        = "held"
	SeatSold        = "sold"
	SeatUnavailable = "unavailable"
)

// Seat is one seat on the map.
type Seat struct {
	ID     uuid.UUID `json:"id"`
	Number string    `json:"number"`
	// X and Y are the layout coordinates, in the same arbitrary units for
	// every seat, so the client can draw the map without knowing the venue.
	X float64 `json:"x"`
	Y float64 `json:"y"`
	// Accessible seats are called out separately because SRS 4.3.1 requires
	// them to be distinguishable, not merely bookable.
	Accessible bool   `json:"accessible"`
	Status     string `json:"status"`
}

// SeatRowInfo is one row of seats.
type SeatRowInfo struct {
	ID    uuid.UUID `json:"id"`
	Label string    `json:"label"`
	Seats []Seat    `json:"seats"`
}

// SeatSection is a priced block of the venue.
type SeatSection struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	PriceCategory string    `json:"price_category"`
	// TicketTypeID is the tier a seat in this section is sold as, and PriceKZT
	// is what it costs - so an attendee sees the price before choosing
	// (SRS 4.3.1).
	TicketTypeID   *uuid.UUID    `json:"ticket_type_id,omitempty"`
	TicketTypeName string        `json:"ticket_type_name,omitempty"`
	PriceKZT       string        `json:"price_kzt,omitempty"`
	Rows           []SeatRowInfo `json:"rows"`
}

// SeatMap is the whole layout for one event.
type SeatMap struct {
	EventID   uuid.UUID     `json:"event_id"`
	VenueID   uuid.UUID     `json:"venue_id"`
	VenueName string        `json:"venue_name"`
	Sections  []SeatSection `json:"sections"`

	// The bounding box of every seat, so the client can set an SVG viewBox
	// without measuring the layout itself.
	MinX float64 `json:"min_x"`
	MinY float64 `json:"min_y"`
	MaxX float64 `json:"max_x"`
	MaxY float64 `json:"max_y"`

	// Totals for the summary line above the map.
	TotalSeats     int `json:"total_seats"`
	AvailableSeats int `json:"available_seats"`
}

// ErrNoSeatingPlan reports an event that is not sold by seat.
var ErrNoSeatingPlan = errors.New("this event does not use assigned seating")

// SeatingStore reads venue layouts and seat availability (SRS 4.3.1).
type SeatingStore struct {
	pool *pgxpool.Pool
}

// NewSeatingStore builds a SeatingStore.
func NewSeatingStore(pool *pgxpool.Pool) *SeatingStore { return &SeatingStore{pool: pool} }

// MapForEvent returns the seat map with every seat's current state.
//
// One query rather than a walk down the tree: a 126-seat hall is three
// sections, eleven rows and 126 seats, and fetching those separately would be
// fifteen round trips to draw one picture.
func (s *SeatingStore) MapForEvent(ctx context.Context, eventID uuid.UUID) (SeatMap, error) {
	var (
		plan      SeatMap
		venueID   *uuid.UUID
		venueName *string
		mode      string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT e.venue_id, v.name, e.seating_mode::text
		  FROM events e
		  LEFT JOIN venues v ON v.id = e.venue_id
		 WHERE e.id = $1`, eventID).Scan(&venueID, &venueName, &mode)
	if errors.Is(err, pgx.ErrNoRows) {
		return SeatMap{}, ErrNotFound
	}
	if err != nil {
		return SeatMap{}, mapError(err)
	}
	if mode != "assigned_seating" || venueID == nil {
		return SeatMap{}, ErrNoSeatingPlan
	}

	plan.EventID = eventID
	plan.VenueID = *venueID
	plan.VenueName = derefOr(venueName, "")

	rows, err := s.pool.Query(ctx, `
		SELECT sec.id, sec.name, sec.price_category, sec.display_order,
		       tt.id, tt.name, tt.price_kzt::text,
		       r.id, r.label, r.display_order,
		       st.id, st.seat_number, st.map_x, st.map_y,
		       st.is_accessible, st.is_available,
		       -- Sold: a live ticket names this seat.
		       EXISTS (SELECT 1 FROM tickets t
		                WHERE t.seat_id = st.id
		                  AND t.status IN ('valid', 'checked_in'))                AS sold,
		       -- Held: somebody's basket has it, and that basket is still alive.
		       EXISTS (SELECT 1 FROM seat_holds h
		                WHERE h.seat_id = st.id
		                  AND h.status = 'active'
		                  AND h.expires_at > now())                               AS held
		  FROM venue_sections sec
		  JOIN seat_rows r  ON r.section_id = sec.id
		  JOIN seats st     ON st.row_id = r.id
		  -- The tier a seat is sold as. Matched on the section's own name
		  -- first, because two sections can share a price category - the demo
		  -- hall's Balcony and Accessible Box both do - and matching on
		  -- category alone would price one of them as the other.
		  LEFT JOIN LATERAL (
		      SELECT t.id, t.name, t.price_kzt
		        FROM ticket_types t
		       WHERE t.event_id = $1
		         AND (t.name = sec.name OR t.price_category = sec.price_category)
		       ORDER BY (t.name = sec.name) DESC, t.display_order
		       LIMIT 1
		  ) tt ON true
		 WHERE sec.venue_id = $2
		 ORDER BY sec.display_order, sec.name, r.display_order, r.label, st.seat_number`,
		eventID, *venueID)
	if err != nil {
		return SeatMap{}, mapError(err)
	}
	defer rows.Close()

	var (
		sectionIndex = map[uuid.UUID]int{}
		rowIndex     = map[uuid.UUID]int{}
		first        = true
	)

	for rows.Next() {
		var (
			secID, rowID, seatID      uuid.UUID
			secName, category, rowLbl string
			secOrder, rowOrder        int
			ticketTypeID              *uuid.UUID
			ticketTypeName, priceKZT  *string
			seatNumber                string
			x, y                      float64
			accessible, available     bool
			sold, held                bool
		)
		if err := rows.Scan(&secID, &secName, &category, &secOrder,
			&ticketTypeID, &ticketTypeName, &priceKZT,
			&rowID, &rowLbl, &rowOrder,
			&seatID, &seatNumber, &x, &y, &accessible, &available,
			&sold, &held); err != nil {
			return SeatMap{}, err
		}

		si, ok := sectionIndex[secID]
		if !ok {
			plan.Sections = append(plan.Sections, SeatSection{
				ID: secID, Name: secName, PriceCategory: category,
				TicketTypeID:   ticketTypeID,
				TicketTypeName: derefOr(ticketTypeName, ""),
				PriceKZT:       derefOr(priceKZT, ""),
			})
			si = len(plan.Sections) - 1
			sectionIndex[secID] = si
		}

		ri, ok := rowIndex[rowID]
		if !ok {
			plan.Sections[si].Rows = append(plan.Sections[si].Rows,
				SeatRowInfo{ID: rowID, Label: rowLbl})
			ri = len(plan.Sections[si].Rows) - 1
			rowIndex[rowID] = ri
		}

		// The order matters: a sold seat that is also held is sold, and a seat
		// taken out of service is unavailable whatever else is true of it.
		status := SeatAvailable
		switch {
		case !available:
			status = SeatUnavailable
		case sold:
			status = SeatSold
		case held:
			status = SeatHeld
		}

		plan.Sections[si].Rows[ri].Seats = append(plan.Sections[si].Rows[ri].Seats, Seat{
			ID: seatID, Number: seatNumber, X: x, Y: y,
			Accessible: accessible, Status: status,
		})

		plan.TotalSeats++
		if status == SeatAvailable {
			plan.AvailableSeats++
		}

		if first {
			plan.MinX, plan.MaxX, plan.MinY, plan.MaxY = x, x, y, y
			first = false
		}
		plan.MinX = min(plan.MinX, x)
		plan.MaxX = max(plan.MaxX, x)
		plan.MinY = min(plan.MinY, y)
		plan.MaxY = max(plan.MaxY, y)
	}
	if err := rows.Err(); err != nil {
		return SeatMap{}, mapError(err)
	}
	if plan.TotalSeats == 0 {
		return SeatMap{}, ErrNoSeatingPlan
	}
	return plan, nil
}

// SeatAssignment is one seat resolved to the tier it is sold as.
type SeatAssignment struct {
	SeatID       uuid.UUID
	TicketTypeID uuid.UUID
}

// SeatUnavailableError reports a seat that cannot be reserved, and why.
type SeatUnavailableError struct {
	SeatID uuid.UUID
	Reason string
}

func (e *SeatUnavailableError) Error() string {
	return fmt.Sprintf("seat %s is %s", e.SeatID, e.Reason)
}

// resolveSeats turns a list of seats into the order lines that buy them.
//
// The client sends seats, not tiers: an attendee picks a place to sit, and
// what that costs follows from where it is. Resolving it server-side is also
// what stops a crafted request from claiming an Orchestra seat at Balcony
// prices.
//
// The seats are locked FOR UPDATE in id order, so two baskets racing for the
// same pair of seats queue rather than deadlock.
func resolveSeats(
	ctx context.Context, tx pgx.Tx, eventID uuid.UUID, seatIDs []uuid.UUID,
) ([]SeatAssignment, error) {
	if len(seatIDs) == 0 {
		return nil, nil
	}

	rows, err := tx.Query(ctx, `
		SELECT st.id, st.is_available, tt.id,
		       EXISTS (SELECT 1 FROM tickets t
		                WHERE t.seat_id = st.id
		                  AND t.status IN ('valid', 'checked_in')) AS sold
		  FROM seats st
		  JOIN seat_rows r        ON r.id = st.row_id
		  JOIN venue_sections sec ON sec.id = r.section_id
		  JOIN events e           ON e.venue_id = sec.venue_id
		  LEFT JOIN LATERAL (
		      SELECT t.id
		        FROM ticket_types t
		       WHERE t.event_id = e.id
		         AND (t.name = sec.name OR t.price_category = sec.price_category)
		       ORDER BY (t.name = sec.name) DESC, t.display_order
		       LIMIT 1
		  ) tt ON true
		 WHERE st.id = ANY($1) AND e.id = $2
		 ORDER BY st.id
		   FOR UPDATE OF st`, seatIDs, eventID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	found := map[uuid.UUID]SeatAssignment{}
	for rows.Next() {
		var (
			seatID       uuid.UUID
			available    bool
			ticketTypeID *uuid.UUID
			sold         bool
		)
		if err := rows.Scan(&seatID, &available, &ticketTypeID, &sold); err != nil {
			return nil, err
		}
		if !available {
			return nil, &SeatUnavailableError{SeatID: seatID, Reason: "not available"}
		}
		if sold {
			return nil, &SeatUnavailableError{SeatID: seatID, Reason: "already sold"}
		}
		if ticketTypeID == nil {
			// A section with no matching tier cannot be priced, and selling a
			// seat at a price nobody set is worse than refusing it.
			return nil, &SeatUnavailableError{SeatID: seatID, Reason: "not on sale"}
		}
		found[seatID] = SeatAssignment{SeatID: seatID, TicketTypeID: *ticketTypeID}
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}

	// Returned in the order asked for, and every requested seat must exist:
	// a seat from another venue silently dropped would produce a basket that
	// does not match what was clicked.
	out := make([]SeatAssignment, 0, len(seatIDs))
	for _, id := range seatIDs {
		assignment, ok := found[id]
		if !ok {
			return nil, &SeatUnavailableError{SeatID: id, Reason: "not part of this event"}
		}
		out = append(out, assignment)
	}
	return out, nil
}
