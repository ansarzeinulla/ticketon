package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	c := newClient(t)

	for _, path := range []string{"/health", "/api/v1/health"} {
		res := c.get(path, "")
		requireStatus(t, res, http.StatusOK)

		if res.Body["status"] != "ok" {
			t.Errorf("%s status = %v, want ok", path, res.Body["status"])
		}
		if res.Body["database"] != "up" {
			t.Errorf("%s database = %v, want up", path, res.Body["database"])
		}
	}
}

func TestUnknownRouteReturnsJSON404(t *testing.T) {
	c := newClient(t)

	res := c.get("/api/v1/does-not-exist", "")
	requireErrorCode(t, res, http.StatusNotFound, "not_found")

	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// A wrong method on a real path must be a 405 with an Allow header, not a 404.
func TestWrongMethodReturns405(t *testing.T) {
	c := newClient(t)

	res := c.get("/api/v1/auth/login", "")
	requireErrorCode(t, res, http.StatusMethodNotAllowed, "method_not_allowed")

	if allow := res.Header.Get("Allow"); !strings.Contains(allow, http.MethodPost) {
		t.Errorf("Allow = %q, want it to mention POST", allow)
	}
	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %q, want application/json", res.Header.Get("Content-Type"))
	}
}

func TestEveryResponseCarriesARequestID(t *testing.T) {
	c := newClient(t)

	res := c.get("/health", "")
	if res.Header.Get("X-Request-ID") == "" {
		t.Error("the response has no X-Request-ID header")
	}

	// A client-supplied id is echoed back so logs can be correlated.
	req, err := http.NewRequest(http.MethodGet, c.server.URL+"/health", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("X-Request-ID", "my-correlation-id")

	got, err := c.server.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer got.Body.Close()

	if id := got.Header.Get("X-Request-ID"); id != "my-correlation-id" {
		t.Errorf("X-Request-ID = %q, want the id supplied by the client", id)
	}
}

func TestCORSPreflight(t *testing.T) {
	c := newClient(t)

	req, err := http.NewRequest(http.MethodOptions, c.server.URL+"/api/v1/events", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	res, err := c.server.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", res.StatusCode)
	}
	if res.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Error("the preflight response has no Access-Control-Allow-Origin header")
	}
	if !strings.Contains(res.Header.Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Error("the preflight response does not allow the Authorization header")
	}
}

func TestNonJSONContentTypeIsRejected(t *testing.T) {
	c := newClient(t)

	req, err := http.NewRequest(http.MethodPost, c.server.URL+"/api/v1/auth/register",
		strings.NewReader("email=a@b.kz&password=correct+horse+battery"))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.server.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a non-JSON content type", res.StatusCode)
	}
}

// A charset parameter is legal on the content type and must not be rejected.
func TestJSONContentTypeWithCharsetIsAccepted(t *testing.T) {
	c := newClient(t)

	req, err := http.NewRequest(http.MethodPost, c.server.URL+"/api/v1/auth/register",
		strings.NewReader(`{"email":"charset@biletflow.test","password":"correct horse battery"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	res, err := c.server.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", res.StatusCode)
	}
}

func TestOversizedBodyIsRejected(t *testing.T) {
	c := newClient(t)

	huge := `{"email":"big@biletflow.test","password":"correct horse battery","full_name":"` +
		strings.Repeat("a", 2<<20) + `"}`

	res := c.post("/api/v1/auth/register", "", huge)
	if res.Status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a body over the 1 MiB limit", res.Status)
	}
}
