package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/biletflow/api/internal/config"
)

// --- registration ------------------------------------------------------------

// TestRegisterWithEmailAndPassword is Phase 2 success criterion 1: a POST with
// an email and a password registers an account.
func TestRegisterWithEmailAndPassword(t *testing.T) {
	c := newClient(t)

	res := c.post("/api/v1/auth/register", "", map[string]any{
		"email":    "criterion.one@biletflow.test",
		"password": "correct horse battery",
	})
	requireStatus(t, res, http.StatusCreated)

	user, ok := res.Body["user"].(map[string]any)
	if !ok {
		t.Fatalf("response has no user object: %s", res.Raw)
	}
	if user["email"] != "criterion.one@biletflow.test" {
		t.Errorf("user.email = %v, want criterion.one@biletflow.test", user["email"])
	}
	if user["status"] != "pending_verification" {
		t.Errorf("user.status = %v, want pending_verification", user["status"])
	}
	if token, _ := res.Body["access_token"].(string); token == "" {
		t.Error("registration did not return an access token")
	}
	if res.Body["token_type"] != "Bearer" {
		t.Errorf("token_type = %v, want Bearer", res.Body["token_type"])
	}

	// The account must exist in PostgreSQL, not just in the response.
	var (
		id       uuid.UUID
		email    string
		fullName string
		status   string
	)
	err := c.pool.QueryRow(t.Context(),
		`SELECT id, email, full_name, status::text FROM users WHERE email = $1`,
		"criterion.one@biletflow.test").Scan(&id, &email, &fullName, &status)
	if err != nil {
		t.Fatalf("the registered user is not in the database: %v", err)
	}
	if fullName != "Criterion One" {
		t.Errorf("full_name = %q, want %q derived from the email local part", fullName, "Criterion One")
	}

	// A new account starts as an attendee and nothing more.
	var roles []string
	rows, err := c.pool.Query(t.Context(), `SELECT role::text FROM user_roles WHERE user_id = $1`, id)
	if err != nil {
		t.Fatalf("read roles: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			t.Fatalf("scan role: %v", err)
		}
		roles = append(roles, role)
	}
	if len(roles) != 1 || roles[0] != "attendee" {
		t.Errorf("roles = %v, want exactly [attendee]", roles)
	}
}

func TestRegisterStoresOnlyAHash(t *testing.T) {
	c := newClient(t)
	acc := c.register("hashcheck")

	var hash string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT password_hash FROM users WHERE id = $1`, acc.ID).Scan(&hash); err != nil {
		t.Fatalf("read password hash: %v", err)
	}

	if hash == acc.Password {
		t.Fatal("the password was stored in plaintext")
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("password_hash = %q, want a bcrypt digest", hash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(acc.Password)); err != nil {
		t.Errorf("the stored hash does not verify against the original password: %v", err)
	}

	// The hash must never appear in an API response.
	res := c.get("/api/v1/auth/me", acc.Token)
	requireStatus(t, res, http.StatusOK)
	if strings.Contains(res.Raw, "password") {
		t.Errorf("a response mentions a password field: %s", res.Raw)
	}
}

func TestRegisterAcceptsOptionalProfileFields(t *testing.T) {
	c := newClient(t)

	res := c.post("/api/v1/auth/register", "", map[string]any{
		"email":     "full.profile@biletflow.test",
		"password":  "correct horse battery",
		"full_name": "Dana Amirova",
		"phone":     "+7 701 111 11 11",
		"locale":    "ru",
	})
	requireStatus(t, res, http.StatusCreated)

	user := res.Body["user"].(map[string]any)
	if user["full_name"] != "Dana Amirova" {
		t.Errorf("full_name = %v, want Dana Amirova", user["full_name"])
	}
	if user["locale"] != "ru" {
		t.Errorf("locale = %v, want ru", user["locale"])
	}
	if user["phone"] != "+7 701 111 11 11" {
		t.Errorf("phone = %v, want the submitted number", user["phone"])
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	c := newClient(t)
	acc := c.register("duplicate")

	res := c.post("/api/v1/auth/register", "", map[string]any{
		"email":    acc.Email,
		"password": "another password",
	})
	requireErrorCode(t, res, http.StatusConflict, "conflict")
}

// Emails are citext in the schema, so case must not create a second account.
func TestRegisterRejectsDuplicateEmailInDifferentCase(t *testing.T) {
	c := newClient(t)
	acc := c.register("casing")

	res := c.post("/api/v1/auth/register", "", map[string]any{
		"email":    strings.ToUpper(acc.Email),
		"password": "another password",
	})
	requireErrorCode(t, res, http.StatusConflict, "conflict")

	var count int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM users WHERE email = $1`, acc.Email).Scan(&count); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Errorf("found %d accounts for %s, want 1", count, acc.Email)
	}
}

