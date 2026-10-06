package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User is an account as the API exposes it. The password hash is deliberately
// absent so it can never be serialised into a response by accident.
type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	FullName        string     `json:"full_name"`
	Phone           *string    `json:"phone,omitempty"`
	Locale          string     `json:"locale"`
	Status          string     `json:"status"`
	Roles           []string   `json:"roles"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Credentials pairs a user with the stored hash, for the login path only.
type Credentials struct {
	User         User
	PasswordHash string
}

// Account status values from the user_status enum.
const (
	StatusPendingVerification = "pending_verification"
	StatusActive              = "active"
	StatusSuspended           = "suspended"
	StatusDeactivated         = "deactivated"
)

// Role values from the user_role enum.
const (
	RoleAttendee      = "attendee"
	RoleOrganizer     = "organizer"
	RoleEventAdmin    = "event_admin"
	RoleSupportStaff  = "support_staff"
	RolePlatformAdmin = "platform_admin"
)

// UserStore reads and writes accounts.
type UserStore struct {
	pool *pgxpool.Pool
}

// NewUserStore builds a UserStore.
func NewUserStore(pool *pgxpool.Pool) *UserStore { return &UserStore{pool: pool} }

// CreateUserParams describes a new account.
type CreateUserParams struct {
	Email        string
	PasswordHash string
	FullName     string
	Phone        *string
	Locale       string
	Roles        []string
}

const userColumns = `id, email, full_name, phone, locale, status,
	email_verified_at, last_login_at, created_at, updated_at`

// Create inserts a user and its initial roles in one transaction.
func (s *UserStore) Create(ctx context.Context, p CreateUserParams) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var u User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, phone, locale)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+userColumns,
		p.Email, p.PasswordHash, p.FullName, p.Phone, p.Locale,
	).Scan(&u.ID, &u.Email, &u.FullName, &u.Phone, &u.Locale, &u.Status,
		&u.EmailVerifiedAt, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return User{}, mapError(err)
	}

	for _, role := range p.Roles {
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`, u.ID, role); err != nil {
			return User{}, mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, mapError(err)
	}

	u.Roles = append([]string{}, p.Roles...)
	return u, nil
}

// GetByID returns a user with its roles.
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.FullName, &u.Phone, &u.Locale, &u.Status,
			&u.EmailVerifiedAt, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, mapError(err)
	}

	if u.Roles, err = s.RolesFor(ctx, u.ID); err != nil {
		return User{}, err
	}
	return u, nil
}

// GetCredentialsByEmail returns the user and password hash for login. The
// lookup is case-insensitive because users.email is a citext column.
func (s *UserStore) GetCredentialsByEmail(ctx context.Context, email string) (Credentials, error) {
	var c Credentials
	err := s.pool.QueryRow(ctx,
		`SELECT `+userColumns+`, password_hash FROM users WHERE email = $1`, email).
		Scan(&c.User.ID, &c.User.Email, &c.User.FullName, &c.User.Phone, &c.User.Locale,
			&c.User.Status, &c.User.EmailVerifiedAt, &c.User.LastLoginAt,
			&c.User.CreatedAt, &c.User.UpdatedAt, &c.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, ErrNotFound
	}
	if err != nil {
		return Credentials{}, mapError(err)
	}

	if c.User.Roles, err = s.RolesFor(ctx, c.User.ID); err != nil {
		return Credentials{}, err
	}
	return c, nil
}

// RolesFor returns the roles granted to a user, alphabetically.
func (s *UserStore) RolesFor(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT role::text FROM user_roles WHERE user_id = $1 ORDER BY role::text`, userID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	roles := []string{}
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

// GrantRole adds a role, ignoring the case where it is already held.
func (s *UserStore) GrantRole(ctx context.Context, userID uuid.UUID, role string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, role)
	return mapError(err)
}

// TouchLastLogin records a successful sign-in.
func (s *UserStore) TouchLastLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, userID)
	return mapError(err)
}

// Restore lifts a suspension (SRS 4.12).
//
// It deliberately does not always return the account to 'active': an account
// that had never confirmed its address goes back to 'pending_verification'.
// Restoring it to 'active' would use a moderation action to grant a
// verification the person never completed.
func (s *UserStore) Restore(ctx context.Context, id uuid.UUID) (User, error) {
	var (
		current  string
		verified *time.Time
	)
	err := s.pool.QueryRow(ctx,
		`SELECT status::text, email_verified_at FROM users WHERE id = $1`, id).
		Scan(&current, &verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, mapError(err)
	}
	if current != StatusSuspended {
		return User{}, ErrStatusUnchanged
	}

	restored := StatusPendingVerification
	if verified != nil {
		restored = StatusActive
	}
	return s.SetStatus(ctx, id, restored)
}

// SetStatus moves an account between the user_status values (SRS 4.12:
// "Suspend users or events").
//
// It returns ErrNotFound for an unknown id and ErrStatusUnchanged when the
// account is already in the requested state, so the caller can tell a
// double-click apart from a successful moderation action rather than reporting
// a no-op as a change.
func (s *UserStore) SetStatus(ctx context.Context, id uuid.UUID, status string) (User, error) {
	var current string
	err := s.pool.QueryRow(ctx, `SELECT status::text FROM users WHERE id = $1`, id).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, mapError(err)
	}
	if current == status {
		return User{}, ErrStatusUnchanged
	}

	var u User
	err = s.pool.QueryRow(ctx,
		`UPDATE users SET status = $2::user_status WHERE id = $1 RETURNING `+userColumns, id, status).
		Scan(&u.ID, &u.Email, &u.FullName, &u.Phone, &u.Locale, &u.Status,
			&u.EmailVerifiedAt, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return User{}, mapError(err)
	}

	if u.Roles, err = s.RolesFor(ctx, u.ID); err != nil {
		return User{}, err
	}
	return u, nil
}
