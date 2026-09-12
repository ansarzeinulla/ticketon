package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Admission token namespaces. An admission QR always starts with TKT_ and a
// campaign QR with CMP_; SRS 4.14 requires that the two can never be confused,
// and the scanner is where that guarantee has to hold.
const (
	AdmissionTokenPrefix = "TKT_"
	CampaignTokenPrefix  = "CMP_"
)

// ScannedKind is what a scanned string turns out to be.
type ScannedKind int

// The kinds a scanner can encounter at a gate.
const (
	ScannedUnknown ScannedKind = iota
	ScannedAdmission
	ScannedCampaign
)

// ClassifyScannedCode decides what a scanned string is.
//
// A campaign QR does not encode the bare CMP_ token: SRS 4.14 requires it to
// encode a trackable HTTPS event link, so what a scanner actually reads is
// something like https://biletflow.kz/events/gala?c=CMP_<uuid>. Matching only
// on a leading prefix would classify that as "unknown code" and leave staff
// guessing, so the token is looked for inside the link too - and rejecting it
// explicitly is precisely the guarantee SRS 4.14 asks for.
func ClassifyScannedCode(raw string) ScannedKind {
	code := strings.TrimSpace(raw)
	if code == "" {
		return ScannedUnknown
	}

	if strings.HasPrefix(code, AdmissionTokenPrefix) {
		return ScannedAdmission
	}
	if strings.HasPrefix(code, CampaignTokenPrefix) {
		return ScannedCampaign
	}

	// A campaign link: look for the token in the query string.
	if parsed, err := url.Parse(code); err == nil && parsed.Scheme != "" {
		for _, values := range parsed.Query() {
			for _, value := range values {
				if strings.HasPrefix(value, CampaignTokenPrefix) {
					return ScannedCampaign
				}
				if strings.HasPrefix(value, AdmissionTokenPrefix) {
					return ScannedAdmission
				}
			}
		}
	}

	// Last resort: a campaign token embedded in text we did not anticipate.
	// Better to name it than to call a promotional code "unknown".
	if strings.Contains(code, CampaignTokenPrefix) {
		return ScannedCampaign
	}

	return ScannedUnknown
}

// CampaignTokenIn extracts the CMP_ token from a scanned campaign link.
func CampaignTokenIn(raw string) string {
	code := strings.TrimSpace(raw)
	if strings.HasPrefix(code, CampaignTokenPrefix) {
		return code
	}

	if parsed, err := url.Parse(code); err == nil && parsed.Scheme != "" {
		for _, values := range parsed.Query() {
			for _, value := range values {
				if strings.HasPrefix(value, CampaignTokenPrefix) {
					return value
				}
			}
		}
	}
	return ""
}

// CheckInResult describes a successful admission.
type CheckInResult struct {
	TicketID       uuid.UUID    `json:"ticket_id"`
	TicketCode     string       `json:"ticket_code"`
	TicketTypeName string       `json:"ticket_type_name"`
	AttendeeName   string       `json:"attendee_name"`
	AttendeeEmail  string       `json:"attendee_email"`
	SeatLabel      string       `json:"seat_label,omitempty"`
	CheckedInAt    time.Time    `json:"checked_in_at"`
	Stats          CheckInStats `json:"stats"`
}

// CheckInStats is the running count an Event Admin sees on the scanner
// (SRS 4.8: "View the total number of registered and checked-in attendees").
type CheckInStats struct {
	Issued    int `json:"issued"`
	CheckedIn int `json:"checked_in"`
}

// AlreadyCheckedInError reports a ticket that has already been admitted. It
// carries who and when, so the scanner can show something more useful than
// "denied".
type AlreadyCheckedInError struct {
	TicketID     uuid.UUID
	AttendeeName string
	CheckedInAt  time.Time
	Stats        CheckInStats
}

func (e *AlreadyCheckedInError) Error() string {
	return fmt.Sprintf("this ticket was already checked in at %s",
		e.CheckedInAt.Format(time.RFC3339))
}

// CampaignTokenError reports that a promotional QR was presented at the gate.
// SRS 4.14 is explicit that this must never be accepted as admission.
type CampaignTokenError struct{ Token string }

func (e *CampaignTokenError) Error() string {
	return "this is a promotional campaign code, not an admission ticket"
}

// WrongEventError reports a ticket that is valid but for a different event.
type WrongEventError struct {
	TicketID   uuid.UUID
	EventTitle string
}

func (e *WrongEventError) Error() string {
	return fmt.Sprintf("this ticket is for %q, not the event being scanned", e.EventTitle)
}

// TicketNotAdmissibleError reports a cancelled or refunded ticket.
type TicketNotAdmissibleError struct {
	TicketID     uuid.UUID
	Status       string
	AttendeeName string
}

func (e *TicketNotAdmissibleError) Error() string {
	return fmt.Sprintf("this ticket is %s and cannot be used for entry", e.Status)
}

// CheckInStore records admissions.
type CheckInStore struct {
	pool *pgxpool.Pool
}

// NewCheckInStore builds a CheckInStore.
func NewCheckInStore(pool *pgxpool.Pool) *CheckInStore { return &CheckInStore{pool: pool} }

