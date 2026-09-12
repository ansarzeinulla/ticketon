package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUserNotFound is returned when assigning staff by an email nobody holds.
var ErrUserNotFound = errors.New("no user with that email")

// StaffAssignment is one person's authority over one event.
type StaffAssignment struct {
	ID         uuid.UUID  `json:"id"`
	EventID    uuid.UUID  `json:"event_id"`
	UserID     uuid.UUID  `json:"user_id"`
	UserName   string     `json:"user_name"`
	UserEmail  string     `json:"user_email"`
	Role       string     `json:"role"`
	AssignedAt time.Time  `json:"assigned_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// ScannableEvent is an event the caller may run check-in for. It is what fills
// the event selector in the scanner app.
type ScannableEvent struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Timezone  string    `json:"timezone"`
	VenueName string    `json:"venue_name,omitempty"`
	Status    string    `json:"status"`
	// How the caller comes to have access: "organizer" or a staff role.
	AccessVia string       `json:"access_via"`
	Stats     CheckInStats `json:"stats"`
}

// StaffStore manages event staff and the events a person may scan.
type StaffStore struct {
	pool *pgxpool.Pool
}

// NewStaffStore builds a StaffStore.
func NewStaffStore(pool *pgxpool.Pool) *StaffStore { return &StaffStore{pool: pool} }

// scannerRoles are the staff roles that may run check-in.
var scannerRoles = []string{"event_admin", "manager"}

// CanScan reports whether a user may check attendees in for an event: they own
// it, or they hold an unrevoked scanner assignment on it.
func (s *StaffStore) CanScan(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	var allowed bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM events WHERE id = $1 AND organizer_id = $2
			UNION ALL
			SELECT 1 FROM staff_assignments
			 WHERE event_id = $1 AND user_id = $2
			   AND revoked_at IS NULL AND role::text = ANY($3)
		)`, eventID, userID, scannerRoles).Scan(&allowed)
	return allowed, mapError(err)
}

// ListScannable returns every event the user may check in for, soonest first.
func (s *StaffStore) ListScannable(ctx context.Context, userID uuid.UUID) ([]ScannableEvent, error) {
	rows, err := s.pool.Query(ctx, `
		WITH accessible AS (
			SELECT e.id, 'organizer' AS access_via
			  FROM events e
			 WHERE e.organizer_id = $1
			UNION
			SELECT sa.event_id, sa.role::text
			  FROM staff_assignments sa
			 WHERE sa.user_id = $1
			   AND sa.revoked_at IS NULL
			   AND sa.role::text = ANY($2)
		)
		SELECT e.id, e.title, e.slug, e.starts_at, e.ends_at, e.timezone,
		       COALESCE(e.venue_name, ''), e.status::text, a.access_via,
		       COALESCE(count(t.id) FILTER (WHERE t.status IN ('valid', 'checked_in')), 0),
		       COALESCE(count(t.id) FILTER (WHERE t.status = 'checked_in'), 0)
		  FROM accessible a
		  JOIN events e ON e.id = a.id
		  LEFT JOIN tickets t ON t.event_id = e.id
		 WHERE e.status <> 'cancelled'
		 GROUP BY e.id, e.title, e.slug, e.starts_at, e.ends_at, e.timezone,
		          e.venue_name, e.status, a.access_via
		 ORDER BY e.starts_at ASC`, userID, scannerRoles)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	events := []ScannableEvent{}
	for rows.Next() {
		var e ScannableEvent
		if err := rows.Scan(&e.ID, &e.Title, &e.Slug, &e.StartsAt, &e.EndsAt, &e.Timezone,
			&e.VenueName, &e.Status, &e.AccessVia, &e.Stats.Issued, &e.Stats.CheckedIn); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// AssignByEmail gives an existing account a scanner role on an event. Staff are
// named by email because an organizer knows their colleague's address, not
// their user id.
func (s *StaffStore) AssignByEmail(
	ctx context.Context, eventID uuid.UUID, email, role string, assignedBy uuid.UUID,
) (StaffAssignment, error) {
	var userID uuid.UUID
	var name, storedEmail string

	err := s.pool.QueryRow(ctx,
		`SELECT id, full_name, email::text FROM users WHERE email = $1`, email).
		Scan(&userID, &name, &storedEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return StaffAssignment{}, ErrUserNotFound
	}
	if err != nil {
		return StaffAssignment{}, mapError(err)
	}

	var a StaffAssignment
	err = s.pool.QueryRow(ctx, `
		INSERT INTO staff_assignments (event_id, user_id, role, assigned_by)
		VALUES ($1, $2, $3::staff_role, $4)
		ON CONFLICT (event_id, user_id, role)
		DO UPDATE SET revoked_at = NULL, assigned_at = now(), assigned_by = $4
		RETURNING id, event_id, user_id, role::text, assigned_at, revoked_at`,
		eventID, userID, role, assignedBy,
	).Scan(&a.ID, &a.EventID, &a.UserID, &a.Role, &a.AssignedAt, &a.RevokedAt)
	if err != nil {
		return StaffAssignment{}, mapError(err)
	}

	// Holding a scanner assignment is what makes someone an Event Admin.
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role) VALUES ($1, 'event_admin')
		 ON CONFLICT DO NOTHING`, userID); err != nil {
		return StaffAssignment{}, mapError(err)
	}

	a.UserName, a.UserEmail = name, storedEmail
	return a, nil
}

// ListForEvent returns an event's staff, revoked assignments included.
func (s *StaffStore) ListForEvent(ctx context.Context, eventID uuid.UUID) ([]StaffAssignment, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sa.id, sa.event_id, sa.user_id, u.full_name, u.email::text,
		       sa.role::text, sa.assigned_at, sa.revoked_at
		  FROM staff_assignments sa
		  JOIN users u ON u.id = sa.user_id
		 WHERE sa.event_id = $1
		 ORDER BY sa.assigned_at`, eventID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	assignments := []StaffAssignment{}
	for rows.Next() {
		var a StaffAssignment
		if err := rows.Scan(&a.ID, &a.EventID, &a.UserID, &a.UserName, &a.UserEmail,
			&a.Role, &a.AssignedAt, &a.RevokedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

// Revoke withdraws an assignment without deleting the record, so the audit
// trail keeps who was authorised and when.
func (s *StaffStore) Revoke(ctx context.Context, assignmentID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE staff_assignments SET revoked_at = now()
		  WHERE id = $1 AND revoked_at IS NULL`, assignmentID)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
