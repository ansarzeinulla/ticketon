package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RosterEntry is one ticket as the scanner app stores it offline (SRS 4.8).
//
// TokenHash, not the token itself. The roster is a list of admission
// credentials for an entire event, and putting it on a phone that can be lost
// or stolen would hand somebody the means to forge every ticket in the house.
// The device hashes what it scans and compares hashes, which answers "is this
// ticket real?" without ever holding the answer to "what are the real
// tickets?".
type RosterEntry struct {
	TicketID       uuid.UUID `json:"ticket_id"`
	TokenHash      string    `json:"token_hash"`
	TicketCode     string    `json:"ticket_code"`
	AttendeeName   string    `json:"attendee_name"`
	TicketTypeName string    `json:"ticket_type_name"`
	SeatLabel      string    `json:"seat_label,omitempty"`
	// Status is what the server believed at download time. A device that has
	// been offline for an hour may be wrong about it, which is why a sync
	// result can still refuse a scan the device admitted.
	Status string `json:"status"`
}

// Roster is a downloadable snapshot of an event's tickets.
type Roster struct {
	EventID     uuid.UUID     `json:"event_id"`
	EventTitle  string        `json:"event_title"`
	GeneratedAt time.Time     `json:"generated_at"`
	Tickets     []RosterEntry `json:"tickets"`
}

// SyncedCheckIn is one admission a device recorded while offline.
type SyncedCheckIn struct {
	TicketID  uuid.UUID `json:"ticket_id"`
	ScannedAt time.Time `json:"scanned_at"`
	Device    string    `json:"device_label"`
}

// SyncResult is what became of one queued admission.
type SyncResult struct {
	TicketID uuid.UUID `json:"ticket_id"`
	// Outcome is "recorded", "already_checked_in", "not_valid" or
	// "unknown_ticket" - the same vocabulary the live gate uses, so the app
	// has one set of cases to handle rather than two.
	Outcome      string     `json:"outcome"`
	AttendeeName string     `json:"attendee_name,omitempty"`
	CheckedInAt  *time.Time `json:"checked_in_at,omitempty"`
}

// OfflineStore backs offline verification and its reconciliation (SRS 4.8).
type OfflineStore struct {
	pool *pgxpool.Pool
}

// NewOfflineStore builds an OfflineStore.
func NewOfflineStore(pool *pgxpool.Pool) *OfflineStore { return &OfflineStore{pool: pool} }

