package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/biletflow/api/internal/email"
	"github.com/biletflow/api/internal/store"
)

// resetTokenFor reads the plaintext token out of the recorded email, which is
// the only place it exists outside the user's inbox.
func (c *client) resetTokenFor(address string) string {
	c.t.Helper()
	c.waitForMail()

	for _, msg := range c.mail.To(address) {
		if msg.Type == email.TypePasswordReset {
			return tokenFromConsole(c.t, msg.Body, "/reset-password?token=")
		}
	}
	return ""
}

// TestPasswordResetDoesNotLeakWhoHasAnAccount is the reason the endpoint
// answers 202 either way.
func TestPasswordResetDoesNotLeakWhoHasAnAccount(t *testing.T) {
	c := newClient(t)
	account := c.register("leakcheck")

	real := c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": account.Email})
	unknown := c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": "nobody.here@biletflow.test"})

	if real.Status != unknown.Status {
		t.Errorf("known address answered %d, unknown answered %d - the difference "+
			"tells an attacker which addresses are registered",
			real.Status, unknown.Status)
	}
	if real.Raw != unknown.Raw {
		t.Errorf("the two responses differ:\n known:   %s\n unknown: %s", real.Raw, unknown.Raw)
	}

	// Only the real one actually produced a token.
	c.waitForMail()
	if got := len(c.mail.To("nobody.here@biletflow.test")); got != 0 {
		t.Errorf("emails to an unregistered address = %d, want 0", got)
	}
	if got := len(c.mail.To(account.Email)); got != 1 {
		t.Errorf("emails to the registered address = %d, want 1", got)
	}
}

// TestPasswordResetSupersedesEarlierTokens: clicking "forgot password" three
// times must not leave three working keys in an inbox.
func TestPasswordResetSupersedesEarlierTokens(t *testing.T) {
	c := newClient(t)
	account := c.register("supersede")

	requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": account.Email}), http.StatusAccepted)
	first := c.resetTokenFor(account.Email)

	c.mail.Reset()
	requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": account.Email}), http.StatusAccepted)
	second := c.resetTokenFor(account.Email)

	if first == "" || second == "" || first == second {
		t.Fatalf("expected two different tokens, got %q and %q", first, second)
	}

	// The older one is dead.
	requireErrorCode(t, c.post("/api/v1/auth/password-reset", "",
		map[string]any{"token": first, "password": "the older passphrase"}),
		http.StatusBadRequest, CodeTokenInvalid)

	// The newer one works.
	requireStatus(t, c.post("/api/v1/auth/password-reset", "",
		map[string]any{"token": second, "password": "the newer passphrase"}), http.StatusOK)
}

func TestPasswordResetRejectsExpiredTokens(t *testing.T) {
	c := newClient(t)
	account := c.register("expiredreset")

	requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": account.Email}), http.StatusAccepted)
	token := c.resetTokenFor(account.Email)

	// Age the token past its window rather than waiting an hour for it. Both
	// timestamps move: user_tokens_expiry_chk holds that a token's window was
	// non-empty when it was written, and an honest "issued two hours ago"
	// respects that where a bare expires_at in the past would not.
	if _, err := c.pool.Exec(t.Context(),
		`UPDATE user_tokens
		    SET created_at = now() - interval '2 hours',
		        expires_at = now() - interval '1 hour'
		  WHERE token_hash = $1`, store.HashToken(token)); err != nil {
		t.Fatalf("age the token: %v", err)
	}

	requireErrorCode(t, c.post("/api/v1/auth/password-reset", "",
		map[string]any{"token": token, "password": "too late for this"}),
		http.StatusBadRequest, CodeTokenInvalid)

	// And the old password still works, because nothing changed.
	requireStatus(t, c.post("/api/v1/auth/login", "",
		map[string]any{"email": account.Email, "password": account.Password}), http.StatusOK)
}

func TestPasswordResetValidatesTheNewPassword(t *testing.T) {
	c := newClient(t)
	account := c.register("weakreset")

	requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": account.Email}), http.StatusAccepted)
	token := c.resetTokenFor(account.Email)

	// Too short: refused, and the token is not burned by the attempt.
	requireStatus(t, c.post("/api/v1/auth/password-reset", "",
		map[string]any{"token": token, "password": "short"}), http.StatusUnprocessableEntity)

	requireStatus(t, c.post("/api/v1/auth/password-reset", "",
		map[string]any{"token": token, "password": "a long enough passphrase"}), http.StatusOK)
}

