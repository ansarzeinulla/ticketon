package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// makePlatformAdmin grants the platform_admin role directly, the way an
// operator would: there is deliberately no self-service route to it.
func (c *client) makePlatformAdmin(userID uuid.UUID) {
	c.t.Helper()
	if _, err := c.pool.Exec(c.t.Context(),
		`INSERT INTO user_roles (user_id, role) VALUES ($1, 'platform_admin')
		 ON CONFLICT DO NOTHING`, userID); err != nil {
		c.t.Fatalf("grant platform_admin: %v", err)
	}
}

// buyAsRegisteredAttendee checks out with a signed-in buyer and returns the
// order id, so a support case can be attached to it.
func (c *client) buyAsRegisteredAttendee(
	buyerToken string, eventID, ticketTypeID uuid.UUID, email string,
) string {
	c.t.Helper()

	res := c.post("/api/v1/events/"+eventID.String()+"/checkout", buyerToken, map[string]any{
		"buyer_name":  "Aisha Nurlanova",
		"buyer_email": email,
		"items":       []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 1}},
	})
	requireStatus(c.t, res, http.StatusCreated)
	return res.Body["order"].(map[string]any)["id"].(string)
}

func supportCase(res response) map[string]any {
	c, _ := res.Body["case"].(map[string]any)
	return c
}

func supportMessages(res response) []any {
	m, _ := res.Body["messages"].([]any)
	return m
}

