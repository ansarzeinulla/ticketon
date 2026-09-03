package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// HoldTTL is how long a basket keeps its seats before the inventory goes back
// on sale (SRS 4.6, "ticket inventory shall be temporarily reserved during
// checkout").
//
// Fifteen minutes is long enough to read a form and find a card, short enough
// that an abandoned tab does not hold the last seat at a sold-out event for an
// afternoon.
const HoldTTL = 15 * time.Minute

// Hold failures the API turns into their own codes.
var (
	// ErrHoldExpired reports a basket whose reservation ran out. The seats are
	// already back on sale by the time this is returned.
	ErrHoldExpired = errors.New("this reservation has expired")
	// ErrHoldNotPending reports an order that is no longer a basket - already
	// paid, cancelled or released.
	ErrHoldNotPending = errors.New("this order is not an open reservation")
)

// Hold is a basket: inventory reserved, nothing sold yet.
type Hold struct {
	OrderID       uuid.UUID   `json:"order_id"`
	OrderNumber   string      `json:"order_number"`
	EventID       uuid.UUID   `json:"event_id"`
	Status        string      `json:"status"`
	Items         []OrderItem `json:"items"`
	SubtotalKZT   string      `json:"subtotal_kzt"`
	ReservedUntil time.Time   `json:"reserved_until"`
	// SecondsRemaining is what a countdown renders. Computed rather than left
	// to the client, whose clock may be anywhere.
	SecondsRemaining int `json:"seconds_remaining"`
	// EstimatedFeeKZT and EstimatedTotalKZT price the basket before a promo
	// code is applied, so an attendee sees what they will pay before
	// committing (SRS 4.3.1).
	EstimatedFeeKZT   string `json:"estimated_processing_fee_kzt"`
	EstimatedTotalKZT string `json:"estimated_total_kzt"`
}

// HoldParams describes a basket to reserve.
type HoldParams struct {
	EventID     uuid.UUID
	BuyerUserID *uuid.UUID
	// Buyer details are optional at hold time: somebody picking seats has not
	// filled in the form yet. They are required to confirm.
	BuyerName  string
	BuyerEmail string
	Items      []CheckoutItem
	// SeatIDs, when the event has assigned seating, are the specific seats
	// this basket is holding (SRS 4.3.1).
	SeatIDs []uuid.UUID
}

// Fees is the processing charge configuration (SRS 3.3).
//
// The fee is what the payment processor takes on a transaction. It is added to
// the attendee's total and never reaches the organizer, which is what SRS 3.3
// means by "deducted from each transaction" - the organizer's proceeds are the
// ticket price, and the charge for moving the money is not theirs to keep.
//
// A free basket is charged nothing, per SRS 3.3's "free events: no platform
// fee". That is a rule about the amount, not about the event: a free ticket
// bought alongside a paid one is part of a transaction that does move money.
type Fees struct {
	// Percent of the discounted subtotal, as a decimal string ("3.5").
	Percent string
	// Fixed component in KZT, as a decimal string.
	FixedKZT string
}

// DefaultFees is what an unconfigured deployment charges.
var DefaultFees = Fees{Percent: "3.5", FixedKZT: "0"}

// CheckoutStore methods below implement the two-step flow. The one-shot
// Checkout in checkout.go composes them inside a single transaction.

