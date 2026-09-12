package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Refund is a completed refund against one order (SRS 4.9).
type Refund struct {
	ID          uuid.UUID  `json:"id"`
	OrderID     uuid.UUID  `json:"order_id"`
	PaymentID   uuid.UUID  `json:"payment_id"`
	AmountKZT   string     `json:"amount_kzt"`
	Status      string     `json:"status"`
	Reason      *string    `json:"reason,omitempty"`
	IsSimulated bool       `json:"is_simulated"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// RefundResult is what the caller needs to answer the request and to notify
// the attendee afterwards.
type RefundResult struct {
	Refund Refund `json:"refund"`
	Order  Order  `json:"order"`
	// VoidedTickets is how many tickets this refund invalidated. Tickets that
	// were already cancelled or refunded are not counted twice.
	VoidedTickets int `json:"voided_tickets"`
	// EventTitle and the buyer's details travel with the result so the caller
	// can build the notification without a second round trip.
	EventTitle string `json:"event_title"`
	BuyerName  string `json:"buyer_name"`
	BuyerEmail string `json:"buyer_email"`
}

// Refund failures that the API turns into specific codes. An organizer
// clicking Refund twice deserves a different answer from one trying to refund
// a cart that was never paid for.
var (
	// ErrAlreadyRefunded reports an order that has already been refunded in
	// full.
	ErrAlreadyRefunded = errors.New("order has already been refunded")
	// ErrOrderNotRefundable reports an order that never completed a payment,
	// so there is nothing to give back.
	ErrOrderNotRefundable = errors.New("order is not in a refundable state")
	// ErrNoSucceededPayment reports a paid order with no succeeded payment
	// row - data that should not exist, but which must not panic if it does.
	ErrNoSucceededPayment = errors.New("order has no succeeded payment to refund")
)

// RefundParams identifies the order and who is refunding it.
type RefundParams struct {
	OrderID uuid.UUID
	ActorID uuid.UUID
	Reason  string
}

// RefundStore performs full refunds.
type RefundStore struct {
	pool *pgxpool.Pool
}

// NewRefundStore builds a RefundStore.
func NewRefundStore(pool *pgxpool.Pool) *RefundStore { return &RefundStore{pool: pool} }

// Refund reverses a paid order in one transaction (SRS 4.9).
//
// Everything moves together or nothing does: the order, its payment, the
// refund record, every issued ticket, the inventory those tickets held, and
// the audit entry. A refund that voided the tickets but left the money marked
// as taken - or vice versa - is exactly the inconsistency a transaction
// exists to prevent.
//
// Only full refunds are supported, which is what SRS 4.9 asks for
// ("initiate full refunds"). Partial refunds would need a per-ticket
// selection UI and a proration policy; the schema carries refunded_kzt and a
// partially_refunded status so that remains open without a migration.
func (s *RefundStore) Refund(ctx context.Context, p RefundParams) (RefundResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RefundResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// --- 1. Lock the order --------------------------------------------------
	// FOR UPDATE serialises two organizers clicking Refund at the same moment:
	// the second waits, then sees status 'refunded' and is told so, rather
	// than writing a second refund row for money that is already back.
	var (
		order      Order
		eventTitle string
	)
	err = tx.QueryRow(ctx, `
		SELECT o.id, o.order_number, o.event_id, o.buyer_user_id, o.buyer_email, o.buyer_name,
		       o.status::text, o.currency, o.subtotal_kzt::text, o.discount_kzt::text,
		       o.processing_fee_kzt::text, o.total_kzt::text, o.placed_at, o.completed_at,
		       o.created_at, e.title
		  FROM orders o
		  JOIN events e ON e.id = o.event_id
		 WHERE o.id = $1
		   FOR UPDATE OF o`, p.OrderID,
	).Scan(&order.ID, &order.OrderNumber, &order.EventID, &order.BuyerUserID,
		&order.BuyerEmail, &order.BuyerName, &order.Status, &order.Currency,
		&order.SubtotalKZT, &order.DiscountKZT, &order.ProcessingFeeKZT, &order.TotalKZT,
		&order.PlacedAt, &order.CompletedAt, &order.CreatedAt, &eventTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefundResult{}, ErrNotFound
	}
	if err != nil {
		return RefundResult{}, mapError(err)
	}

	switch order.Status {
	case "refunded":
		return RefundResult{}, ErrAlreadyRefunded
	case "paid", "completed", "partially_refunded":
		// refundable
	default:
		return RefundResult{}, ErrOrderNotRefundable
	}

	// A free registration files a zero-value payment, so it would otherwise
	// reach the INSERT below and trip refunds_amount_chk, which requires a
	// positive amount - a constraint violation surfacing as a 500 on what is
	// really a user error. It is cancelled instead (SRS 4.9), so say so.
	if isZeroAmount(order.TotalKZT) {
		return RefundResult{}, ErrFreeOrderNotRefundable
	}

	// --- 2. Find the payment being reversed ---------------------------------
	var (
		paymentID uuid.UUID
		paidKZT   string
	)
	err = tx.QueryRow(ctx, `
		SELECT id, amount_kzt::text
		  FROM payments
		 WHERE order_id = $1 AND status = 'succeeded'
		 ORDER BY created_at
		 LIMIT 1
		   FOR UPDATE`, order.ID).Scan(&paymentID, &paidKZT)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefundResult{}, ErrNoSucceededPayment
	}
	if err != nil {
		return RefundResult{}, mapError(err)
	}

	// --- 3. Void the tickets and hand back what they were holding ----------
	// SRS 4.9: "Refunded or cancelled tickets shall become invalid." The gate
	// already refuses a ticket whose status is 'refunded' (Phase 6), so this
	// is what stops a refunded QR at the door. Shared with cancellation,
	// which owes the attendee the same two effects.
	voided, err := voidOrderTickets(ctx, tx, order.ID, "refunded")
	if err != nil {
		return RefundResult{}, err
	}

	// --- 5. The money --------------------------------------------------------
	if _, err := tx.Exec(ctx, `
		UPDATE payments SET status = 'refunded' WHERE id = $1`, paymentID); err != nil {
		return RefundResult{}, mapError(err)
	}

	// refunded_kzt is set from total_kzt in SQL rather than from a value this
	// process computed, so orders_refund_not_above_total_chk cannot be tripped
	// by a rounding difference between Go and PostgreSQL.
	err = tx.QueryRow(ctx, `
		UPDATE orders
		   SET status        = 'refunded',
		       refunded_kzt  = total_kzt,
		       cancelled_at  = now()
		 WHERE id = $1
		RETURNING status::text, total_kzt::text`, order.ID,
	).Scan(&order.Status, &order.TotalKZT)
	if err != nil {
		return RefundResult{}, mapError(err)
	}

	var refund Refund
	reason := nullableString(p.Reason)
	err = tx.QueryRow(ctx, `
		INSERT INTO refunds (payment_id, order_id, amount_kzt, status, reason,
		                     initiated_by, provider_refund_ref, is_simulated, processed_at)
		VALUES ($1, $2, $3::numeric, 'succeeded', $4, $5, $6, true, now())
		RETURNING id, order_id, payment_id, amount_kzt::text, status::text, reason,
		          is_simulated, processed_at, created_at`,
		paymentID, order.ID, order.TotalKZT, reason, p.ActorID, "sim_refund_"+order.OrderNumber,
	).Scan(&refund.ID, &refund.OrderID, &refund.PaymentID, &refund.AmountKZT,
		&refund.Status, &refund.Reason, &refund.IsSimulated, &refund.ProcessedAt,
		&refund.CreatedAt)
	if err != nil {
		return RefundResult{}, mapError(err)
	}

	// --- 6. The audit entry --------------------------------------------------
	// SRS 4.9: "All payment and refund actions shall be recorded in an audit
	// log." audit_logs rejects UPDATE and DELETE by trigger, so this line is
	// permanent once the transaction commits.
	description := fmt.Sprintf("Refunded order %s for %s KZT, voiding %d ticket(s)",
		order.OrderNumber, order.TotalKZT, voided)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id,
		                        description, metadata)
		VALUES ($1, $2, 'order.refunded', 'order', $3, $4,
		        jsonb_build_object('simulated', true, 'amount_kzt', $5::text,
		                           'voided_tickets', $6::int, 'refund_id', $7::text))`,
		order.EventID, p.ActorID, order.ID.String(), description,
		order.TotalKZT, voided, refund.ID.String(),
	); err != nil {
		return RefundResult{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return RefundResult{}, mapError(err)
	}

	return RefundResult{
		Refund:        refund,
		Order:         order,
		VoidedTickets: voided,
		EventTitle:    eventTitle,
		BuyerName:     order.BuyerName,
		BuyerEmail:    order.BuyerEmail,
	}, nil
}