// CheckIn admits the holder of qrToken to eventID.
//
// Duplicate entry is prevented by check_in_one_active_per_ticket_uidx, the
// unique partial index over check_in_records where reversed_at IS NULL. Two
// scanners hitting the same ticket at the same instant therefore cannot both
// succeed: one insert wins and the other comes back as a unique violation,
// which is translated into AlreadyCheckedInError. The ticket row is also locked
// first, so the two paths agree on what happened.
func (s *CheckInStore) CheckIn(
	ctx context.Context, eventID uuid.UUID, qrToken string, byUserID uuid.UUID, device string,
) (CheckInResult, error) {
	token := strings.TrimSpace(qrToken)

	// Rejected before touching the database: a campaign QR is not an admission
	// credential, and saying so precisely beats a generic "not found".
	switch ClassifyScannedCode(token) {
	case ScannedCampaign:
		return CheckInResult{}, &CampaignTokenError{Token: token}
	case ScannedAdmission:
		// carry on
	default:
		return CheckInResult{}, ErrNotFound
	}

	return s.admit(ctx, eventID, "t.qr_token = $1", token, byUserID, device)
}

// CheckInByTicketID admits a ticket the staff member found by name rather than
// by camera (SRS 4.8, "search for attendees manually").
//
// It runs the same transaction as a scan - the same row lock, the same unique
// index, the same duplicate handling - so a manual admission is neither weaker
// nor different from a scanned one. Only the way the ticket was located
// differs, and that is recorded in the device label.
func (s *CheckInStore) CheckInByTicketID(
	ctx context.Context, eventID, ticketID uuid.UUID, byUserID uuid.UUID, device string,
) (CheckInResult, error) {
	return s.admit(ctx, eventID, "t.id = $1", ticketID, byUserID, device)
}

// admit is the shared body. `predicate` is one of the two compile-time literals
// above and never contains anything a caller supplied; the value it compares
// against is always a bound parameter.
func (s *CheckInStore) admit(
	ctx context.Context, eventID uuid.UUID, predicate string, subject any,
	byUserID uuid.UUID, device string,
) (CheckInResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CheckInResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		ticketID       uuid.UUID
		ticketCode     string
		ticketEventID  uuid.UUID
		eventTitle     string
		status         string
		ticketTypeName string
		attendeeName   string
		attendeeEmail  string
		section        *string
		row            *string
		seatNumber     *string
	)

	err = tx.QueryRow(ctx, `
		SELECT t.id, t.ticket_code, t.event_id, e.title, t.status::text, tt.name,
		       COALESCE(a.full_name, o.buyer_name),
		       COALESCE(a.email::text, o.buyer_email::text),
		       t.seat_section, t.seat_row, t.seat_number
		  FROM tickets t
		  JOIN events e        ON e.id = t.event_id
		  JOIN ticket_types tt ON tt.id = t.ticket_type_id
		  JOIN orders o        ON o.id = t.order_id
		  LEFT JOIN attendees a ON a.id = t.attendee_id
		 WHERE `+predicate+`
		   FOR UPDATE OF t`, subject,
	).Scan(&ticketID, &ticketCode, &ticketEventID, &eventTitle, &status, &ticketTypeName,
		&attendeeName, &attendeeEmail, &section, &row, &seatNumber)

	if errors.Is(err, pgx.ErrNoRows) {
		return CheckInResult{}, ErrNotFound
	}
	if err != nil {
		return CheckInResult{}, mapError(err)
	}

	if ticketEventID != eventID {
		return CheckInResult{}, &WrongEventError{TicketID: ticketID, EventTitle: eventTitle}
	}

	switch status {
	case "cancelled", "refunded":
		return CheckInResult{}, &TicketNotAdmissibleError{
			TicketID: ticketID, Status: status, AttendeeName: attendeeName,
		}
	}

	// The insert goes through a savepoint. A unique violation aborts the
	// enclosing transaction in PostgreSQL, and we still need to query who used
	// the ticket and when in order to answer usefully - so the failure has to
	// be contained rather than allowed to poison the transaction.
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		return CheckInResult{}, mapError(err)
	}

	// The timestamp comes from PostgreSQL, not from this process.
	//
	// check_in_reversal_chk requires reversed_at >= checked_in_at, and a
	// reversal is stamped by the database. Mixing an application clock into one
	// side of that comparison makes reversal fail whenever the two machines
	// disagree by a few milliseconds - which they do, routinely, when the
	// database is in a container or on another host.
	var now time.Time
	insertErr := savepoint.QueryRow(ctx, `
		INSERT INTO check_in_records (ticket_id, event_id, checked_in_by, checked_in_at, device_label)
		VALUES ($1, $2, $3, now(), $4)
		RETURNING checked_in_at`,
		ticketID, eventID, nullableUUID(byUserID), nullableString(device)).Scan(&now)

	if insertErr != nil {
		_ = savepoint.Rollback(ctx)

		if isUniqueViolation(insertErr, "check_in_one_active_per_ticket_uidx") {
			// Someone already walked through with this ticket.
			previous, statsErr := s.alreadyCheckedIn(ctx, tx, ticketID, eventID, attendeeName)
			if statsErr != nil {
				return CheckInResult{}, statsErr
			}
			return CheckInResult{}, previous
		}
		return CheckInResult{}, mapError(insertErr)
	}

	if err := savepoint.Commit(ctx); err != nil {
		return CheckInResult{}, mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE tickets SET status = 'checked_in', checked_in_at = $2 WHERE id = $1`,
		ticketID, now); err != nil {
		return CheckInResult{}, mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description)
		VALUES ($1, $2, 'ticket.checked_in', 'ticket', $3, $4)`,
		eventID, nullableUUID(byUserID), ticketID.String(),
		"Checked in "+attendeeName+" ("+ticketCode+")",
	); err != nil {
		return CheckInResult{}, mapError(err)
	}

	stats, err := checkInStats(ctx, tx, eventID)
	if err != nil {
		return CheckInResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CheckInResult{}, mapError(err)
	}

	return CheckInResult{
		TicketID:       ticketID,
		TicketCode:     ticketCode,
		TicketTypeName: ticketTypeName,
		AttendeeName:   attendeeName,
		AttendeeEmail:  attendeeEmail,
		SeatLabel:      seatLabel(section, row, seatNumber),
		CheckedInAt:    now,
		Stats:          stats,
	}, nil
}

