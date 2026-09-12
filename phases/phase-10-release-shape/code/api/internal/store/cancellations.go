package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Cancelling a free registration (SRS 4.9).
//
// This is deliberately a separate verb from a refund rather than a special
// case inside it. A refund reverses money: it writes a refunds row, and
// refunds_amount_chk requires that row to carry a positive amount. A free
// registration has no money to reverse, so routing it through the refund path
// tripped that constraint and surfaced as a 500. The two operations also mean
// different things to the person receiving the email - "your money is coming
// back" versus "you are off the list" - so they get different notifications.
//
// What they share is the consequence: SRS 4.9 says refunded *or cancelled*
// tickets become invalid, so both void the tickets and hand the inventory
// back. That shared part lives in voidOrderTickets.
var (
	// ErrAlreadyCancelled reports an order that is already cancelled.
	ErrAlreadyCancelled = errors.New("order has already been cancelled")
	// ErrPaidOrderNeedsRefund reports an attempt to cancel an order that took
	// money. Writing a paid order off as "cancelled" would leave the payment
	// standing with no refund against it.
	ErrPaidOrderNeedsRefund = errors.New("a paid order must be refunded, not cancelled")
	// ErrOrderNotCancellable reports an order in a state where cancellation is
	// meaningless - already refunded, expired, or failed.
	ErrOrderNotCancellable = errors.New("order is not in a cancellable state")
	// ErrFreeOrderNotRefundable reports a zero-value order sent to the refund
	// endpoint. It is cancellable instead.
	ErrFreeOrderNotRefundable = errors.New("a free registration is cancelled, not refunded")
)

// CancelParams identifies the registration and who is cancelling it.
type CancelParams struct {
	OrderID uuid.UUID
	ActorID uuid.UUID
	Reason  string
}

// CancelResult is what the caller needs to answer the request and notify the
// attendee afterwards.
type CancelResult struct {
	Order Order `json:"order"`
	// CancelledTickets is how many tickets this cancellation invalidated.
	CancelledTickets int    `json:"cancelled_tickets"`
	EventTitle       string `json:"event_title"`
	BuyerName        string `json:"buyer_name"`
	BuyerEmail       string `json:"buyer_email"`
}

// voidOrderTickets moves an order's live tickets to the given status and hands
// the inventory they were holding back to their ticket types. It is the half
// of a refund that a cancellation also needs.
//
// checked_in tickets are voided too: somebody who was admitted and then had
// their registration cancelled must not be readmitted on a second scan.
// checked_in_at is cleared with the status because
// tickets_checked_in_consistency_chk holds that the column is set if and only
// if the ticket is currently checked in. The check_in_records row is left
// exactly as it was - the person really did walk through the door.
func voidOrderTickets(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, status string) (int, error) {
	rows, err := tx.Query(ctx, `
		UPDATE tickets
		   SET status = $2::ticket_status, checked_in_at = NULL
		 WHERE order_id = $1
		   AND status IN ('valid', 'checked_in')
		RETURNING ticket_type_id`, orderID, status)
	if err != nil {
		return 0, mapError(err)
	}

	perType := map[uuid.UUID]int{}
	voided := 0
	for rows.Next() {
		var typeID uuid.UUID
		if err := rows.Scan(&typeID); err != nil {
			rows.Close()
			return 0, err
		}
		perType[typeID]++
		voided++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, mapError(err)
	}

	// quantity_sold tracks live tickets, so a voided ticket has to release the
	// inventory it was holding or the event would sell out against tickets
	// that no longer admit anybody. quantity_refunded keeps the history that
	// the decrement would otherwise erase - it counts places given back,
	// whether by refund or by cancellation.
	for typeID, n := range perType {
		if _, err := tx.Exec(ctx, `
			UPDATE ticket_types
			   SET quantity_sold     = GREATEST(quantity_sold - $2::int, 0),
			       quantity_refunded = quantity_refunded + $2::int
			 WHERE id = $1`, typeID, n); err != nil {
			return 0, mapError(err)
		}
	}

	return voided, nil
}

// Cancel voids a free registration in one transaction (SRS 4.9).
//
// The order, its tickets, the inventory those tickets held and the audit entry
// all move together or none of them do.
func (s *RefundStore) Cancel(ctx context.Context, p CancelParams) (CancelResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CancelResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// FOR UPDATE serialises two organizers clicking Cancel at the same moment:
	// the second waits, sees status 'cancelled' and is told so, rather than
	// releasing the same place into inventory twice.
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
		return CancelResult{}, ErrNotFound
	}
	if err != nil {
		return CancelResult{}, mapError(err)
	}

	switch order.Status {
	case "cancelled":
		return CancelResult{}, ErrAlreadyCancelled
	case "pending", "awaiting_payment", "paid", "completed":
		// candidates - the money question is decided below
	default:
		// refunded, partially_refunded, expired, failed
		return CancelResult{}, ErrOrderNotCancellable
	}

	// The money test is the total, not the presence of a payment row: a free
	// checkout still files a zero-value simulated payment, so "has a payment"
	// would wrongly classify every free registration as paid.
	if !isZeroAmount(order.TotalKZT) {
		return CancelResult{}, ErrPaidOrderNeedsRefund
	}

	cancelled, err := voidOrderTickets(ctx, tx, order.ID, "cancelled")
	if err != nil {
		return CancelResult{}, err
	}

	// The zero-value payment row is deliberately left alone. payment_status
	// has no 'cancelled' member, and the two values that do exist would both
	// be untrue: 'refunded' claims money went back, 'failed' claims the
	// transaction never completed. The order's own status is the authoritative
	// state, and analytics reads orders and tickets, not payments.

	err = tx.QueryRow(ctx, `
		UPDATE orders
		   SET status       = 'cancelled',
		       cancelled_at = now()
		 WHERE id = $1
		RETURNING status::text`, order.ID).Scan(&order.Status)
	if err != nil {
		return CancelResult{}, mapError(err)
	}

	// SRS 4.9 requires the action in the audit log; SRS 4.16 makes that log
	// append-only, so this line is permanent once the transaction commits.
	noun := "tickets"
	if cancelled == 1 {
		noun = "ticket"
	}
	description := fmt.Sprintf("Cancelled free registration %s, voiding %d %s",
		order.OrderNumber, cancelled, noun)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id,
		                        description, metadata)
		VALUES ($1, $2, 'order.cancelled', 'order', $3, $4,
		        jsonb_build_object('free', true, 'cancelled_tickets', $5::int,
		                           'reason', $6::text))`,
		order.EventID, p.ActorID, order.ID.String(), description,
		cancelled, nullableString(p.Reason),
	); err != nil {
		return CancelResult{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return CancelResult{}, mapError(err)
	}

	return CancelResult{
		Order:            order,
		CancelledTickets: cancelled,
		EventTitle:       eventTitle,
		BuyerName:        order.BuyerName,
		BuyerEmail:       order.BuyerEmail,
	}, nil
}

// isZeroAmount reports whether a numeric-as-text amount is zero. The value
// arrives as PostgreSQL rendered it from numeric(14,2) - "0.00" - but a bare
// "0" is accepted too so the helper does not depend on the exact rendering.
func isZeroAmount(amount string) bool {
	for _, r := range amount {
		if r >= '1' && r <= '9' {
			return false
		}
	}
	return true
}
