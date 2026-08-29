package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/email"
	"github.com/biletflow/api/internal/testutil"
)

// TestPhase10SuccessCriteria walks the three criteria for the phase end to
// end, over real HTTP against the real database.
//
//  1. an organizer refunds a past order and the tickets immediately become
//     invalid - the Phase 6 scanner will not admit them;
//  2. a paid ticket type cannot be sold until the organizer has clicked
//     through the activation checklist;
//  3. a completed checkout logs a formatted mock email to the console.
func TestPhase10SuccessCriteria(t *testing.T) {
	t.Run("criterion 1: refund invalidates the tickets", func(t *testing.T) {
		c := newClient(t)
		organizer := c.register("p10refund")
		eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Refund Criterion Fest", "5000", 10)

		bought := c.buy(eventID, ticketTypeID, 2, "Aliya Refundova", "aliya.refund@biletflow.test")
		requireStatus(t, bought, http.StatusCreated)

		orderID := bought.Body["order"].(map[string]any)["id"].(string)
		tickets := bought.Body["tickets"].([]any)
		if len(tickets) != 2 {
			t.Fatalf("expected 2 tickets, got %d", len(tickets))
		}
		firstQR := tickets[0].(map[string]any)["qr_token"].(string)
		firstID := tickets[0].(map[string]any)["id"].(string)

		// The ticket works before the refund, which is what makes the "after"
		// meaningful rather than a ticket that never worked.
		requireStatus(t, c.scan(organizer.Token, eventID, firstQR), http.StatusOK)

		// The organizer refunds from their attendee view.
		refund := c.post("/api/v1/orders/"+orderID+"/refund", organizer.Token,
			map[string]any{"reason": "Attendee could not travel"})
		requireStatus(t, refund, http.StatusOK)

		if voided := refund.Body["voided_tickets"]; voided != float64(2) {
			t.Errorf("voided_tickets = %v, want 2", voided)
		}
		if status := refund.Body["order"].(map[string]any)["status"]; status != "refunded" {
			t.Errorf("order status = %v, want refunded", status)
		}

		// --- the tickets are invalid, in the database ------------------------
		var valid, refunded int
		if err := c.pool.QueryRow(t.Context(), `
			SELECT count(*) FILTER (WHERE status IN ('valid','checked_in')),
			       count(*) FILTER (WHERE status = 'refunded')
			  FROM tickets WHERE order_id = $1`, orderID).Scan(&valid, &refunded); err != nil {
			t.Fatalf("count tickets: %v", err)
		}
		if valid != 0 || refunded != 2 {
			t.Errorf("tickets: %d still valid, %d refunded; want 0 and 2", valid, refunded)
		}

		// --- and the Phase 6 scanner refuses them ----------------------------
		second := c.scan(organizer.Token, eventID, firstQR)
		if second.Status == http.StatusOK {
			t.Fatalf("a refunded ticket was admitted: %s", second.Raw)
		}
		if second.Status != http.StatusConflict {
			t.Errorf("scan status = %d, want 409; body = %s", second.Status, second.Raw)
		}

		// The money and the paperwork moved too.
		var (
			orderStatus, refundedKZT, paymentStatus, refundStatus string
			auditEntries                                          int
		)
		if err := c.pool.QueryRow(t.Context(), `
			SELECT o.status::text, o.refunded_kzt::text, p.status::text, r.status::text,
			       (SELECT count(*) FROM audit_logs
			         WHERE entity_id = o.id::text AND action = 'order.refunded')
			  FROM orders o
			  JOIN payments p ON p.order_id = o.id
			  JOIN refunds  r ON r.order_id = o.id
			 WHERE o.id = $1`, orderID,
		).Scan(&orderStatus, &refundedKZT, &paymentStatus, &refundStatus, &auditEntries); err != nil {
			t.Fatalf("read refund rows: %v", err)
		}
		if orderStatus != "refunded" || paymentStatus != "refunded" || refundStatus != "succeeded" {
			t.Errorf("order=%s payment=%s refund=%s; want refunded/refunded/succeeded",
				orderStatus, paymentStatus, refundStatus)
		}
		if refundedKZT != "10000.00" {
			t.Errorf("refunded_kzt = %s, want 10000.00", refundedKZT)
		}
		if auditEntries != 1 {
			t.Errorf("audit entries for the refund = %d, want 1", auditEntries)
		}

		t.Logf("criterion 1 OK: refunded %s KZT, 2 tickets voided, scanner refused ticket %s",
			refundedKZT, firstID)
	})

	t.Run("criterion 2: paid sales need the activation checklist", func(t *testing.T) {
		c := newClient(t)
		organizer := c.register("p10activate")
		eventID, slug := func() (uuid.UUID, string) {
			id, created := c.createEvent(organizer.Token, "Activation Criterion Fest")
			return id, created.eventString("slug")
		}()

		ticketTypeID, _ := c.createTicketType(organizer.Token, eventID,
			ticketTypeBody("General Admission", "5000", 10))
		requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/publish",
			organizer.Token, nil), http.StatusOK)

		// --- before the checklist, the public cannot buy ---------------------
		blocked := c.buy(eventID, ticketTypeID, 1, "Too Early", "early@biletflow.test")
		requireErrorCode(t, blocked, http.StatusForbidden, CodePaidSalesNotActive)

		var ticketsIssued int
		if err := c.pool.QueryRow(t.Context(),
			`SELECT count(*) FROM tickets WHERE event_id = $1`, eventID).Scan(&ticketsIssued); err != nil {
			t.Fatalf("count tickets: %v", err)
		}
		if ticketsIssued != 0 {
			t.Fatalf("a blocked purchase still issued %d ticket(s)", ticketsIssued)
		}

		// The public page says so rather than offering a form that will fail.
		page := c.get("/api/v1/public/events/"+slug, "")
		requireStatus(t, page, http.StatusOK)
		if page.Body["paid_sales_active"] != false || page.Body["paid_sales_required"] != true {
			t.Errorf("public page: paid_sales_active=%v paid_sales_required=%v; want false/true",
				page.Body["paid_sales_active"], page.Body["paid_sales_required"])
		}
		if page.Body["on_sale"] != false {
			t.Errorf("public page on_sale = %v, want false", page.Body["on_sale"])
		}

		// --- the organizer sees the checklist --------------------------------
		state := c.get("/api/v1/events/"+eventID.String()+"/activation", organizer.Token)
		requireStatus(t, state, http.StatusOK)
		activation := state.Body["activation"].(map[string]any)
		if activation["status"] != "not_started" {
			t.Errorf("status = %v, want not_started", activation["status"])
		}
		if outstanding := activation["outstanding"].([]any); len(outstanding) != 4 {
			t.Errorf("outstanding steps = %v, want all four", outstanding)
		}

		// --- a partial checklist is not enough -------------------------------
		partial := c.post("/api/v1/events/"+eventID.String()+"/activation", organizer.Token,
			map[string]any{"accept_terms": true, "confirm_identity": true})
		requireStatus(t, partial, http.StatusOK)
		if active := partial.Body["activation"].(map[string]any)["is_active"]; active != false {
			t.Errorf("half a checklist activated the event: %s", partial.Raw)
		}

		stillBlocked := c.buy(eventID, ticketTypeID, 1, "Still Early", "still@biletflow.test")
		requireErrorCode(t, stillBlocked, http.StatusForbidden, CodePaidSalesNotActive)

		// --- completing it opens sales ---------------------------------------
		done := c.post("/api/v1/events/"+eventID.String()+"/activation", organizer.Token,
			map[string]any{"confirm_payout": true, "pay_activation_fee": true})
		requireStatus(t, done, http.StatusOK)

		final := done.Body["activation"].(map[string]any)
		if final["is_active"] != true || final["status"] != "active" {
			t.Fatalf("checklist complete but activation is %v: %s", final["status"], done.Raw)
		}
		if outstanding := final["outstanding"].([]any); len(outstanding) != 0 {
			t.Errorf("outstanding = %v, want none", outstanding)
		}

		// The fee was recorded, and recorded as simulated (SRS 4.5, 4.6).
		var feeKZT string
		var simulated bool
		if err := c.pool.QueryRow(t.Context(), `
			SELECT p.amount_kzt::text, p.is_simulated
			  FROM payments p
			  JOIN paid_sales_activations a ON a.activation_payment_id = p.id
			 WHERE a.event_id = $1`, eventID).Scan(&feeKZT, &simulated); err != nil {
			t.Fatalf("read the activation fee: %v", err)
		}
		if feeKZT != "5000.00" || !simulated {
			t.Errorf("activation fee = %s (simulated=%v), want 5000.00 simulated", feeKZT, simulated)
		}

		// --- and now the same purchase succeeds ------------------------------
		allowed := c.buy(eventID, ticketTypeID, 1, "Right On Time", "ontime@biletflow.test")
		requireStatus(t, allowed, http.StatusCreated)

		t.Logf("criterion 2 OK: purchase refused until all four steps were done, then accepted")
	})

	t.Run("criterion 3: checkout logs a formatted mock email", func(t *testing.T) {
		// This uses the real ConsoleSender - the same type main() wires - with
		// its writer pointed at a buffer. What is asserted here is exactly what
		// would appear on stdout.
		var console bytes.Buffer

		pool := testutil.Pool(t)
		testutil.Reset(t, pool)

		cfg := testConfig(t)
		cfg.APIBaseURL = "http://localhost:8080"
		cfg.WebBaseURL = "http://localhost:3000"

		srv := NewWithSender(cfg, pool, email.NewConsoleSender(&console))
		ts := httptest.NewServer(srv.Handler())
		t.Cleanup(ts.Close)

		c := &client{t: t, server: ts, pool: pool, api: srv}

		organizer := c.register("p10email")
		eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Console Email Fest", "5000", 10)

		bought := c.buy(eventID, ticketTypeID, 2, "Bekzat Sailau", "bekzat@biletflow.test")
		requireStatus(t, bought, http.StatusCreated)

		// Delivery is asynchronous by design, so the test says where the
		// asynchrony ends instead of sleeping.
		srv.Mailer().Wait()

		out := console.String()
		if out == "" {
			t.Fatal("nothing was written to the console")
		}

		orderNumber := bought.Body["order"].(map[string]any)["order_number"].(string)
		firstTicketID := bought.Body["tickets"].([]any)[0].(map[string]any)["id"].(string)

		for _, want := range []string{
			"MOCK EMAIL",                     // labelled as simulated
			"To:      bekzat@biletflow.test", // the attendee
			"Subject: Your Tickets to Console Email Fest",
			"Hi Bekzat,",
			orderNumber,
			"10000.00", // what they paid
			"http://localhost:8080/api/v1/tickets/" + firstTicketID + "/pdf",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("console output is missing %q", want)
			}
		}

		// The outbox row proves the send is recorded, not only printed.
		orderID := uuid.MustParse(bought.Body["order"].(map[string]any)["id"].(string))
		notes, err := srv.notifications.ListForOrder(t.Context(), orderID)
		if err != nil {
			t.Fatalf("read notifications: %v", err)
		}
		if len(notes) != 1 {
			t.Fatalf("notifications recorded = %d, want 1", len(notes))
		}
		if notes[0].Type != email.TypeOrderConfirmation || notes[0].Status != "sent" {
			t.Errorf("notification = %s/%s, want %s/sent",
				notes[0].Type, notes[0].Status, email.TypeOrderConfirmation)
		}

		t.Logf("criterion 3 OK: console email to %s for order %s, %d line(s)",
			notes[0].RecipientEmail, orderNumber, strings.Count(out, "\n"))
	})
}
