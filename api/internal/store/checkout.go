package store

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Order mirrors a row of the orders table. Money is carried as a decimal string
// so a numeric(14,2) never passes through a float64.
type Order struct {
	ID               uuid.UUID  `json:"id"`
	OrderNumber      string     `json:"order_number"`
	EventID          uuid.UUID  `json:"event_id"`
	BuyerUserID      *uuid.UUID `json:"buyer_user_id,omitempty"`
	BuyerEmail       string     `json:"buyer_email"`
	BuyerName        string     `json:"buyer_name"`
	Status           string     `json:"status"`
	Currency         string     `json:"currency"`
	SubtotalKZT      string     `json:"subtotal_kzt"`
	DiscountKZT      string     `json:"discount_kzt"`
	ProcessingFeeKZT string     `json:"processing_fee_kzt"`
	TotalKZT         string     `json:"total_kzt"`
	PlacedAt         *time.Time `json:"placed_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// OrderItem is one line of an order.
type OrderItem struct {
	ID             uuid.UUID `json:"id"`
	OrderID        uuid.UUID `json:"order_id"`
	TicketTypeID   uuid.UUID `json:"ticket_type_id"`
	TicketTypeName string    `json:"ticket_type_name"`
	Quantity       int       `json:"quantity"`
	UnitPriceKZT   string    `json:"unit_price_kzt"`
	LineTotalKZT   string    `json:"line_total_kzt"`
}

// Attendee is the person the tickets are issued to.
type Attendee struct {
	ID       uuid.UUID `json:"id"`
	OrderID  uuid.UUID `json:"order_id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

// Ticket is one issued admission ticket.
type Ticket struct {
	ID             uuid.UUID `json:"id"`
	TicketCode     string    `json:"ticket_code"`
	QRToken        string    `json:"qr_token"`
	TicketTypeID   uuid.UUID `json:"ticket_type_id"`
	TicketTypeName string    `json:"ticket_type_name"`
	Status         string    `json:"status"`
	IssuedAt       time.Time `json:"issued_at"`
}

// Payment is the simulated payment recorded against the order.
type Payment struct {
	ID          uuid.UUID `json:"id"`
	AmountKZT   string    `json:"amount_kzt"`
	Status      string    `json:"status"`
	Provider    string    `json:"provider"`
	IsSimulated bool      `json:"is_simulated"`
	PaidAt      time.Time `json:"paid_at"`
}

// AppliedPromo describes the discount an order actually received.
type AppliedPromo struct {
	CampaignID   uuid.UUID `json:"campaign_id"`
	CampaignName string    `json:"campaign_name"`
	Code         string    `json:"code"`
	DiscountKZT  string    `json:"discount_kzt"`
}

// CheckoutResult is everything the confirmation screen needs.
type CheckoutResult struct {
	Order    Order         `json:"order"`
	Items    []OrderItem   `json:"items"`
	Attendee Attendee      `json:"attendee"`
	Tickets  []Ticket      `json:"tickets"`
	Payment  Payment       `json:"payment"`
	Promo    *AppliedPromo `json:"promo,omitempty"`
}

// CheckoutItem is one requested line.
type CheckoutItem struct {
	TicketTypeID uuid.UUID
	Quantity     int
}

// CheckoutParams describes a purchase attempt.
type CheckoutParams struct {
	EventID     uuid.UUID
	BuyerUserID *uuid.UUID
	BuyerName   string
	BuyerEmail  string
	BuyerPhone  *string
	Items       []CheckoutItem

	// Promo is the campaign to apply, already resolved and validated by the
	// caller. It is re-checked here under a row lock, because validation and
	// purchase are separate moments and the last redemption may have gone in
	// between them.
	Promo *Campaign
}

// InsufficientInventoryError reports that a ticket type cannot cover the
// request. It carries the numbers so the API can say exactly what is left.
type InsufficientInventoryError struct {
	TicketTypeID   uuid.UUID
	TicketTypeName string
	Requested      int
	Remaining      int
}

func (e *InsufficientInventoryError) Error() string {
	if e.Remaining <= 0 {
		return fmt.Sprintf("%q is sold out", e.TicketTypeName)
	}
	return fmt.Sprintf("only %d ticket(s) left for %q, but %d were requested",
		e.Remaining, e.TicketTypeName, e.Requested)
}

// NotOnSaleError reports a ticket type that exists but cannot be bought now.
type NotOnSaleError struct {
	TicketTypeID   uuid.UUID
	TicketTypeName string
	Reason         string
}

func (e *NotOnSaleError) Error() string {
	return fmt.Sprintf("%q is not on sale: %s", e.TicketTypeName, e.Reason)
}

// ExceedsMaxPerOrderError reports a request above the per-order limit.
type ExceedsMaxPerOrderError struct {
	TicketTypeID   uuid.UUID
	TicketTypeName string
	Requested      int
	MaxPerOrder    int
}

func (e *ExceedsMaxPerOrderError) Error() string {
	return fmt.Sprintf("at most %d ticket(s) of %q may be bought in one order, but %d were requested",
		e.MaxPerOrder, e.TicketTypeName, e.Requested)
}

// PaidSalesNotActiveError reports a paid ticket bought before the organizer
// finished the activation checklist (SRS 4.5).
type PaidSalesNotActiveError struct {
	// Status is the activation's current state, which decides whether the
	// attendee is told "not yet on sale" or "suspended".
	Status string
}

func (e *PaidSalesNotActiveError) Error() string {
	if e.Status == ActivationSuspended {
		return "paid ticket sales for this event have been suspended"
	}
	return "paid ticket sales for this event have not been activated yet"
}

// isPositiveAmount reports whether a decimal money string is above zero.
//
// The string comes from numeric(14,2), so it is always a plain decimal - but
// it is parsed rather than compared against "0.00" because "0" and "0.000"
// are the same amount written differently.
func isPositiveAmount(s string) bool {
	amount, ok := new(big.Rat).SetString(s)
	if !ok {
		// An unparseable price is not a licence to sell it for free.
		return true
	}
	return amount.Sign() > 0
}

// CheckoutStore performs the purchase transaction.
type CheckoutStore struct {
	pool *pgxpool.Pool
}

// NewCheckoutStore builds a CheckoutStore.
func NewCheckoutStore(pool *pgxpool.Pool) *CheckoutStore { return &CheckoutStore{pool: pool} }

// lockedTicketType is the inventory snapshot read under a row lock.
type lockedTicketType struct {
	id           uuid.UUID
	name         string
	priceKZT     string
	remaining    int
	maxPerOrder  int
	isHidden     bool
	salesStartAt *time.Time
	salesEndAt   *time.Time
}

// Checkout sells tickets in a single transaction.
//
// Overselling is prevented by taking a row lock on each ticket type with
// SELECT ... FOR UPDATE before reading the remaining count. Concurrent
// checkouts for the same type therefore queue behind each other instead of
// both reading the same "remaining" and both succeeding. Rows are locked in a
// deterministic id order so two orders touching the same pair of ticket types
// cannot deadlock.
//
// The ticket_types_inventory_chk constraint is the backstop: even if this
// logic were wrong, the database would refuse to record the oversold row.
func (s *CheckoutStore) Checkout(ctx context.Context, p CheckoutParams) (CheckoutResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CheckoutResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()

	// --- 1. Lock the inventory rows this order touches -----------------------
	ids := make([]uuid.UUID, 0, len(p.Items))
	for _, item := range p.Items {
		ids = append(ids, item.TicketTypeID)
	}

	rows, err := tx.Query(ctx, `
		SELECT id, name, price_kzt::text,
		       quantity_total - quantity_sold - quantity_reserved AS remaining,
		       max_per_order, is_hidden, sales_start_at, sales_end_at
		  FROM ticket_types
		 WHERE id = ANY($1) AND event_id = $2
		 ORDER BY id
		   FOR UPDATE`, ids, p.EventID)
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}

	locked := map[uuid.UUID]lockedTicketType{}
	for rows.Next() {
		var t lockedTicketType
		if err := rows.Scan(&t.id, &t.name, &t.priceKZT, &t.remaining,
			&t.maxPerOrder, &t.isHidden, &t.salesStartAt, &t.salesEndAt); err != nil {
			rows.Close()
			return CheckoutResult{}, err
		}
		locked[t.id] = t
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return CheckoutResult{}, mapError(err)
	}

	// --- 2. Validate every line against the locked snapshot ------------------
	for _, item := range p.Items {
		t, ok := locked[item.TicketTypeID]
		if !ok {
			return CheckoutResult{}, ErrNotFound
		}
		if t.isHidden {
			return CheckoutResult{}, &NotOnSaleError{t.id, t.name, "it is not currently offered"}
		}
		if t.salesStartAt != nil && now.Before(*t.salesStartAt) {
			return CheckoutResult{}, &NotOnSaleError{t.id, t.name, "sales have not opened yet"}
		}
		if t.salesEndAt != nil && !now.Before(*t.salesEndAt) {
			return CheckoutResult{}, &NotOnSaleError{t.id, t.name, "sales have closed"}
		}
		if item.Quantity > t.maxPerOrder {
			return CheckoutResult{}, &ExceedsMaxPerOrderError{t.id, t.name, item.Quantity, t.maxPerOrder}
		}
		if item.Quantity > t.remaining {
			return CheckoutResult{}, &InsufficientInventoryError{t.id, t.name, item.Quantity, t.remaining}
		}
	}

	// --- 2b. Paid tickets need an activated event (SRS 4.5) ------------------
	// The check lives inside the transaction, next to the prices it depends
	// on. Reading the activation in the handler instead would leave a window
	// where an admin suspends paid sales between the check and the sale.
	//
	// Free tickets are unaffected: activation exists to gate money, and a free
	// registration takes none.
	paid := false
	for _, item := range p.Items {
		if isPositiveAmount(locked[item.TicketTypeID].priceKZT) {
			paid = true
			break
		}
	}
	if paid {
		var activationStatus string
		err := tx.QueryRow(ctx, `
			SELECT status::text FROM paid_sales_activations WHERE event_id = $1`,
			p.EventID).Scan(&activationStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			activationStatus = ActivationNotStarted
		} else if err != nil {
			return CheckoutResult{}, mapError(err)
		}
		if activationStatus != ActivationActive {
			return CheckoutResult{}, &PaidSalesNotActiveError{Status: activationStatus}
		}
	}

	// --- 2c. The simulated decline (SRS 4.6, 4.10) --------------------------
	// SRS 4.10 requires a "payment failure" notification, and SRS 4.6 requires
	// that "Failed or abandoned transactions shall not create valid tickets".
	// Neither could be demonstrated while the simulated gateway always
	// succeeded, so it needs a way to say no.
	//
	// The trigger is a reserved buyer address rather than a field in the
	// request body: a client-supplied "payment_outcome" would look like the
	// caller deciding whether their own payment succeeded. A magic address is
	// the same idiom the real card sandboxes use, and it cannot be reached by
	// accident - nobody owns decline.simulator.biletflow.kz.
	//
	// This lands before the inventory is taken, so a declined payment never
	// holds stock away from a buyer whose payment would have gone through.
	if paid && isDeclineSimulation(p.BuyerEmail) {
		return CheckoutResult{}, &PaymentDeclinedError{
			Reason: "The simulated payment provider declined this card.",
		}
	}

	// --- 3. Take the inventory ----------------------------------------------
	// The WHERE clause repeats the inventory test so the decrement can never
	// exceed what is available, even if the snapshot above were somehow stale.
	for _, item := range p.Items {
		tag, err := tx.Exec(ctx, `
			UPDATE ticket_types
			   SET quantity_sold = quantity_sold + $2
			 WHERE id = $1
			   AND quantity_sold + quantity_reserved + $2 <= quantity_total`,
			item.TicketTypeID, item.Quantity)
		if err != nil {
			return CheckoutResult{}, mapError(err)
		}
		if tag.RowsAffected() == 0 {
			t := locked[item.TicketTypeID]
			return CheckoutResult{}, &InsufficientInventoryError{t.id, t.name, item.Quantity, t.remaining}
		}
	}

	// --- 4. Record the order -------------------------------------------------
	orderNumber, err := newCode("BF", "-", 10)
	if err != nil {
		return CheckoutResult{}, err
	}

	var order Order
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (order_number, event_id, buyer_user_id, buyer_email, buyer_name,
		                    buyer_phone, status, subtotal_kzt, discount_kzt,
		                    processing_fee_kzt, total_kzt, placed_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', 0, 0, 0, 0, $7, $7)
		RETURNING id, order_number, event_id, buyer_user_id, buyer_email, buyer_name,
		          status::text, currency, subtotal_kzt::text, discount_kzt::text,
		          processing_fee_kzt::text, total_kzt::text, placed_at, completed_at, created_at`,
		orderNumber, p.EventID, p.BuyerUserID, p.BuyerEmail, p.BuyerName, p.BuyerPhone, now,
	).Scan(&order.ID, &order.OrderNumber, &order.EventID, &order.BuyerUserID,
		&order.BuyerEmail, &order.BuyerName, &order.Status, &order.Currency,
		&order.SubtotalKZT, &order.DiscountKZT, &order.ProcessingFeeKZT, &order.TotalKZT,
		&order.PlacedAt, &order.CompletedAt, &order.CreatedAt)
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}

	// --- 5. Order items, with the money arithmetic done by PostgreSQL --------
	items := make([]OrderItem, 0, len(p.Items))
	for _, item := range p.Items {
		t := locked[item.TicketTypeID]

		var oi OrderItem
		err := tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, ticket_type_id, quantity,
			                         unit_price_kzt, discount_kzt, line_total_kzt)
			-- Both uses of $3 are cast, otherwise PostgreSQL cannot deduce one
			-- type for a parameter that is an integer column and a numeric
			-- operand in the same statement (SQLSTATE 42P08).
			VALUES ($1, $2, $3::int, $4::numeric, 0, $4::numeric * $3::int)
			RETURNING id, order_id, ticket_type_id, quantity,
			          unit_price_kzt::text, line_total_kzt::text`,
			order.ID, item.TicketTypeID, item.Quantity, t.priceKZT,
		).Scan(&oi.ID, &oi.OrderID, &oi.TicketTypeID, &oi.Quantity,
			&oi.UnitPriceKZT, &oi.LineTotalKZT)
		if err != nil {
			return CheckoutResult{}, mapError(err)
		}
		oi.TicketTypeName = t.name
		items = append(items, oi)
	}

	// Totals are summed in SQL for the same reason: numeric all the way.
	//
	// The subtotal has to land before any discount does: the order row carries
	// both orders_discount_not_above_subtotal_chk and orders_total_math_chk, so
	// writing a discount against a still-zero subtotal would be rejected. Every
	// statement below leaves the row satisfying both.
	err = tx.QueryRow(ctx, `
		UPDATE orders
		   SET subtotal_kzt = sums.subtotal,
		       total_kzt    = sums.subtotal - discount_kzt + processing_fee_kzt
		  FROM (SELECT COALESCE(sum(line_total_kzt), 0) AS subtotal
		          FROM order_items WHERE order_id = $1) AS sums
		 WHERE orders.id = $1
		RETURNING subtotal_kzt::text, total_kzt::text`,
		order.ID,
	).Scan(&order.SubtotalKZT, &order.TotalKZT)
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}

	// --- 5b. The promo discount, decided entirely by the server -------------
	applied, err := applyPromo(ctx, tx, p.Promo, order.ID, p.BuyerUserID)
	if err != nil {
		return CheckoutResult{}, err
	}

	// The simulated payment succeeds immediately, so the order is paid.
	err = tx.QueryRow(ctx, `
		UPDATE orders SET status = 'paid' WHERE id = $1
		RETURNING subtotal_kzt::text, discount_kzt::text, total_kzt::text, status::text`,
		order.ID,
	).Scan(&order.SubtotalKZT, &order.DiscountKZT, &order.TotalKZT, &order.Status)
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}
	if applied != nil {
		applied.DiscountKZT = order.DiscountKZT
	}

	// --- 6. The attendee and their tickets ----------------------------------
	var attendee Attendee
	err = tx.QueryRow(ctx, `
		INSERT INTO attendees (order_id, user_id, full_name, email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, order_id, full_name, email`,
		order.ID, p.BuyerUserID, p.BuyerName, p.BuyerEmail,
	).Scan(&attendee.ID, &attendee.OrderID, &attendee.FullName, &attendee.Email)
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}

	tickets := []Ticket{}
	for i, item := range p.Items {
		t := locked[item.TicketTypeID]

		for n := 0; n < item.Quantity; n++ {
			ticketCode, err := newCode("BF-TKT", "-", 10)
			if err != nil {
				return CheckoutResult{}, err
			}
			// The TKT_ prefix is enforced by tickets_qr_token_prefix_chk and is
			// what keeps an admission QR distinct from a campaign QR (SRS 4.14).
			//
			// The body is a fresh random UUID, not the ticket's own id: a ticket
			// id travels in URLs and logs, and it must not double as a working
			// admission credential.
			qrToken := "TKT_" + uuid.NewString()

			var ticket Ticket
			err = tx.QueryRow(ctx, `
				INSERT INTO tickets (ticket_code, order_id, order_item_id, event_id,
				                     ticket_type_id, attendee_id, qr_token, status)
				VALUES ($1, $2, $3, $4, $5, $6, $7, 'valid')
				RETURNING id, ticket_code, qr_token, ticket_type_id, status::text, issued_at`,
				ticketCode, order.ID, items[i].ID, p.EventID,
				item.TicketTypeID, attendee.ID, qrToken,
			).Scan(&ticket.ID, &ticket.TicketCode, &ticket.QRToken,
				&ticket.TicketTypeID, &ticket.Status, &ticket.IssuedAt)
			if err != nil {
				return CheckoutResult{}, mapError(err)
			}
			ticket.TicketTypeName = t.name
			tickets = append(tickets, ticket)
		}
	}

	// --- 7. The simulated payment -------------------------------------------
	// is_simulated defaults to true in the schema and is set explicitly here:
	// SRS 4.6 requires that demonstration payments are never presented as real
	// financial transactions.
	var payment Payment
	err = tx.QueryRow(ctx, `
		INSERT INTO payments (purpose, order_id, payer_user_id, amount_kzt, status,
		                      provider, provider_payment_ref, is_simulated, paid_at)
		VALUES ('ticket_order', $1, $2, $3::numeric, 'succeeded', 'simulated', $4, true, $5)
		RETURNING id, amount_kzt::text, status::text, provider, is_simulated, paid_at`,
		order.ID, p.BuyerUserID, order.TotalKZT, "sim_"+order.OrderNumber, now,
	).Scan(&payment.ID, &payment.AmountKZT, &payment.Status,
		&payment.Provider, &payment.IsSimulated, &payment.PaidAt)
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}

	// --- 8. Timeline entry ---------------------------------------------------
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description, metadata)
		VALUES ($1, $2, 'order.created', 'order', $3, $4, jsonb_build_object('simulated', true))`,
		p.EventID, p.BuyerUserID, order.ID.String(),
		fmt.Sprintf("Simulated order %s for %s KZT", order.OrderNumber, order.TotalKZT),
	); err != nil {
		return CheckoutResult{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return CheckoutResult{}, mapError(err)
	}

	return CheckoutResult{
		Order: order, Items: items, Attendee: attendee,
		Tickets: tickets, Payment: payment, Promo: applied,
	}, nil
}

// applyPromo redeems a campaign against an order, inside the checkout
// transaction.
//
// The campaign row is locked with FOR UPDATE and its limit re-checked here,
// not just when the attendee typed the code: validation and payment are
// separate moments, and the last redemption may have been taken in between.
// campaigns_redemption_limit_chk is the backstop if this logic were ever wrong.
func applyPromo(
	ctx context.Context, tx pgx.Tx, promo *Campaign, orderID uuid.UUID, userID *uuid.UUID,
) (*AppliedPromo, error) {
	if promo == nil {
		return nil, nil
	}

	var (
		discountType   string
		discountValue  string
		maxRedemptions *int
		redeemed       int
		status         string
	)
	err := tx.QueryRow(ctx, `
		SELECT discount_type::text, discount_value::text, max_redemptions,
		       redemption_count, status::text
		  FROM campaigns WHERE id = $1 FOR UPDATE`, promo.ID,
	).Scan(&discountType, &discountValue, &maxRedemptions, &redeemed, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPromoNotFound
	}
	if err != nil {
		return nil, mapError(err)
	}

	// Exhaustion is checked before the general status, and both before the
	// discount is computed. The winner of a race marks the campaign exhausted,
	// so everyone behind it must be told "fully redeemed" rather than the
	// vaguer "not active" - the attendee needs to know the code was real but
	// is gone, not that they mistyped it.
	if status == CampaignExhausted ||
		(maxRedemptions != nil && redeemed >= *maxRedemptions) {
		return nil, ErrPromoExhausted
	}
	if status != CampaignActive {
		return nil, ErrPromoNotActive
	}

	// The discount is computed by PostgreSQL over the order's own lines, so no
	// amount is ever taken from the client and no float touches the money.
	// Only lines the campaign covers count towards it.
	var discount string
	err = tx.QueryRow(ctx, `
		WITH eligible AS (
			SELECT COALESCE(sum(oi.line_total_kzt), 0) AS base
			  FROM order_items oi
			 WHERE oi.order_id = $1
			   AND (NOT EXISTS (SELECT 1 FROM campaign_ticket_types WHERE campaign_id = $2)
			        OR oi.ticket_type_id IN (
			             SELECT ticket_type_id FROM campaign_ticket_types WHERE campaign_id = $2))
		)
		SELECT CASE
		         WHEN $3 = 'percentage' THEN round(base * $4::numeric / 100, 2)
		         ELSE least($4::numeric, base)
		       END::text
		  FROM eligible`,
		orderID, promo.ID, discountType, discountValue).Scan(&discount)
	if err != nil {
		return nil, mapError(err)
	}

	// discount and total move together: orders_total_math_chk requires
	// total = subtotal - discount + fee to hold after every statement.
	if _, err := tx.Exec(ctx, `
		UPDATE orders
		   SET discount_kzt = $2::numeric,
		       total_kzt    = subtotal_kzt - $2::numeric + processing_fee_kzt,
		       campaign_id  = $3,
		       promo_code_id = $4
		 WHERE id = $1`, orderID, discount, promo.ID, promo.CodeID); err != nil {
		return nil, mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO promo_redemptions (campaign_id, promo_code_id, order_id, user_id, discount_kzt)
		VALUES ($1, $2, $3, $4, $5::numeric)`,
		promo.ID, promo.CodeID, orderID, userID, discount); err != nil {
		return nil, mapError(err)
	}

	// Increment last, so the constraint fires here if anything slipped through.
	var newCount int
	if err := tx.QueryRow(ctx, `
		UPDATE campaigns SET redemption_count = redemption_count + 1
		 WHERE id = $1
		RETURNING redemption_count`, promo.ID).Scan(&newCount); err != nil {
		var constraintErr *ConstraintError
		mapped := mapError(err)
		if errors.As(mapped, &constraintErr) &&
			constraintErr.Constraint == "campaigns_redemption_limit_chk" {
			return nil, ErrPromoExhausted
		}
		return nil, mapped
	}

	// A campaign that has just run out is marked so, which is what makes the
	// organizer dashboard and the checkout agree without a background job.
	if maxRedemptions != nil && newCount >= *maxRedemptions {
		if _, err := tx.Exec(ctx,
			`UPDATE campaigns SET status = 'exhausted' WHERE id = $1`, promo.ID); err != nil {
			return nil, mapError(err)
		}
	}

	return &AppliedPromo{
		CampaignID:   promo.ID,
		CampaignName: promo.Name,
		Code:         promo.Code,
	}, nil
}