// Roster returns every issued ticket for an event, hashed.
//
// Cancelled and refunded tickets are included rather than filtered out: a
// device needs to be able to say "this ticket was refunded" at the door, and a
// ticket simply missing from the roster is indistinguishable from a forgery.
func (s *OfflineStore) Roster(ctx context.Context, eventID uuid.UUID) (Roster, error) {
	var roster Roster

	err := s.pool.QueryRow(ctx,
		`SELECT id, title FROM events WHERE id = $1`, eventID).
		Scan(&roster.EventID, &roster.EventTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return Roster{}, ErrNotFound
	}
	if err != nil {
		return Roster{}, mapError(err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT t.id,
		       -- Hashed in the database, so the plaintext token never leaves
		       -- the row it lives in.
		       encode(digest(t.qr_token, 'sha256'), 'hex'),
		       t.ticket_code,
		       COALESCE(a.full_name, o.buyer_name),
		       tt.name,
		       COALESCE(
		           NULLIF(concat_ws(' ', t.seat_section, t.seat_row, t.seat_number), ''),
		           ''),
		       t.status::text
		  FROM tickets t
		  JOIN ticket_types tt ON tt.id = t.ticket_type_id
		  JOIN orders o        ON o.id = t.order_id
		  LEFT JOIN attendees a ON a.id = t.attendee_id
		 WHERE t.event_id = $1
		 ORDER BY t.ticket_code`, eventID)
	if err != nil {
		return Roster{}, mapError(err)
	}
	defer rows.Close()

	roster.Tickets = []RosterEntry{}
	for rows.Next() {
		var entry RosterEntry
		if err := rows.Scan(&entry.TicketID, &entry.TokenHash, &entry.TicketCode,
			&entry.AttendeeName, &entry.TicketTypeName, &entry.SeatLabel,
			&entry.Status); err != nil {
			return Roster{}, err
		}
		roster.Tickets = append(roster.Tickets, entry)
	}
	if err := rows.Err(); err != nil {
		return Roster{}, mapError(err)
	}

	roster.GeneratedAt = time.Now().UTC()
	return roster, nil
}

// SyncCheckIns records admissions a device made while it was offline.
//
// Each is applied independently and reported on separately: one ticket that
// was refunded in the meantime must not discard the twenty good admissions
// queued behind it.
//
// The winner of a conflict is whoever synced first, which is the only rule
// that can be applied consistently - the devices' clocks cannot be trusted
// against each other, and that is exactly what the Phase 6 clock-skew bug
// taught.
func (s *OfflineStore) SyncCheckIns(
	ctx context.Context, eventID, byUserID uuid.UUID, entries []SyncedCheckIn,
) ([]SyncResult, error) {
	results := make([]SyncResult, 0, len(entries))

	for _, entry := range entries {
		result, err := s.syncOne(ctx, eventID, byUserID, entry)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *OfflineStore) syncOne(
	ctx context.Context, eventID, byUserID uuid.UUID, entry SyncedCheckIn,
) (SyncResult, error) {
	result := SyncResult{TicketID: entry.TicketID}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		status       string
		attendeeName string
		ticketEvent  uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		SELECT t.status::text, COALESCE(a.full_name, o.buyer_name), t.event_id
		  FROM tickets t
		  JOIN orders o ON o.id = t.order_id
		  LEFT JOIN attendees a ON a.id = t.attendee_id
		 WHERE t.id = $1
		   FOR UPDATE OF t`, entry.TicketID).Scan(&status, &attendeeName, &ticketEvent)
	if errors.Is(err, pgx.ErrNoRows) {
		result.Outcome = "unknown_ticket"
		return result, nil
	}
	if err != nil {
		return result, mapError(err)
	}

	result.AttendeeName = attendeeName

	if ticketEvent != eventID {
		result.Outcome = "unknown_ticket"
		return result, nil
	}

	switch status {
	case "cancelled", "refunded":
		// Voided after the device went offline. The person was let in on a
		// ticket that is no longer valid, and the organizer needs to know -
		// which is what reporting it rather than silently recording it does.
		result.Outcome = "not_valid"
		return result, nil
	case "checked_in":
		var checkedInAt *time.Time
		_ = tx.QueryRow(ctx,
			`SELECT checked_in_at FROM check_in_records
			  WHERE ticket_id = $1 AND reversed_at IS NULL`, entry.TicketID).Scan(&checkedInAt)
		result.Outcome = "already_checked_in"
		result.CheckedInAt = checkedInAt
		return result, nil
	}

	// The device's own clock decides when this happened, clamped so it cannot
	// be in the future. A phone an hour fast would otherwise write a check-in
	// that had not happened yet, and check_in_reversal_chk would later refuse
	// to reverse it.
	scannedAt := entry.ScannedAt
	if scannedAt.IsZero() || scannedAt.After(time.Now()) {
		scannedAt = time.Now()
	}

	device := entry.Device
	if device == "" {
		device = "offline scanner"
	}

	var recordedAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO check_in_records (ticket_id, event_id, checked_in_by, checked_in_at, device_label)
		VALUES ($1, $2, $3, LEAST($4::timestamptz, now()), $5)
		RETURNING checked_in_at`,
		entry.TicketID, eventID, nullableUUID(byUserID), scannedAt, device).Scan(&recordedAt)
	if isUniqueViolation(err, "check_in_one_active_per_ticket_uidx") {
		// Another device synced the same admission first.
		result.Outcome = "already_checked_in"
		return result, nil
	}
	if err != nil {
		return result, mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE tickets SET status = 'checked_in', checked_in_at = $2 WHERE id = $1`,
		entry.TicketID, recordedAt); err != nil {
		return result, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return result, mapError(err)
	}

	result.Outcome = "recorded"
	result.CheckedInAt = &recordedAt
	return result, nil
}
