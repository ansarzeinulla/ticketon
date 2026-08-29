package api

import (
	"net/http"
	"testing"
)

// SRS 4.1: "Organizers shall have a profile containing contact and payout
// information", and "Users shall be able to sign in, sign out, and reset
// passwords".
//
// The organizer_profiles table existed from Phase 1 but was only ever written
// as a side effect of the activation checklist. There was no way to read a
// profile back, no way to edit it, and no way to change a password while
// signed in.

func profileOf(t *testing.T, res response) map[string]any {
	t.Helper()
	p, ok := res.Body["profile"].(map[string]any)
	if !ok {
		t.Fatalf("no profile in response: %s", res.Raw)
	}
	return p
}

// TestProfileStartsEmptyRatherThanMissing - the dashboard wants a form to
// render, and "no profile yet" and "a blank profile" are the same thing to the
// person filling it in.
func TestProfileStartsEmptyRatherThanMissing(t *testing.T) {
	c := newClient(t)
	organizer := c.register("profilenew")

	res := c.get("/api/v1/auth/profile", organizer.Token)
	requireStatus(t, res, http.StatusOK)

	p := profileOf(t, res)
	if p["user_id"] != organizer.ID.String() {
		t.Errorf("user_id = %v, want %s", p["user_id"], organizer.ID)
	}
	if accounts, ok := p["payout_accounts"].([]any); !ok || len(accounts) != 0 {
		t.Errorf("payout_accounts = %v, want an empty list", p["payout_accounts"])
	}
}

// TestOrganizerCanFillInAndEditTheirProfile is the requirement itself.
func TestOrganizerCanFillInAndEditTheirProfile(t *testing.T) {
	c := newClient(t)
	organizer := c.register("profileedit")

	res := c.patch("/api/v1/auth/profile", organizer.Token, map[string]any{
		"display_name":  "Dana Events",
		"legal_name":    "Dana Events LLP",
		"contact_email": "hello@danaevents.kz",
		"contact_phone": "+7 700 000 0000",
		"description":   "Independent promoter in Almaty.",
		"website_url":   "https://danaevents.kz",
	})
	requireStatus(t, res, http.StatusOK)

	p := profileOf(t, res)
	if p["display_name"] != "Dana Events" {
		t.Errorf("display_name = %v", p["display_name"])
	}
	if p["contact_email"] != "hello@danaevents.kz" {
		t.Errorf("contact_email = %v", p["contact_email"])
	}

	// It reads back on a fresh request, not just in the write's response.
	p = profileOf(t, c.get("/api/v1/auth/profile", organizer.Token))
	if p["website_url"] != "https://danaevents.kz" {
		t.Errorf("website_url = %v after re-reading", p["website_url"])
	}

	// A PATCH leaves absent fields alone.
	res = c.patch("/api/v1/auth/profile", organizer.Token,
		map[string]any{"description": "Now booking for 2027."})
	p = profileOf(t, res)
	if p["display_name"] != "Dana Events" {
		t.Errorf("display_name = %v after a partial patch, want it untouched", p["display_name"])
	}
	if p["description"] != "Now booking for 2027." {
		t.Errorf("description = %v", p["description"])
	}

	// An explicit null clears a field - the tri-state the event PATCH uses.
	res = c.patch("/api/v1/auth/profile", organizer.Token,
		map[string]any{"legal_name": nil})
	if _, present := profileOf(t, res)["legal_name"]; present {
		t.Errorf("legal_name survived an explicit null: %s", res.Raw)
	}
}

// TestProfileValidation catches what would otherwise be a constraint violation
// or a silently stored non-address.
func TestProfileValidation(t *testing.T) {
	c := newClient(t)
	organizer := c.register("profilevalid")

	cases := []struct {
		name  string
		body  map[string]any
		field string
	}{
		{"blank display name", map[string]any{"display_name": "   "}, "display_name"},
		{"not an email", map[string]any{"contact_email": "not-an-address"}, "contact_email"},
		{"bare domain website", map[string]any{"website_url": "danaevents.kz"}, "website_url"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := c.patch("/api/v1/auth/profile", organizer.Token, tc.body)
			requireStatus(t, res, http.StatusUnprocessableEntity)
			if _, ok := res.errorFields()[tc.field]; !ok {
				t.Errorf("no field error on %s: %s", tc.field, res.Raw)
			}
		})
	}
}

// TestProfileIsPrivateToItsOwner - one organizer cannot read or write another's.
func TestProfileIsPrivateToItsOwner(t *testing.T) {
	c := newClient(t)
	owner := c.register("profileowner")
	stranger := c.register("profilestranger")

	requireStatus(t, c.patch("/api/v1/auth/profile", owner.Token,
		map[string]any{"display_name": "The Owner"}), http.StatusOK)

	// The endpoint is addressed by the token, never by a user id in the path,
	// so a stranger simply gets their own - empty - profile back.
	p := profileOf(t, c.get("/api/v1/auth/profile", stranger.Token))
	if p["display_name"] == "The Owner" {
		t.Error("a stranger read the owner's profile")
	}

	requireStatus(t, c.get("/api/v1/auth/profile", ""), http.StatusUnauthorized)
	requireStatus(t, c.patch("/api/v1/auth/profile", "", map[string]any{}),
		http.StatusUnauthorized)
}

