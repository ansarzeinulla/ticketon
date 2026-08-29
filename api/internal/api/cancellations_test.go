package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/email"
)

// SRS 4.9: "Organizers shall be able to cancel free registrations."
//
// A free registration produces a zero-value order with a zero-value payment,
// so it can never travel the refund path: refunds_amount_chk requires a
// positive amount. Cancellation is its own verb, and these tests pin the whole
// behaviour rather than only the happy path.

// freeOrder registers an organizer's free event and books one place on it,
// returning the order id and the id of the single issued ticket.
func (c *client) freeOrder(token, title, buyerEmail string) (eventID, ticketTypeID uuid.UUID, orderID, ticketID string) {
	c.t.Helper()

	eventID, _, ticketTypeID = c.sellableEvent(token, title, "0", 10)
	bought := c.buy(eventID, ticketTypeID, 1, "Free Attendee", buyerEmail)
	requireStatus(c.t, bought, http.StatusCreated)

	orderID = orderIDOf(c.t, bought)
	ticketID = bought.Body["tickets"].([]any)[0].(map[string]any)["id"].(string)
	return eventID, ticketTypeID, orderID, ticketID
}

func (c *client) orderStatus(orderID string) string {
	c.t.Helper()
	var status string
	if err := c.pool.QueryRow(c.t.Context(),
		`SELECT status::text FROM orders WHERE id = $1`, uuid.MustParse(orderID)).
		Scan(&status); err != nil {
		c.t.Fatalf("read order status: %v", err)
	}
	return status
}

// TestFreeRegistrationCanBeCancelled is the requirement itself.
func TestFreeRegistrationCanBeCancelled(t *testing.T) {
	c := newClient(t)
	organizer := c.register("cancelowner")

	_, ticketTypeID, orderID, ticketID := c.freeOrder(
		organizer.Token, "Cancellable Free Event", "free.attendee@biletflow.test")

	if sold, _ := c.soldFor(ticketTypeID); sold != 1 {
		t.Fatalf("quantity_sold = %d before the cancellation, want 1", sold)
	}

	res := c.post("/api/v1/orders/"+orderID+"/cancel", organizer.Token,
		map[string]any{"reason": "the attendee asked to be taken off the list"})
	requireStatus(t, res, http.StatusOK)

	if got := res.Body["cancelled_tickets"]; got != float64(1) {
		t.Errorf("cancelled_tickets = %v, want 1; body = %s", got, res.Raw)
	}
	if got := c.orderStatus(orderID); got != "cancelled" {
		t.Errorf("order status = %q, want cancelled", got)
	}
	// SRS 4.9: "Refunded or cancelled tickets shall become invalid."
	if got := c.ticketStatus(ticketID); got != "cancelled" {
		t.Errorf("ticket status = %q, want cancelled", got)
	}
	// The place goes back on sale, exactly as a refund returns inventory.
	if sold, _ := c.soldFor(ticketTypeID); sold != 0 {
		t.Errorf("quantity_sold = %d after the cancellation, want 0", sold)
	}
}

