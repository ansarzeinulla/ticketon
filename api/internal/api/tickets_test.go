package api

import (
	"bytes"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/ticketpdf"
)

// assertQRImageEncodes proves that a served PNG is the QR for `want`.
//
// By byte comparison against the encoder, not by decoding. gozxing is
// deterministic for a given image but fails to locate roughly 1-3% of perfect
// codes, so decoding freshly generated random tokens made this suite fail about
// one run in fifteen for no product reason.
//
// The two halves of the guarantee are proven separately and exactly:
//   - that the encoder round-trips a string is asserted in package ticketpdf,
//     against pinned inputs, with a real QR reader;
//   - that the API serves exactly that encoder's output for the stored token is
//     asserted here.
//
// Together they say the same thing the flaky decode said, and they say it every
// time.
func assertQRImageEncodes(t *testing.T, served []byte, want string) {
	t.Helper()

	expected, err := ticketpdf.QRPNG(want)
	if err != nil {
		t.Fatalf("encode the expected QR: %v", err)
	}

	if !bytes.Equal(served, expected) {
		t.Fatalf("the served QR image is not the code for %q (%d bytes served, %d expected)",
			want, len(served), len(expected))
	}
}

// tokenShape is the exact format the Phase 5 criteria call for.
var tokenShape = regexp.MustCompile(
	`^TKT_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// buyOneTicket runs a checkout for a single ticket and returns its id.
func (c *client) buyOneTicket(token, title string) (ticketID string, qrToken string) {
	c.t.Helper()

	eventID, _, ticketTypeID := c.sellableEvent(token, title, "5000", 5)

	res := c.buy(eventID, ticketTypeID, 1, "Nurlan Amanov", "nurlan@biletflow.test")
	requireStatus(c.t, res, http.StatusCreated)

	tickets, _ := res.Body["tickets"].([]any)
	if len(tickets) != 1 {
		c.t.Fatalf("%d tickets issued, want 1", len(tickets))
	}

	ticket := tickets[0].(map[string]any)
	return ticket["id"].(string), ticket["qr_token"].(string)
}

// TestPhase5SuccessCriteria walks the Phase 5 acceptance criteria: buy one
// ticket, fetch its PDF, and confirm what a scanner reads off the QR.
func TestPhase5SuccessCriteria(t *testing.T) {
	c := newClient(t)
	organizer := c.register("phase5organizer")

	// --- 1. A simulated checkout for exactly 1 ticket -----------------------
	ticketID, qrToken := c.buyOneTicket(organizer.Token, "Phase 5 Concert")
	t.Logf("criterion 1 OK: one ticket issued, id %s", ticketID)

	// --- 2 & 3. The PDF downloads and is a well-formed A4 document ----------
	pdf := c.getBinary("/api/v1/tickets/"+ticketID+"/pdf", "")
	if pdf.Status != http.StatusOK {
		t.Fatalf("criterion 2: status = %d, want 200; body = %s", pdf.Status, pdf.Body)
	}

	if ct := pdf.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("criterion 3: Content-Type = %q, want application/pdf", ct)
	}
	disposition := pdf.Header.Get("Content-Disposition")
	if !strings.HasPrefix(disposition, "attachment;") {
		t.Errorf("criterion 2: Content-Disposition = %q, want an attachment so the browser downloads it",
			disposition)
	}
	if !strings.Contains(disposition, ".pdf") {
		t.Errorf("criterion 2: Content-Disposition = %q, want a .pdf filename", disposition)
	}

	if !bytes.HasPrefix(pdf.Body, []byte("%PDF-")) {
		t.Fatalf("criterion 3: the body is not a PDF: %q", pdf.Body[:min(16, len(pdf.Body))])
	}
	if !bytes.Contains(pdf.Body, []byte("%%EOF")) {
		t.Error("criterion 3: the PDF has no EOF trailer, so a viewer would call it corrupt")
	}
	// A4 in PostScript points.
	if !bytes.Contains(pdf.Body, []byte("595.28")) || !bytes.Contains(pdf.Body, []byte("841.89")) {
		t.Error("criterion 3: the page is not A4")
	}
	t.Logf("criterion 3 OK: %d-byte A4 PDF served as an attachment", len(pdf.Body))

	// --- 4. It shows the event, attendee and venue --------------------------
	// The PDF's own text is asserted by the ticketpdf package's tests, which can
	// inflate its content streams. Here we confirm the served document is built
	// from this ticket, by checking the token printed on it as the scan
	// fallback and the ticket code in the filename.
	var (
		ticketCode string
		eventTitle string
		attendee   string
		venue      string
	)
	err := c.pool.QueryRow(t.Context(), `
		SELECT t.ticket_code, e.title, COALESCE(a.full_name, o.buyer_name), COALESCE(e.venue_name, '')
		  FROM tickets t
		  JOIN events e ON e.id = t.event_id
		  JOIN orders o ON o.id = t.order_id
		  LEFT JOIN attendees a ON a.id = t.attendee_id
		 WHERE t.id = $1`, uuid.MustParse(ticketID)).
		Scan(&ticketCode, &eventTitle, &attendee, &venue)
	if err != nil {
		t.Fatalf("criterion 4: read ticket: %v", err)
	}

	if !strings.Contains(disposition, safeFilename(ticketCode)) {
		t.Errorf("criterion 4: filename %q does not name ticket %q", disposition, ticketCode)
	}
	if eventTitle != "Phase 5 Concert" || attendee != "Nurlan Amanov" {
		t.Fatalf("criterion 4: ticket belongs to (%q, %q), not the purchase under test",
			eventTitle, attendee)
	}
	t.Logf("criterion 4 OK: PDF for %q, attendee %q, venue %q", eventTitle, attendee, venue)

	// --- 5. A scanner reads exactly TKT_<uuid> ------------------------------
	qr := c.getBinary("/api/v1/tickets/"+ticketID+"/qr.png", "")
	if qr.Status != http.StatusOK {
		t.Fatalf("criterion 5: QR status = %d, want 200", qr.Status)
	}
	if ct := qr.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("criterion 5: Content-Type = %q, want image/png", ct)
	}

	// What the database actually holds for this ticket.
	var stored string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT qr_token FROM tickets WHERE id = $1`, uuid.MustParse(ticketID)).Scan(&stored); err != nil {
		t.Fatalf("criterion 5: read token: %v", err)
	}
	if stored != qrToken {
		t.Fatalf("criterion 5: the API returned %q but the database stores %q", qrToken, stored)
	}
	if !tokenShape.MatchString(stored) {
		t.Fatalf("criterion 5: token is %q, want the exact form TKT_<uuid>", stored)
	}

	// The served image is the QR for exactly that token.
	assertQRImageEncodes(t, qr.Body, stored)
	t.Logf("criterion 5 OK: the QR encodes %s", stored)
}

