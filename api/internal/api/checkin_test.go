package api

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// scan posts a QR token at an event's gate.
func (c *client) scan(token string, eventID uuid.UUID, qrToken string) response {
	return c.post("/api/v1/events/"+eventID.String()+"/check-in", token, map[string]any{
		"qr_token":     qrToken,
		"device_label": "Scanner-01",
	})
}

// checkIn returns the check_in object from a successful scan.
func checkIn(res response) map[string]any {
	c, _ := res.Body["check_in"].(map[string]any)
	return c
}

// ticketStatus reads a ticket's status straight from PostgreSQL.
func (c *client) ticketStatus(ticketID string) string {
	c.t.Helper()
	var status string
	if err := c.pool.QueryRow(c.t.Context(),
		`SELECT status::text FROM tickets WHERE id = $1`, uuid.MustParse(ticketID)).
		Scan(&status); err != nil {
		c.t.Fatalf("read ticket status: %v", err)
	}
	return status
}

// activeCheckIns counts unreversed check-in records for a ticket.
func (c *client) activeCheckIns(ticketID string) int {
	c.t.Helper()
	var n int
	if err := c.pool.QueryRow(c.t.Context(),
		`SELECT count(*) FROM check_in_records WHERE ticket_id = $1 AND reversed_at IS NULL`,
		uuid.MustParse(ticketID)).Scan(&n); err != nil {
		c.t.Fatalf("count check-ins: %v", err)
	}
	return n
}

// TestPhase6SuccessCriteria walks the scanner acceptance criteria: a first scan
// admits the attendee, the database records it, and an immediate second scan is
// refused as already used.
func TestPhase6SuccessCriteria(t *testing.T) {
	c := newClient(t)
	organizer := c.register("phase6organizer")

	ticketID, qrToken := c.buyOneTicket(organizer.Token, "Phase 6 Gate Test")

	var eventID uuid.UUID
	if err := c.pool.QueryRow(t.Context(),
		`SELECT event_id FROM tickets WHERE id = $1`, uuid.MustParse(ticketID)).
		Scan(&eventID); err != nil {
		t.Fatalf("read event id: %v", err)
	}

	// --- the event selector offers this event -------------------------------
	scannable := c.get("/api/v1/events/scannable", organizer.Token)
	requireStatus(t, scannable, http.StatusOK)

	events, _ := scannable.Body["events"].([]any)
	found := false
	for _, raw := range events {
		if raw.(map[string]any)["id"] == eventID.String() {
			found = true
			if raw.(map[string]any)["access_via"] != "organizer" {
				t.Errorf("access_via = %v, want organizer", raw.(map[string]any)["access_via"])
			}
		}
	}
	if !found {
		t.Fatalf("the event is missing from the scanner's event list: %s", scannable.Raw)
	}
	t.Log("selector OK: the event appears in /events/scannable")

	// --- 3. The first scan admits the attendee ------------------------------
	first := c.scan(organizer.Token, eventID, qrToken)
	requireStatus(t, first, http.StatusOK)

	if first.Body["result"] != "valid" {
		t.Errorf("result = %v, want valid", first.Body["result"])
	}

	admitted := checkIn(first)
	if admitted["attendee_name"] != "Nurlan Amanov" {
		t.Errorf("attendee_name = %v, want Nurlan Amanov", admitted["attendee_name"])
	}
	if admitted["ticket_type_name"] != "General Admission" {
		t.Errorf("ticket_type_name = %v, want General Admission", admitted["ticket_type_name"])
	}

	stats, _ := admitted["stats"].(map[string]any)
	if checkedIn, _ := stats["checked_in"].(float64); int(checkedIn) != 1 {
		t.Errorf("stats.checked_in = %v, want 1", stats["checked_in"])
	}
	t.Logf("criterion 3 OK: green result for %v", admitted["attendee_name"])

	// --- 4. The database records it -----------------------------------------
	if got := c.ticketStatus(ticketID); got != "checked_in" {
		t.Errorf("criterion 4: ticket status = %q, want checked_in", got)
	}
	if got := c.activeCheckIns(ticketID); got != 1 {
		t.Errorf("criterion 4: %d active check_in_records, want exactly 1", got)
	}

	var (
		recordedBy uuid.UUID
		device     *string
		at         time.Time
	)
	if err := c.pool.QueryRow(t.Context(), `
		SELECT checked_in_by, device_label, checked_in_at
		  FROM check_in_records WHERE ticket_id = $1 AND reversed_at IS NULL`,
		uuid.MustParse(ticketID)).Scan(&recordedBy, &device, &at); err != nil {
		t.Fatalf("criterion 4: read check-in record: %v", err)
	}
	if recordedBy != organizer.ID {
		t.Errorf("criterion 4: checked_in_by = %v, want the scanning user %v", recordedBy, organizer.ID)
	}
	if device == nil || *device != "Scanner-01" {
		t.Errorf("criterion 4: device_label = %v, want Scanner-01", device)
	}
	t.Logf("criterion 4 OK: ticket is checked_in with 1 record, scanned by %v at %s",
		recordedBy, at.Format(time.RFC3339))

	// --- 5. An immediate second scan is refused -----------------------------
	second := c.scan(organizer.Token, eventID, qrToken)
	if second.Status != http.StatusConflict {
		t.Fatalf("criterion 5: status = %d, want 409; body = %s", second.Status, second.Raw)
	}
	if second.errorCode() != CodeAlreadyCheckedIn {
		t.Errorf("criterion 5: code = %q, want %q", second.errorCode(), CodeAlreadyCheckedIn)
	}

	errObj, _ := second.Body["error"].(map[string]any)
	if errObj["attendee_name"] != "Nurlan Amanov" {
		t.Errorf("criterion 5: the refusal should name the attendee, got %v", errObj["attendee_name"])
	}
	if errObj["checked_in_at"] == nil {
		t.Error("criterion 5: the refusal should say when the ticket was first used")
	}

	// The refused scan changed nothing.
	if got := c.activeCheckIns(ticketID); got != 1 {
		t.Errorf("criterion 5: %d active check-ins after the repeat scan, want still 1", got)
	}
	t.Logf("criterion 5 OK: refused with %q", second.errorCode())
}

