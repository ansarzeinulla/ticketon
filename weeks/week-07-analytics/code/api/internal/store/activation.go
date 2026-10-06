package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Activation statuses, mirroring the activation_status enum.
const (
	ActivationNotStarted = "not_started"
	ActivationInProgress = "in_progress"
	ActivationActive     = "active"
	ActivationSuspended  = "suspended"
)

// SimulatedActivationFeeKZT is the fallback activation fee.
//
// SRS 4.5 requires the system to record payment of an activation fee. Nothing
// is charged to anybody: the payment row it produces carries is_simulated =
// true, and SRS 4.6 forbids presenting demonstration records as real
// transactions.
//
// SRS 4.12 requires administrators to be able to "configure activation fees",
// so the live value comes from platform_settings. This constant is what the
// store falls back to when that row is missing or unreadable - a misconfigured
// setting must not stop an organizer activating paid sales.
const SimulatedActivationFeeKZT = "5000.00"

// feeKZT reads the configured activation fee. It is a method so the store can
// consult platform_settings without every caller having to.
func (s *ActivationStore) feeKZT(ctx context.Context) string {
	var raw []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT value FROM platform_settings WHERE key = 'activation_fee_kzt'`,
	).Scan(&raw); err != nil {
		return SimulatedActivationFeeKZT
	}
	var fee string
	if err := json.Unmarshal(raw, &fee); err != nil || blankAmount(fee) {
		return SimulatedActivationFeeKZT
	}
	return fee
}

// Activation is an event's paid-sales activation state (SRS 4.5).
type Activation struct {
	EventID uuid.UUID `json:"event_id"`
	Status  string    `json:"status"`

	// The checklist. Each timestamp is the moment that step was completed;
	// nil means it is still outstanding.
	IdentityVerifiedAt *time.Time `json:"identity_verified_at,omitempty"`
	PayoutVerifiedAt   *time.Time `json:"payout_verified_at,omitempty"`
	TermsAcceptedAt    *time.Time `json:"terms_accepted_at,omitempty"`
	FeePaidAt          *time.Time `json:"fee_paid_at,omitempty"`

	ActivationFeeKZT string     `json:"activation_fee_kzt"`
	ActivatedAt      *time.Time `json:"activated_at,omitempty"`
	SuspendedAt      *time.Time `json:"suspended_at,omitempty"`
	SuspensionReason *string    `json:"suspension_reason,omitempty"`

	// Derived, so a client never has to reimplement the rule.
	IsActive bool `json:"is_active"`
	// RequiredForSales reports whether this event actually has paid tickets.
	// A free event never needs activating, and the dashboard should not nag
	// about a checklist that gates nothing.
	RequiredForSales bool `json:"required_for_sales"`
	// Outstanding names the steps still to do, in the order the UI shows them.
	Outstanding []string `json:"outstanding"`
}

// Checklist step identifiers, used in the request body and in Outstanding.
const (
	StepIdentity = "identity"
	StepPayout   = "payout"
	StepTerms    = "terms"
	StepFee      = "fee"
)

// ErrPaidSalesSuspended reports an activation a platform admin has stopped.
var ErrPaidSalesSuspended = errors.New("paid sales are suspended for this event")

// ActivationSteps is what an organizer is completing in one request. A step
// that is false is left alone rather than cleared: the checklist only ever
// moves forwards, and a half-filled form should not undo yesterday's progress.
type ActivationSteps struct {
	Identity bool
	Payout   bool
	Terms    bool
	Fee      bool
}

// ActivationStore reads and advances paid-sales activation.
type ActivationStore struct {
	pool *pgxpool.Pool
}

// NewActivationStore builds an ActivationStore.
func NewActivationStore(pool *pgxpool.Pool) *ActivationStore {
	return &ActivationStore{pool: pool}
}

// activationColumns is shared by every read so the row shape stays in one
// place. fee_paid_at is derived from the linked payment rather than stored
// twice.
const activationColumns = `
	a.event_id, a.status::text, a.identity_verified_at, a.payout_verified_at,
	a.terms_accepted_at, a.activation_fee_kzt::text, a.activated_at,
	a.suspended_at, a.suspension_reason, p.paid_at`

func scanActivation(row pgx.Row) (Activation, error) {
	var a Activation
	err := row.Scan(&a.EventID, &a.Status, &a.IdentityVerifiedAt, &a.PayoutVerifiedAt,
		&a.TermsAcceptedAt, &a.ActivationFeeKZT, &a.ActivatedAt,
		&a.SuspendedAt, &a.SuspensionReason, &a.FeePaidAt)
	if err != nil {
		return Activation{}, err
	}
	a.finish()
	return a, nil
}

// finish fills in the derived fields.
func (a *Activation) finish() {
	a.IsActive = a.Status == ActivationActive

	a.Outstanding = []string{}
	if a.IdentityVerifiedAt == nil {
		a.Outstanding = append(a.Outstanding, StepIdentity)
	}
	if a.PayoutVerifiedAt == nil {
		a.Outstanding = append(a.Outstanding, StepPayout)
	}
	if a.TermsAcceptedAt == nil {
		a.Outstanding = append(a.Outstanding, StepTerms)
	}
	if a.FeePaidAt == nil {
		a.Outstanding = append(a.Outstanding, StepFee)
	}
}

// ForEvent returns the activation state, whether or not a row exists yet.
//
// An event with no row has simply never started, which is a state worth
// reporting rather than an error: the dashboard needs to render the checklist
// before anybody has touched it.
func (s *ActivationStore) ForEvent(ctx context.Context, eventID uuid.UUID) (Activation, error) {
	a, err := scanActivation(s.pool.QueryRow(ctx, `
		SELECT `+activationColumns+`
		  FROM paid_sales_activations a
		  LEFT JOIN payments p ON p.id = a.activation_payment_id
		 WHERE a.event_id = $1`, eventID))

	if errors.Is(err, pgx.ErrNoRows) {
		a = Activation{
			EventID:          eventID,
			Status:           ActivationNotStarted,
			ActivationFeeKZT: s.feeKZT(ctx),
		}
		a.finish()
	} else if err != nil {
		return Activation{}, mapError(err)
	}

	required, err := s.paidTicketsExist(ctx, eventID)
	if err != nil {
		return Activation{}, err
	}
	a.RequiredForSales = required
	return a, nil
}

// paidTicketsExist reports whether the event sells anything above zero KZT.
func (s *ActivationStore) paidTicketsExist(ctx context.Context, eventID uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM ticket_types
		                WHERE event_id = $1 AND price_kzt > 0)`, eventID).Scan(&exists)
	if err != nil {
		return false, mapError(err)
	}
	return exists, nil
}