// TestPasswordResetStoresOnlyAHash: a leaked backup must not contain working
// reset links.
func TestPasswordResetStoresOnlyAHash(t *testing.T) {
	c := newClient(t)
	account := c.register("hashonly")

	requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": account.Email}), http.StatusAccepted)
	token := c.resetTokenFor(account.Email)

	var stored string
	if err := c.pool.QueryRow(t.Context(), `
		SELECT token_hash FROM user_tokens WHERE consumed_at IS NULL`).Scan(&stored); err != nil {
		t.Fatalf("read the stored token: %v", err)
	}
	if stored == token {
		t.Fatal("the plaintext token was stored in the database")
	}
	if stored != store.HashToken(token) {
		t.Error("the stored value is not the token's hash")
	}
}

func TestEmailVerificationActivatesTheAccount(t *testing.T) {
	c := newClient(t)
	account := c.register("verifyme")

	// A new account starts unverified.
	var status string
	var verifiedAt *time.Time
	if err := c.pool.QueryRow(t.Context(),
		`SELECT status::text, email_verified_at FROM users WHERE id = $1`,
		account.ID).Scan(&status, &verifiedAt); err != nil {
		t.Fatalf("read the account: %v", err)
	}
	if status != "pending_verification" || verifiedAt != nil {
		t.Fatalf("a new account is %s / verified %v", status, verifiedAt)
	}

	c.mail.Reset()
	requireStatus(t, c.post("/api/v1/auth/verify-email/request", account.Token, nil),
		http.StatusAccepted)
	c.waitForMail()

	sent := c.mail.To(account.Email)
	if len(sent) != 1 || sent[0].Type != email.TypeEmailVerification {
		t.Fatalf("verification emails = %d", len(sent))
	}
	token := tokenFromConsole(t, sent[0].Body, "/verify-email?token=")
	if token == "" {
		t.Fatalf("no token in the email:\n%s", sent[0].Body)
	}

	// Verifying needs no session: the link is followed from an inbox.
	verified := c.post("/api/v1/auth/verify-email", "", map[string]any{"token": token})
	requireStatus(t, verified, http.StatusOK)
	if verified.Body["account_status"] != "active" {
		t.Errorf("account_status = %v, want active", verified.Body["account_status"])
	}

	if err := c.pool.QueryRow(t.Context(),
		`SELECT status::text, email_verified_at FROM users WHERE id = $1`,
		account.ID).Scan(&status, &verifiedAt); err != nil {
		t.Fatalf("re-read the account: %v", err)
	}
	if status != "active" || verifiedAt == nil {
		t.Errorf("after verifying, the account is %s / verified %v", status, verifiedAt)
	}

	// Good once.
	requireErrorCode(t, c.post("/api/v1/auth/verify-email", "", map[string]any{"token": token}),
		http.StatusBadRequest, CodeTokenInvalid)

	// Asking again says there is nothing to do.
	requireStatus(t, c.post("/api/v1/auth/verify-email/request", account.Token, nil),
		http.StatusOK)
}

func TestVerifyEmailRejectsGarbage(t *testing.T) {
	c := newClient(t)

	requireStatus(t, c.post("/api/v1/auth/verify-email", "", map[string]any{"token": ""}),
		http.StatusUnprocessableEntity)
	requireErrorCode(t, c.post("/api/v1/auth/verify-email", "",
		map[string]any{"token": "not-a-real-token"}), http.StatusBadRequest, CodeTokenInvalid)
}

// TestResetPasswordInvalidatesOtherTokens: somebody recovering a compromised
// account should not leave the attacker holding a working verification link.
func TestResetPasswordInvalidatesOtherTokens(t *testing.T) {
	c := newClient(t)
	account := c.register("cleansweep")

	c.mail.Reset()
	requireStatus(t, c.post("/api/v1/auth/verify-email/request", account.Token, nil),
		http.StatusAccepted)
	c.waitForMail()
	verification := tokenFromConsole(t, c.mail.To(account.Email)[0].Body, "/verify-email?token=")

	c.mail.Reset()
	requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": account.Email}), http.StatusAccepted)
	reset := c.resetTokenFor(account.Email)

	requireStatus(t, c.post("/api/v1/auth/password-reset", "",
		map[string]any{"token": reset, "password": "recovered this account"}), http.StatusOK)

	// The verification token issued before the reset is gone too.
	requireErrorCode(t, c.post("/api/v1/auth/verify-email", "",
		map[string]any{"token": verification}), http.StatusBadRequest, CodeTokenInvalid)
}

// TestResetPasswordIssuesNoSession: a stolen reset link must not skip login.
func TestResetPasswordIssuesNoSession(t *testing.T) {
	c := newClient(t)
	account := c.register("nosession")

	requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": account.Email}), http.StatusAccepted)
	token := c.resetTokenFor(account.Email)

	done := c.post("/api/v1/auth/password-reset", "",
		map[string]any{"token": token, "password": "changed it just now"})
	requireStatus(t, done, http.StatusOK)

	if _, issued := done.Body["access_token"]; issued {
		t.Error("resetting a password handed back a session token")
	}
}
