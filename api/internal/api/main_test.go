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
	"github.com/biletflow/api/internal/testutil"
)

// testConfig mirrors production settings except for the bcrypt cost, which is
// dropped to the minimum so the suite is not dominated by hashing time.
func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		// Uploads land in a directory the test framework removes afterwards,
		// so a test run leaves nothing behind on disk.
		UploadDir:      t.TempDir(),
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

	pool := testutil.Pool(t)
	testutil.Reset(t, pool)

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

// activatePaidSales completes the whole activation checklist in one call
// (SRS 4.5). Anything selling a paid ticket needs this first.
func (c *client) activatePaidSales(token string, eventID uuid.UUID) {
	c.t.Helper()

	res := c.post("/api/v1/events/"+eventID.String()+"/activation", token, map[string]any{
		"confirm_identity":   true,
		"confirm_payout":     true,
		"accept_terms":       true,
		"pay_activation_fee": true,
	})
	if res.Status != http.StatusOK {
		c.t.Fatalf("activate paid sales: status = %d, body = %s", res.Status, res.Raw)
	}
	if active, _ := res.Body["activation"].(map[string]any)["is_active"].(bool); !active {
		c.t.Fatalf("activate paid sales: activation is not active; body = %s", res.Raw)
	}
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

// event returns the "event" object from a response body.
func (r response) event() map[string]any {
	e, _ := r.Body["event"].(map[string]any)
	return e
}

func (r response) eventString(key string) string {
	v, _ := r.event()[key].(string)
	return v
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

// binaryResponse is a response whose body is not JSON - a PDF or a PNG.
type binaryResponse struct {
	Status int
	Header http.Header
	Body   []byte
}

// getBinary fetches a non-JSON resource without trying to parse it.
func (c *client) getBinary(path, token string) binaryResponse {
	c.t.Helper()

	req, err := http.NewRequest(http.MethodGet, c.server.URL+path, nil)
	if err != nil {
		c.t.Fatalf("build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := c.server.Client().Do(req)
	if err != nil {
		c.t.Fatalf("GET %s: %v", path, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatalf("read body: %v", err)
	}

	return binaryResponse{Status: res.StatusCode, Header: res.Header, Body: body}
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

// validEventBody is a complete, valid create-event payload.
func validEventBody(title string) map[string]any {
	start := time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Second)
	return map[string]any{
		"title":         title,
		"description":   "An event created by the integration test suite.",
		"category":      "music",
		"venue_name":    "Almaty Demo Hall",
		"venue_address": "Abay Avenue 44, Almaty",
		"starts_at":     start.Format(time.RFC3339),
		"ends_at":       start.Add(3 * time.Hour).Format(time.RFC3339),
		"timezone":      "Asia/Almaty",
		"capacity":      200,
	}
}

// createEvent posts a valid event and returns its id.
func (c *client) createEvent(token, title string) (uuid.UUID, response) {
	c.t.Helper()

	res := c.post("/api/v1/events", token, validEventBody(title))
	if res.Status != http.StatusCreated {
		c.t.Fatalf("create event %q: status = %d, body = %s", title, res.Status, res.Raw)
	}

	id, err := uuid.Parse(res.eventString("id"))
	if err != nil {
		c.t.Fatalf("create event %q: id %q is not a uuid", title, res.eventString("id"))
	}
	return id, res
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
