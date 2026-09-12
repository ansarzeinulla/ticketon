package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/email"
)

// orderIDOf pulls the order id out of a checkout response.
func orderIDOf(t *testing.T, res response) string {
	t.Helper()
	order, ok := res.Body["order"].(map[string]any)
	if !ok {
		t.Fatalf("no order in response: %s", res.Raw)
	}
	return order["id"].(string)
}

func TestRefundIsOrganizerOnly(t *testing.T) {
	c := newClient(t)
	organizer := c.register("refundowner")
	stranger := c.register("refundstranger")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Owner Only Refunds", "5000", 10)
	bought := c.buy(eventID, ticketTypeID, 1, "Buyer", "buyer@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	orderID := orderIDOf(t, bought)

	// Anonymous.
	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/refund", "", nil), http.StatusUnauthorized)

	// Another organizer.
	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/refund", stranger.Token, nil),
		http.StatusForbidden)

	// The tickets are untouched by the refused attempts.
	if got := c.ticketStatus(bought.Body["tickets"].([]any)[0].(map[string]any)["id"].(string)); got != "valid" {
		t.Errorf("ticket status = %q after refused refunds, want valid", got)
	}

	// The owner may.
	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/refund", organizer.Token, nil), http.StatusOK)
}

func TestRefundIsIdempotentlyRefused(t *testing.T) {
	c := newClient(t)
	organizer := c.register("refundtwice")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Twice Refunded", "5000", 10)

	bought := c.buy(eventID, ticketTypeID, 1, "Buyer", "twice@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	orderID := orderIDOf(t, bought)

	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/refund", organizer.Token, nil), http.StatusOK)

	// The second attempt is refused rather than writing a second refund row.
	requireErrorCode(t, c.post("/api/v1/orders/"+orderID+"/refund", organizer.Token, nil),
		http.StatusConflict, CodeAlreadyRefunded)

	var refunds int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM refunds WHERE order_id = $1`, orderID).Scan(&refunds); err != nil {
		t.Fatalf("count refunds: %v", err)
	}
	if refunds != 1 {
		t.Errorf("refund rows = %d, want exactly 1", refunds)
	}
}

// TestRefundReturnsInventory is the behaviour that makes a refund useful to an
// organizer: the seat goes back on sale.
func TestRefundReturnsInventory(t *testing.T) {
	c := newClient(t)
	organizer := c.register("refundstock")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Inventory Returns", "5000", 3)

	bought := c.buy(eventID, ticketTypeID, 3, "Bulk Buyer", "bulk@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)

	if sold, total := c.soldFor(ticketTypeID); sold != 3 || total != 3 {
		t.Fatalf("sold %d/%d before the refund, want 3/3", sold, total)
	}

	// Sold out, so the next attendee is turned away.
	requireErrorCode(t, c.buy(eventID, ticketTypeID, 1, "Late", "late@biletflow.test"),
		http.StatusConflict, CodeInsufficientInventory)

	requireStatus(t, c.post("/api/v1/orders/"+orderIDOf(t, bought)+"/refund", organizer.Token, nil),
		http.StatusOK)

	if sold, _ := c.soldFor(ticketTypeID); sold != 0 {
		t.Errorf("quantity_sold = %d after refunding every ticket, want 0", sold)
	}

	var refundedCount int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT quantity_refunded FROM ticket_types WHERE id = $1`,
		ticketTypeID).Scan(&refundedCount); err != nil {
		t.Fatalf("read quantity_refunded: %v", err)
	}
	if refundedCount != 3 {
		t.Errorf("quantity_refunded = %d, want 3", refundedCount)
	}

	// And the seats really are back on sale.
	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Second Chance", "second@biletflow.test"),
		http.StatusCreated)
}

// TestRefundKeepsQuantitySoldEqualToLiveTickets pins the invariant the refund
// path relies on: quantity_sold is the number of tickets that still admit
// somebody, not the number ever issued.
func TestRefundKeepsQuantitySoldEqualToLiveTickets(t *testing.T) {
	c := newClient(t)
	organizer := c.register("refundinvariant")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Invariant Fest", "5000", 10)

	first := c.buy(eventID, ticketTypeID, 2, "First", "first@biletflow.test")
	requireStatus(t, first, http.StatusCreated)
	second := c.buy(eventID, ticketTypeID, 3, "Second", "second@biletflow.test")
	requireStatus(t, second, http.StatusCreated)

	requireStatus(t, c.post("/api/v1/orders/"+orderIDOf(t, first)+"/refund", organizer.Token, nil),
		http.StatusOK)

	var quantitySold, liveTickets int
	if err := c.pool.QueryRow(t.Context(), `
		SELECT tt.quantity_sold,
		       (SELECT count(*) FROM tickets t
		         WHERE t.ticket_type_id = tt.id AND t.status IN ('valid','checked_in'))
		  FROM ticket_types tt WHERE tt.id = $1`, ticketTypeID).Scan(&quantitySold, &liveTickets); err != nil {
		t.Fatalf("read the invariant: %v", err)
	}
	if quantitySold != liveTickets {
		t.Errorf("quantity_sold = %d but %d tickets are live", quantitySold, liveTickets)
	}
	if liveTickets != 3 {
		t.Errorf("live tickets = %d, want the 3 from the un-refunded order", liveTickets)
	}
}

