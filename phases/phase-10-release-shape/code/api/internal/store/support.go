package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Support case statuses, matching the support_case_status enum.
const (
	SupportOpen               = "open"
	SupportInProgress         = "in_progress"
	SupportWaitingForCustomer = "waiting_for_customer"
	SupportResolved           = "resolved"
)

// Support case kinds, matching the support_case_kind enum.
const (
	SupportKindAttendee  = "attendee"
	SupportKindOrganizer = "organizer"
)

// SupportCategories are the issue categories an attendee picks from (SRS 4.13).
var SupportCategories = []string{
	"ticket_delivery", "payment", "refund", "seating",
	"event_information", "check_in", "account", "technical",
}

// SupportCase is one conversation, with the context it was opened from.
type SupportCase struct {
	ID            uuid.UUID  `json:"id"`
	CaseNumber    string     `json:"case_number"`
	Kind          string     `json:"kind"`
	Category      string     `json:"category"`
	Status        string     `json:"status"`
	Subject       string     `json:"subject"`
	RequesterID   uuid.UUID  `json:"requester_user_id"`
	RequesterName string     `json:"requester_name"`
	AssignedToID  *uuid.UUID `json:"assigned_to_user_id,omitempty"`
	AssignedName  *string    `json:"assigned_to_name,omitempty"`

	EventID     *uuid.UUID `json:"event_id,omitempty"`
	EventTitle  *string    `json:"event_title,omitempty"`
	OrderID     *uuid.UUID `json:"order_id,omitempty"`
	OrderNumber *string    `json:"order_number,omitempty"`
	TicketID    *uuid.UUID `json:"ticket_id,omitempty"`
	TicketCode  *string    `json:"ticket_code,omitempty"`

	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	MessageCount  int        `json:"message_count"`
}

// SupportMessage is one post in a case thread.
type SupportMessage struct {
	ID         uuid.UUID  `json:"id"`
	CaseID     uuid.UUID  `json:"support_case_id"`
	SenderID   *uuid.UUID `json:"sender_user_id,omitempty"`
	SenderName string     `json:"sender_name"`
	// SenderRole tells the UI which side of the conversation posted: the
	// attendee who opened it, or the organizer answering.
	SenderRole     string    `json:"sender_role"`
	Body           string    `json:"body"`
	IsInternalNote bool      `json:"is_internal_note"`
	CreatedAt      time.Time `json:"created_at"`

	// Attachment describes the uploaded file backing this message, if any
	// (bonus, SRS 4.13). Nil when the message is text alone.
	Attachment *MessageAttachment `json:"attachment,omitempty"`
}

// MessageAttachment is a file a support message carries.
//
// The four fields move together - support_messages_attachment_chk enforces
// that in the database - so a half-populated attachment cannot reach a client
// and render as a broken link.
type MessageAttachment struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Bytes    int64  `json:"bytes"`
}

// SupportStore reads and writes support conversations.
type SupportStore struct {
	pool *pgxpool.Pool
}

// NewSupportStore builds a SupportStore.
func NewSupportStore(pool *pgxpool.Pool) *SupportStore { return &SupportStore{pool: pool} }

const supportCaseColumns = `c.id, c.case_number, c.kind::text, c.category::text, c.status::text,
	c.subject, c.requester_user_id, requester.full_name,
	c.assigned_to_user_id, assignee.full_name,
	c.event_id, e.title, c.order_id, o.order_number, c.ticket_id, t.ticket_code,
	c.last_message_at, c.resolved_at, c.created_at,
	(SELECT count(*) FROM support_messages m WHERE m.support_case_id = c.id)`

const supportCaseJoins = `
	FROM support_cases c
	JOIN users requester ON requester.id = c.requester_user_id
	LEFT JOIN users assignee ON assignee.id = c.assigned_to_user_id
	LEFT JOIN events e ON e.id = c.event_id
	LEFT JOIN orders o ON o.id = c.order_id
	LEFT JOIN tickets t ON t.id = c.ticket_id`