// Two scanners hitting the same ticket at the same instant: exactly one may win.
func TestCheckInIsAtomicUnderConcurrency(t *testing.T) {
	c := newClient(t)
	organizer := c.register("raceGate")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Race Gate Event", "1000", 5)
	bought := c.buy(eventID, ticketTypeID, 1, "Race Attendee", "race@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)

	ticket := bought.Body["tickets"].([]any)[0].(map[string]any)
	ticketID := ticket["id"].(string)
	qrToken := ticket["qr_token"].(string)

	const scanners = 8
	results := make([]response, scanners)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := range scanners {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			results[index] = c.scan(organizer.Token, eventID, qrToken)
		}(i)
	}

	close(start)
	wg.Wait()

	admitted, refused := 0, 0
	for i, res := range results {
		switch {
		case res.Status == http.StatusOK:
			admitted++
		case res.Status == http.StatusConflict && res.errorCode() == CodeAlreadyCheckedIn:
			refused++
		default:
			t.Errorf("scanner %d got %d with code %q, want 200 or a 409 already-used",
				i, res.Status, res.errorCode())
		}
	}

	if admitted != 1 {
		t.Errorf("%d scanners admitted the same ticket, want exactly 1", admitted)
	}
	if refused != scanners-1 {
		t.Errorf("%d scanners were refused, want %d", refused, scanners-1)
	}
	if got := c.activeCheckIns(ticketID); got != 1 {
		t.Fatalf("DOUBLE ADMISSION: %d active check_in_records for one ticket", got)
	}
}

