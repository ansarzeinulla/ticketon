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

// StaffStore manages the people an organizer puts on an event's door.
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