func TestRegisterValidation(t *testing.T) {
	c := newClient(t)

	tests := []struct {
		name      string
		body      map[string]any
		wantField string
	}{
		{"missing email", map[string]any{"password": "correct horse battery"}, "email"},
		{"blank email", map[string]any{"email": "  ", "password": "correct horse battery"}, "email"},
		{"malformed email", map[string]any{"email": "not-an-email", "password": "correct horse battery"}, "email"},
		{"email without dot", map[string]any{"email": "user@localhost", "password": "correct horse battery"}, "email"},
		{"missing password", map[string]any{"email": "a@b.kz"}, "password"},
		{"short password", map[string]any{"email": "a@b.kz", "password": "short"}, "password"},
		{"overlong password", map[string]any{"email": "a@b.kz", "password": strings.Repeat("x", 73)}, "password"},
		{"blank full name", map[string]any{"email": "a@b.kz", "password": "correct horse battery", "full_name": "   "}, "full_name"},
		{"unknown locale", map[string]any{"email": "a@b.kz", "password": "correct horse battery", "locale": "fr"}, "locale"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.post("/api/v1/auth/register", "", tt.body)
			requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")

			if _, ok := res.errorFields()[tt.wantField]; !ok {
				t.Errorf("error fields = %v, want an entry for %q", res.errorFields(), tt.wantField)
			}
		})
	}
}