func TestCheckoutIssuesUUIDTokens(t *testing.T) {
	c := newClient(t)
	owner := c.register("uuidtokens")
	eventID, _, ticketTypeID := c.sellableEvent(owner.Token, "Token Shape Event", "1000", 10)

	res := c.buy(eventID, ticketTypeID, 3, "Token Buyer", "tokens@biletflow.test")
	requireStatus(t, res, http.StatusCreated)

	tickets, _ := res.Body["tickets"].([]any)
	seen := map[string]bool{}

	for _, raw := range tickets {
		token := raw.(map[string]any)["qr_token"].(string)

		if !tokenShape.MatchString(token) {
			t.Errorf("qr_token = %q, want TKT_<uuid>", token)
		}
		if seen[token] {
			t.Errorf("duplicate qr_token %q across tickets in one order", token)
		}
		seen[token] = true

		// The token must not simply repeat the ticket id: an id travels in URLs
		// and logs, and must not double as an admission credential.
		if strings.TrimPrefix(token, "TKT_") == raw.(map[string]any)["id"].(string) {
			t.Error("the admission token is just the ticket id")
		}
	}
}

func TestTicketPDFAndQRNotFound(t *testing.T) {
	c := newClient(t)

	for _, path := range []string{"/pdf", "/qr.png", ""} {
		unknown := c.getBinary("/api/v1/tickets/"+uuid.NewString()+path, "")
		if unknown.Status != http.StatusNotFound {
			t.Errorf("GET %s for an unknown ticket = %d, want 404", path, unknown.Status)
		}

		malformed := c.getBinary("/api/v1/tickets/not-a-uuid"+path, "")
		if malformed.Status != http.StatusBadRequest {
			t.Errorf("GET %s with a malformed id = %d, want 400", path, malformed.Status)
		}
	}
}