func scanSupportCase(row pgx.Row) (SupportCase, error) {
	var c SupportCase
	err := row.Scan(&c.ID, &c.CaseNumber, &c.Kind, &c.Category, &c.Status,
		&c.Subject, &c.RequesterID, &c.RequesterName,
		&c.AssignedToID, &c.AssignedName,
		&c.EventID, &c.EventTitle, &c.OrderID, &c.OrderNumber, &c.TicketID, &c.TicketCode,
		&c.LastMessageAt, &c.ResolvedAt, &c.CreatedAt, &c.MessageCount)
	return c, err
}

// OpenCaseParams describes a new support request.
type OpenCaseParams struct {
	RequesterID uuid.UUID
	Kind        string
	Category    string
	Subject     string
	Body        string
	EventID     *uuid.UUID
	OrderID     *uuid.UUID
	TicketID    *uuid.UUID
}

// Open creates a case and its first message in one transaction, so a case can
// never exist without the question that prompted it.
func (s *SupportStore) Open(ctx context.Context, p OpenCaseParams) (SupportCase, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SupportCase{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	caseNumber, err := newCode("SC", "-", 8)
	if err != nil {
		return SupportCase{}, err
	}

	var caseID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO support_cases (case_number, kind, category, status, subject,
		                           requester_user_id, event_id, order_id, ticket_id,
		                           last_message_at)
		VALUES ($1, $2::support_case_kind, $3::support_case_category, 'open', $4,
		        $5, $6, $7, $8, now())
		RETURNING id`,
		caseNumber, p.Kind, p.Category, p.Subject,
		p.RequesterID, p.EventID, p.OrderID, p.TicketID).Scan(&caseID)
	if err != nil {
		return SupportCase{}, mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO support_messages (support_case_id, sender_user_id, body)
		VALUES ($1, $2, $3)`, caseID, p.RequesterID, p.Body); err != nil {
		return SupportCase{}, mapError(err)
	}

	if p.EventID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description)
			VALUES ($1, $2, 'support_case.opened', 'support_case', $3, $4)`,
			*p.EventID, p.RequesterID, caseID.String(),
			"Support case "+caseNumber+" opened: "+p.Subject); err != nil {
			return SupportCase{}, mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SupportCase{}, mapError(err)
	}
	return s.GetByID(ctx, caseID)
}

// GetByID returns one case with its context.
func (s *SupportStore) GetByID(ctx context.Context, id uuid.UUID) (SupportCase, error) {
	c, err := scanSupportCase(s.pool.QueryRow(ctx,
		`SELECT `+supportCaseColumns+supportCaseJoins+` WHERE c.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return SupportCase{}, ErrNotFound
	}
	if err != nil {
		return SupportCase{}, mapError(err)
	}
	return c, nil
}

// ListForRequester returns the cases a person opened, most recent activity first.
func (s *SupportStore) ListForRequester(ctx context.Context, userID uuid.UUID) ([]SupportCase, error) {
	return s.list(ctx, `WHERE c.requester_user_id = $1`, userID)
}

// ListForEvent returns every case attached to an event - the organizer's inbox.
func (s *SupportStore) ListForEvent(ctx context.Context, eventID uuid.UUID) ([]SupportCase, error) {
	return s.list(ctx, `WHERE c.event_id = $1`, eventID)
}

// ListForOrder returns the cases opened against one order.
func (s *SupportStore) ListForOrder(ctx context.Context, orderID uuid.UUID) ([]SupportCase, error) {
	return s.list(ctx, `WHERE c.order_id = $1`, orderID)
}

func (s *SupportStore) list(ctx context.Context, where string, arg any) ([]SupportCase, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+supportCaseColumns+supportCaseJoins+` `+where+
			` ORDER BY COALESCE(c.last_message_at, c.created_at) DESC`, arg)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	cases := []SupportCase{}
	for rows.Next() {
		c, err := scanSupportCase(rows)
		if err != nil {
			return nil, err
		}
		cases = append(cases, c)
	}
	return cases, rows.Err()
}