// TestRefundShowsInAnalyticsAndTimeline ties Phase 10 back to Phase 9: the
// dashboard has to tell the truth after a refund, not just before one.
func TestRefundShowsInAnalyticsAndTimeline(t *testing.T) {
	c := newClient(t)
	organizer := c.register("refundanalytics")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Refund Analytics", "5000", 10)

	bought := c.buy(eventID, ticketTypeID, 2, "Buyer", "analytics@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	requireStatus(t, c.post("/api/v1/orders/"+orderIDOf(t, bought)+"/refund", organizer.Token, nil),
		http.StatusOK)

	analytics := c.get("/api/v1/events/"+eventID.String()+"/analytics", organizer.Token)
	requireStatus(t, analytics, http.StatusOK)
	figures := analytics.Body["analytics"].(map[string]any)

	// The tickets no longer count as sold - they admit nobody.
	if figures["tickets_sold"] != float64(0) {
		t.Errorf("tickets_sold = %v after refunding everything, want 0", figures["tickets_sold"])
	}
	// The revenue still counts: the sale happened, and a refund is a separate
	// event rather than a rewrite of history.
	if figures["gross_revenue_kzt"] != "10350.00" {
		t.Errorf("gross_revenue_kzt = %v, want 10350.00 (a refund does not unmake the sale)",
			figures["gross_revenue_kzt"])
	}

	timeline := c.get("/api/v1/events/"+eventID.String()+"/timeline", organizer.Token)
	requireStatus(t, timeline, http.StatusOK)

	found := false
	for _, entry := range timeline.Body["entries"].([]any) {
		if entry.(map[string]any)["action"] == "order.refunded" {
			found = true
		}
	}
	if !found {
		t.Errorf("the refund is missing from the event history: %s", timeline.Raw)
	}
}

func TestRefundNotifiesTheAttendee(t *testing.T) {
	c := newClient(t)
	organizer := c.register("refundmail")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Refund Mail Fest", "5000", 10)

	bought := c.buy(eventID, ticketTypeID, 1, "Dana Kim", "dana.refund@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)

	c.mail.Reset()
	requireStatus(t, c.post("/api/v1/orders/"+orderIDOf(t, bought)+"/refund", organizer.Token,
		map[string]any{"reason": "Event rescheduled"}), http.StatusOK)
	c.waitForMail()

	sent := c.mail.To("dana.refund@biletflow.test")
	if len(sent) != 1 {
		t.Fatalf("refund emails = %d, want 1", len(sent))
	}
	if sent[0].Type != email.TypeRefundCompleted {
		t.Errorf("type = %q, want %q", sent[0].Type, email.TypeRefundCompleted)
	}
	if sent[0].Subject != "Refund for Refund Mail Fest" {
		t.Errorf("subject = %q", sent[0].Subject)
	}
	for _, want := range []string{"Hi Dana,", "5175.00", "Event rescheduled", "no longer be admitted"} {
		if !contains(sent[0].Body, want) {
			t.Errorf("refund email is missing %q; body:\n%s", want, sent[0].Body)
		}
	}
}

func TestRefundRejectsUnpaidAndUnknownOrders(t *testing.T) {
	c := newClient(t)
	organizer := c.register("refundunknown")

	// An order id that does not exist.
	requireStatus(t, c.post("/api/v1/orders/"+uuid.NewString()+"/refund", organizer.Token, nil),
		http.StatusNotFound)

	// A malformed id.
	requireStatus(t, c.post("/api/v1/orders/not-a-uuid/refund", organizer.Token, nil),
		http.StatusBadRequest)
}

// TestOrganizerSeesTheirOrders covers the view the Refund button lives in.
func TestOrganizerSeesTheirOrders(t *testing.T) {
	c := newClient(t)
	organizer := c.register("orderlist")
	stranger := c.register("orderlistnosy")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Attendee List Fest", "5000", 10)

	requireStatus(t, c.buy(eventID, ticketTypeID, 2, "Aliya T", "aliya@biletflow.test"),
		http.StatusCreated)
	second := c.buy(eventID, ticketTypeID, 1, "Bekzat S", "bekzat@biletflow.test")
	requireStatus(t, second, http.StatusCreated)

	// Someone else's event is none of their business.
	requireStatus(t, c.get("/api/v1/events/"+eventID.String()+"/orders", stranger.Token),
		http.StatusForbidden)

	listed := c.get("/api/v1/events/"+eventID.String()+"/orders", organizer.Token)
	requireStatus(t, listed, http.StatusOK)

	orders := listed.Body["orders"].([]any)
	if len(orders) != 2 {
		t.Fatalf("orders = %d, want 2", len(orders))
	}
	// Newest first.
	first := orders[0].(map[string]any)
	if first["buyer_name"] != "Bekzat S" {
		t.Errorf("first row = %v, want the newest order", first["buyer_name"])
	}
	if first["refundable"] != true {
		t.Errorf("a paid order is not marked refundable: %v", first)
	}
	if first["ticket_count"] != float64(1) || first["live_tickets"] != float64(1) {
		t.Errorf("ticket counts = %v/%v, want 1/1", first["ticket_count"], first["live_tickets"])
	}

	// After a refund the row says so, and the button can be disabled.
	requireStatus(t, c.post("/api/v1/orders/"+orderIDOf(t, second)+"/refund", organizer.Token, nil),
		http.StatusOK)

	after := c.get("/api/v1/events/"+eventID.String()+"/orders", organizer.Token)
	refunded := after.Body["orders"].([]any)[0].(map[string]any)
	if refunded["status"] != "refunded" || refunded["refundable"] != false {
		t.Errorf("refunded row = %v, want status refunded and refundable false", refunded)
	}
	if refunded["live_tickets"] != float64(0) {
		t.Errorf("live_tickets = %v after the refund, want 0", refunded["live_tickets"])
	}
}