// EventOrder is one row of the organizer's order list.
type EventOrder struct {
	ID          uuid.UUID  `json:"id"`
	OrderNumber string     `json:"order_number"`
	BuyerName   string     `json:"buyer_name"`
	BuyerEmail  string     `json:"buyer_email"`
	Status      string     `json:"status"`
	TotalKZT    string     `json:"total_kzt"`
	DiscountKZT string     `json:"discount_kzt"`
	RefundedKZT string     `json:"refunded_kzt"`
	TicketCount int        `json:"ticket_count"`
	LiveTickets int        `json:"live_tickets"`
	CheckedIn   int        `json:"checked_in"`
	PlacedAt    *time.Time `json:"placed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	// Refundable and Cancellable mirror the rules the two endpoints enforce,
	// so the dashboard can disable a button instead of offering an action that
	// is certain to fail. They are mutually exclusive: money either moved and
	// has to come back, or it never moved and the registration is simply
	// withdrawn (SRS 4.9).
	Refundable  bool `json:"refundable"`
	Cancellable bool `json:"cancellable"`
}

// ListEventOrders returns an event's orders, newest first, for the organizer's
// attendee view.
func (s *RefundStore) ListEventOrders(ctx context.Context, eventID uuid.UUID, limit int) ([]EventOrder, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.order_number, o.buyer_name, o.buyer_email, o.status::text,
		       o.total_kzt::text, o.discount_kzt::text, o.refunded_kzt::text,
		       count(t.id)                                              AS ticket_count,
		       count(t.id) FILTER (WHERE t.status IN ('valid','checked_in')) AS live_tickets,
		       count(t.id) FILTER (WHERE t.status = 'checked_in')        AS checked_in,
		       o.placed_at, o.created_at
		  FROM orders o
		  LEFT JOIN tickets t ON t.order_id = o.id
		 WHERE o.event_id = $1
		 GROUP BY o.id
		 ORDER BY o.created_at DESC
		 LIMIT $2`, eventID, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	orders := []EventOrder{}
	for rows.Next() {
		var o EventOrder
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.BuyerName, &o.BuyerEmail, &o.Status,
			&o.TotalKZT, &o.DiscountKZT, &o.RefundedKZT, &o.TicketCount,
			&o.LiveTickets, &o.CheckedIn, &o.PlacedAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		switch o.Status {
		case "paid", "completed", "partially_refunded":
			if isZeroAmount(o.TotalKZT) {
				o.Cancellable = true
			} else {
				o.Refundable = true
			}
		case "pending", "awaiting_payment":
			if isZeroAmount(o.TotalKZT) {
				o.Cancellable = true
			}
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