func TestRegisterRejectsMalformedBodies(t *testing.T) {
	c := newClient(t)

	tests := []struct {
		name string
		body any
	}{
		{"not json", "this is not json"},
		{"empty body", ""},
		{"array instead of object", "[]"},
		{"unknown field", `{"email":"a@b.kz","password":"correct horse battery","admin":true}`},
		{"wrong field type", `{"email":123,"password":"correct horse battery"}`},
		{"two objects", `{"email":"a@b.kz","password":"correct horse battery"}{"email":"c@d.kz"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.post("/api/v1/auth/register", "", tt.body)
			requireErrorCode(t, res, http.StatusBadRequest, "invalid_json")
		})
	}
}

// --- login -------------------------------------------------------------------

// TestLoginReturnsUsableToken is Phase 2 success criterion 2.
func TestLoginReturnsUsableToken(t *testing.T) {
	c := newClient(t)
	acc := c.register("login")

	res := c.post("/api/v1/auth/login", "", map[string]any{
		"email":    acc.Email,
		"password": acc.Password,
	})
	requireStatus(t, res, http.StatusOK)

	token, _ := res.Body["access_token"].(string)
	if token == "" {
		t.Fatalf("login returned no token: %s", res.Raw)
	}
	if strings.Count(token, ".") != 2 {
		t.Fatalf("access_token = %q, want a three-part JWT", token)
	}

	// "Valid" means the token actually authorises a protected request.
	me := c.get("/api/v1/auth/me", token)
	requireStatus(t, me, http.StatusOK)

	user := me.Body["user"].(map[string]any)
	if user["id"] != acc.ID.String() {
		t.Errorf("the token authenticated %v, want %v", user["id"], acc.ID)
	}

	// The claims must survive a round trip through the server's own parser.
	claims, err := c.api.Tokens().Parse(token)
	if err != nil {
		t.Fatalf("the server cannot parse the token it issued: %v", err)
	}
	if got, _ := claims.UserID(); got != acc.ID {
		t.Errorf("token subject = %v, want %v", got, acc.ID)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		t.Error("the issued token is already expired")
	}
}

func TestLoginIsCaseInsensitive(t *testing.T) {
	c := newClient(t)
	acc := c.register("caseinsensitive")

	res := c.post("/api/v1/auth/login", "", map[string]any{
		"email":    strings.ToUpper(acc.Email),
		"password": acc.Password,
	})
	requireStatus(t, res, http.StatusOK)
}

func TestLoginRecordsLastLogin(t *testing.T) {
	c := newClient(t)
	acc := c.register("lastlogin")

	var before *time.Time
	if err := c.pool.QueryRow(t.Context(),
		`SELECT last_login_at FROM users WHERE id = $1`, acc.ID).Scan(&before); err != nil {
		t.Fatalf("read last_login_at: %v", err)
	}
	if before != nil {
		t.Fatalf("last_login_at = %v before any login, want NULL", before)
	}

	requireStatus(t, c.post("/api/v1/auth/login", "", map[string]any{
		"email": acc.Email, "password": acc.Password,
	}), http.StatusOK)

	var after *time.Time
	if err := c.pool.QueryRow(t.Context(),
		`SELECT last_login_at FROM users WHERE id = $1`, acc.ID).Scan(&after); err != nil {
		t.Fatalf("read last_login_at: %v", err)
	}
	if after == nil {
		t.Error("last_login_at is still NULL after a successful login")
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	c := newClient(t)
	acc := c.register("badcreds")

	tests := []struct {
		name  string
		email string
		pass  string
	}{
		{"wrong password", acc.Email, "not the password"},
		{"unknown email", "nobody" + acc.Email, acc.Password},
		{"password of another account", acc.Email, "correct horse batteryy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.post("/api/v1/auth/login", "", map[string]any{
				"email": tt.email, "password": tt.pass,
			})
			requireErrorCode(t, res, http.StatusUnauthorized, "invalid_credentials")

			// The message must not reveal whether the address is registered.
			if strings.Contains(strings.ToLower(res.Raw), "no such user") ||
				strings.Contains(strings.ToLower(res.Raw), "not found") {
				t.Errorf("the error leaks whether the account exists: %s", res.Raw)
			}
		})
	}
}

func TestLoginValidatesRequiredFields(t *testing.T) {
	c := newClient(t)

	for _, body := range []map[string]any{
		{},
		{"email": "a@b.kz"},
		{"password": "correct horse battery"},
	} {
		res := c.post("/api/v1/auth/login", "", body)
		requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")
	}
}

func TestLoginRejectsSuspendedAccount(t *testing.T) {
	c := newClient(t)
	acc := c.register("suspended")

	if _, err := c.pool.Exec(t.Context(),
		`UPDATE users SET status = 'suspended' WHERE id = $1`, acc.ID); err != nil {
		t.Fatalf("suspend user: %v", err)
	}

	res := c.post("/api/v1/auth/login", "", map[string]any{
		"email": acc.Email, "password": acc.Password,
	})
	requireErrorCode(t, res, http.StatusForbidden, "forbidden")
}

// --- protected routes --------------------------------------------------------

func TestMeRequiresValidToken(t *testing.T) {
	c := newClient(t)
	acc := c.register("metoken")

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
		{"wrong scheme", "Basic " + acc.Token, http.StatusUnauthorized},
		{"garbage token", "Bearer not.a.token", http.StatusUnauthorized},
		{"tampered token", "Bearer " + acc.Token + "x", http.StatusUnauthorized},
		{"missing scheme", acc.Token, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, c.server.URL+"/api/v1/auth/me", nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			res, err := c.server.Client().Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	c := newClient(t)
	acc := c.register("expiry")

	// Mint a token that expired an hour ago using the server's own signer.
	tokens := c.api.Tokens()
	tokens.SetClock(func() time.Time { return time.Now().Add(-2 * time.Hour) })
	expired, _, err := tokens.Issue(acc.ID, acc.Email, []string{"attendee"})
	if err != nil {
		t.Fatalf("issue expired token: %v", err)
	}
	tokens.SetClock(time.Now)

	res := c.get("/api/v1/auth/me", expired)
	requireErrorCode(t, res, http.StatusUnauthorized, "unauthorized")
	if !strings.Contains(strings.ToLower(res.Raw), "expired") {
		t.Errorf("the message should say the token expired: %s", res.Raw)
	}
}

// A token signed with a different secret must never be accepted, even if every
// claim inside it looks correct.
func TestForgedTokenIsRejected(t *testing.T) {
	c := newClient(t)
	acc := c.register("forged")

	forger := New(config.Config{
		Env: "test", JWTSecret: "a-different-secret", JWTIssuer: "biletflow-test",
		AccessTokenTTL: time.Hour, BcryptCost: bcrypt.MinCost,
	}, c.pool)

	forged, _, err := forger.Tokens().Issue(acc.ID, acc.Email, []string{"platform_admin"})
	if err != nil {
		t.Fatalf("issue forged token: %v", err)
	}

	requireErrorCode(t, c.get("/api/v1/auth/me", forged), http.StatusUnauthorized, "unauthorized")
}

// A token stays syntactically valid after suspension, so authorisation has to
// re-check the account rather than trusting the claims.
func TestTokenStopsWorkingWhenAccountIsSuspended(t *testing.T) {
	c := newClient(t)
	acc := c.register("revoked")

	requireStatus(t, c.get("/api/v1/auth/me", acc.Token), http.StatusOK)

	if _, err := c.pool.Exec(t.Context(),
		`UPDATE users SET status = 'suspended' WHERE id = $1`, acc.ID); err != nil {
		t.Fatalf("suspend user: %v", err)
	}

	requireErrorCode(t, c.get("/api/v1/auth/me", acc.Token), http.StatusForbidden, "forbidden")
}

func TestTokenForDeletedAccountIsRejected(t *testing.T) {
	c := newClient(t)
	acc := c.register("deleted")

	if _, err := c.pool.Exec(t.Context(), `DELETE FROM users WHERE id = $1`, acc.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	requireErrorCode(t, c.get("/api/v1/auth/me", acc.Token), http.StatusUnauthorized, "unauthorized")
}