// AdvanceParams carries one checklist submission.
type AdvanceParams struct {
	EventID     uuid.UUID
	OrganizerID uuid.UUID
	Steps       ActivationSteps
}

// Advance completes the checklist steps in one transaction and activates paid
// sales once every step is done (SRS 4.5).
//
// The steps are deliberately not four separate endpoints. An organizer ticking
// the last box expects sales to open in that same click, and splitting the
// final transition into its own call would leave a window where the checklist
// reads complete while sales are still shut.
func (s *ActivationStore) Advance(ctx context.Context, p AdvanceParams) (Activation, error) {
	// Read once, so the checklist row, the payment row and the audit line all
	// quote the same figure even if an administrator changes it mid-flight.
	fee := s.feeKZT(ctx)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Activation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// --- 1. The organizer profile the activation belongs to ------------------
	// organizer_profile_id is NOT NULL, and a profile is created lazily rather
	// than at registration: most accounts never organize anything, and an
	// empty profile row for every signup is noise.
	var profileID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO organizer_profiles (user_id, display_name, contact_email)
		SELECT id, COALESCE(NULLIF(btrim(full_name), ''), email::text), email
		  FROM users WHERE id = $1
		ON CONFLICT (user_id) DO UPDATE SET updated_at = now()
		RETURNING id`, p.OrganizerID).Scan(&profileID)
	if err != nil {
		return Activation{}, mapError(err)
	}

	// --- 2. The activation row, locked ---------------------------------------
	var (
		activationID uuid.UUID
		status       string
		paymentID    *uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		INSERT INTO paid_sales_activations (event_id, organizer_profile_id,
		                                    activation_fee_kzt, status)
		VALUES ($1, $2, $3::numeric, 'in_progress')
		ON CONFLICT (event_id) DO UPDATE SET updated_at = now()
		RETURNING id, status::text, activation_payment_id`,
		p.EventID, profileID, fee,
	).Scan(&activationID, &status, &paymentID)
	if err != nil {
		return Activation{}, mapError(err)
	}

	// A platform admin's suspension outranks the organizer's checklist
	// (SRS 4.5, final bullet). Only an admin can lift it.
	if status == ActivationSuspended {
		return Activation{}, ErrPaidSalesSuspended
	}

	// --- 3. The simulated fee payment ----------------------------------------
	// Written once. Re-submitting the form must not mint a second payment row
	// for a fee that was already "paid".
	if p.Steps.Fee && paymentID == nil {
		var newPaymentID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO payments (purpose, event_id, payer_user_id, amount_kzt, status,
			                      provider, provider_payment_ref, is_simulated, paid_at)
			VALUES ('paid_sales_activation', $1, $2, $3::numeric, 'succeeded',
			        'simulated', $4, true, now())
			RETURNING id`,
			p.EventID, p.OrganizerID, fee,
			"sim_activation_"+p.EventID.String(),
		).Scan(&newPaymentID)
		if err != nil {
			return Activation{}, mapError(err)
		}
		paymentID = &newPaymentID
	}

	// --- 3b. The simulated payout destination -------------------------------
	// SRS 4.5 step 4 is "Connects or registers a valid payout account", and
	// SRS 4.1 says the organizer's profile carries payout information. Ticking
	// the box used to set a timestamp and nothing else, so the destination the
	// organizer had supposedly registered existed nowhere.
	//
	// NFR section 7: "Payment-card data shall not be stored directly by the
	// platform." What is stored is an opaque provider reference and a masked
	// display value - never an account number.
	if p.Steps.Payout {
		if _, err := tx.Exec(ctx, `
			INSERT INTO payout_accounts (organizer_profile_id, provider,
			                             provider_account_ref, masked_account,
			                             status, is_simulated, verified_at)
			VALUES ($1, 'simulated', $2, $3, 'verified', true, now())
			ON CONFLICT (organizer_profile_id, provider, provider_account_ref)
			DO NOTHING`,
			profileID,
			"sim_acct_"+p.OrganizerID.String(),
			"**** "+lastFour(p.OrganizerID.String()),
		); err != nil {
			return Activation{}, mapError(err)
		}
	}

	// --- 4. Stamp the completed steps ----------------------------------------
	// COALESCE keeps the first completion time: re-ticking a box already done
	// should not rewrite when it happened.
	_, err = tx.Exec(ctx, `
		UPDATE paid_sales_activations
		   SET identity_verified_at = CASE WHEN $2 THEN COALESCE(identity_verified_at, now())
		                                   ELSE identity_verified_at END,
		       payout_verified_at   = CASE WHEN $3 THEN COALESCE(payout_verified_at, now())
		                                   ELSE payout_verified_at END,
		       terms_accepted_at    = CASE WHEN $4 THEN COALESCE(terms_accepted_at, now())
		                                   ELSE terms_accepted_at END,
		       activation_payment_id = COALESCE($5, activation_payment_id)
		 WHERE id = $1`,
		activationID, p.Steps.Identity, p.Steps.Payout, p.Steps.Terms, paymentID)
	if err != nil {
		return Activation{}, mapError(err)
	}

	// --- 5. Activate, if and only if the checklist is complete ---------------
	// The condition is evaluated in SQL against the row as it now stands, not
	// against what this process believes it wrote. activations_checklist_chk
	// enforces the same rule as a constraint, so an 'active' row with a gap in
	// its checklist cannot exist even if this statement were wrong.
	_, err = tx.Exec(ctx, `
		UPDATE paid_sales_activations
		   SET status       = 'active',
		       activated_at = COALESCE(activated_at, now())
		 WHERE id = $1
		   AND status <> 'suspended'
		   AND identity_verified_at   IS NOT NULL
		   AND payout_verified_at     IS NOT NULL
		   AND terms_accepted_at      IS NOT NULL
		   AND activation_payment_id  IS NOT NULL`, activationID)
	if err != nil {
		return Activation{}, mapError(err)
	}

	// events.paid_sales_enabled is the flag the rest of the schema reads. It
	// is kept in step with the activation rather than being a second source of
	// truth an organizer could toggle independently.
	if _, err := tx.Exec(ctx, `
		UPDATE events e
		   SET paid_sales_enabled = (a.status = 'active')
		  FROM paid_sales_activations a
		 WHERE a.id = $1 AND e.id = a.event_id`, activationID); err != nil {
		return Activation{}, mapError(err)
	}

	activation, err := scanActivation(tx.QueryRow(ctx, `
		SELECT `+activationColumns+`
		  FROM paid_sales_activations a
		  LEFT JOIN payments p ON p.id = a.activation_payment_id
		 WHERE a.id = $1`, activationID))
	if err != nil {
		return Activation{}, mapError(err)
	}

	// --- 6. The audit entry --------------------------------------------------
	action, description := "event.activation_updated", "Paid-sales activation checklist updated"
	if activation.IsActive && status != ActivationActive {
		action = "event.paid_sales_activated"
		description = "Paid sales activated; the simulated " +
			fee + " KZT activation fee was recorded"
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id,
		                        description, metadata)
		VALUES ($1, $2, $3, 'event', $4, $5, jsonb_build_object('simulated', true))`,
		p.EventID, p.OrganizerID, action, p.EventID.String(), description,
	); err != nil {
		return Activation{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Activation{}, mapError(err)
	}

	required, err := s.paidTicketsExist(ctx, p.EventID)
	if err != nil {
		return Activation{}, err
	}
	activation.RequiredForSales = required
	return activation, nil
}

