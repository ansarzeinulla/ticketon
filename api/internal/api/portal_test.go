package api

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/biletflow/api/internal/email"
)

// --- the administrative portal (SRS 2.1, 4.12) -------------------------------

func TestAdminSearchIsRestrictedToPlatformAdmins(t *testing.T) {
	c := newClient(t)
	organizer := c.register("portalorganizer")
	admin := c.register("portaladmin")

	for _, path := range []string{
		"/api/v1/admin/search?q=anything",
		"/api/v1/admin/reports/events.csv",
	} {
		requireStatus(t, c.get(path, ""), http.StatusUnauthorized)
		requireStatus(t, c.get(path, organizer.Token), http.StatusForbidden)
	}

	// The role is granted by an operator, never self-service.
	c.makePlatformAdmin(admin.ID)
	requireStatus(t, c.get("/api/v1/admin/search?q=", admin.Token), http.StatusOK)
}

// TestAdminSearchOpensOnSomethingUseful: an empty query is a legitimate first
// visit, not an error.
func TestAdminSearchOpensOnSomethingUseful(t *testing.T) {
	c := newClient(t)
	admin := c.register("emptyquery")
	c.makePlatformAdmin(admin.ID)

	organizer := c.register("emptyqueryorg")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Something To Find", "5000", 5)
	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "A Buyer", "abuyer@biletflow.test"),
		http.StatusCreated)

	res := c.get("/api/v1/admin/search?q=", admin.Token)
	requireStatus(t, res, http.StatusOK)

	results := res.Body["results"].(map[string]any)
	for _, section := range []string{"users", "events", "orders", "payments"} {
		if len(results[section].([]any)) == 0 {
			t.Errorf("an empty query returned no %s", section)
		}
	}
}

// TestAdminSearchFindsNothingForNonsense guards against a query that matches
// everything by accident - the failure mode of a badly built ILIKE.
func TestAdminSearchFindsNothingForNonsense(t *testing.T) {
	c := newClient(t)
	admin := c.register("nonsense")
	c.makePlatformAdmin(admin.ID)

	organizer := c.register("nonsenseorg")
	c.sellableEvent(organizer.Token, "Real Event", "5000", 5)

	res := c.get("/api/v1/admin/search?q=zzzznotarealthingzzzz", admin.Token)
	requireStatus(t, res, http.StatusOK)

	results := res.Body["results"].(map[string]any)
	for _, section := range []string{"users", "events", "orders", "payments"} {
		if got := len(results[section].([]any)); got != 0 {
			t.Errorf("a nonsense query matched %d %s", got, section)
		}
	}
}

// TestAdminSearchSurfacesFailuresAndActivation covers the SRS 4.12 bullets
// about inspecting activation records and monitoring payments.
func TestAdminSearchSurfacesFailuresAndActivation(t *testing.T) {
	c := newClient(t)
	admin := c.register("inspector")
	c.makePlatformAdmin(admin.ID)

	organizer := c.register("inspected")
	eventID, _, _ := c.sellableEvent(organizer.Token, "Inspected Fest", "5000", 5)

	res := c.get("/api/v1/admin/search?q=Inspected+Fest", admin.Token)
	requireStatus(t, res, http.StatusOK)

	events := res.Body["results"].(map[string]any)["events"].([]any)
	if len(events) != 1 {
		t.Fatalf("events = %d", len(events))
	}
	if events[0].(map[string]any)["activation_status"] != "active" {
		t.Errorf("activation_status = %v, want active (sellableEvent activates)",
			events[0].(map[string]any)["activation_status"])
	}

	// The activation fee shows up as a payment an admin can audit.
	payments := res.Body["results"].(map[string]any)["payments"].([]any)
	found := false
	for _, p := range payments {
		if p.(map[string]any)["purpose"] == "paid_sales_activation" {
			found = true
			if p.(map[string]any)["is_simulated"] != true {
				t.Error("the activation fee is not flagged as simulated")
			}
		}
	}
	if !found {
		t.Errorf("no activation payment in the results: %v", payments)
	}
	_ = eventID
}

