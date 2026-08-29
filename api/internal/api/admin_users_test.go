package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// SRS 4.12: "Suspend users or events."
//
// Event suspension has existed since Phase 8. User suspension did not: the
// users.status enum carried 'suspended' and every authorised request already
// honoured it, but the only way to set it was raw SQL. These tests pin the
// endpoint that closes that half of the requirement.

// promoteToPlatformAdmin grants the platform_admin role directly, the way the
// smoke script does - there is deliberately no endpoint that hands out this
// role, since anyone who could call it would already have to be an admin.
func (c *client) promoteToPlatformAdmin(userID uuid.UUID) {
	c.t.Helper()
	if _, err := c.pool.Exec(c.t.Context(),
		`INSERT INTO user_roles (user_id, role) VALUES ($1, 'platform_admin')
		 ON CONFLICT DO NOTHING`, userID); err != nil {
		c.t.Fatalf("grant platform_admin: %v", err)
	}
}

// adminAccount registers an account and promotes it, returning a fresh token
// that carries the new role.
func (c *client) adminAccount(prefix string) account {
	c.t.Helper()
	admin := c.register(prefix)
	c.promoteToPlatformAdmin(admin.ID)

	// The role is read from the account on every request, but the token also
	// carries a role list, so a re-login keeps the two consistent.
	res := c.post("/api/v1/auth/login", "", map[string]any{
		"email": admin.Email, "password": admin.Password,
	})
	requireStatus(c.t, res, http.StatusOK)
	admin.Token = res.Body["access_token"].(string)
	return admin
}

func (c *client) userStatus(userID uuid.UUID) string {
	c.t.Helper()
	var status string
	if err := c.pool.QueryRow(c.t.Context(),
		`SELECT status::text FROM users WHERE id = $1`, userID).Scan(&status); err != nil {
		c.t.Fatalf("read user status: %v", err)
	}
	return status
}

// TestAdminCanSuspendAndRestoreAUser is the requirement itself.
func TestAdminCanSuspendAndRestoreAUser(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("usersuspendadmin")
	target := c.register("usersuspendtarget")

	res := c.post("/api/v1/admin/users/"+target.ID.String()+"/suspend", admin.Token,
		map[string]any{"reason": "repeated fraudulent listings"})
	requireStatus(t, res, http.StatusOK)

	if got := c.userStatus(target.ID); got != store.StatusSuspended {
		t.Fatalf("user status = %q, want suspended", got)
	}

	// Restoring puts an unverified account back where it was, not straight to
	// active: lifting a suspension must not double as granting a verification.
	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/unsuspend",
		admin.Token, nil), http.StatusOK)
	if got := c.userStatus(target.ID); got != store.StatusPendingVerification {
		t.Errorf("user status = %q after restore, want pending_verification", got)
	}

	// A verified account does go back to active.
	if _, err := c.pool.Exec(t.Context(),
		`UPDATE users SET email_verified_at = now(), status = 'active' WHERE id = $1`,
		target.ID); err != nil {
		t.Fatalf("mark verified: %v", err)
	}
	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/suspend",
		admin.Token, nil), http.StatusOK)
	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/unsuspend",
		admin.Token, nil), http.StatusOK)
	if got := c.userStatus(target.ID); got != store.StatusActive {
		t.Errorf("verified account status = %q after restore, want active", got)
	}
}

// TestSuspensionTakesEffectOnTheNextRequest is what makes the endpoint worth
// having: the target's existing token has to stop working at once, not when it
// expires 24 hours later.
func TestSuspensionTakesEffectOnTheNextRequest(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("suspendnowadmin")
	target := c.register("suspendnowtarget")

	// The token works before.
	requireStatus(t, c.get("/api/v1/auth/me", target.Token), http.StatusOK)

	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/suspend",
		admin.Token, nil), http.StatusOK)

	// And is refused immediately after, with the same token.
	requireStatus(t, c.get("/api/v1/auth/me", target.Token), http.StatusForbidden)

	// They cannot sign in again either.
	requireStatus(t, c.post("/api/v1/auth/login", "", map[string]any{
		"email": target.Email, "password": target.Password,
	}), http.StatusForbidden)

	// Restoring gives the account back, old token included.
	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/unsuspend",
		admin.Token, nil), http.StatusOK)
	requireStatus(t, c.get("/api/v1/auth/me", target.Token), http.StatusOK)
}