// Messages returns a case thread oldest first.
//
// includeInternal is false for the attendee: staff notes are written on the
// same thread but must never be shown to the person who opened the case.
func (s *SupportStore) Messages(
	ctx context.Context, caseID, requesterID uuid.UUID, includeInternal bool,
) ([]SupportMessage, error) {
	query := `
		SELECT m.id, m.support_case_id, m.sender_user_id,
		       COALESCE(u.full_name, 'BiletFlow'),
		       CASE WHEN m.sender_user_id = $2 THEN 'requester' ELSE 'staff' END,
		       m.body, m.is_internal_note, m.created_at,
		       m.attachment_url, m.attachment_filename,
		       m.attachment_mime_type, m.attachment_bytes
		  FROM support_messages m
		  LEFT JOIN users u ON u.id = m.sender_user_id
		 WHERE m.support_case_id = $1`
	if !includeInternal {
		query += ` AND m.is_internal_note = false`
	}
	query += ` ORDER BY m.created_at ASC`

	rows, err := s.pool.Query(ctx, query, caseID, requesterID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	messages := []SupportMessage{}
	for rows.Next() {
		var (
			m                   SupportMessage
			url, filename, mime *string
			size                *int64
		)
		if err := rows.Scan(&m.ID, &m.CaseID, &m.SenderID, &m.SenderName,
			&m.SenderRole, &m.Body, &m.IsInternalNote, &m.CreatedAt,
			&url, &filename, &mime, &size); err != nil {
			return nil, err
		}
		// The database guarantees all four columns move together, so testing
		// one is enough to know whether there is a file here.
		if url != nil {
			m.Attachment = &MessageAttachment{
				URL: *url, Filename: derefOr(filename, ""),
				MimeType: derefOr(mime, ""), Bytes: derefOrInt64(size, 0),
			}
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// PostMessage adds a reply and bumps the case's activity timestamp.
//
// A staff reply also moves an untouched case to in_progress, so an organizer
// does not have to remember to change the status by hand.
func (s *SupportStore) PostMessage(
	ctx context.Context, caseID, senderID uuid.UUID, body string, internal, fromStaff bool,
	attachment *MessageAttachment,
) (SupportMessage, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SupportMessage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		url, filename, mime *string
		size                *int64
	)
	if attachment != nil {
		url, filename, mime, size =
			&attachment.URL, &attachment.Filename, &attachment.MimeType, &attachment.Bytes
	}

	var m SupportMessage
	err = tx.QueryRow(ctx, `
		INSERT INTO support_messages (support_case_id, sender_user_id, body, is_internal_note,
		                              attachment_url, attachment_filename,
		                              attachment_mime_type, attachment_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, support_case_id, sender_user_id, body, is_internal_note, created_at`,
		caseID, senderID, body, internal, url, filename, mime, size,
	).Scan(&m.ID, &m.CaseID, &m.SenderID, &m.Body, &m.IsInternalNote, &m.CreatedAt)
	if err != nil {
		return SupportMessage{}, mapError(err)
	}
	m.Attachment = attachment

	// An internal note is not a reply to the customer, so it does not change
	// the status the customer sees.
	if fromStaff && !internal {
		if _, err := tx.Exec(ctx, `
			UPDATE support_cases
			   SET last_message_at = now(),
			       status = CASE WHEN status = 'open' THEN 'in_progress'::support_case_status
			                     ELSE status END,
			       assigned_to_user_id = COALESCE(assigned_to_user_id, $2)
			 WHERE id = $1`, caseID, senderID); err != nil {
			return SupportMessage{}, mapError(err)
		}
	} else {
		if _, err := tx.Exec(ctx,
			`UPDATE support_cases SET last_message_at = now() WHERE id = $1`, caseID); err != nil {
			return SupportMessage{}, mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SupportMessage{}, mapError(err)
	}
	return m, nil
}

// SetStatus moves a case through open -> in_progress -> resolved.
func (s *SupportStore) SetStatus(
	ctx context.Context, caseID uuid.UUID, status string, actorID uuid.UUID,
) (SupportCase, error) {
	// support_cases_resolved_chk requires resolved_at whenever the status is
	// resolved, and reopening has to clear it again.
	tag, err := s.pool.Exec(ctx, `
		UPDATE support_cases
		   SET status = $2::support_case_status,
		       resolved_at = CASE WHEN $2 = 'resolved' THEN COALESCE(resolved_at, now()) END,
		       assigned_to_user_id = COALESCE(assigned_to_user_id, $3)
		 WHERE id = $1`, caseID, status, actorID)
	if err != nil {
		return SupportCase{}, mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return SupportCase{}, ErrNotFound
	}
	return s.GetByID(ctx, caseID)
}

// Assign puts a case in a named person's hands (SRS 4.13: "Authorized staff
// shall be able to assign a case").
//
// Until now assignment only ever happened implicitly, as a COALESCE side
// effect of the first reply, so a case could not be handed to a colleague who
// had not spoken yet. Passing a nil assignee unassigns the case.
func (s *SupportStore) Assign(
	ctx context.Context, caseID uuid.UUID, assignee *uuid.UUID,
) (SupportCase, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE support_cases SET assigned_to_user_id = $2 WHERE id = $1`, caseID, assignee)
	if err != nil {
		return SupportCase{}, mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return SupportCase{}, ErrNotFound
	}
	return s.GetByID(ctx, caseID)
}

// UserByEmail resolves the person a case is being assigned to.
func (s *SupportStore) UserByEmail(ctx context.Context, email string) (uuid.UUID, string, error) {
	var (
		id   uuid.UUID
		name string
	)
	err := s.pool.QueryRow(ctx,
		`SELECT id, full_name FROM users WHERE email = $1`, email).Scan(&id, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", ErrNotFound
	}
	return id, name, mapError(err)
}

// RequesterContact returns the person who opened a case, for notifications.
func (s *SupportStore) RequesterContact(
	ctx context.Context, caseID uuid.UUID,
) (CaseParticipant, error) {
	var c CaseParticipant
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.full_name, u.email
		  FROM support_cases sc
		  JOIN users u ON u.id = sc.requester_user_id
		 WHERE sc.id = $1`, caseID).Scan(&c.UserID, &c.FullName, &c.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return CaseParticipant{}, ErrNotFound
	}
	return c, mapError(err)
}

// OrderBelongsTo reports whether an order is the user's own.
//
// It matches on the account id or the buyer email: a guest checkout stores no
// user id, so someone who buys as a guest and registers later with the same
// address should still be able to ask about their own order.
func (s *SupportStore) OrderBelongsTo(ctx context.Context, orderID, userID uuid.UUID) (bool, error) {
	var owns bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM orders o
			 WHERE o.id = $1
			   AND (o.buyer_user_id = $2
			        OR o.buyer_email = (SELECT email FROM users WHERE id = $2))
		)`, orderID, userID).Scan(&owns)
	return owns, mapError(err)
}

// CaseParticipant is the other side of a support conversation.
type CaseParticipant struct {
	UserID   *uuid.UUID
	FullName string
	Email    string
}

// Counterpart returns whoever should be told about a new message: the event's
// organizer when the requester wrote it, and the requester when staff did
// (SRS 4.10, "new support message").
//
// Returns ErrNotFound when there is nobody to tell - a case with no event, or
// one where the sender is somehow both sides.
func (s *SupportStore) Counterpart(
	ctx context.Context, caseID, senderID uuid.UUID,
) (CaseParticipant, error) {
	var (
		requesterID    uuid.UUID
		requesterName  string
		requesterEmail string
		organizerID    *uuid.UUID
		organizerName  *string
		organizerEmail *string
	)

	err := s.pool.QueryRow(ctx, `
		SELECT c.requester_user_id, r.full_name, r.email::text,
		       o.id, o.full_name, o.email::text
		  FROM support_cases c
		  JOIN users r ON r.id = c.requester_user_id
		  LEFT JOIN events e ON e.id = c.event_id
		  LEFT JOIN users o  ON o.id = e.organizer_id
		 WHERE c.id = $1`, caseID,
	).Scan(&requesterID, &requesterName, &requesterEmail,
		&organizerID, &organizerName, &organizerEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return CaseParticipant{}, ErrNotFound
	}
	if err != nil {
		return CaseParticipant{}, mapError(err)
	}

	// The requester wrote it, so the organizer hears about it.
	if senderID == requesterID {
		if organizerID == nil || organizerEmail == nil {
			return CaseParticipant{}, ErrNotFound
		}
		return CaseParticipant{
			UserID: organizerID, FullName: derefOr(organizerName, ""), Email: *organizerEmail,
		}, nil
	}

	// Staff wrote it, so the requester hears about it.
	return CaseParticipant{
		UserID: &requesterID, FullName: requesterName, Email: requesterEmail,
	}, nil
}

func derefOrInt64(p *int64, fallback int64) int64 {
	if p == nil {
		return fallback
	}
	return *p
}