// Hold reserves inventory for a basket and returns the reservation (SRS 4.6).
func (s *CheckoutStore) Hold(ctx context.Context, p HoldParams) (Hold, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Hold{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	held, err := s.hold(ctx, tx, p)
	if err != nil {
		return Hold{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Hold{}, mapError(err)
	}
	return held, nil
}

// hold does the work inside a caller's transaction.
func (s *CheckoutStore) hold(ctx context.Context, tx pgx.Tx, p HoldParams) (Hold, error) {
	// Expired baskets are swept before anything is counted. The background
	// sweeper does this too, but doing it here as well means a stale hold can
	// never be the reason a sale is refused - even if the sweeper is wedged.
	if err := releaseExpiredForEvent(ctx, tx, p.EventID); err != nil {
		return Hold{}, err
	}

	// Assigned seating: the attendee picked seats, not tiers. Each seat is
	// resolved to the tier it is sold as and becomes its own line of one, so
	// every ticket issued later carries exactly one seat. Deriving the tier
	// here rather than trusting the request is also what stops a crafted
	// basket claiming an Orchestra seat at Balcony prices (SRS 4.3.1).
	if len(p.SeatIDs) > 0 {
		assignments, err := resolveSeats(ctx, tx, p.EventID, p.SeatIDs)
		if err != nil {
			return Hold{}, err
		}

		items := make([]CheckoutItem, 0, len(assignments))
		seats := make([]uuid.UUID, 0, len(assignments))
		for _, assignment := range assignments {
			items = append(items, CheckoutItem{
				TicketTypeID: assignment.TicketTypeID, Quantity: 1,
			})
			seats = append(seats, assignment.SeatID)
		}
		p.Items = items
		p.SeatIDs = seats
	}

	if len(p.Items) == 0 {
		return Hold{}, ErrNotFound
	}

	locked, err := lockTicketTypes(ctx, tx, p.EventID, p.Items)
	if err != nil {
		return Hold{}, err
	}
	if err := validateBasket(ctx, tx, p.EventID, p.Items, locked, time.Now().UTC()); err != nil {
		return Hold{}, err
	}

	// --- take the inventory as *reserved*, not sold ------------------------
	// The WHERE clause repeats the availability test so the increment can
	// never exceed what is free, even if the snapshot above were stale.
	// ticket_types_inventory_chk is the backstop underneath that.
	for _, item := range p.Items {
		tag, err := tx.Exec(ctx, `
			UPDATE ticket_types
			   SET quantity_reserved = quantity_reserved + $2::int
			 WHERE id = $1
			   AND quantity_sold + quantity_reserved + $2::int <= quantity_total`,
			item.TicketTypeID, item.Quantity)
		if err != nil {
			return Hold{}, mapError(err)
		}
		if tag.RowsAffected() == 0 {
			t := locked[item.TicketTypeID]
			return Hold{}, &InsufficientInventoryError{t.id, t.name, item.Quantity, t.remaining}
		}
	}

	orderNumber, err := newCode("BF", "-", 10)
	if err != nil {
		return Hold{}, err
	}

	var (
		held      Hold
		expiresAt time.Time
	)
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (order_number, event_id, buyer_user_id, buyer_email, buyer_name,
		                    status, subtotal_kzt, discount_kzt, processing_fee_kzt,
		                    total_kzt, reserved_until)
		VALUES ($1, $2, $3, $4, $5, 'pending', 0, 0, 0, 0, now() + $6::interval)
		RETURNING id, order_number, reserved_until`,
		orderNumber, p.EventID, p.BuyerUserID,
		nullableOrPlaceholder(p.BuyerEmail, "reserved@biletflow.invalid"),
		nullableOrPlaceholder(p.BuyerName, "Reserved"), HoldTTL.String(),
	).Scan(&held.OrderID, &held.OrderNumber, &expiresAt)
	if err != nil {
		return Hold{}, mapError(err)
	}

	items, err := insertOrderItems(ctx, tx, held.OrderID, p.Items, locked, p.SeatIDs)
	if err != nil {
		return Hold{}, err
	}

	// The subtotal is summed in SQL, for the same reason every other amount is.
	var subtotal string
	err = tx.QueryRow(ctx, `
		UPDATE orders
		   SET subtotal_kzt = sums.subtotal,
		       total_kzt    = sums.subtotal
		  FROM (SELECT COALESCE(sum(line_total_kzt), 0) AS subtotal
		          FROM order_items WHERE order_id = $1) AS sums
		 WHERE orders.id = $1
		RETURNING subtotal_kzt::text`, held.OrderID).Scan(&subtotal)
	if err != nil {
		return Hold{}, mapError(err)
	}

	// --- hold the seats themselves, for assigned seating (SRS 4.3.1) -------
	if err := holdSeats(ctx, tx, p.EventID, held.OrderID, p.SeatIDs, expiresAt); err != nil {
		return Hold{}, err
	}

	fee, total, err := estimateFee(ctx, tx, held.OrderID, s.fees)
	if err != nil {
		return Hold{}, err
	}

	held.EventID = p.EventID
	held.Status = "pending"
	held.Items = items
	held.SubtotalKZT = subtotal
	held.ReservedUntil = expiresAt
	held.SecondsRemaining = secondsUntil(expiresAt)
	held.EstimatedFeeKZT = fee
	held.EstimatedTotalKZT = total
	return held, nil
}

// ConfirmParams turns a basket into a sale.
type ConfirmParams struct {
	OrderID     uuid.UUID
	BuyerUserID *uuid.UUID
	BuyerName   string
	BuyerEmail  string
	BuyerPhone  *string
	Promo       *Campaign
}

// Confirm pays for a held basket and issues its tickets.
func (s *CheckoutStore) Confirm(ctx context.Context, p ConfirmParams) (CheckoutResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CheckoutResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := s.confirm(ctx, tx, p)
	if err != nil {
		// A basket that has run out is swept before the error goes back, so the
		// seats are on sale again the moment somebody discovers they were too
		// slow - rather than lingering until the next timer tick. The rollback
		// above has already undone this transaction, so the sweep needs its
		// own.
		if errors.Is(err, ErrHoldExpired) {
			_ = tx.Rollback(ctx)
			if _, sweepErr := s.ReleaseExpired(ctx); sweepErr != nil {
				return CheckoutResult{}, sweepErr
			}
		}
		return CheckoutResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CheckoutResult{}, mapError(err)
	}
	return result, nil
}

// confirm does the work inside a caller's transaction.
func (s *CheckoutStore) confirm(
	ctx context.Context, tx pgx.Tx, p ConfirmParams,
) (CheckoutResult, error) {
	now := time.Now().UTC()

	// --- 1. Lock the basket -------------------------------------------------
	// FOR UPDATE serialises a double-submitted payment: the second waits, sees
	// a status that is no longer pending, and is told so.
	var (
		order     Order
		expiresAt *time.Time
	)
	err := tx.QueryRow(ctx, `
		SELECT id, order_number, event_id, buyer_user_id, buyer_email, buyer_name,
		       status::text, currency, subtotal_kzt::text, discount_kzt::text,
		       processing_fee_kzt::text, total_kzt::text, placed_at, completed_at,
		       created_at, reserved_until
		  FROM orders WHERE id = $1
		   FOR UPDATE`, p.OrderID,
	).Scan(&order.ID, &order.OrderNumber, &order.EventID, &order.BuyerUserID,
		&order.BuyerEmail, &order.BuyerName, &order.Status, &order.Currency,
		&order.SubtotalKZT, &order.DiscountKZT, &order.ProcessingFeeKZT, &order.TotalKZT,
		&order.PlacedAt, &order.CompletedAt, &order.CreatedAt, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CheckoutResult{}, ErrNotFound
	}
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}

	if order.Status == "expired" {
		return CheckoutResult{}, ErrHoldExpired
	}
	if order.Status != "pending" {
		return CheckoutResult{}, ErrHoldNotPending
	}
	if expiresAt != nil && !now.Before(*expiresAt) {
		// Reported, not released, and deliberately so: this returns an error,
		// which rolls the transaction back - so a release written here would be
		// undone on the way out. Confirm() does it afterwards, in a
		// transaction that actually commits.
		return CheckoutResult{}, ErrHoldExpired
	}

	// --- 1b. The simulated decline (SRS 4.6, 4.10) --------------------------
	// SRS 4.10 requires a "payment failure" notification and SRS 4.6 that
	// "failed or abandoned transactions shall not create valid tickets".
	// Neither could be demonstrated while the simulated gateway always said
	// yes, so it needs a way to say no.
	//
	// The trigger is a reserved buyer address rather than a field in the
	// request body: a client-supplied "payment_outcome" would look like the
	// caller deciding whether their own payment succeeded. A magic address is
	// the idiom the real card sandboxes use, and nobody owns
	// decline.simulator.biletflow.kz.
	//
	// It is checked here, before anything is written. A one-shot checkout
	// rolls the whole transaction back, so a declined card holds no stock; a
	// two-step basket keeps its reservation, so the attendee can try another
	// card without losing their seats.
	declineEmail := p.BuyerEmail
	if declineEmail == "" {
		declineEmail = order.BuyerEmail
	}
	if isPositiveAmount(order.SubtotalKZT) && isDeclineSimulation(declineEmail) {
		return CheckoutResult{}, &PaymentDeclinedError{
			Reason: "The simulated payment provider declined this card.",
		}
	}

	// --- 2. Whose order it is ----------------------------------------------
	if err := tx.QueryRow(ctx, `
		UPDATE orders
		   SET buyer_name    = COALESCE(NULLIF($2, ''), buyer_name),
		       buyer_email   = COALESCE(NULLIF($3, '')::citext, buyer_email),
		       buyer_phone   = COALESCE($4, buyer_phone),
		       buyer_user_id = COALESCE($5, buyer_user_id),
		       placed_at     = COALESCE(placed_at, now())
		 WHERE id = $1
		RETURNING buyer_name, buyer_email::text`,
		p.OrderID, p.BuyerName, p.BuyerEmail, p.BuyerPhone, p.BuyerUserID,
	).Scan(&order.BuyerName, &order.BuyerEmail); err != nil {
		return CheckoutResult{}, mapError(err)
	}

	// --- 3. The lines being sold -------------------------------------------
	items, err := orderItemsFor(ctx, tx, p.OrderID)
	if err != nil {
		return CheckoutResult{}, err
	}
	if len(items) == 0 {
		return CheckoutResult{}, ErrNotFound
	}

	// --- 4. Reserved becomes sold ------------------------------------------
	// The two counters move together, so quantity_sold + quantity_reserved is
	// unchanged and ticket_types_inventory_chk cannot be tripped midway.
	for _, item := range items {
		if _, err := tx.Exec(ctx, `
			UPDATE ticket_types
			   SET quantity_reserved = GREATEST(quantity_reserved - $2::int, 0),
			       quantity_sold     = quantity_sold + $2::int
			 WHERE id = $1`, item.TicketTypeID, item.Quantity); err != nil {
			return CheckoutResult{}, mapError(err)
		}
	}

	// --- 5. The discount, decided entirely by the server --------------------
	applied, err := applyPromo(ctx, tx, p.Promo, p.OrderID, p.BuyerUserID)
	if err != nil {
		return CheckoutResult{}, err
	}

	// --- 6. The processing charge (SRS 3.3) ---------------------------------
	// After the discount, because a fee is charged on what is actually paid.
	if err := chargeFee(ctx, tx, p.OrderID, s.fees); err != nil {
		return CheckoutResult{}, err
	}

	// The simulated payment succeeds immediately, so the order is paid.
	err = tx.QueryRow(ctx, `
		UPDATE orders
		   SET status = 'paid', reserved_until = NULL, completed_at = now()
		 WHERE id = $1
		RETURNING subtotal_kzt::text, discount_kzt::text, processing_fee_kzt::text,
		          total_kzt::text, status::text, placed_at, completed_at`,
		p.OrderID,
	).Scan(&order.SubtotalKZT, &order.DiscountKZT, &order.ProcessingFeeKZT,
		&order.TotalKZT, &order.Status, &order.PlacedAt, &order.CompletedAt)
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}
	if applied != nil {
		applied.DiscountKZT = order.DiscountKZT
	}

	// --- 7. The attendee and their tickets ----------------------------------
	attendee, tickets, err := issueTickets(ctx, tx, order, items)
	if err != nil {
		return CheckoutResult{}, err
	}

	// Seats held for this basket become sold with it.
	if _, err := tx.Exec(ctx, `
		UPDATE seat_holds SET status = 'converted', released_at = now()
		 WHERE order_id = $1 AND status = 'active'`, p.OrderID); err != nil {
		return CheckoutResult{}, mapError(err)
	}

	// --- 8. The simulated payment -------------------------------------------
	payment, err := recordPayment(ctx, tx, order, now)
	if err != nil {
		return CheckoutResult{}, err
	}

	// --- 9. Timeline entry ---------------------------------------------------
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id,
		                        description, metadata)
		VALUES ($1, $2, 'order.created', 'order', $3, $4,
		        jsonb_build_object('simulated', true, 'processing_fee_kzt', $5::text))`,
		order.EventID, p.BuyerUserID, order.ID.String(),
		fmt.Sprintf("Simulated order %s for %s KZT", order.OrderNumber, order.TotalKZT),
		order.ProcessingFeeKZT,
	); err != nil {
		return CheckoutResult{}, mapError(err)
	}

	return CheckoutResult{
		Order: order, Items: items, Attendee: attendee,
		Tickets: tickets, Payment: payment, Promo: applied,
	}, nil
}

// Release cancels a basket and puts its inventory back immediately.
//
// The attendee closing a tab is the common case, and waiting fifteen minutes
// to resell a seat somebody has visibly walked away from is pure waste.
func (s *CheckoutStore) Release(ctx context.Context, orderID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	err = tx.QueryRow(ctx,
		`SELECT status::text FROM orders WHERE id = $1 FOR UPDATE`, orderID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return mapError(err)
	}
	if status != "pending" {
		return ErrHoldNotPending
	}

	if err := releaseHold(ctx, tx, orderID, "cancelled"); err != nil {
		return err
	}
	return mapError(tx.Commit(ctx))
}

// ReleaseExpired puts back the inventory of every basket whose time is up, and
// reports how many it released. This is what the background sweeper calls.
func (s *CheckoutStore) ReleaseExpired(ctx context.Context) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	released, err := releaseExpired(ctx, tx, nil)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, mapError(err)
	}
	return released, nil
}

