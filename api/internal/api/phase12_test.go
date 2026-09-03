package api

import (
	"bytes"
	"encoding/csv"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/email"
	"github.com/biletflow/api/internal/testutil"
)

// TestPhase12SuccessCriteria walks the four criteria for the phase.
//
//  1. a platform admin searches for a user and downloads a CSV report;
//  2. a forgotten password is reset with the token printed to the console;
//  3. an organizer's banner and refund policy reach the public event page;
//  4. staff find an attendee by name and check them in without a QR code.
func TestPhase12SuccessCriteria(t *testing.T) {
	t.Run("criterion 1: admin search and CSV report", func(t *testing.T) {
		c := newClient(t)
		admin := c.register("p12admin")
		c.makePlatformAdmin(admin.ID)

		organizer := c.register("p12organizer")
		eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Admin Portal Fest", "5000", 10)
		requireStatus(t, c.buy(eventID, ticketTypeID, 2, "Searchable Buyer",
			"searchable@biletflow.test"), http.StatusCreated)

		// --- the portal is for admins only ----------------------------------
		requireStatus(t, c.get("/api/v1/admin/search?q=", ""), http.StatusUnauthorized)
		requireStatus(t, c.get("/api/v1/admin/search?q=", organizer.Token), http.StatusForbidden)

		// --- searching for a user -------------------------------------------
		found := c.get("/api/v1/admin/search?q="+organizer.Email, admin.Token)
		requireStatus(t, found, http.StatusOK)

		results := found.Body["results"].(map[string]any)
		users := results["users"].([]any)
		if len(users) != 1 {
			t.Fatalf("users found = %d, want exactly the one searched for", len(users))
		}
		user := users[0].(map[string]any)
		if user["email"] != organizer.Email {
			t.Errorf("found %v, want %v", user["email"], organizer.Email)
		}
		if user["event_count"] != float64(1) {
			t.Errorf("event_count = %v, want 1", user["event_count"])
		}

		// The same query reaches events, orders and payments (SRS 4.12).
		byOrder := c.get("/api/v1/admin/search?q=searchable@biletflow.test", admin.Token)
		requireStatus(t, byOrder, http.StatusOK)
		orders := byOrder.Body["results"].(map[string]any)["orders"].([]any)
		if len(orders) != 1 {
			t.Fatalf("orders found by buyer email = %d, want 1", len(orders))
		}

		byEvent := c.get("/api/v1/admin/search?q=Admin+Portal+Fest", admin.Token)
		requireStatus(t, byEvent, http.StatusOK)
		events := byEvent.Body["results"].(map[string]any)["events"].([]any)
		if len(events) != 1 {
			t.Fatalf("events found by title = %d, want 1", len(events))
		}
		if events[0].(map[string]any)["tickets_sold"] != float64(2) {
			t.Errorf("tickets_sold = %v, want 2", events[0].(map[string]any)["tickets_sold"])
		}

		// The dashboard header counts the platform.
		stats := found.Body["stats"].(map[string]any)
		if stats["events"] != float64(1) || stats["tickets_sold"] != float64(2) {
			t.Errorf("stats = %v", stats)
		}

		// --- the CSV report --------------------------------------------------
		report := c.getBinary("/api/v1/admin/reports/events.csv", admin.Token)
		if report.Status != http.StatusOK {
			t.Fatalf("report status = %d", report.Status)
		}
		if ct := report.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
			t.Errorf("Content-Type = %q, want text/csv", ct)
		}
		if cd := report.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
			t.Errorf("Content-Disposition = %q, want an attachment", cd)
		}

		records, err := csv.NewReader(bytes.NewReader(report.Body)).ReadAll()
		if err != nil {
			t.Fatalf("the report is not valid CSV: %v", err)
		}
		if len(records) != 2 {
			t.Fatalf("report rows = %d (plus header), want 1 event", len(records)-1)
		}

		header, row := records[0], records[1]
		column := func(name string) string {
			for i, h := range header {
				if h == name {
					return row[i]
				}
			}
			t.Fatalf("the report has no %q column: %v", name, header)
			return ""
		}
		if column("title") != "Admin Portal Fest" {
			t.Errorf("title = %q", column("title"))
		}
		if column("tickets_sold") != "2" || column("gross_kzt") != "10350.00" {
			t.Errorf("sold = %q, gross = %q; want 2 and 10350.00 (10000 plus the 3.5 percent fee)",
				column("tickets_sold"), column("gross_kzt"))
		}
		if column("organizer_email") != organizer.Email {
			t.Errorf("organizer_email = %q", column("organizer_email"))
		}

		t.Logf("criterion 1 OK: found %s, report has %d column(s) and %d event row(s)",
			organizer.Email, len(header), len(records)-1)
	})

	t.Run("criterion 2: forgotten password reset from the console token", func(t *testing.T) {
		var console bytes.Buffer

		pool := testutil.Pool(t)
		testutil.Reset(t, pool)

		cfg := testConfig(t)
		cfg.WebBaseURL = "http://localhost:3000"
		srv := NewWithSender(cfg, pool, email.NewConsoleSender(&console))
		ts := httptest.NewServer(srv.Handler())
		t.Cleanup(ts.Close)

		c := &client{t: t, server: ts, pool: pool, api: srv}
		account := c.register("p12forgot")

		// --- ask for a reset -------------------------------------------------
		requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
			map[string]any{"email": account.Email}), http.StatusAccepted)
		srv.Mailer().Wait()

		out := console.String()
		if !strings.Contains(out, "Subject: Reset your BiletFlow password") {
			t.Fatalf("no reset email on the console:\n%s", out)
		}

		// The token is read out of the console exactly as a person would.
		token := tokenFromConsole(t, out, "/reset-password?token=")
		if token == "" {
			t.Fatalf("no token in the console output:\n%s", out)
		}

		// --- use it ----------------------------------------------------------
		const newPassword = "a brand new passphrase"
		requireStatus(t, c.post("/api/v1/auth/password-reset", "",
			map[string]any{"token": token, "password": newPassword}), http.StatusOK)

		// The old password is dead, the new one works.
		requireStatus(t, c.post("/api/v1/auth/login", "",
			map[string]any{"email": account.Email, "password": account.Password}),
			http.StatusUnauthorized)

		signedIn := c.post("/api/v1/auth/login", "",
			map[string]any{"email": account.Email, "password": newPassword})
		requireStatus(t, signedIn, http.StatusOK)
		if signedIn.Body["access_token"] == nil {
			t.Error("the new password did not produce a session")
		}

		// A reset code is good exactly once.
		requireErrorCode(t, c.post("/api/v1/auth/password-reset", "",
			map[string]any{"token": token, "password": "yet another passphrase"}),
			http.StatusBadRequest, CodeTokenInvalid)

		t.Logf("criterion 2 OK: reset with the console token, old password refused")
	})

	t.Run("criterion 3: banner and refund policy reach the public page", func(t *testing.T) {
		c := newClient(t)
		organizer := c.register("p12image")

		// --- upload a banner --------------------------------------------------
		uploaded := c.uploadImage(organizer.Token, "banner.png", pngBytes())
		requireStatus(t, uploaded, http.StatusCreated)

		imageURL, _ := uploaded.Body["url"].(string)
		if imageURL == "" || !strings.HasSuffix(imageURL, ".png") {
			t.Fatalf("upload returned %q, want a .png URL", imageURL)
		}

		// The stored file is served back, so the <img> on the page resolves.
		served := c.getBinary(strings.TrimPrefix(imageURL, c.api.cfg.APIBaseURL), "")
		if served.Status != http.StatusOK {
			t.Fatalf("serving the banner returned %d", served.Status)
		}
		if !bytes.Equal(served.Body, pngBytes()) {
			t.Error("the served bytes are not the bytes uploaded")
		}

		// --- an event carrying both fields ------------------------------------
		const policy = "Full refunds up to 7 days before the event. No refunds after that."
		body := validEventBody("Illustrated Fest")
		body["cover_image_url"] = imageURL
		body["refund_policy"] = policy

		created := c.post("/api/v1/events", organizer.Token, body)
		requireStatus(t, created, http.StatusCreated)
		eventID := uuid.MustParse(created.eventString("id"))
		slug := created.eventString("slug")

		c.createTicketType(organizer.Token, eventID, ticketTypeBody("GA", "0", 10))
		requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/publish",
			organizer.Token, nil), http.StatusOK)

		// --- and the public page shows them -----------------------------------
		page := c.get("/api/v1/public/events/"+slug, "")
		requireStatus(t, page, http.StatusOK)

		event := page.Body["event"].(map[string]any)
		if event["cover_image_url"] != imageURL {
			t.Errorf("public cover_image_url = %v, want %v", event["cover_image_url"], imageURL)
		}
		if event["refund_policy"] != policy {
			t.Errorf("public refund_policy = %v", event["refund_policy"])
		}

		// Both survive an edit, which is where an Optional field usually breaks.
		const revised = "Refunds are handled case by case. Write to us."
		requireStatus(t, c.patch("/api/v1/events/"+eventID.String(), organizer.Token,
			map[string]any{"refund_policy": revised}), http.StatusOK)

		after := c.get("/api/v1/public/events/"+slug, "")
		afterEvent := after.Body["event"].(map[string]any)
		if afterEvent["refund_policy"] != revised {
			t.Errorf("after editing, refund_policy = %v", afterEvent["refund_policy"])
		}
		if afterEvent["cover_image_url"] != imageURL {
			t.Errorf("editing the policy dropped the banner: %v", afterEvent["cover_image_url"])
		}

		t.Logf("criterion 3 OK: banner served from %s and both fields on the public page",
			imageURL)
	})

	t.Run("criterion 4: manual attendee search and check-in", func(t *testing.T) {
		c := newClient(t)
		organizer := c.register("p12manual")
		eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Manual Door Fest", "5000", 20)

		requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Aigerim Serikova",
			"aigerim@biletflow.test"), http.StatusCreated)
		requireStatus(t, c.buy(eventID, ticketTypeID, 2, "Bekzat Sailau",
			"bekzat@biletflow.test"), http.StatusCreated)

		search := func(token, q string) response {
			return c.get("/api/v1/events/"+eventID.String()+"/attendees?q="+q, token)
		}

		// --- an attendee list is not public ----------------------------------
		requireStatus(t, search("", "Aigerim"), http.StatusUnauthorized)
		stranger := c.register("p12nosy")
		requireStatus(t, search(stranger.Token, "Aigerim"), http.StatusForbidden)

		// --- staff type a name ------------------------------------------------
		found := search(organizer.Token, "aigerim")
		requireStatus(t, found, http.StatusOK)

		attendees := found.Body["attendees"].([]any)
		if len(attendees) != 1 {
			t.Fatalf("matches for \"aigerim\" = %d, want 1", len(attendees))
		}

		match := attendees[0].(map[string]any)
		if match["attendee_name"] != "Aigerim Serikova" {
			t.Errorf("attendee_name = %v", match["attendee_name"])
		}
		if match["status"] != "valid" || match["admissible"] != true {
			t.Errorf("an unused ticket reads %v/%v, want valid and admissible",
				match["status"], match["admissible"])
		}
		// The search must not hand out an admission credential.
		if _, leaked := match["qr_token"]; leaked {
			t.Error("the attendee search exposed a QR token")
		}

		// --- and check them in, with no QR code -------------------------------
		ticketID := match["ticket_id"].(string)
		admitted := c.post("/api/v1/events/"+eventID.String()+"/check-in/manual",
			organizer.Token, map[string]any{"ticket_id": ticketID})
		requireStatus(t, admitted, http.StatusOK)

		if got := c.ticketStatus(ticketID); got != "checked_in" {
			t.Errorf("ticket status = %q after a manual check-in, want checked_in", got)
		}

		// It is a real check-in: the same duplicate protection applies.
		requireErrorCode(t, c.post("/api/v1/events/"+eventID.String()+"/check-in/manual",
			organizer.Token, map[string]any{"ticket_id": ticketID}),
			http.StatusConflict, CodeAlreadyCheckedIn)

		// And the row records how they got in.
		var device string
		if err := c.pool.QueryRow(t.Context(), `
			SELECT COALESCE(device_label, '') FROM check_in_records
			 WHERE ticket_id = $1 AND reversed_at IS NULL`, ticketID).Scan(&device); err != nil {
			t.Fatalf("read the check-in record: %v", err)
		}
		if device != "manual search" {
			t.Errorf("device_label = %q, want the manual marker", device)
		}

		// The list now shows them as checked in and no longer admissible.
		after := search(organizer.Token, "aigerim")
		updated := after.Body["attendees"].([]any)[0].(map[string]any)
		if updated["status"] != "checked_in" || updated["admissible"] != false {
			t.Errorf("after check-in the row reads %v/%v, want checked_in and not admissible",
				updated["status"], updated["admissible"])
		}

		t.Logf("criterion 4 OK: found by name, admitted without a QR, duplicate refused")
	})
}

// tokenFromConsole pulls a token out of a printed link, the way a person
// reading the console would.
func tokenFromConsole(t *testing.T, output, marker string) string {
	t.Helper()

	index := strings.Index(output, marker)
	if index < 0 {
		return ""
	}
	rest := output[index+len(marker):]
	if end := strings.IndexAny(rest, " \n\r\t"); end >= 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

// pngBytes is a real 256x256 PNG. It has to clear the uploader's minimum
// dimension check (200x200), so a 1x1 pixel will not do; the pattern is
// deterministic so repeated calls return byte-identical content, which the
// round-trip assertion relies on.
func pngBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// uploadImage posts a multipart image the way the browser does.
func (c *client) uploadImage(token, filename string, content []byte) response {
	c.t.Helper()

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile(uploadFormKey, filename)
	if err != nil {
		c.t.Fatalf("build the upload form: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		c.t.Fatalf("write the upload body: %v", err)
	}
	if err := form.Close(); err != nil {
		c.t.Fatalf("close the upload form: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost,
		c.server.URL+"/api/v1/uploads/images", &body)
	if err != nil {
		c.t.Fatalf("build the upload request: %v", err)
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.send(req)
}