// SRS 4.14: a campaign QR must never open the gate.
func TestCheckInRejectsCampaignToken(t *testing.T) {
	c := newClient(t)
	organizer := c.register("campaignGate")
	eventID, _, _ := c.sellableEvent(organizer.Token, "Campaign Gate Event", "1000", 5)

	// A real campaign row, so this is not just a prefix check on a made-up value.
	var campaignToken string
	err := c.pool.QueryRow(t.Context(), `
		INSERT INTO campaigns (event_id, name, discount_type, discount_value, qr_token, status)
		VALUES ($1, 'Student Promo', 'percentage', 10, $2, 'active')
		RETURNING qr_token`, eventID, "CMP_"+uuid.NewString()).Scan(&campaignToken)
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	res := c.scan(organizer.Token, eventID, campaignToken)
	if res.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a campaign code; body = %s", res.Status, res.Raw)
	}
	if res.errorCode() != CodeCampaignToken {
		t.Errorf("code = %q, want %q", res.errorCode(), CodeCampaignToken)
	}

	var records int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM check_in_records WHERE event_id = $1`, eventID).Scan(&records); err != nil {
		t.Fatalf("count check-ins: %v", err)
	}
	if records != 0 {
		t.Errorf("%d check-ins recorded from a campaign code, want 0", records)
	}
}

func TestCheckInRejectsUnknownAndMalformedTokens(t *testing.T) {
	c := newClient(t)
	organizer := c.register("unknownGate")
	eventID, _, _ := c.sellableEvent(organizer.Token, "Unknown Token Event", "1000", 5)

	tests := []struct {
		name  string
		token string
	}{
		{"unknown admission token", "TKT_" + uuid.NewString()},
		{"not a token at all", "hello world"},
		{"a bare uuid", uuid.NewString()},
		{"a url", "https://biletflow.kz/events/some-event"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.scan(organizer.Token, eventID, tt.token)
			if res.Status != http.StatusNotFound {
				t.Errorf("status = %d, want 404; body = %s", res.Status, res.Raw)
			}
			if res.errorCode() != CodeUnknownTicket {
				t.Errorf("code = %q, want %q", res.errorCode(), CodeUnknownTicket)
			}
		})
	}

	requireErrorCode(t, c.scan(organizer.Token, eventID, "  "),
		http.StatusUnprocessableEntity, "validation_failed")
}

// A ticket for another event must not open this gate.
func TestCheckInRejectsTicketFromAnotherEvent(t *testing.T) {
	c := newClient(t)
	organizer := c.register("wrongEventGate")

	_, otherToken := c.buyOneTicket(organizer.Token, "The Other Concert")

	gateEventID, _, _ := c.sellableEvent(organizer.Token, "The Gate Event", "1000", 5)

	res := c.scan(organizer.Token, gateEventID, otherToken)
	if res.Status != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", res.Status, res.Raw)
	}
	if res.errorCode() != CodeWrongEvent {
		t.Errorf("code = %q, want %q", res.errorCode(), CodeWrongEvent)
	}
	errObj, _ := res.Body["error"].(map[string]any)
	if msg, _ := errObj["message"].(string); msg == "" {
		t.Error("the refusal should name the event the ticket is actually for")
	}
}

func TestCheckInRejectsRefundedTicket(t *testing.T) {
	c := newClient(t)
	organizer := c.register("refundedGate")

	ticketID, qrToken := c.buyOneTicket(organizer.Token, "Refunded Gate Event")

	var eventID uuid.UUID
	if err := c.pool.QueryRow(t.Context(),
		`SELECT event_id FROM tickets WHERE id = $1`, uuid.MustParse(ticketID)).Scan(&eventID); err != nil {
		t.Fatalf("read event id: %v", err)
	}
	if _, err := c.pool.Exec(t.Context(),
		`UPDATE tickets SET status = 'refunded' WHERE id = $1`, uuid.MustParse(ticketID)); err != nil {
		t.Fatalf("refund the ticket: %v", err)
	}

	res := c.scan(organizer.Token, eventID, qrToken)
	if res.Status != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", res.Status, res.Raw)
	}
	if res.errorCode() != CodeTicketNotValid {
		t.Errorf("code = %q, want %q", res.errorCode(), CodeTicketNotValid)
	}
	if got := c.activeCheckIns(ticketID); got != 0 {
		t.Errorf("%d check-ins recorded for a refunded ticket, want 0", got)
	}
}

// SRS 4.8: staff see and act on their assigned events only.
func TestCheckInAuthorization(t *testing.T) {
	c := newClient(t)
	organizer := c.register("gateOwner")
	outsider := c.register("gateOutsider")
	scanner := c.register("gateScanner")

	ticketID, qrToken := c.buyOneTicket(organizer.Token, "Authorized Gate Event")

	var eventID uuid.UUID
	if err := c.pool.QueryRow(t.Context(),
		`SELECT event_id FROM tickets WHERE id = $1`, uuid.MustParse(ticketID)).Scan(&eventID); err != nil {
		t.Fatalf("read event id: %v", err)
	}

	requireErrorCode(t, c.scan("", eventID, qrToken), http.StatusUnauthorized, "unauthorized")
	requireErrorCode(t, c.scan(outsider.Token, eventID, qrToken), http.StatusForbidden, "forbidden")

	// An unassigned account sees no events to scan.
	list := c.get("/api/v1/events/scannable", scanner.Token)
	requireStatus(t, list, http.StatusOK)
	if events, _ := list.Body["events"].([]any); len(events) != 0 {
		t.Errorf("an unassigned account can see %d events, want 0", len(events))
	}

	// The organizer assigns them, and the gate opens.
	assign := c.post("/api/v1/events/"+eventID.String()+"/staff", organizer.Token, map[string]any{
		"email": scanner.Email,
		"role":  "event_admin",
	})
	requireStatus(t, assign, http.StatusCreated)

	list = c.get("/api/v1/events/scannable", scanner.Token)
	requireStatus(t, list, http.StatusOK)
	events, _ := list.Body["events"].([]any)
	if len(events) != 1 {
		t.Fatalf("the assigned scanner sees %d events, want 1", len(events))
	}
	if events[0].(map[string]any)["access_via"] != "event_admin" {
		t.Errorf("access_via = %v, want event_admin", events[0].(map[string]any)["access_via"])
	}

	requireStatus(t, c.scan(scanner.Token, eventID, qrToken), http.StatusOK)

	// Being an Event Admin does not make them an organizer of anything.
	mine := c.get("/api/v1/events/mine", scanner.Token)
	requireStatus(t, mine, http.StatusOK)
	if own, _ := mine.Body["events"].([]any); len(own) != 0 {
		t.Errorf("the scanner appears to own %d events, want 0", len(own))
	}
}

func TestRevokedStaffLoseTheGate(t *testing.T) {
	c := newClient(t)
	organizer := c.register("revokeOwner")
	scanner := c.register("revokeScanner")

	ticketID, qrToken := c.buyOneTicket(organizer.Token, "Revoke Gate Event")
	var eventID uuid.UUID
	if err := c.pool.QueryRow(t.Context(),
		`SELECT event_id FROM tickets WHERE id = $1`, uuid.MustParse(ticketID)).Scan(&eventID); err != nil {
		t.Fatalf("read event id: %v", err)
	}

	assign := c.post("/api/v1/events/"+eventID.String()+"/staff", organizer.Token,
		map[string]any{"email": scanner.Email})
	requireStatus(t, assign, http.StatusCreated)
	assignmentID := assign.Body["assignment"].(map[string]any)["id"].(string)

	requireStatus(t, c.get("/api/v1/events/scannable", scanner.Token), http.StatusOK)

	requireStatus(t, c.delete(
		"/api/v1/events/"+eventID.String()+"/staff/"+assignmentID, organizer.Token),
		http.StatusNoContent)

	requireErrorCode(t, c.scan(scanner.Token, eventID, qrToken),
		http.StatusForbidden, "forbidden")

	list := c.get("/api/v1/events/scannable", scanner.Token)
	requireStatus(t, list, http.StatusOK)
	if events, _ := list.Body["events"].([]any); len(events) != 0 {
		t.Errorf("a revoked scanner still sees %d events, want 0", len(events))
	}
}

func TestAssignStaffValidation(t *testing.T) {
	c := newClient(t)
	organizer := c.register("assignValidation")
	eventID, _ := c.createEvent(organizer.Token, "Staff Validation Event")
	path := "/api/v1/events/" + eventID.String() + "/staff"

	res := c.post(path, organizer.Token, map[string]any{"email": "nobody@biletflow.test"})
	requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")
	if _, ok := res.errorFields()["email"]; !ok {
		t.Errorf("error fields = %v, want an entry for email", res.errorFields())
	}

	res = c.post(path, organizer.Token, map[string]any{"email": "not-an-email"})
	requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")

	other := c.register("assignRole")
	res = c.post(path, organizer.Token, map[string]any{"email": other.Email, "role": "wizard"})
	requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")
}

// SRS 4.8: an accidental check-in may be reversed, which frees the ticket.
func TestReverseCheckInAllowsRescan(t *testing.T) {
	c := newClient(t)
	organizer := c.register("reverseGate")

	ticketID, qrToken := c.buyOneTicket(organizer.Token, "Reverse Gate Event")
	var eventID uuid.UUID
	if err := c.pool.QueryRow(t.Context(),
		`SELECT event_id FROM tickets WHERE id = $1`, uuid.MustParse(ticketID)).Scan(&eventID); err != nil {
		t.Fatalf("read event id: %v", err)
	}

	requireStatus(t, c.scan(organizer.Token, eventID, qrToken), http.StatusOK)
	requireErrorCode(t, c.scan(organizer.Token, eventID, qrToken),
		http.StatusConflict, CodeAlreadyCheckedIn)

	reverse := c.post("/api/v1/tickets/"+ticketID+"/check-in/reverse", organizer.Token,
		map[string]any{"reason": "scanned by mistake"})
	requireStatus(t, reverse, http.StatusOK)

	if got := c.ticketStatus(ticketID); got != "valid" {
		t.Errorf("ticket status = %q after a reversal, want valid", got)
	}
	if got := c.activeCheckIns(ticketID); got != 0 {
		t.Errorf("%d active check-ins after a reversal, want 0", got)
	}

	// The reversed record is kept for the audit trail.
	var total int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM check_in_records WHERE ticket_id = $1`,
		uuid.MustParse(ticketID)).Scan(&total); err != nil {
		t.Fatalf("count records: %v", err)
	}
	if total != 1 {
		t.Errorf("%d total check-in records, want the reversed one retained", total)
	}

	// And the ticket works again.
	requireStatus(t, c.scan(organizer.Token, eventID, qrToken), http.StatusOK)
	if got := c.activeCheckIns(ticketID); got != 1 {
		t.Errorf("%d active check-ins after the re-scan, want 1", got)
	}

	requireErrorCode(t, c.post("/api/v1/tickets/"+ticketID+"/check-in/reverse", "", nil),
		http.StatusUnauthorized, "unauthorized")
}

