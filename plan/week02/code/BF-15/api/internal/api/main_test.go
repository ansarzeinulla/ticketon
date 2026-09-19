package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/biletflow/api/internal/config"
	"github.com/biletflow/api/internal/email"
)

// testConfig mirrors production settings except for the bcrypt cost, which is
// dropped to the minimum so the suite is not dominated by hashing time.

func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		Env:            "test",
		JWTSecret:      "integration-test-secret",
		JWTIssuer:      "biletflow-test",
		AccessTokenTTL: time.Hour,
		BcryptCost:     bcrypt.MinCost,
	}
}

// client drives the real HTTP stack: router, middleware, handlers and database.
type client struct {
	t      *testing.T
	server *httptest.Server
	pool   *pgxpool.Pool
	api    *Server
	// mail captures the notifications the server sends, so a test can assert
	// on them without reading the console.
	mail *email.Recorder
}

// newClient starts a server backed by the test database, emptied beforehand.
func newClient(t *testing.T) *client {
	t.Helper()

	pool := testPool(t)
	resetDB(t, pool)

	recorder := email.NewRecorder()
	srv := NewWithSender(testConfig(t), pool, recorder)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	return &client{t: t, server: ts, pool: pool, api: srv, mail: recorder}
}

// waitForMail blocks until every notification queued so far has been
// dispatched. Delivery is asynchronous by design (SRS 4.10 does not ask the
// buyer to wait on a mail server), so a test that asserts on it has to say
// where the asynchrony ends rather than sleeping and hoping.
func (c *client) waitForMail() {
	c.t.Helper()
	c.api.Mailer().Wait()
}

// response is a decoded HTTP response.
type response struct {
	Status int
	Header http.Header
	Body   map[string]any
	Raw    string
}

// errorCode returns the machine-readable code from an error envelope.
func (r response) errorCode() string {
	errObj, ok := r.Body["error"].(map[string]any)
	if !ok {
		return ""
	}
	code, _ := errObj["code"].(string)
	return code
}

// errorFields returns the per-field validation messages.
func (r response) errorFields() map[string]any {
	errObj, ok := r.Body["error"].(map[string]any)
	if !ok {
		return nil
	}
	fields, _ := errObj["fields"].(map[string]any)
	return fields
}

// do performs a request. An empty token means no Authorization header.
func (c *client) do(method, path, token string, body any) response {
	c.t.Helper()

	var reader io.Reader
	switch v := body.(type) {
	case nil:
		reader = nil
	case string:
		reader = bytes.NewBufferString(v)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			c.t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, c.server.URL+path, reader)
	if err != nil {
		c.t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.send(req)
}

// send performs a prepared request and decodes the JSON envelope. Split out of
// `do` so a multipart upload - which cannot be built from a JSON body - still
// goes through exactly the same client and decoding.
func (c *client) send(req *http.Request) response {
	c.t.Helper()

	res, err := c.server.Client().Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatalf("read response body: %v", err)
	}

	out := response{Status: res.StatusCode, Header: res.Header, Raw: string(raw)}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out.Body); err != nil {
			c.t.Fatalf("%s %s returned non-JSON body %q", req.Method, req.URL.Path, string(raw))
		}
	}
	return out
}

func (c *client) post(path, token string, body any) response {
	return c.do(http.MethodPost, path, token, body)
}

func (c *client) get(path, token string) response {
	return c.do(http.MethodGet, path, token, nil)
}

func (c *client) patch(path, token string, body any) response {
	return c.do(http.MethodPatch, path, token, body)
}

func (c *client) delete(path, token string) response {
	return c.do(http.MethodDelete, path, token, nil)
}

// account is a registered user plus its access token.
type account struct {
	ID       uuid.UUID
	Email    string
	Password string
	Token    string
}

// uniqueEmail keeps registrations from colliding across sub-tests.
var emailCounter int

func nextEmail(prefix string) string {
	emailCounter++
	return fmt.Sprintf("%s%d@biletflow.test", prefix, emailCounter)
}

// register creates an account through the real registration endpoint.
func (c *client) register(prefix string) account {
	c.t.Helper()

	email := nextEmail(prefix)
	const password = "correct horse battery"

	res := c.post("/api/v1/auth/register", "", map[string]any{
		"email":    email,
		"password": password,
	})
	if res.Status != http.StatusCreated {
		c.t.Fatalf("register %s: status = %d, body = %s", email, res.Status, res.Raw)
	}

	token, _ := res.Body["access_token"].(string)
	user, _ := res.Body["user"].(map[string]any)
	idString, _ := user["id"].(string)
	id, err := uuid.Parse(idString)
	if err != nil {
		c.t.Fatalf("register %s: user id %q is not a uuid", email, idString)
	}

	return account{ID: id, Email: email, Password: password, Token: token}
}

// --- assertion helpers -------------------------------------------------------

func requireStatus(t *testing.T, res response, want int) {
	t.Helper()
	if res.Status != want {
		t.Fatalf("status = %d, want %d; body = %s", res.Status, want, res.Raw)
	}
}

func requireErrorCode(t *testing.T, res response, wantStatus int, wantCode string) {
	t.Helper()
	if res.Status != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", res.Status, wantStatus, res.Raw)
	}
	if got := res.errorCode(); got != wantCode {
		t.Fatalf("error code = %q, want %q; body = %s", got, wantCode, res.Raw)
	}
}
