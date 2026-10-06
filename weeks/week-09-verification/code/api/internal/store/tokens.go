package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Token purposes, mirroring the user_token_purpose enum.
const (
	TokenEmailVerification = "email_verification"
	TokenPasswordReset     = "password_reset"
)

// Token lifetimes.
//
// A reset token is short-lived because it is a password in disguise: anyone
// holding it can take the account. A verification token is only a claim about
// an address, so it can afford to survive a weekend in an inbox.
const (
	PasswordResetTTL     = 1 * time.Hour
	EmailVerificationTTL = 72 * time.Hour
)

// ErrTokenInvalid covers every way a token can fail to work: unknown, expired,
// or already used.
//
// They are deliberately one error. Telling a caller that a token exists but has
// expired confirms the token was real, which is exactly the fact an attacker
// guessing tokens is trying to learn.
var ErrTokenInvalid = errors.New("token is invalid, expired or already used")

// IssuedToken is what the caller emails out. The plaintext exists only in this
// struct and in the message; the database sees the hash.
type IssuedToken struct {
	Token     string
	ExpiresAt time.Time
	UserID    uuid.UUID
	Email     string
	FullName  string
}

// TokenStore issues and consumes single-use account tokens (SRS 4.1).
type TokenStore struct {
	pool *pgxpool.Pool
}

// NewTokenStore builds a TokenStore.
func NewTokenStore(pool *pgxpool.Pool) *TokenStore { return &TokenStore{pool: pool} }

// HashToken is the one-way function between what the user holds and what is
// stored. Exported so tests can look a token up the way the database does.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// newToken returns a URL-safe random string with 256 bits of entropy.
func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// Issue creates a token for a user, invalidating any earlier one for the same
// purpose.
//
// Superseding rather than accumulating matters: someone who clicks "forgot
// password" three times should not leave three working keys to their account
// lying in an inbox.
func (s *TokenStore) Issue(
	ctx context.Context, userID uuid.UUID, purpose string, ttl time.Duration,
) (IssuedToken, error) {
	token, err := newToken()
	if err != nil {
		return IssuedToken{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return IssuedToken{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE user_tokens SET consumed_at = now()
		 WHERE user_id = $1 AND purpose = $2::user_token_purpose AND consumed_at IS NULL`,
		userID, purpose); err != nil {
		return IssuedToken{}, mapError(err)
	}

	issued := IssuedToken{Token: token, UserID: userID}
	err = tx.QueryRow(ctx, `
		INSERT INTO user_tokens (user_id, purpose, token_hash, expires_at)
		VALUES ($1, $2::user_token_purpose, $3, now() + $4::interval)
		RETURNING expires_at`,
		userID, purpose, HashToken(token), ttl.String(),
	).Scan(&issued.ExpiresAt)
	if err != nil {
		return IssuedToken{}, mapError(err)
	}

	if err := tx.QueryRow(ctx,
		`SELECT email::text, full_name FROM users WHERE id = $1`, userID,
	).Scan(&issued.Email, &issued.FullName); err != nil {
		return IssuedToken{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return IssuedToken{}, mapError(err)
	}
	return issued, nil
}

// IssueForEmail issues a token addressed by email rather than by id.
//
// It returns ErrNotFound for an unknown address. The *handler* must not pass
// that distinction on to the caller - "no account with this email" turns the
// password-reset form into a way of testing which addresses are registered -
// but the store still reports honestly and lets the handler decide.
func (s *TokenStore) IssueForEmail(
	ctx context.Context, email, purpose string, ttl time.Duration,
) (IssuedToken, error) {
	var userID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return IssuedToken{}, ErrNotFound
	}
	if err != nil {
		return IssuedToken{}, mapError(err)
	}
	return s.Issue(ctx, userID, purpose, ttl)
}

// consume validates a token and marks it used, returning the user it belongs
// to. The whole check happens in one UPDATE so a token cannot be redeemed
// twice by two concurrent requests.
func (s *TokenStore) consume(ctx context.Context, tx pgx.Tx, token, purpose string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := tx.QueryRow(ctx, `
		UPDATE user_tokens
		   SET consumed_at = now()
		 WHERE token_hash = $1
		   AND purpose = $2::user_token_purpose
		   AND consumed_at IS NULL
		   AND expires_at > now()
		RETURNING user_id`, HashToken(token), purpose).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrTokenInvalid
	}
	if err != nil {
		return uuid.Nil, mapError(err)
	}
	return userID, nil
}

// VerifiedUser is the account state after a token has done its work.
type VerifiedUser struct {
	ID       uuid.UUID
	Email    string
	FullName string
	Status   string
}

// ConsumeEmailVerification marks an address verified (SRS 4.1).
func (s *TokenStore) ConsumeEmailVerification(ctx context.Context, token string) (VerifiedUser, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return VerifiedUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	userID, err := s.consume(ctx, tx, token, TokenEmailVerification)
	if err != nil {
		return VerifiedUser{}, err
	}

	// An account only leaves pending_verification here. A suspended account
	// verifying its address stays suspended: confirming an inbox is not an
	// appeal against moderation.
	var user VerifiedUser
	err = tx.QueryRow(ctx, `
		UPDATE users
		   SET email_verified_at = COALESCE(email_verified_at, now()),
		       status = CASE WHEN status = 'pending_verification'
		                     THEN 'active'::user_status ELSE status END
		 WHERE id = $1
		RETURNING id, email::text, full_name, status::text`, userID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.Status)
	if err != nil {
		return VerifiedUser{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return VerifiedUser{}, mapError(err)
	}
	return user, nil
}

// ConsumePasswordReset sets a new password hash and invalidates every other
// outstanding token for the account.
//
// The second part matters: someone recovering an account they lost control of
// should not leave the attacker holding a working verification link.
func (s *TokenStore) ConsumePasswordReset(
	ctx context.Context, token, passwordHash string,
) (VerifiedUser, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return VerifiedUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	userID, err := s.consume(ctx, tx, token, TokenPasswordReset)
	if err != nil {
		return VerifiedUser{}, err
	}

	var user VerifiedUser
	err = tx.QueryRow(ctx, `
		UPDATE users SET password_hash = $2 WHERE id = $1
		RETURNING id, email::text, full_name, status::text`, userID, passwordHash,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.Status)
	if err != nil {
		return VerifiedUser{}, mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE user_tokens SET consumed_at = now()
		 WHERE user_id = $1 AND consumed_at IS NULL`, userID); err != nil {
		return VerifiedUser{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return VerifiedUser{}, mapError(err)
	}
	return user, nil
}