// TestCancelledFreeTicketStopsScanning is the consequence that matters at the
// door: a cancelled registration must not admit anybody.
func TestCancelledFreeTicketStopsScanning(t *testing.T) {
	c := newClient(t)
	organizer := c.register("cancelgate")

	eventID, _, orderID, ticketID := c.freeOrder(
		organizer.Token, "Cancelled At The Gate", "gate.free@biletflow.test")

	var qrToken string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT qr_token FROM tickets WHERE id = $1`, uuid.MustParse(ticketID)).
		Scan(&qrToken); err != nil {
		t.Fatalf("read qr token: %v", err)
	}

	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/cancel", organizer.Token, nil),
		http.StatusOK)

	requireErrorCode(t, c.post("/api/v1/events/"+eventID.String()+"/check-in", organizer.Token,
		map[string]any{"qr_token": qrToken}), http.StatusConflict, CodeTicketNotValid)
}

// TestCancelIsOrganizerOnly mirrors the refund authorization rules: a
// cancellation destroys somebody's admission, so the buyer cannot do it either.
func TestCancelIsOrganizerOnly(t *testing.T) {
	c := newClient(t)
	organizer := c.register("cancelauthz")
	stranger := c.register("cancelstranger")

	_, _, orderID, ticketID := c.freeOrder(
		organizer.Token, "Owner Only Cancellations", "authz.free@biletflow.test")

	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/cancel", "", nil), http.StatusUnauthorized)
	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/cancel", stranger.Token, nil),
		http.StatusForbidden)

	if got := c.ticketStatus(ticketID); got != "valid" {
		t.Errorf("ticket status = %q after refused cancellations, want valid", got)
	}

	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/cancel", organizer.Token, nil),
		http.StatusOK)
}

// TestCancelIsIdempotentlyRefused stops a double click from voiding inventory
// twice.
func TestCancelIsIdempotentlyRefused(t *testing.T) {
	c := newClient(t)
	organizer := c.register("canceltwice")

	_, ticketTypeID, orderID, _ := c.freeOrder(
		organizer.Token, "Cancelled Twice", "twice.free@biletflow.test")

	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/cancel", organizer.Token, nil),
		http.StatusOK)
	requireErrorCode(t, c.post("/api/v1/orders/"+orderID+"/cancel", organizer.Token, nil),
		http.StatusConflict, CodeAlreadyCancelled)

	// The second attempt must not have driven the counter below zero or
	// released a second place.
	if sold, _ := c.soldFor(ticketTypeID); sold != 0 {
		t.Errorf("quantity_sold = %d after two cancellations, want 0", sold)
	}
}

// TestPaidOrderIsRefundedNotCancelled keeps the two verbs apart: money that was
// taken has to be given back, not quietly written off.
func TestPaidOrderIsRefundedNotCancelled(t *testing.T) {
	c := newClient(t)
	organizer := c.register("cancelpaid")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Paid Not Cancellable", "5000", 10)
	bought := c.buy(eventID, ticketTypeID, 1, "Paying Buyer", "paid.buyer@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	orderID := orderIDOf(t, bought)

	requireErrorCode(t, c.post("/api/v1/orders/"+orderID+"/cancel", organizer.Token, nil),
		http.StatusConflict, CodeUseRefund)

	// And the order is untouched by the refusal.
	if got := c.orderStatus(orderID); got != "paid" {
		t.Errorf("order status = %q after a refused cancellation, want paid", got)
	}
}

// TestFreeOrderIsRefusedByTheRefundEndpoint is the bug this work started from:
// a free order used to reach the refunds table and trip refunds_amount_chk,
// which surfaced as a 500. It must be a clean, actionable 409 instead.
func TestFreeOrderIsRefusedByTheRefundEndpoint(t *testing.T) {
	c := newClient(t)
	organizer := c.register("freerefund")

	_, _, orderID, _ := c.freeOrder(organizer.Token, "Free Cannot Be Refunded",
		"norefund.free@biletflow.test")

	res := c.post("/api/v1/orders/"+orderID+"/refund", organizer.Token, nil)
	requireErrorCode(t, res, http.StatusConflict, CodeNotRefundable)

	// The message has to send the organizer somewhere useful.
	msg, _ := res.Body["error"].(map[string]any)["message"].(string)
	if !strings.Contains(strings.ToLower(msg), "cancel") {
		t.Errorf("message = %q, want it to point at cancellation", msg)
	}
}

// TestOrderListFlagsCancellableSeparatelyFromRefundable stops the dashboard
// offering a button that is certain to fail.
func TestOrderListFlagsCancellableSeparatelyFromRefundable(t *testing.T) {
	c := newClient(t)
	organizer := c.register("cancelflags")

	freeEventID, _, freeOrderID, _ := c.freeOrder(
		organizer.Token, "Flagged Free Event", "flags.free@biletflow.test")

	orders := c.get("/api/v1/events/"+freeEventID.String()+"/orders", organizer.Token)
	requireStatus(t, orders, http.StatusOK)
	row := orders.Body["orders"].([]any)[0].(map[string]any)

	if row["refundable"] != false {
		t.Errorf("free order refundable = %v, want false", row["refundable"])
	}
	if row["cancellable"] != true {
		t.Errorf("free order cancellable = %v, want true", row["cancellable"])
	}

	// After cancelling, it is neither.
	requireStatus(t, c.post("/api/v1/orders/"+freeOrderID+"/cancel", organizer.Token, nil),
		http.StatusOK)
	orders = c.get("/api/v1/events/"+freeEventID.String()+"/orders", organizer.Token)
	row = orders.Body["orders"].([]any)[0].(map[string]any)
	if row["refundable"] != false || row["cancellable"] != false {
		t.Errorf("cancelled order flags = refundable %v / cancellable %v, want false / false",
			row["refundable"], row["cancellable"])
	}

	// A paid order is the mirror image.
	paidEventID, _, paidTypeID := c.sellableEvent(organizer.Token, "Flagged Paid Event", "5000", 10)
	requireStatus(t, c.buy(paidEventID, paidTypeID, 1, "Payer", "flags.paid@biletflow.test"),
		http.StatusCreated)
	paid := c.get("/api/v1/events/"+paidEventID.String()+"/orders", organizer.Token)
	paidRow := paid.Body["orders"].([]any)[0].(map[string]any)
	if paidRow["refundable"] != true || paidRow["cancellable"] != false {
		t.Errorf("paid order flags = refundable %v / cancellable %v, want true / false",
			paidRow["refundable"], paidRow["cancellable"])
	}
}

// TestCancellationIsAudited - SRS 4.9 requires the action in the audit log, and
// SRS 4.16 makes that log append-only, so the entry is permanent.
func TestCancellationIsAudited(t *testing.T) {
	c := newClient(t)
	organizer := c.register("cancelaudit")

	eventID, _, orderID, _ := c.freeOrder(organizer.Token, "Audited Cancellation",
		"audit.free@biletflow.test")

	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/cancel", organizer.Token,
		map[string]any{"reason": "duplicate registration"}), http.StatusOK)

	var description string
	if err := c.pool.QueryRow(t.Context(), `
		SELECT description FROM audit_logs
		 WHERE event_id = $1 AND action = 'order.cancelled'`, eventID).Scan(&description); err != nil {
		t.Fatalf("read audit entry: %v", err)
	}
	if !strings.Contains(description, "1 ticket") {
		t.Errorf("audit description = %q, want the ticket count in it", description)
	}

	// It shows up on the organizer's timeline (SRS 4.16).
	timeline := c.get("/api/v1/events/"+eventID.String()+"/timeline", organizer.Token)
	requireStatus(t, timeline, http.StatusOK)
	if !strings.Contains(timeline.Raw, "order.cancelled") {
		t.Errorf("timeline does not mention the cancellation: %s", timeline.Raw)
	}
}

// TestCancellationNotifiesTheAttendee - SRS 4.10 lists event/registration
// changes among the notifications the platform sends.
func TestCancellationNotifiesTheAttendee(t *testing.T) {
	c := newClient(t)
	organizer := c.register("cancelnotify")

	_, _, orderID, _ := c.freeOrder(organizer.Token, "Notified Cancellation",
		"notified.free@biletflow.test")

	requireStatus(t, c.post("/api/v1/orders/"+orderID+"/cancel", organizer.Token, nil),
		http.StatusOK)
	c.waitForMail()

	sent := c.mail.To("notified.free@biletflow.test")
	var msg *email.Message
	for i := range sent {
		if sent[i].Type == email.TypeRegistrationCancelled {
			msg = &sent[i]
		}
	}
	if msg == nil {
		t.Fatalf("no cancellation email reached the attendee; sent = %d message(s)", len(sent))
	}
	if !strings.Contains(msg.Body, "Notified Cancellation") {
		t.Errorf("email does not name the event: %s", msg.Body)
	}

	// And it is recorded in the outbox (SRS 4.10).
	var outbox int
	if err := c.pool.QueryRow(t.Context(), `
		SELECT count(*) FROM notifications
		 WHERE recipient_email = $1 AND type = $2`,
		"notified.free@biletflow.test", email.TypeRegistrationCancelled).Scan(&outbox); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if outbox != 1 {
		t.Errorf("notifications rows = %d, want 1", outbox)
	}
}