// TestPhase8SuccessCriteria walks the four Phase 8 acceptance criteria.
func TestPhase8SuccessCriteria(t *testing.T) {
	c := newClient(t)
	organizer := c.register("phase8organizer")
	attendee := c.register("phase8attendee")
	admin := c.register("phase8admin")
	c.makePlatformAdmin(admin.ID)

	eventID, slug, ticketTypeID := c.sellableEvent(organizer.Token, "Phase 8 Open Air", "5000", 20)
	orderID := c.buyAsRegisteredAttendee(attendee.Token, eventID, ticketTypeID, attendee.Email)

	// --- 1. The attendee opens a case on their order ------------------------
	opened := c.post("/api/v1/support/cases", attendee.Token, map[string]any{
		"category": "event_information",
		"subject":  "Parking",
		"message":  "Where is parking?",
		"order_id": orderID,
	})
	requireStatus(t, opened, http.StatusCreated)

	caseID := supportCase(opened)["id"].(string)
	if supportCase(opened)["status"] != "open" {
		t.Errorf("criterion 1: status = %v, want open", supportCase(opened)["status"])
	}
	if supportCase(opened)["category"] != "event_information" {
		t.Errorf("criterion 1: category = %v", supportCase(opened)["category"])
	}
	// The order, and through it the event, is captured automatically (SRS 4.13).
	if supportCase(opened)["order_id"] != orderID {
		t.Errorf("criterion 1: the case is not linked to the order")
	}
	if supportCase(opened)["event_id"] != eventID.String() {
		t.Errorf("criterion 1: the case did not pick up the order's event")
	}
	if got := supportMessages(opened); len(got) != 1 ||
		got[0].(map[string]any)["body"] != "Where is parking?" {
		t.Fatalf("criterion 1: the opening message is missing: %s", opened.Raw)
	}
	t.Logf("criterion 1 OK: case %v opened on the order", supportCase(opened)["case_number"])

	// --- 2. The organizer replies and resolves; the attendee sees both ------
	inbox := c.get("/api/v1/events/"+eventID.String()+"/support/cases", organizer.Token)
	requireStatus(t, inbox, http.StatusOK)
	if cases, _ := inbox.Body["cases"].([]any); len(cases) != 1 {
		t.Fatalf("criterion 2: the organizer's inbox has %d cases, want 1", len(cases))
	}

	replied := c.post("/api/v1/support/cases/"+caseID+"/messages", organizer.Token,
		map[string]any{"message": "Parking is in Zone B"})
	requireStatus(t, replied, http.StatusCreated)

	// Replying moves an untouched case to in_progress on its own.
	if supportCase(replied)["status"] != "in_progress" {
		t.Errorf("criterion 2: status = %v, want in_progress after the first reply",
			supportCase(replied)["status"])
	}

	resolved := c.patch("/api/v1/support/cases/"+caseID, organizer.Token,
		map[string]any{"status": "resolved"})
	requireStatus(t, resolved, http.StatusOK)
	if supportCase(resolved)["status"] != "resolved" {
		t.Fatalf("criterion 2: status = %v, want resolved", supportCase(resolved)["status"])
	}

	// The attendee refreshes: the reply and the new status are both there.
	refreshed := c.get("/api/v1/support/cases/"+caseID, attendee.Token)
	requireStatus(t, refreshed, http.StatusOK)

	if refreshed.Body["case"].(map[string]any)["status"] != "resolved" {
		t.Errorf("criterion 2: the attendee sees status %v, want resolved",
			supportCase(refreshed)["status"])
	}

	bodies := []string{}
	for _, raw := range supportMessages(refreshed) {
		bodies = append(bodies, raw.(map[string]any)["body"].(string))
	}
	if len(bodies) != 2 || bodies[0] != "Where is parking?" || bodies[1] != "Parking is in Zone B" {
		t.Fatalf("criterion 2: the attendee's thread is %v, want both messages in order", bodies)
	}
	if supportMessages(refreshed)[1].(map[string]any)["sender_role"] != "staff" {
		t.Errorf("criterion 2: the reply is not attributed to the organizer")
	}
	t.Logf("criterion 2 OK: attendee sees %q and status %v",
		bodies[1], supportCase(refreshed)["status"])

	// --- 3. A platform admin suspends the event -----------------------------
	// Neither the organizer nor the attendee can do this.
	requireErrorCode(t, c.post("/api/v1/admin/events/"+eventID.String()+"/suspend",
		organizer.Token, map[string]any{"reason": "trying it on"}),
		http.StatusForbidden, "forbidden")
	requireErrorCode(t, c.post("/api/v1/admin/events/"+eventID.String()+"/suspend",
		attendee.Token, nil), http.StatusForbidden, "forbidden")
	requireErrorCode(t, c.post("/api/v1/admin/events/"+eventID.String()+"/suspend", "", nil),
		http.StatusUnauthorized, "unauthorized")

	suspended := c.post("/api/v1/admin/events/"+eventID.String()+"/suspend", admin.Token,
		map[string]any{"reason": "Reported for misleading description"})
	requireStatus(t, suspended, http.StatusOK)
	if suspended.event()["status"] != "suspended" {
		t.Fatalf("criterion 3: status = %v, want suspended", suspended.event()["status"])
	}

	var dbStatus string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT status::text FROM events WHERE id = $1`, eventID).Scan(&dbStatus); err != nil {
		t.Fatalf("criterion 3: read event: %v", err)
	}
	if dbStatus != "suspended" {
		t.Fatalf("criterion 3: db status = %q, want suspended", dbStatus)
	}
	t.Logf("criterion 3 OK: the event is %s in PostgreSQL", dbStatus)

	// --- 4. Checkout is blocked and no ticket is issued ---------------------
	var ticketsBefore int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM tickets WHERE event_id = $1`, eventID).Scan(&ticketsBefore); err != nil {
		t.Fatalf("criterion 4: count tickets: %v", err)
	}

	blocked := c.buy(eventID, ticketTypeID, 1, "Late Buyer", "late@biletflow.test")
	if blocked.Status != http.StatusForbidden {
		t.Fatalf("criterion 4: status = %d, want 403; body = %s", blocked.Status, blocked.Raw)
	}
	if blocked.errorCode() != CodeEventSuspended {
		t.Errorf("criterion 4: code = %q, want %q", blocked.errorCode(), CodeEventSuspended)
	}

	var ticketsAfter, ordersAfter int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT (SELECT count(*) FROM tickets WHERE event_id = $1),
		        (SELECT count(*) FROM orders WHERE event_id = $1)`, eventID).
		Scan(&ticketsAfter, &ordersAfter); err != nil {
		t.Fatalf("criterion 4: count after: %v", err)
	}
	if ticketsAfter != ticketsBefore {
		t.Fatalf("criterion 4: %d tickets issued for a suspended event", ticketsAfter-ticketsBefore)
	}
	t.Logf("criterion 4 OK: checkout refused with %q, no ticket issued", blocked.errorCode())

	// The public page still resolves, with the banner flag set.
	public := c.get("/api/v1/public/events/"+slug, "")
	requireStatus(t, public, http.StatusOK)
	if public.Body["suspended"] != true {
		t.Errorf("criterion 4: public page suspended = %v, want true", public.Body["suspended"])
	}
	if public.Body["on_sale"] != false {
		t.Errorf("criterion 4: public page on_sale = %v, want false", public.Body["on_sale"])
	}
}

// SRS 4.13: a case is visible only to the requester, the event's organizer and
// platform admins.
func TestSupportCaseAccessControl(t *testing.T) {
	c := newClient(t)
	organizer := c.register("supportowner")
	attendee := c.register("supportattendee")
	stranger := c.register("supportstranger")
	admin := c.register("supportadmin")
	c.makePlatformAdmin(admin.ID)

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Access Control Event", "1000", 10)
	orderID := c.buyAsRegisteredAttendee(attendee.Token, eventID, ticketTypeID, attendee.Email)

	opened := c.post("/api/v1/support/cases", attendee.Token, map[string]any{
		"category": "payment", "subject": "Charged twice?",
		"message": "I think I was charged twice.", "order_id": orderID,
	})
	requireStatus(t, opened, http.StatusCreated)
	caseID := supportCase(opened)["id"].(string)
	path := "/api/v1/support/cases/" + caseID

	requireStatus(t, c.get(path, attendee.Token), http.StatusOK)
	requireStatus(t, c.get(path, organizer.Token), http.StatusOK)
	requireStatus(t, c.get(path, admin.Token), http.StatusOK)

	// A stranger gets 404, not 403: a 403 would confirm the case exists.
	requireErrorCode(t, c.get(path, stranger.Token), http.StatusNotFound, "not_found")
	requireErrorCode(t, c.get(path, ""), http.StatusUnauthorized, "unauthorized")

	requireErrorCode(t, c.post(path+"/messages", stranger.Token,
		map[string]any{"message": "let me in"}), http.StatusNotFound, "not_found")

	// The requester may reply but may not declare their own case resolved.
	requireStatus(t, c.post(path+"/messages", attendee.Token,
		map[string]any{"message": "Any update?"}), http.StatusCreated)
	requireErrorCode(t, c.patch(path, attendee.Token, map[string]any{"status": "resolved"}),
		http.StatusForbidden, "forbidden")

	// A stranger cannot see it in their own case list either.
	mine := c.get("/api/v1/support/cases", stranger.Token)
	requireStatus(t, mine, http.StatusOK)
	if cases, _ := mine.Body["cases"].([]any); len(cases) != 0 {
		t.Errorf("a stranger sees %d cases, want 0", len(cases))
	}

	// And not in another organizer's event inbox.
	requireErrorCode(t, c.get("/api/v1/events/"+eventID.String()+"/support/cases", stranger.Token),
		http.StatusForbidden, "forbidden")
}

// Internal notes are written on the same thread but never shown to the person
// who opened the case.
func TestInternalNotesAreStaffOnly(t *testing.T) {
	c := newClient(t)
	organizer := c.register("noteowner")
	attendee := c.register("noteattendee")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Internal Note Event", "1000", 10)
	orderID := c.buyAsRegisteredAttendee(attendee.Token, eventID, ticketTypeID, attendee.Email)

	opened := c.post("/api/v1/support/cases", attendee.Token, map[string]any{
		"category": "refund", "subject": "Refund please",
		"message": "Can I get a refund?", "order_id": orderID,
	})
	requireStatus(t, opened, http.StatusCreated)
	caseID := supportCase(opened)["id"].(string)
	path := "/api/v1/support/cases/" + caseID

	requireStatus(t, c.post(path+"/messages", organizer.Token, map[string]any{
		"message": "Checking with finance", "internal_note": true,
	}), http.StatusCreated)
	requireStatus(t, c.post(path+"/messages", organizer.Token, map[string]any{
		"message": "We can refund you this week",
	}), http.StatusCreated)

	staffView := c.get(path, organizer.Token)
	requireStatus(t, staffView, http.StatusOK)
	if got := len(supportMessages(staffView)); got != 3 {
		t.Errorf("the organizer sees %d messages, want 3 including the note", got)
	}

	attendeeView := c.get(path, attendee.Token)
	requireStatus(t, attendeeView, http.StatusOK)
	messages := supportMessages(attendeeView)
	if len(messages) != 2 {
		t.Fatalf("the attendee sees %d messages, want 2 with the note hidden", len(messages))
	}
	for _, raw := range messages {
		if raw.(map[string]any)["body"] == "Checking with finance" {
			t.Error("an internal note leaked to the attendee")
		}
	}

	// An attendee cannot write an internal note either.
	requireStatus(t, c.post(path+"/messages", attendee.Token, map[string]any{
		"message": "sneaky", "internal_note": true,
	}), http.StatusCreated)

	var internalFromAttendee int
	if err := c.pool.QueryRow(t.Context(), `
		SELECT count(*) FROM support_messages
		 WHERE support_case_id = $1 AND is_internal_note = true AND sender_user_id = $2`,
		uuid.MustParse(caseID), attendee.ID).Scan(&internalFromAttendee); err != nil {
		t.Fatalf("count notes: %v", err)
	}
	if internalFromAttendee != 0 {
		t.Errorf("the attendee wrote %d internal notes, want 0", internalFromAttendee)
	}
}

func TestSupportCaseStatusFlow(t *testing.T) {
	c := newClient(t)
	organizer := c.register("statusowner")
	attendee := c.register("statusattendee")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Status Flow Event", "1000", 10)
	orderID := c.buyAsRegisteredAttendee(attendee.Token, eventID, ticketTypeID, attendee.Email)

	opened := c.post("/api/v1/support/cases", attendee.Token, map[string]any{
		"category": "seating", "subject": "Seat swap",
		"message": "Can we sit together?", "order_id": orderID,
	})
	requireStatus(t, opened, http.StatusCreated)
	path := "/api/v1/support/cases/" + supportCase(opened)["id"].(string)

	for _, status := range []string{"in_progress", "waiting_for_customer", "resolved"} {
		res := c.patch(path, organizer.Token, map[string]any{"status": status})
		requireStatus(t, res, http.StatusOK)
		if supportCase(res)["status"] != status {
			t.Errorf("status = %v, want %v", supportCase(res)["status"], status)
		}
	}

	// Resolving stamps resolved_at, and reopening clears it - the schema's
	// support_cases_resolved_chk demands they stay consistent.
	var resolvedAt *string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT resolved_at::text FROM support_cases WHERE id = $1`,
		uuid.MustParse(supportCase(opened)["id"].(string))).Scan(&resolvedAt); err != nil {
		t.Fatalf("read resolved_at: %v", err)
	}
	if resolvedAt == nil {
		t.Error("resolved_at was not stamped when the case was resolved")
	}

	reopened := c.patch(path, organizer.Token, map[string]any{"status": "open"})
	requireStatus(t, reopened, http.StatusOK)
	if err := c.pool.QueryRow(t.Context(),
		`SELECT resolved_at::text FROM support_cases WHERE id = $1`,
		uuid.MustParse(supportCase(opened)["id"].(string))).Scan(&resolvedAt); err != nil {
		t.Fatalf("read resolved_at: %v", err)
	}
	if resolvedAt != nil {
		t.Error("resolved_at survived reopening the case")
	}

	requireErrorCode(t, c.patch(path, organizer.Token, map[string]any{"status": "on_fire"}),
		http.StatusUnprocessableEntity, "validation_failed")
}