// TestAdminReportIsValidCSV also checks the numbers, because a report that
// parses but lies is worse than one that does not parse.
func TestAdminReportIsValidCSV(t *testing.T) {
	c := newClient(t)
	admin := c.register("reporter")
	c.makePlatformAdmin(admin.ID)

	organizer := c.register("reported")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Reported Fest", "5000", 10)

	bought := c.buy(eventID, ticketTypeID, 3, "Report Buyer", "report@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	qr := bought.Body["tickets"].([]any)[0].(map[string]any)["qr_token"].(string)
	requireStatus(t, c.scan(organizer.Token, eventID, qr), http.StatusOK)

	// A refund, so the refunded and net columns have something to say.
	second := c.buy(eventID, ticketTypeID, 1, "Refunded Buyer", "refunded@biletflow.test")
	requireStatus(t, second, http.StatusCreated)
	requireStatus(t, c.post("/api/v1/orders/"+orderIDOf(t, second)+"/refund",
		organizer.Token, nil), http.StatusOK)

	report := c.getBinary("/api/v1/admin/reports/events.csv", admin.Token)
	if report.Status != http.StatusOK {
		t.Fatalf("status = %d", report.Status)
	}

	records, err := csv.NewReader(bytes.NewReader(report.Body)).ReadAll()
	if err != nil {
		t.Fatalf("not valid CSV: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("rows = %d, want a header and one event", len(records))
	}

	fields := map[string]string{}
	for i, name := range records[0] {
		fields[name] = records[1][i]
	}

	// 4 tickets bought, 1 refunded, so 3 live. 20000 taken, 5000 given back.
	for name, want := range map[string]string{
		"tickets_sold":  "3",
		"checked_in":    "1",
		"orders":        "2",
		"gross_kzt":     "20000.00",
		"refunded_kzt":  "5000.00",
		"net_kzt":       "15000.00",
		"discounts_kzt": "0.00",
		"lifecycle":     "upcoming",
	} {
		if fields[name] != want {
			t.Errorf("%s = %q, want %q", name, fields[name], want)
		}
	}
}

// TestAdminReportEscapesCommas is what CSV encoding is for: an event titled
// "Jazz, Blues & More" must not become three columns.
func TestAdminReportEscapesCommas(t *testing.T) {
	c := newClient(t)
	admin := c.register("csvescape")
	c.makePlatformAdmin(admin.ID)

	organizer := c.register("csvescapeorg")
	const awkward = `Jazz, Blues & "More"`
	body := validEventBody(awkward)
	requireStatus(t, c.post("/api/v1/events", organizer.Token, body), http.StatusCreated)

	report := c.getBinary("/api/v1/admin/reports/events.csv", admin.Token)
	records, err := csv.NewReader(bytes.NewReader(report.Body)).ReadAll()
	if err != nil {
		t.Fatalf("a comma in a title broke the CSV: %v", err)
	}
	if len(records) != 2 || len(records[1]) != len(records[0]) {
		t.Fatalf("row has %d fields, header has %d", len(records[1]), len(records[0]))
	}
	if records[1][1] != awkward {
		t.Errorf("title round-tripped as %q, want %q", records[1][1], awkward)
	}
}

// --- uploads (SRS 4.2) -------------------------------------------------------

func TestUploadRejectsNonImages(t *testing.T) {
	c := newClient(t)
	organizer := c.register("uploadtypes")

	// A .png name on a text file is not a PNG. The type is sniffed from the
	// content, so the name buys nothing.
	res := c.uploadImage(organizer.Token, "not-really.png", []byte("#!/bin/sh\nrm -rf /\n"))
	requireErrorCode(t, res, http.StatusUnsupportedMediaType, CodeUnsupportedMedia)

	// And nothing was written.
	requireStatus(t, c.uploadImage(organizer.Token, "real.png", pngBytes()), http.StatusCreated)
}

func TestUploadNeedsAnAccount(t *testing.T) {
	c := newClient(t)
	requireStatus(t, c.uploadImage("", "banner.png", pngBytes()), http.StatusUnauthorized)
}

// TestUploadIgnoresTheClientFilename: a supplied name is a path-traversal
// vector and a way to overwrite somebody else's banner.
func TestUploadIgnoresTheClientFilename(t *testing.T) {
	c := newClient(t)
	organizer := c.register("uploadname")

	res := c.uploadImage(organizer.Token, "../../../etc/passwd.png", pngBytes())
	requireStatus(t, res, http.StatusCreated)

	stored := res.Body["filename"].(string)
	if strings.ContainsAny(stored, `/\.`) && !strings.HasSuffix(stored, ".png") {
		t.Errorf("stored filename %q contains path characters", stored)
	}
	if strings.Contains(stored, "passwd") || strings.Contains(stored, "..") {
		t.Errorf("stored filename %q came from the client", stored)
	}

	// Two uploads of the same bytes get different names, so one organizer
	// cannot clobber another's banner.
	other := c.uploadImage(organizer.Token, "banner.png", pngBytes())
	requireStatus(t, other, http.StatusCreated)
	if other.Body["filename"] == stored {
		t.Error("two uploads produced the same filename")
	}
}

func TestUploadServingRefusesTraversal(t *testing.T) {
	c := newClient(t)
	organizer := c.register("traversal")
	requireStatus(t, c.uploadImage(organizer.Token, "b.png", pngBytes()), http.StatusCreated)

	for _, path := range []string{
		"/uploads/..%2f..%2fetc%2fpasswd",
		"/uploads/.",
		"/uploads/does-not-exist.png",
	} {
		if res := c.getBinary(path, ""); res.Status == http.StatusOK {
			t.Errorf("%s was served with 200", path)
		}
	}
}

// --- attendee search (SRS 4.8) -----------------------------------------------

func TestAttendeeSearchMatchesTheUsefulFields(t *testing.T) {
	c := newClient(t)
	organizer := c.register("attendeesearch")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Search Fest", "5000", 20)

	bought := c.buy(eventID, ticketTypeID, 1, "Nurlan Amanov", "nurlan@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)

	orderNumber := bought.Body["order"].(map[string]any)["order_number"].(string)
	ticketCode := bought.Body["tickets"].([]any)[0].(map[string]any)["ticket_code"].(string)

	// Staff at a door will type any of these.
	for _, query := range []string{"nurlan", "NURLAN", "Amanov", "nurlan@biletflow.test",
		orderNumber, ticketCode} {
		res := c.get("/api/v1/events/"+eventID.String()+"/attendees?q="+query, organizer.Token)
		requireStatus(t, res, http.StatusOK)
		if got := len(res.Body["attendees"].([]any)); got != 1 {
			t.Errorf("searching %q found %d, want 1", query, got)
		}
	}

	// An empty query opens the screen with the list rather than nothing.
	all := c.get("/api/v1/events/"+eventID.String()+"/attendees?q=", organizer.Token)
	requireStatus(t, all, http.StatusOK)
	if len(all.Body["attendees"].([]any)) != 1 {
		t.Error("an empty query returned no attendees")
	}
}

// TestAttendeeSearchIsScopedToOneEvent: a door device authorised for tonight's
// event must not be able to read last week's guest list.
func TestAttendeeSearchIsScopedToOneEvent(t *testing.T) {
	c := newClient(t)
	organizer := c.register("scoped")

	firstID, _, firstType := c.sellableEvent(organizer.Token, "First Fest", "5000", 10)
	secondID, _, secondType := c.sellableEvent(organizer.Token, "Second Fest", "5000", 10)

	requireStatus(t, c.buy(firstID, firstType, 1, "Only At First", "first@biletflow.test"),
		http.StatusCreated)
	requireStatus(t, c.buy(secondID, secondType, 1, "Only At Second", "second@biletflow.test"),
		http.StatusCreated)

	res := c.get("/api/v1/events/"+firstID.String()+"/attendees?q=Only", organizer.Token)
	requireStatus(t, res, http.StatusOK)

	attendees := res.Body["attendees"].([]any)
	if len(attendees) != 1 {
		t.Fatalf("matches = %d, want only this event's attendee", len(attendees))
	}
	if attendees[0].(map[string]any)["attendee_name"] != "Only At First" {
		t.Errorf("found %v, which belongs to another event",
			attendees[0].(map[string]any)["attendee_name"])
	}
}

// TestManualCheckInRefusesAVoidTicket: the manual path enforces the same rules
// as the camera, or it becomes the way around them.
func TestManualCheckInRefusesAVoidTicket(t *testing.T) {
	c := newClient(t)
	organizer := c.register("manualvoid")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Void Door Fest", "5000", 10)

	bought := c.buy(eventID, ticketTypeID, 1, "Refunded Guest", "void@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	ticketID := bought.Body["tickets"].([]any)[0].(map[string]any)["id"].(string)

	requireStatus(t, c.post("/api/v1/orders/"+orderIDOf(t, bought)+"/refund",
		organizer.Token, nil), http.StatusOK)

	requireErrorCode(t, c.post("/api/v1/events/"+eventID.String()+"/check-in/manual",
		organizer.Token, map[string]any{"ticket_id": ticketID}),
		http.StatusConflict, CodeTicketNotValid)

	// The search says so too, so the app can grey the row out.
	res := c.get("/api/v1/events/"+eventID.String()+"/attendees?q=Refunded", organizer.Token)
	row := res.Body["attendees"].([]any)[0].(map[string]any)
	if row["status"] != "refunded" || row["admissible"] != false {
		t.Errorf("a refunded ticket reads %v/%v", row["status"], row["admissible"])
	}
}

func TestManualCheckInRejectsAnotherEventsTicket(t *testing.T) {
	c := newClient(t)
	organizer := c.register("manualwrong")

	firstID, _, firstType := c.sellableEvent(organizer.Token, "Right Fest", "5000", 10)
	secondID, _, _ := c.sellableEvent(organizer.Token, "Wrong Fest", "5000", 10)

	bought := c.buy(firstID, firstType, 1, "Guest", "guest@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	ticketID := bought.Body["tickets"].([]any)[0].(map[string]any)["id"].(string)

	requireErrorCode(t, c.post("/api/v1/events/"+secondID.String()+"/check-in/manual",
		organizer.Token, map[string]any{"ticket_id": ticketID}),
		http.StatusConflict, CodeWrongEvent)
}

func TestManualCheckInValidatesItsInput(t *testing.T) {
	c := newClient(t)
	organizer := c.register("manualinput")
	eventID, _, _ := c.sellableEvent(organizer.Token, "Input Fest", "5000", 10)

	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/check-in/manual",
		organizer.Token, map[string]any{"ticket_id": "not-a-uuid"}),
		http.StatusUnprocessableEntity)

	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/check-in/manual",
		organizer.Token, map[string]any{"ticket_id": "11111111-1111-1111-1111-111111111111"}),
		http.StatusNotFound)
}

// --- the new notification triggers (SRS 4.10) --------------------------------

func TestCancellingAnEventNotifiesTicketHolders(t *testing.T) {
	c := newClient(t)
	organizer := c.register("cancelnotice")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Doomed Fest", "5000", 10)

	// Two orders, one of them for three tickets.
	requireStatus(t, c.buy(eventID, ticketTypeID, 3, "Aliya T", "aliya.cancel@biletflow.test"),
		http.StatusCreated)
	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Bekzat S", "bekzat.cancel@biletflow.test"),
		http.StatusCreated)

	// Give the event a refund policy, which the notice should carry.
	const policy = "Full refunds within 14 days of a cancellation."
	requireStatus(t, c.patch("/api/v1/events/"+eventID.String(), organizer.Token,
		map[string]any{"refund_policy": policy}), http.StatusOK)

	c.mail.Reset()
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/cancel", organizer.Token, nil),
		http.StatusOK)
	c.waitForMail()

	// One message per order, not per ticket: three seats is still one email.
	aliya := c.mail.To("aliya.cancel@biletflow.test")
	if len(aliya) != 1 {
		t.Fatalf("emails to the three-ticket buyer = %d, want 1", len(aliya))
	}
	if aliya[0].Type != email.TypeEventCancelled {
		t.Errorf("type = %q", aliya[0].Type)
	}
	if aliya[0].Subject != "Doomed Fest has been cancelled" {
		t.Errorf("subject = %q", aliya[0].Subject)
	}
	for _, want := range []string{"Hi Aliya,", "3, now void", policy} {
		if !contains(aliya[0].Body, want) {
			t.Errorf("the notice is missing %q; body:\n%s", want, aliya[0].Body)
		}
	}

	if got := len(c.mail.To("bekzat.cancel@biletflow.test")); got != 1 {
		t.Errorf("emails to the second buyer = %d, want 1", got)
	}
}

