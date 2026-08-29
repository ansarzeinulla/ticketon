package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestPhase2SuccessCriteria walks the three Phase 2 acceptance criteria in
// order, in one test, using nothing but HTTP calls and a direct SQL check -
// the same path a person would take with Postman or cURL.
func TestPhase2SuccessCriteria(t *testing.T) {
	c := newClient(t)

	const (
		email    = "phase2@biletflow.test"
		password = "correct horse battery"
	)

	// --- 1. POST email + password registers an account ----------------------
	registered := c.post("/api/v1/auth/register", "", map[string]any{
		"email":    email,
		"password": password,
	})
	requireStatus(t, registered, http.StatusCreated)

	userID, err := uuid.Parse(registered.Body["user"].(map[string]any)["id"].(string))
	if err != nil {
		t.Fatalf("criterion 1: registration returned a non-uuid id: %v", err)
	}

	var dbEmail string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT email FROM users WHERE id = $1`, userID).Scan(&dbEmail); err != nil {
		t.Fatalf("criterion 1: the account is not in PostgreSQL: %v", err)
	}
	t.Logf("criterion 1 OK: registered %s as %s", dbEmail, userID)

	// --- 2. POST to login returns a valid token -----------------------------
	loggedIn := c.post("/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	})
	requireStatus(t, loggedIn, http.StatusOK)

	token, _ := loggedIn.Body["access_token"].(string)
	if token == "" {
		t.Fatalf("criterion 2: login returned no token: %s", loggedIn.Raw)
	}

	// "Valid" is proven by using it, not by inspecting it.
	me := c.get("/api/v1/auth/me", token)
	requireStatus(t, me, http.StatusOK)
	if me.Body["user"].(map[string]any)["id"] != userID.String() {
		t.Fatalf("criterion 2: the token authenticated the wrong user: %s", me.Raw)
	}
	t.Logf("criterion 2 OK: token authenticates as %s", userID)

	// --- 3. POST /events with that token returns 201 and lands in the DB ----
	start := time.Now().Add(45 * 24 * time.Hour).UTC().Truncate(time.Second)
	created := c.post("/api/v1/events", token, map[string]any{
		"title":         "Phase 2 Verification Concert",
		"description":   "Created by the Phase 2 acceptance test.",
		"category":      "music",
		"venue_name":    "Almaty Demo Hall",
		"venue_address": "Abay Avenue 44, Almaty",
		"starts_at":     start.Format(time.RFC3339),
		"ends_at":       start.Add(3 * time.Hour).Format(time.RFC3339),
		"timezone":      "Asia/Almaty",
		"capacity":      300,
	})
	requireStatus(t, created, http.StatusCreated)

	eventID, err := uuid.Parse(created.eventString("id"))
	if err != nil {
		t.Fatalf("criterion 3: the created event has a non-uuid id: %v", err)
	}

	var (
		title       string
		status      string
		organizerID uuid.UUID
		capacity    int
	)
	err = c.pool.QueryRow(t.Context(), `
		SELECT title, status::text, organizer_id, capacity
		  FROM events WHERE id = $1`, eventID).
		Scan(&title, &status, &organizerID, &capacity)
	if err != nil {
		t.Fatalf("criterion 3: the event is not in PostgreSQL: %v", err)
	}

	if title != "Phase 2 Verification Concert" {
		t.Errorf("criterion 3: db title = %q, want the submitted title", title)
	}
	if organizerID != userID {
		t.Errorf("criterion 3: db organizer_id = %v, want the registered user %v", organizerID, userID)
	}
	if capacity != 300 {
		t.Errorf("criterion 3: db capacity = %d, want 300", capacity)
	}
	t.Logf("criterion 3 OK: event %s (%q, status=%s) is in the events table", eventID, title, status)
}