func TestOpenCaseValidation(t *testing.T) {
	c := newClient(t)
	organizer := c.register("casevalidationowner")
	attendee := c.register("casevalidation")
	stranger := c.register("casevalidationstranger")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Case Validation Event", "1000", 10)
	orderID := c.buyAsRegisteredAttendee(attendee.Token, eventID, ticketTypeID, attendee.Email)

	tests := []struct {
		name      string
		body      map[string]any
		wantField string
	}{
		{"missing category", map[string]any{"subject": "s", "message": "m"}, "category"},
		{"unknown category", map[string]any{"category": "lost_dog", "subject": "s", "message": "m"}, "category"},
		{"missing subject", map[string]any{"category": "payment", "message": "m"}, "subject"},
		{"blank subject", map[string]any{"category": "payment", "subject": "  ", "message": "m"}, "subject"},
		{"missing message", map[string]any{"category": "payment", "subject": "s"}, "message"},
		{"blank message", map[string]any{"category": "payment", "subject": "s", "message": " "}, "message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.post("/api/v1/support/cases", attendee.Token, tt.body)
			requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")
			if _, ok := res.errorFields()[tt.wantField]; !ok {
				t.Errorf("error fields = %v, want %q", res.errorFields(), tt.wantField)
			}
		})
	}

	// Someone else's order is not a valid context.
	requireErrorCode(t, c.post("/api/v1/support/cases", stranger.Token, map[string]any{
		"category": "payment", "subject": "Nosy", "message": "About your order",
		"order_id": orderID,
	}), http.StatusForbidden, "forbidden")

	requireErrorCode(t, c.post("/api/v1/support/cases", "", map[string]any{
		"category": "payment", "subject": "s", "message": "m",
	}), http.StatusUnauthorized, "unauthorized")

	var count int
	if err := c.pool.QueryRow(t.Context(), `SELECT count(*) FROM support_cases`).Scan(&count); err != nil {
		t.Fatalf("count cases: %v", err)
	}
	if count != 0 {
		t.Errorf("%d invalid cases reached the database, want 0", count)
	}
}