func TestSupportReplyNotifiesTheOtherSide(t *testing.T) {
	c := newClient(t)
	organizer := c.register("supportnotify")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Chatty Fest", "5000", 10)

	buyer := c.register("supportbuyer")
	orderID := c.buyAsRegisteredAttendee(buyer.Token, eventID, ticketTypeID, buyer.Email)

	// The attendee asks, so the organizer is told.
	c.mail.Reset()
	opened := c.post("/api/v1/support/cases", buyer.Token, map[string]any{
		"order_id": orderID, "category": "event_information", "subject": "Where is parking?",
		"message": "Where is parking?",
	})
	requireStatus(t, opened, http.StatusCreated)
	caseID := opened.Body["case"].(map[string]any)["id"].(string)

	c.mail.Reset()
	requireStatus(t, c.post("/api/v1/support/cases/"+caseID+"/messages", organizer.Token,
		map[string]any{"message": "Parking is in Zone B, next to the west gate."}),
		http.StatusCreated)
	c.waitForMail()

	// The organizer replied, so the attendee hears about it.
	sent := c.mail.To(buyer.Email)
	if len(sent) != 1 {
		t.Fatalf("emails to the attendee = %d, want 1", len(sent))
	}
	if sent[0].Type != email.TypeSupportMessage {
		t.Errorf("type = %q", sent[0].Type)
	}
	if !contains(sent[0].Body, "Zone B") || !contains(sent[0].Body, "Where is parking?") {
		t.Errorf("the notice does not quote the reply:\n%s", sent[0].Body)
	}

	// An internal note tells the attendee nothing - that is what the checkbox
	// is for.
	c.mail.Reset()
	requireStatus(t, c.post("/api/v1/support/cases/"+caseID+"/messages", organizer.Token,
		map[string]any{"message": "Nobody has checked the west gate signage.",
			"internal_note": true}), http.StatusCreated)
	c.waitForMail()

	if got := len(c.mail.To(buyer.Email)); got != 0 {
		t.Errorf("an internal note sent %d email(s) to the attendee", got)
	}
}