func TestCheckInStatsTrackProgress(t *testing.T) {
	c := newClient(t)
	organizer := c.register("statsGate")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Stats Gate Event", "1000", 10)
	bought := c.buy(eventID, ticketTypeID, 3, "Group Buyer", "group@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)

	statsPath := "/api/v1/events/" + eventID.String() + "/check-in/stats"

	res := c.get(statsPath, organizer.Token)
	requireStatus(t, res, http.StatusOK)
	stats, _ := res.Body["stats"].(map[string]any)
	if issued, _ := stats["issued"].(float64); int(issued) != 3 {
		t.Errorf("issued = %v, want 3", stats["issued"])
	}
	if checkedIn, _ := stats["checked_in"].(float64); int(checkedIn) != 0 {
		t.Errorf("checked_in = %v, want 0", stats["checked_in"])
	}

	tickets, _ := bought.Body["tickets"].([]any)
	for i, raw := range tickets[:2] {
		token := raw.(map[string]any)["qr_token"].(string)
		requireStatus(t, c.scan(organizer.Token, eventID, token), http.StatusOK)
		_ = i
	}

	res = c.get(statsPath, organizer.Token)
	requireStatus(t, res, http.StatusOK)
	stats, _ = res.Body["stats"].(map[string]any)
	if checkedIn, _ := stats["checked_in"].(float64); int(checkedIn) != 2 {
		t.Errorf("checked_in = %v, want 2 after two scans", stats["checked_in"])
	}
}