// A guest buys, registers later with the same address, and can still ask about
// the order they placed.
func TestGuestBuyerCanOpenCaseAfterRegistering(t *testing.T) {
	c := newClient(t)
	organizer := c.register("guestcaseowner")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Guest Case Event", "1000", 10)

	const guestEmail = "guest.buyer@biletflow.test"

	bought := c.buy(eventID, ticketTypeID, 1, "Guest Buyer", guestEmail)
	requireStatus(t, bought, http.StatusCreated)
	orderID := bought.Body["order"].(map[string]any)["id"].(string)

	registered := c.post("/api/v1/auth/register", "", map[string]any{
		"email": guestEmail, "password": "correct horse battery",
	})
	requireStatus(t, registered, http.StatusCreated)
	token := registered.Body["access_token"].(string)

	opened := c.post("/api/v1/support/cases", token, map[string]any{
		"category": "ticket_delivery", "subject": "No email",
		"message": "My ticket never arrived.", "order_id": orderID,
	})
	requireStatus(t, opened, http.StatusCreated)
	if supportCase(opened)["order_id"] != orderID {
		t.Error("the case was not linked to the guest order")
	}
}

func TestSupportCategoriesAreServed(t *testing.T) {
	c := newClient(t)

	res := c.get("/api/v1/support/categories", "")
	requireStatus(t, res, http.StatusOK)

	categories, _ := res.Body["categories"].([]any)
	if len(categories) != 8 {
		t.Errorf("%d categories, want the 8 from the schema enum", len(categories))
	}

	// Every value must be accepted by the support_case_category enum.
	for _, raw := range categories {
		var valid bool
		if err := c.pool.QueryRow(c.t.Context(),
			`SELECT $1::text = ANY (enum_range(NULL::support_case_category)::text[])`,
			raw.(string)).Scan(&valid); err != nil {
			t.Fatalf("check category: %v", err)
		}
		if !valid {
			t.Errorf("category %q is not in the support_case_category enum", raw)
		}
	}
}