// TestPayoutDestinationIsVisibleAndMasked - SRS 4.1 asks the profile to carry
// payout information; NFR section 7 forbids storing card data, so what comes
// back is a masked display value and nothing else.
func TestPayoutDestinationIsVisibleAndMasked(t *testing.T) {
	c := newClient(t)
	organizer := c.register("profilepayout")

	// Completing the activation checklist is what registers a payout account.
	eventID, _ := c.createEvent(organizer.Token, "Payout Visible")
	c.createTicketType(organizer.Token, eventID, ticketTypeBody("Standard", "5000", 5))
	c.activatePaidSales(organizer.Token, eventID)

	p := profileOf(t, c.get("/api/v1/auth/profile", organizer.Token))
	accounts, _ := p["payout_accounts"].([]any)
	if len(accounts) == 0 {
		t.Fatalf("no payout account after activation: %v", p)
	}

	account := accounts[0].(map[string]any)
	if account["is_simulated"] != true {
		t.Errorf("is_simulated = %v, want true - the MVP moves no real money",
			account["is_simulated"])
	}
	if account["currency"] != "KZT" {
		t.Errorf("currency = %v, want KZT", account["currency"])
	}
	// Nothing resembling a full account number may appear.
	for _, forbidden := range []string{"provider_account_ref", "account_number", "iban"} {
		if _, present := account[forbidden]; present {
			t.Errorf("payout account exposes %q", forbidden)
		}
	}
}

// --- Password change ---------------------------------------------------------

// TestUserCanChangeTheirPassword is the requirement itself.
func TestUserCanChangeTheirPassword(t *testing.T) {
	c := newClient(t)
	user := c.register("passwordchange")

	requireStatus(t, c.post("/api/v1/auth/password", user.Token, map[string]any{
		"current_password": user.Password,
		"new_password":     "a brand new passphrase",
	}), http.StatusNoContent)

	// The old password stops working and the new one starts.
	requireStatus(t, c.post("/api/v1/auth/login", "", map[string]any{
		"email": user.Email, "password": user.Password,
	}), http.StatusUnauthorized)
	requireStatus(t, c.post("/api/v1/auth/login", "", map[string]any{
		"email": user.Email, "password": "a brand new passphrase",
	}), http.StatusOK)
}

// TestPasswordChangeNeedsTheCurrentPassword - a token left behind on a shared
// machine must not be enough to lock its owner out of their own account.
func TestPasswordChangeNeedsTheCurrentPassword(t *testing.T) {
	c := newClient(t)
	user := c.register("passwordproof")

	requireErrorCode(t, c.post("/api/v1/auth/password", user.Token, map[string]any{
		"current_password": "not the right one",
		"new_password":     "a brand new passphrase",
	}), http.StatusForbidden, CodeWrongPassword)

	// The password is unchanged.
	requireStatus(t, c.post("/api/v1/auth/login", "", map[string]any{
		"email": user.Email, "password": user.Password,
	}), http.StatusOK)

	requireStatus(t, c.post("/api/v1/auth/password", "", map[string]any{
		"current_password": user.Password, "new_password": "another passphrase",
	}), http.StatusUnauthorized)
}

// TestPasswordChangeValidatesTheNewPassword holds a change to the same
// standard as registration.
func TestPasswordChangeValidatesTheNewPassword(t *testing.T) {
	c := newClient(t)
	user := c.register("passwordweak")

	res := c.post("/api/v1/auth/password", user.Token, map[string]any{
		"current_password": user.Password,
		"new_password":     "short",
	})
	requireStatus(t, res, http.StatusUnprocessableEntity)
	if _, ok := res.errorFields()["new_password"]; !ok {
		t.Errorf("no field error on new_password: %s", res.Raw)
	}

	res = c.post("/api/v1/auth/password", user.Token, map[string]any{
		"new_password": "a perfectly fine passphrase",
	})
	requireStatus(t, res, http.StatusUnprocessableEntity)
	if _, ok := res.errorFields()["current_password"]; !ok {
		t.Errorf("no field error on current_password: %s", res.Raw)
	}
}

// TestPasswordChangeStoresOnlyAHash - NFR section 7.
func TestPasswordChangeStoresOnlyAHash(t *testing.T) {
	c := newClient(t)
	user := c.register("passwordhash")

	requireStatus(t, c.post("/api/v1/auth/password", user.Token, map[string]any{
		"current_password": user.Password,
		"new_password":     "a brand new passphrase",
	}), http.StatusNoContent)

	var hash string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT password_hash FROM users WHERE id = $1`, user.ID).Scan(&hash); err != nil {
		t.Fatalf("read hash: %v", err)
	}
	if hash == "a brand new passphrase" {
		t.Fatal("the password is stored in plaintext")
	}
	if len(hash) < 20 {
		t.Errorf("stored hash looks too short to be bcrypt: %q", hash)
	}
}

// TestChangingAPasswordClosesOutstandingResetLinks - a reset link already in
// flight must stop working, or it becomes a way back in for whoever requested
// it.
func TestChangingAPasswordClosesOutstandingResetLinks(t *testing.T) {
	c := newClient(t)
	user := c.register("passwordreset")

	// Ask for a reset, so there is a live token.
	requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": user.Email}), http.StatusAccepted)

	var live int
	if err := c.pool.QueryRow(t.Context(), `
		SELECT count(*) FROM user_tokens
		 WHERE user_id = $1 AND purpose = 'password_reset' AND consumed_at IS NULL`,
		user.ID).Scan(&live); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if live != 1 {
		t.Fatalf("live reset tokens = %d before the change, want 1", live)
	}

	requireStatus(t, c.post("/api/v1/auth/password", user.Token, map[string]any{
		"current_password": user.Password,
		"new_password":     "a brand new passphrase",
	}), http.StatusNoContent)

	if err := c.pool.QueryRow(t.Context(), `
		SELECT count(*) FROM user_tokens
		 WHERE user_id = $1 AND purpose = 'password_reset' AND consumed_at IS NULL`,
		user.ID).Scan(&live); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if live != 0 {
		t.Errorf("live reset tokens = %d after the change, want 0", live)
	}
}