// SetSuspended stops or restores paid sales for one event (SRS 4.5, final
// bullet: platform administrators may suspend paid sales where fraud or a
// policy violation is suspected).
//
// This is narrower than the Phase 8 event suspension, which stops an event
// selling anything at all. Suspending paid sales leaves free registration
// working, which is the proportionate response when the concern is about money
// rather than about the event itself.
func (s *ActivationStore) SetSuspended(
	ctx context.Context, eventID, adminID uuid.UUID, suspended bool, reason string,
) (Activation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Activation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var activationID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM paid_sales_activations WHERE event_id = $1`,
		eventID).Scan(&activationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Activation{}, ErrNotFound
	}
	if err != nil {
		return Activation{}, mapError(err)
	}

	if suspended {
		_, err = tx.Exec(ctx, `
			UPDATE paid_sales_activations
			   SET status = 'suspended', suspended_at = now(),
			       suspended_by = $2, suspension_reason = $3
			 WHERE id = $1`, activationID, adminID, nullableString(reason))
	} else {
		// Lifting returns the event to in_progress, not straight to active:
		// the same UPDATE that activates on completion decides that, and it
		// re-checks the whole checklist rather than trusting the old status.
		_, err = tx.Exec(ctx, `
			UPDATE paid_sales_activations
			   SET status = CASE
			           WHEN identity_verified_at IS NOT NULL
			            AND payout_verified_at   IS NOT NULL
			            AND terms_accepted_at    IS NOT NULL
			            AND activation_payment_id IS NOT NULL THEN 'active'::activation_status
			           ELSE 'in_progress'::activation_status END,
			       suspended_at = NULL, suspended_by = NULL, suspension_reason = NULL
			 WHERE id = $1`, activationID)
	}
	if err != nil {
		return Activation{}, mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE events e
		   SET paid_sales_enabled = (a.status = 'active')
		  FROM paid_sales_activations a
		 WHERE a.id = $1 AND e.id = a.event_id`, activationID); err != nil {
		return Activation{}, mapError(err)
	}

	action, description := "event.paid_sales_unsuspended", "Paid sales restored by BiletFlow"
	if suspended {
		action, description = "event.paid_sales_suspended", "Paid sales suspended by BiletFlow"
		if reason != "" {
			description += ": " + reason
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description)
		VALUES ($1, $2, $3, 'event', $4, $5)`,
		eventID, adminID, action, eventID.String(), description); err != nil {
		return Activation{}, mapError(err)
	}

	activation, err := scanActivation(tx.QueryRow(ctx, `
		SELECT `+activationColumns+`
		  FROM paid_sales_activations a
		  LEFT JOIN payments p ON p.id = a.activation_payment_id
		 WHERE a.id = $1`, activationID))
	if err != nil {
		return Activation{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Activation{}, mapError(err)
	}

	required, err := s.paidTicketsExist(ctx, eventID)
	if err != nil {
		return Activation{}, err
	}
	activation.RequiredForSales = required
	return activation, nil
}

// lastFour renders the trailing digits of an opaque reference as a masked
// display value. It is cosmetic: there is no account number to mask, because
// the platform never receives one (NFR section 7).
func lastFour(ref string) string {
	digits := make([]rune, 0, 4)
	for i := len(ref) - 1; i >= 0 && len(digits) < 4; i-- {
		if ref[i] >= '0' && ref[i] <= '9' {
			digits = append([]rune{rune(ref[i])}, digits...)
		}
	}
	for len(digits) < 4 {
		digits = append(digits, '0')
	}
	return string(digits)
}