func TestTicketPDFIsNotCached(t *testing.T) {
	c := newClient(t)
	owner := c.register("pdfcache")
	ticketID, _ := c.buyOneTicket(owner.Token, "Cache Event")

	res := c.getBinary("/api/v1/tickets/"+ticketID+"/pdf", "")
	requireBinaryStatus(t, res, http.StatusOK)

	// A ticket can be cancelled or refunded, so a stale cached copy is unsafe.
	cache := res.Header.Get("Cache-Control")
	if !strings.Contains(cache, "no-store") {
		t.Errorf("Cache-Control = %q, want no-store", cache)
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("the PDF response should be served with X-Content-Type-Options: nosniff")
	}
}

func TestGetTicketReturnsDeliveryLinks(t *testing.T) {
	c := newClient(t)
	owner := c.register("ticketlinks")
	ticketID, qrToken := c.buyOneTicket(owner.Token, "Links Event")

	res := c.get("/api/v1/tickets/"+ticketID, "")
	requireStatus(t, res, http.StatusOK)

	ticket, _ := res.Body["ticket"].(map[string]any)
	if ticket["qr_token"] != qrToken {
		t.Errorf("qr_token = %v, want %q", ticket["qr_token"], qrToken)
	}
	if ticket["status"] != "valid" {
		t.Errorf("status = %v, want valid", ticket["status"])
	}
	if ticket["pdf_url"] != "/api/v1/tickets/"+ticketID+"/pdf" {
		t.Errorf("pdf_url = %v", ticket["pdf_url"])
	}
	if ticket["qr_url"] != "/api/v1/tickets/"+ticketID+"/qr.png" {
		t.Errorf("qr_url = %v", ticket["qr_url"])
	}
	if ticket["attendee_name"] != "Nurlan Amanov" {
		t.Errorf("attendee_name = %v, want Nurlan Amanov", ticket["attendee_name"])
	}
}

// Every ticket in a multi-ticket order gets its own distinct PDF.
func TestEachTicketHasItsOwnPDF(t *testing.T) {
	c := newClient(t)
	owner := c.register("multipdf")
	eventID, _, ticketTypeID := c.sellableEvent(owner.Token, "Multi PDF Event", "1000", 10)

	res := c.buy(eventID, ticketTypeID, 2, "Two Tickets", "two@biletflow.test")
	requireStatus(t, res, http.StatusCreated)

	tickets, _ := res.Body["tickets"].([]any)
	if len(tickets) != 2 {
		t.Fatalf("%d tickets, want 2", len(tickets))
	}

	scanned := map[string]bool{}
	for _, raw := range tickets {
		id := raw.(map[string]any)["id"].(string)

		pdf := c.getBinary("/api/v1/tickets/"+id+"/pdf", "")
		requireBinaryStatus(t, pdf, http.StatusOK)
		if !bytes.HasPrefix(pdf.Body, []byte("%PDF-")) {
			t.Fatalf("ticket %s did not produce a PDF", id)
		}

		token := raw.(map[string]any)["qr_token"].(string)

		qr := c.getBinary("/api/v1/tickets/"+id+"/qr.png", "")
		requireBinaryStatus(t, qr, http.StatusOK)
		assertQRImageEncodes(t, qr.Body, token)

		if scanned[token] {
			t.Errorf("two tickets in one order carry the same token %q", token)
		}
		scanned[token] = true
	}
}

func requireBinaryStatus(t *testing.T, res binaryResponse, want int) {
	t.Helper()
	if res.Status != want {
		t.Fatalf("status = %d, want %d; body = %s", res.Status, want, res.Body)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
