package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// eventIDOf reads the event a ticket belongs to.
func (c *client) eventIDOf(ticketID string) uuid.UUID {
	c.t.Helper()
	var eventID uuid.UUID
	if err := c.pool.QueryRow(c.t.Context(),
		`SELECT event_id FROM tickets WHERE id = $1`, uuid.MustParse(ticketID)).
		Scan(&eventID); err != nil {
		c.t.Fatalf("read event id: %v", err)
	}
	return eventID
}

// rosterEntries pulls the tickets array out of a roster response.
func rosterEntries(res response) []any {
	roster, _ := res.Body["roster"].(map[string]any)
	tickets, _ := roster["tickets"].([]any)
	return tickets
}

// TestOfflineRosterHashesTokens is the core security property of SRS 4.8: the
// downloadable roster carries the SHA-256 of each admission token, never the
// token itself, so a lost scanner cannot be turned into a ticket forge.
func TestOfflineRosterHashesTokens(t *testing.T) {
	c := newClient(t)
	organizer := c.register("offlineroster")

	ticketID, qrToken := c.buyOneTicket(organizer.Token, "Offline Roster Show")
	eventID := c.eventIDOf(ticketID)

	res := c.get("/api/v1/events/"+eventID.String()+"/roster", organizer.Token)
	requireStatus(t, res, http.StatusOK)

	if cc := res.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}

	entries := rosterEntries(res)
	if len(entries) != 1 {
		t.Fatalf("%d roster entries, want 1", len(entries))
	}
	entry := entries[0].(map[string]any)

	sum := sha256.Sum256([]byte(qrToken))
	want := hex.EncodeToString(sum[:])
	if got := entry["token_hash"]; got != want {
		t.Errorf("token_hash = %v, want %v (sha256 of the token)", got, want)
	}

	// The plaintext token must appear nowhere in the payload.
	if strings.Contains(res.Raw, qrToken) {
		t.Error("the roster response contains the plaintext QR token")
	}

	if entry["status"] != "valid" {
		t.Errorf("status = %v, want valid", entry["status"])
	}
	if entry["ticket_id"] != ticketID {
		t.Errorf("ticket_id = %v, want %v", entry["ticket_id"], ticketID)
	}
}

// TestOfflineSyncRecordsAndDeduplicates walks the reconciliation path: a queued
// admission is recorded, and re-syncing the same admission is reported as
// already checked in rather than double-counted.
func TestOfflineSyncRecordsAndDeduplicates(t *testing.T) {
	c := newClient(t)
	organizer := c.register("offlinesync")

	ticketID, _ := c.buyOneTicket(organizer.Token, "Offline Sync Show")
	eventID := c.eventIDOf(ticketID)

	scannedAt := time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339)
	body := map[string]any{
		"check_ins": []map[string]any{
			{"ticket_id": ticketID, "scanned_at": scannedAt, "device_label": "Gate-Offline"},
		},
	}

	res := c.post("/api/v1/events/"+eventID.String()+"/check-in/sync", organizer.Token, body)
	requireStatus(t, res, http.StatusOK)

	if n, _ := res.Body["recorded"].(float64); n != 1 {
		t.Errorf("recorded = %v, want 1", res.Body["recorded"])
	}
	if c.ticketStatus(ticketID) != "checked_in" {
		t.Errorf("ticket status = %q, want checked_in", c.ticketStatus(ticketID))
	}
	if got := c.activeCheckIns(ticketID); got != 1 {
		t.Fatalf("active check-ins = %d, want 1", got)
	}

	// Re-syncing the very same admission is idempotent.
	again := c.post("/api/v1/events/"+eventID.String()+"/check-in/sync", organizer.Token, body)
	requireStatus(t, again, http.StatusOK)
	if n, _ := again.Body["already_checked_in"].(float64); n != 1 {
		t.Errorf("already_checked_in = %v, want 1 on re-sync", again.Body["already_checked_in"])
	}
	if got := c.activeCheckIns(ticketID); got != 1 {
		t.Errorf("active check-ins after re-sync = %d, want still 1", got)
	}
}

// TestOfflineSyncRejectsVoidedTicket proves an admission the device made while
// offline is reported as invalid when the ticket was refunded in the meantime,
// so the organizer learns somebody was let in on a dead ticket.
func TestOfflineSyncRejectsVoidedTicket(t *testing.T) {
	c := newClient(t)
	organizer := c.register("offlinevoid")

	ticketID, _ := c.buyOneTicket(organizer.Token, "Offline Void Show")
	eventID := c.eventIDOf(ticketID)

	// Void it the way a refund does, after the device is imagined to have gone
	// offline with the ticket still valid.
	if _, err := c.pool.Exec(t.Context(),
		`UPDATE tickets SET status = 'refunded' WHERE id = $1`, uuid.MustParse(ticketID)); err != nil {
		t.Fatalf("void ticket: %v", err)
	}

	body := map[string]any{
		"check_ins": []map[string]any{{"ticket_id": ticketID}},
	}
	res := c.post("/api/v1/events/"+eventID.String()+"/check-in/sync", organizer.Token, body)
	requireStatus(t, res, http.StatusOK)

	results, _ := res.Body["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("%d results, want 1", len(results))
	}
	if outcome := results[0].(map[string]any)["outcome"]; outcome != "not_valid" {
		t.Errorf("outcome = %v, want not_valid", outcome)
	}
	if n, _ := res.Body["rejected"].(float64); n != 1 {
		t.Errorf("rejected = %v, want 1", res.Body["rejected"])
	}
	if got := c.activeCheckIns(ticketID); got != 0 {
		t.Errorf("a voided ticket was admitted anyway: %d check-ins", got)
	}
}

// TestOfflineSyncUnknownTicket reports a ticket id that is not this event's (or
// not a ticket at all) without failing the batch.
func TestOfflineSyncUnknownTicket(t *testing.T) {
	c := newClient(t)
	organizer := c.register("offlineunknown")

	ticketID, _ := c.buyOneTicket(organizer.Token, "Offline Unknown Show")
	eventID := c.eventIDOf(ticketID)

	body := map[string]any{
		"check_ins": []map[string]any{{"ticket_id": uuid.NewString()}},
	}
	res := c.post("/api/v1/events/"+eventID.String()+"/check-in/sync", organizer.Token, body)
	requireStatus(t, res, http.StatusOK)

	results, _ := res.Body["results"].([]any)
	if len(results) != 1 || results[0].(map[string]any)["outcome"] != "unknown_ticket" {
		t.Fatalf("results = %v, want one unknown_ticket", res.Body["results"])
	}
}

// TestOfflineRosterRequiresScanRights denies the roster to somebody with no
// claim to the door - it is the guest list plus the means to validate it.
func TestOfflineRosterRequiresScanRights(t *testing.T) {
	c := newClient(t)
	owner := c.register("offlineowner")
	stranger := c.register("offlinestranger")

	ticketID, _ := c.buyOneTicket(owner.Token, "Offline Private Show")
	eventID := c.eventIDOf(ticketID)

	res := c.get("/api/v1/events/"+eventID.String()+"/roster", stranger.Token)
	if res.Status == http.StatusOK {
		t.Fatalf("a stranger downloaded the roster: status %d", res.Status)
	}
}
