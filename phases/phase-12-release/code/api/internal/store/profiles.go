package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Organizer profiles (SRS 4.1: "Organizers shall have a profile containing
// contact and payout information").
//
// The organizer_profiles table existed from Phase 1, but the only thing that
// ever wrote to it was the paid-sales activation checklist, which filled in a
// display name derived from the account and nothing else. An organizer had no
// way to see or edit their own profile, and no way to read back the payout
// destination they had registered.

// OrganizerProfile is an organizer's public-facing and contact detail.
type OrganizerProfile struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	DisplayName  string    `json:"display_name"`
	LegalName    *string   `json:"legal_name,omitempty"`
	ContactEmail *string   `json:"contact_email,omitempty"`
	ContactPhone *string   `json:"contact_phone,omitempty"`
	Description  *string   `json:"description,omitempty"`
	WebsiteURL   *string   `json:"website_url,omitempty"`
	// IdentityVerifiedAt is the simulated KYC timestamp (SRS 3.2). Production
	// identity verification is explicitly out of scope in SRS 8.
	IdentityVerifiedAt *time.Time `json:"identity_verified_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	// PayoutAccounts carries the masked destinations only. NFR section 7:
	// "Payment-card data shall not be stored directly by the platform", so
	// there is nothing here but an opaque reference and a masked display value.
	PayoutAccounts []PayoutAccount `json:"payout_accounts"`
}

// PayoutAccount is a masked payout destination.
type PayoutAccount struct {
	ID            uuid.UUID  `json:"id"`
	Provider      string     `json:"provider"`
	MaskedAccount *string    `json:"masked_account,omitempty"`
	Currency      string     `json:"currency"`
	Status        string     `json:"status"`
	IsSimulated   bool       `json:"is_simulated"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ProfileStore reads and writes organizer profiles.
type ProfileStore struct {
	pool *pgxpool.Pool
}

// NewProfileStore builds a ProfileStore.
func NewProfileStore(pool *pgxpool.Pool) *ProfileStore { return &ProfileStore{pool: pool} }

const profileColumns = `id, user_id, display_name, legal_name, contact_email,
	contact_phone, description, website_url, identity_verified_at,
	created_at, updated_at`

// Get returns an organizer's profile with its masked payout accounts.
func (s *ProfileStore) Get(ctx context.Context, userID uuid.UUID) (OrganizerProfile, error) {
	var p OrganizerProfile
	err := s.pool.QueryRow(ctx,
		`SELECT `+profileColumns+` FROM organizer_profiles WHERE user_id = $1`, userID).
		Scan(&p.ID, &p.UserID, &p.DisplayName, &p.LegalName, &p.ContactEmail,
			&p.ContactPhone, &p.Description, &p.WebsiteURL, &p.IdentityVerifiedAt,
			&p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizerProfile{}, ErrNotFound
	}
	if err != nil {
		return OrganizerProfile{}, mapError(err)
	}

	if p.PayoutAccounts, err = s.payoutAccounts(ctx, p.ID); err != nil {
		return OrganizerProfile{}, err
	}
	return p, nil
}

func (s *ProfileStore) payoutAccounts(ctx context.Context, profileID uuid.UUID) ([]PayoutAccount, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, provider, masked_account, currency, status::text, is_simulated,
		       verified_at, created_at
		  FROM payout_accounts
		 WHERE organizer_profile_id = $1
		 ORDER BY created_at`, profileID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	accounts := []PayoutAccount{}
	for rows.Next() {
		var a PayoutAccount
		if err := rows.Scan(&a.ID, &a.Provider, &a.MaskedAccount, &a.Currency,
			&a.Status, &a.IsSimulated, &a.VerifiedAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// ProfileUpdate is a PATCH body. Every field is Optional so an absent key
// leaves the column alone and an explicit null clears it, which is the same
// tri-state the event PATCH uses.
type ProfileUpdate struct {
	DisplayName  Optional[string]
	LegalName    Optional[string]
	ContactEmail Optional[string]
	ContactPhone Optional[string]
	Description  Optional[string]
	WebsiteURL   Optional[string]
}

// Upsert creates or updates an organizer's profile (SRS 4.1).
//
// It creates on first write rather than requiring a separate "create profile"
// step: an organizer who fills in their contact details expects that to be the
// whole action. The display name falls back to the account's own name, so the
// NOT NULL column is always satisfied without asking for it twice.
func (s *ProfileStore) Upsert(
	ctx context.Context, userID uuid.UUID, u ProfileUpdate,
) (OrganizerProfile, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OrganizerProfile{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var fallbackName string
	if err := tx.QueryRow(ctx, `SELECT full_name FROM users WHERE id = $1`, userID).
		Scan(&fallbackName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrganizerProfile{}, ErrNotFound
		}
		return OrganizerProfile{}, mapError(err)
	}

	displayName := fallbackName
	if u.DisplayName.Set && u.DisplayName.Valid {
		displayName = u.DisplayName.Value
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO organizer_profiles (user_id, display_name)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO NOTHING`, userID, displayName); err != nil {
		return OrganizerProfile{}, mapError(err)
	}

	// COALESCE on the "was this key present" flag rather than on the value, so
	// an explicit null clears the column instead of being read as "absent".
	if _, err := tx.Exec(ctx, `
		UPDATE organizer_profiles
		   SET display_name  = CASE WHEN $2::boolean THEN $3  ELSE display_name  END,
		       legal_name    = CASE WHEN $4::boolean THEN $5  ELSE legal_name    END,
		       contact_email = CASE WHEN $6::boolean THEN $7  ELSE contact_email END,
		       contact_phone = CASE WHEN $8::boolean THEN $9  ELSE contact_phone END,
		       description   = CASE WHEN $10::boolean THEN $11 ELSE description  END,
		       website_url   = CASE WHEN $12::boolean THEN $13 ELSE website_url  END
		 WHERE user_id = $1`,
		userID,
		u.DisplayName.Set && u.DisplayName.Valid, displayName,
		u.LegalName.Set, u.LegalName.Ptr(),
		u.ContactEmail.Set, u.ContactEmail.Ptr(),
		u.ContactPhone.Set, u.ContactPhone.Ptr(),
		u.Description.Set, u.Description.Ptr(),
		u.WebsiteURL.Set, u.WebsiteURL.Ptr(),
	); err != nil {
		return OrganizerProfile{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return OrganizerProfile{}, mapError(err)
	}
	return s.Get(ctx, userID)
}

// ChangePassword replaces an account's password hash (SRS 4.1: "Users shall be
// able to sign in, sign out, and reset passwords").
//
// The caller has already verified the current password. This exists as its own
// method so the hash column is written in exactly one more place than
// registration, and so the write can be paired with invalidating outstanding
// reset tokens in a single transaction: changing a password has to close any
// reset link that was already in flight.
func (s *ProfileStore) ChangePassword(ctx context.Context, userID uuid.UUID, hash string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, hash)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE user_tokens SET consumed_at = now()
		 WHERE user_id = $1 AND purpose = 'password_reset' AND consumed_at IS NULL`,
		userID); err != nil {
		return mapError(err)
	}

	return mapError(tx.Commit(ctx))
}