// GetOrder returns a previously placed order with its items and tickets.
func (s *CheckoutStore) GetOrder(ctx context.Context, id uuid.UUID) (CheckoutResult, error) {
	var order Order
	err := s.pool.QueryRow(ctx, `
		SELECT id, order_number, event_id, buyer_user_id, buyer_email, buyer_name,
		       status::text, currency, subtotal_kzt::text, discount_kzt::text,
		       processing_fee_kzt::text, total_kzt::text, placed_at, completed_at, created_at
		  FROM orders WHERE id = $1`, id,
	).Scan(&order.ID, &order.OrderNumber, &order.EventID, &order.BuyerUserID,
		&order.BuyerEmail, &order.BuyerName, &order.Status, &order.Currency,
		&order.SubtotalKZT, &order.DiscountKZT, &order.ProcessingFeeKZT, &order.TotalKZT,
		&order.PlacedAt, &order.CompletedAt, &order.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CheckoutResult{}, ErrNotFound
	}
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}

	itemRows, err := s.pool.Query(ctx, `
		SELECT oi.id, oi.order_id, oi.ticket_type_id, tt.name, oi.quantity,
		       oi.unit_price_kzt::text, oi.line_total_kzt::text
		  FROM order_items oi
		  JOIN ticket_types tt ON tt.id = oi.ticket_type_id
		 WHERE oi.order_id = $1
		 ORDER BY oi.created_at`, id)
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}
	defer itemRows.Close()

	items := []OrderItem{}
	for itemRows.Next() {
		var oi OrderItem
		if err := itemRows.Scan(&oi.ID, &oi.OrderID, &oi.TicketTypeID, &oi.TicketTypeName,
			&oi.Quantity, &oi.UnitPriceKZT, &oi.LineTotalKZT); err != nil {
			return CheckoutResult{}, err
		}
		items = append(items, oi)
	}
	if err := itemRows.Err(); err != nil {
		return CheckoutResult{}, mapError(err)
	}

	ticketRows, err := s.pool.Query(ctx, `
		SELECT t.id, t.ticket_code, t.qr_token, t.ticket_type_id, tt.name,
		       t.status::text, t.issued_at
		  FROM tickets t
		  JOIN ticket_types tt ON tt.id = t.ticket_type_id
		 WHERE t.order_id = $1
		 ORDER BY t.issued_at, t.ticket_code`, id)
	if err != nil {
		return CheckoutResult{}, mapError(err)
	}
	defer ticketRows.Close()

	tickets := []Ticket{}
	for ticketRows.Next() {
		var t Ticket
		if err := ticketRows.Scan(&t.ID, &t.TicketCode, &t.QRToken, &t.TicketTypeID,
			&t.TicketTypeName, &t.Status, &t.IssuedAt); err != nil {
			return CheckoutResult{}, err
		}
		tickets = append(tickets, t)
	}
	if err := ticketRows.Err(); err != nil {
		return CheckoutResult{}, mapError(err)
	}

	return CheckoutResult{Order: order, Items: items, Tickets: tickets}, nil
}

// newCode returns prefix + separator + `length` random base32 characters,
// using crypto-quality randomness so codes cannot be guessed or enumerated.
func newCode(prefix, separator string, length int) (string, error) {
	// base32 packs 5 bits per character, so this is comfortably enough entropy.
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	return prefix + separator + encoded[:length], nil
}

// DeclineSimulationDomain is the reserved address suffix that makes the
// simulated payment provider decline (SRS 4.6: payments are "a clearly
// labelled internal simulation").
//
// It is a domain nobody can own, so a real attendee cannot trip it by typing
// their own address.
const DeclineSimulationDomain = "@decline.simulator.biletflow.kz"

// PaymentDeclinedError reports a simulated payment the provider refused.
type PaymentDeclinedError struct {
	Reason string
}

func (e *PaymentDeclinedError) Error() string {
	return "payment declined: " + e.Reason
}

func isDeclineSimulation(email string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(email)), DeclineSimulationDomain)
}
