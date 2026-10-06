package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Notification is one queued or delivered message (SRS 4.10).
type Notification struct {
	ID             uuid.UUID
	RecipientEmail string
	Type           string
	Subject        string
	Body           string
	Status         string
}

// NotificationParams is what the API records before dispatching.
type NotificationParams struct {
	UserID         *uuid.UUID
	RecipientEmail string
	Type           string
	Subject        string
	Body           string
	EventID        *uuid.UUID
	OrderID        *uuid.UUID
	// SupportCaseID links a support notification back to its conversation.
	// The column existed from Phase 1; nothing wrote it until the assignment
	// and status-change notifications needed somewhere to point.
	SupportCaseID *uuid.UUID
}

// NotificationStore records notifications.
//
// The row is written before the message is dispatched, so a notification that
// was attempted is visible even if delivery then failed. That ordering is what
// makes the table an outbox rather than a log of successes.
type NotificationStore struct {
	pool *pgxpool.Pool
}

// NewNotificationStore builds a NotificationStore.
func NewNotificationStore(pool *pgxpool.Pool) *NotificationStore {
	return &NotificationStore{pool: pool}
}

// Queue records a pending notification and returns its id.
func (s *NotificationStore) Queue(ctx context.Context, p NotificationParams) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, recipient_email, channel, type, subject, body,
		                           status, event_id, order_id, support_case_id)
		VALUES ($1, $2, 'email', $3, $4, $5, 'pending', $6, $7, $8)
		RETURNING id`,
		p.UserID, nullableString(p.RecipientEmail), p.Type, p.Subject, p.Body,
		p.EventID, p.OrderID, p.SupportCaseID).Scan(&id)
	return id, mapError(err)
}

// MarkSent stamps a notification as delivered.
func (s *NotificationStore) MarkSent(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications SET status = 'sent', sent_at = now() WHERE id = $1`, id)
	return mapError(err)
}

// MarkFailed records that delivery did not succeed.
func (s *NotificationStore) MarkFailed(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications SET status = 'failed' WHERE id = $1`, id)
	return mapError(err)
}

// ListForOrder returns the notifications recorded against one order, oldest
// first. Tests use it to prove the outbox row was written.
func (s *NotificationStore) ListForOrder(ctx context.Context, orderID uuid.UUID) ([]Notification, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(recipient_email::text, ''), type, COALESCE(subject, ''),
		       COALESCE(body, ''), status::text
		  FROM notifications
		 WHERE order_id = $1
		 ORDER BY created_at`, orderID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.RecipientEmail, &n.Type, &n.Subject,
			&n.Body, &n.Status); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