// GetHold reads a basket back, so a page reloaded mid-checkout can pick up
// where it left off.
func (s *CheckoutStore) GetHold(ctx context.Context, orderID uuid.UUID) (Hold, error) {
	var (
		held      Hold
		expiresAt *time.Time
		status    string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, order_number, event_id, status::text, subtotal_kzt::text, reserved_until
		  FROM orders WHERE id = $1`, orderID,
	).Scan(&held.OrderID, &held.OrderNumber, &held.EventID, &status,
		&held.SubtotalKZT, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, ErrNotFound
	}
	if err != nil {
		return Hold{}, mapError(err)
	}

	held.Status = status
	switch status {
	case "pending":
		// still open
	case "expired":
		// Swept already, either by the timer or by somebody else's purchase.
		// Saying so precisely is what lets the UI offer "pick them again"
		// rather than a flat "this is no longer open".
		return Hold{}, ErrHoldExpired
	default:
		return Hold{}, ErrHoldNotPending
	}

	// Still marked pending, but its time is up. Release it here and now rather
	// than reporting an expiry and leaving the stock reserved until a timer
	// notices: this is the moment somebody discovered they were too slow, and
	// it is the moment the seats should go back on sale.
	//
	// It has to be its own committed transaction. Every path out of here
	// returns an error, and an error rolls back - so a release written into
	// the caller's transaction would be quietly undone.
	if expiresAt == nil || !time.Now().UTC().Before(*expiresAt) {
		if _, err := s.ReleaseExpired(ctx); err != nil {
			return Hold{}, err
		}
		return Hold{}, ErrHoldExpired
	}
	held.ReservedUntil = *expiresAt
	held.SecondsRemaining = secondsUntil(*expiresAt)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Hold{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if held.Items, err = orderItemsFor(ctx, tx, orderID); err != nil {
		return Hold{}, err
	}
	if held.EstimatedFeeKZT, held.EstimatedTotalKZT, err =
		estimateFee(ctx, tx, orderID, s.fees); err != nil {
		return Hold{}, err
	}
	return held, mapError(tx.Commit(ctx))
}

// --- shared helpers ----------------------------------------------------------

// releaseHold puts one basket's inventory back and closes the order.
func releaseHold(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, status string) error {
	if _, err := tx.Exec(ctx, `
		UPDATE ticket_types tt
		   SET quantity_reserved = GREATEST(tt.quantity_reserved - held.qty, 0)
		  FROM (SELECT ticket_type_id, sum(quantity)::int AS qty
		          FROM order_items WHERE order_id = $1
		         GROUP BY ticket_type_id) AS held
		 WHERE tt.id = held.ticket_type_id`, orderID); err != nil {
		return mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE seat_holds SET status = 'released', released_at = now()
		 WHERE order_id = $1 AND status = 'active'`, orderID); err != nil {
		return mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE orders
		   SET status = $2::order_status, reserved_until = NULL, cancelled_at = now()
		 WHERE id = $1`, orderID, status); err != nil {
		return mapError(err)
	}
	return nil
}

// releaseExpired sweeps expired baskets, optionally narrowed to one event.
//
// It is one statement: the orders are closed and their inventory returned
// inside the same data-modifying CTE, so no window exists in which a basket is
// marked expired while its seats are still counted as reserved.
func releaseExpired(ctx context.Context, tx pgx.Tx, eventID *uuid.UUID) (int, error) {
	tag, err := tx.Exec(ctx, `
		WITH expired AS (
		    UPDATE orders
		       SET status = 'expired', reserved_until = NULL, cancelled_at = now()
		     WHERE status = 'pending'
		       AND reserved_until IS NOT NULL
		       AND reserved_until <= now()
		       AND ($1::uuid IS NULL OR event_id = $1)
		    RETURNING id
		),
		seats AS (
		    UPDATE seat_holds SET status = 'expired', released_at = now()
		     WHERE status = 'active' AND order_id IN (SELECT id FROM expired)
		    RETURNING 1
		),
		returned AS (
		    SELECT oi.ticket_type_id, sum(oi.quantity)::int AS qty
		      FROM order_items oi
		      JOIN expired e ON e.id = oi.order_id
		     GROUP BY oi.ticket_type_id
		)
		UPDATE ticket_types tt
		   SET quantity_reserved = GREATEST(tt.quantity_reserved - r.qty, 0)
		  FROM returned r
		 WHERE tt.id = r.ticket_type_id`, eventID)
	if err != nil {
		return 0, mapError(err)
	}
	return int(tag.RowsAffected()), nil
}

func releaseExpiredForEvent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error {
	_, err := releaseExpired(ctx, tx, &eventID)
	return err
}

// estimateFee prices a basket without writing anything.
func estimateFee(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, fees Fees) (string, string, error) {
	var fee, total string
	err := tx.QueryRow(ctx, `
		SELECT f.fee::text, (subtotal_kzt - discount_kzt + f.fee)::text
		  FROM orders,
		       LATERAL (SELECT CASE WHEN subtotal_kzt - discount_kzt > 0
		                            THEN round((subtotal_kzt - discount_kzt)
		                                       * $2::numeric / 100 + $3::numeric, 2)
		                            ELSE 0 END AS fee) AS f
		 WHERE orders.id = $1`, orderID, fees.Percent, fees.FixedKZT).Scan(&fee, &total)
	if err != nil {
		return "", "", mapError(err)
	}
	return fee, total, nil
}

// chargeFee writes the processing charge and the resulting total (SRS 3.3).
//
// orders_total_math_chk holds that total = subtotal - discount + fee, so both
// columns have to move in the same statement or the row would be rejected
// halfway.
func chargeFee(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, fees Fees) error {
	_, err := tx.Exec(ctx, `
		UPDATE orders o
		   SET processing_fee_kzt = f.fee,
		       total_kzt          = o.subtotal_kzt - o.discount_kzt + f.fee
		  FROM (SELECT CASE WHEN subtotal_kzt - discount_kzt > 0
		                    THEN round((subtotal_kzt - discount_kzt)
		                               * $2::numeric / 100 + $3::numeric, 2)
		                    ELSE 0 END AS fee
		          FROM orders WHERE id = $1) AS f
		 WHERE o.id = $1`, orderID, fees.Percent, fees.FixedKZT)
	return mapError(err)
}

// holdSeats reserves specific seats for an assigned-seating basket.
//
// seat_holds_one_active_per_seat_uidx is a unique partial index over active
// holds, so two baskets cannot hold the same seat: the second insert is a
// unique violation, which is reported as the seat being taken (SRS 4.3.1,
// "the system shall prevent two orders from purchasing the same seat").
func holdSeats(
	ctx context.Context, tx pgx.Tx, eventID, orderID uuid.UUID,
	seatIDs []uuid.UUID, expiresAt time.Time,
) error {
	for _, seatID := range seatIDs {
		_, err := tx.Exec(ctx, `
			INSERT INTO seat_holds (seat_id, event_id, order_id, status, expires_at)
			VALUES ($1, $2, $3, 'active', $4)`, seatID, eventID, orderID, expiresAt)
		if isUniqueViolation(err, "seat_holds_one_active_per_seat_uidx") {
			return &SeatTakenError{SeatID: seatID}
		}
		if err != nil {
			return mapError(err)
		}
	}
	return nil
}

// SeatTakenError reports a seat somebody else is holding or has bought.
type SeatTakenError struct {
	SeatID uuid.UUID
}

func (e *SeatTakenError) Error() string {
	return "that seat has just been taken"
}

// secondsUntil is the countdown a client renders, floored at zero.
func secondsUntil(deadline time.Time) int {
	remaining := int(time.Until(deadline).Seconds())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// nullableOrPlaceholder fills a NOT NULL column for a basket that has no buyer
// details yet. The placeholder is overwritten at confirm, and an order that is
// never confirmed is expired rather than kept.
func nullableOrPlaceholder(value, placeholder string) string {
	if value == "" {
		return placeholder
	}
	return value
}

// Sweeper runs ReleaseExpired on a timer.
//
// It is a belt to the braces of the opportunistic release inside hold: that one
// guarantees correctness for the seats somebody is actively trying to buy, and
// this one keeps the counters honest for everything else, so the number an
// event page shows as remaining is right even when nobody is shopping.
type Sweeper struct {
	store    *CheckoutStore
	interval time.Duration
	onSweep  func(released int, err error)
}

// NewSweeper builds a sweeper. A zero interval means the default of one minute.
func NewSweeper(store *CheckoutStore, interval time.Duration, onSweep func(int, error)) *Sweeper {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Sweeper{store: store, interval: interval, onSweep: onSweep}
}

// Run sweeps until the context is cancelled.
func (s *Sweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			released, err := s.store.ReleaseExpired(ctx)
			if s.onSweep != nil && (released > 0 || err != nil) {
				s.onSweep(released, err)
			}
		}
	}
}