// TestUserSuspensionIsPlatformAdminOnly - the power to lock somebody out of
// their account cannot be self-service.
func TestUserSuspensionIsPlatformAdminOnly(t *testing.T) {
	c := newClient(t)
	ordinary := c.register("suspendauthzuser")
	target := c.register("suspendauthztarget")

	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/suspend", "", nil),
		http.StatusUnauthorized)
	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/suspend",
		ordinary.Token, nil), http.StatusForbidden)

	if got := c.userStatus(target.ID); got == store.StatusSuspended {
		t.Error("user was suspended by an account with no right to do it")
	}
}

// TestAdminCannotSuspendThemselves stops an administrator locking the platform
// out of its own moderation tools by accident.
func TestAdminCannotSuspendThemselves(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("selfsuspendadmin")

	requireErrorCode(t, c.post("/api/v1/admin/users/"+admin.ID.String()+"/suspend",
		admin.Token, nil), http.StatusConflict, CodeCannotSuspendSelf)

	if got := c.userStatus(admin.ID); got == store.StatusSuspended {
		t.Error("the admin suspended themselves")
	}
}

// TestUserSuspensionIsIdempotentlyRefused gives a double click a clear answer.
func TestUserSuspensionIsIdempotentlyRefused(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("suspendtwiceadmin")
	target := c.register("suspendtwicetarget")

	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/suspend",
		admin.Token, nil), http.StatusOK)
	requireErrorCode(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/suspend",
		admin.Token, nil), http.StatusConflict, httpx.CodeConflict)

	// And un-suspending something that is not suspended is refused too.
	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/unsuspend",
		admin.Token, nil), http.StatusOK)
	requireErrorCode(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/unsuspend",
		admin.Token, nil), http.StatusConflict, httpx.CodeConflict)
}

// TestUnknownUserIsNotFound - a UUID that belongs to nobody.
func TestUnknownUserIsNotFound(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("suspendmissingadmin")

	requireStatus(t, c.post("/api/v1/admin/users/"+uuid.NewString()+"/suspend",
		admin.Token, nil), http.StatusNotFound)
	requireStatus(t, c.post("/api/v1/admin/users/not-a-uuid/suspend", admin.Token, nil),
		http.StatusBadRequest)
}

// TestUserSuspensionIsAudited - SRS 7 requires administrator actions to be
// auditable, and SRS 4.16 makes the log append-only.
func TestUserSuspensionIsAudited(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("suspendauditadmin")
	target := c.register("suspendaudittarget")

	requireStatus(t, c.post("/api/v1/admin/users/"+target.ID.String()+"/suspend",
		admin.Token, map[string]any{"reason": "policy violation"}), http.StatusOK)

	var description string
	var actor uuid.UUID
	if err := c.pool.QueryRow(t.Context(), `
		SELECT description, actor_user_id FROM audit_logs
		 WHERE action = 'user.suspended' AND entity_id = $1`,
		target.ID.String()).Scan(&description, &actor); err != nil {
		t.Fatalf("read audit entry: %v", err)
	}
	if actor != admin.ID {
		t.Errorf("actor = %s, want the admin %s", actor, admin.ID)
	}
	if description == "" {
		t.Error("audit description is empty")
	}
}

// TestSuspendedOrganizerStopsSellingTickets is the consequence SRS 4.12 is
// reaching for: suspending a fraudulent organizer has to stop their events
// taking money, not merely block their login.
func TestSuspendedOrganizerStopsSellingTickets(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("suspendsellsadmin")
	organizer := c.register("suspendsellsorganizer")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Sales Stop Here", "5000", 10)

	// Selling works before the suspension.
	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Early Buyer", "early@biletflow.test"),
		http.StatusCreated)

	requireStatus(t, c.post("/api/v1/admin/users/"+organizer.ID.String()+"/suspend",
		admin.Token, map[string]any{"reason": "fraud"}), http.StatusOK)

	// And is refused afterwards, before any inventory moves.
	res := c.buy(eventID, ticketTypeID, 1, "Late Buyer", "late@biletflow.test")
	requireErrorCode(t, res, http.StatusForbidden, CodeOrganizerSuspended)

	if sold, _ := c.soldFor(ticketTypeID); sold != 1 {
		t.Errorf("quantity_sold = %d, want 1 - the refused checkout took inventory", sold)
	}
}