// alreadyCheckedIn builds the error for a repeat scan, including when the
// ticket was first used.
func (s *CheckInStore) alreadyCheckedIn(
	ctx context.Context, tx pgx.Tx, ticketID, eventID uuid.UUID, attendeeName string,
) (*AlreadyCheckedInError, error) {
	var checkedInAt time.Time
	err := tx.QueryRow(ctx, `
		SELECT checked_in_at FROM check_in_records
		 WHERE ticket_id = $1 AND reversed_at IS NULL
		 ORDER BY checked_in_at DESC LIMIT 1`, ticketID).Scan(&checkedInAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, mapError(err)
	}

	stats, err := checkInStats(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}

	return &AlreadyCheckedInError{
		TicketID:     ticketID,
		AttendeeName: attendeeName,
		CheckedInAt:  checkedInAt,
		Stats:        stats,
	}, nil
}

// Reverse undoes an accidental check-in (SRS 4.8).
func (s *CheckInStore) Reverse(
	ctx context.Context, ticketID, byUserID uuid.UUID, reason string,
) (CheckInStats, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CheckInStats{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var eventID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE check_in_records
		   -- GREATEST guards check_in_reversal_chk against any row whose
		   -- checked_in_at was written by a clock other than this database's.
		   SET reversed_at = GREATEST(now(), checked_in_at),
		       reversed_by = $2,
		       reversal_reason = $3
		 WHERE ticket_id = $1 AND reversed_at IS NULL
		RETURNING event_id`,
		ticketID, nullableUUID(byUserID), nullableString(reason)).Scan(&eventID)

	if errors.Is(err, pgx.ErrNoRows) {
		return CheckInStats{}, ErrNotFound
	}
	if err != nil {
		return CheckInStats{}, mapError(err)
	}

	// The reversal frees the ticket for a legitimate re-scan.
	if _, err := tx.Exec(ctx,
		`UPDATE tickets SET status = 'valid', checked_in_at = NULL WHERE id = $1`,
		ticketID); err != nil {
		return CheckInStats{}, mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description)
		VALUES ($1, $2, 'ticket.check_in_reversed', 'ticket', $3, $4)`,
		eventID, nullableUUID(byUserID), ticketID.String(), "Check-in reversed",
	); err != nil {
		return CheckInStats{}, mapError(err)
	}

	stats, err := checkInStats(ctx, tx, eventID)
	if err != nil {
		return CheckInStats{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CheckInStats{}, mapError(err)
	}
	return stats, nil
}

// Stats returns the live counts for an event.
func (s *CheckInStore) Stats(ctx context.Context, eventID uuid.UUID) (CheckInStats, error) {
	return checkInStats(ctx, s.pool, eventID)
}

// querier is the shared surface of a pool and a transaction.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func checkInStats(ctx context.Context, q querier, eventID uuid.UUID) (CheckInStats, error) {
	var stats CheckInStats
	err := q.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status IN ('valid', 'checked_in')) AS issued,
		       count(*) FILTER (WHERE status = 'checked_in')             AS checked_in
		  FROM tickets WHERE event_id = $1`, eventID).Scan(&stats.Issued, &stats.CheckedIn)
	if err != nil {
		return CheckInStats{}, mapError(err)
	}
	return stats, nil
}

func seatLabel(section, row, number *string) string {
	if section == nil && row == nil && number == nil {
		return ""
	}
	parts := []string{}
	if section != nil && *section != "" {
		parts = append(parts, *section)
	}
	if row != nil && *row != "" {
		parts = append(parts, "Row "+*row)
	}
	if number != nil && *number != "" {
		parts = append(parts, "Seat "+*number)
	}
	return strings.Join(parts, ", ")
}

func nullableUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func nullableString(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
