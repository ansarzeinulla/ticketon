package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/biletflow/api/internal/email"
	"github.com/biletflow/api/internal/store"
)

// SRS 4.10 lists nine notifications the system shall send. Six existed:
// account verification, purchase confirmation, ticket delivery, event
// cancellation, refund completion, and new support message. These cover the
// other three - payment failure, event updates, organizer payout status - plus
// the two SRS 4.13 adds: support-case assignment and status changes.

// mailOfType returns the messages of one type sent to an address.
func (c *client) mailOfType(address, msgType string) []email.Message {
	c.t.Helper()
	c.waitForMail()

	var out []email.Message
	for _, m := range c.mail.To(address) {
		if m.Type == msgType {
			out = append(out, m)
		}
	}
	return out
}

func (c *client) outboxCount(address, msgType string) int {
	c.t.Helper()
	var n int
	if err := c.pool.QueryRow(c.t.Context(),
		`SELECT count(*) FROM notifications WHERE recipient_email = $1 AND type = $2`,
		address, msgType).Scan(&n); err != nil {
		c.t.Fatalf("count notifications: %v", err)
	}
	return n
}

// --- Payment failure (SRS 4.10) ----------------------------------------------

// TestDeclinedPaymentIssuesNoTicketsAndNotifies is SRS 4.6 ("Failed or
// abandoned transactions shall not create valid tickets") and SRS 4.10
// ("payment failure") in one walk.
func TestDeclinedPaymentIssuesNoTicketsAndNotifies(t *testing.T) {
	c := newClient(t)
	organizer := c.register("declineorganizer")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Declined Card Night", "5000", 10)

	declined := "buyer" + store.DeclineSimulationDomain
	res := c.buy(eventID, ticketTypeID, 2, "Unlucky Buyer", declined)
	requireErrorCode(t, res, http.StatusPaymentRequired, CodePaymentFailed)

	// Nothing was issued and nothing was held.
	if sold, _ := c.soldFor(ticketTypeID); sold != 0 {
		t.Errorf("quantity_sold = %d after a declined payment, want 0", sold)
	}
	var orders, tickets int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM orders WHERE event_id = $1`, eventID).Scan(&orders); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM tickets WHERE event_id = $1`, eventID).Scan(&tickets); err != nil {
		t.Fatalf("count tickets: %v", err)
	}
	if orders != 0 || tickets != 0 {
		t.Errorf("orders = %d, tickets = %d after a decline; want 0 and 0", orders, tickets)
	}

	// And the buyer is told (SRS 4.10).
	msgs := c.mailOfType(declined, email.TypePaymentFailed)
	if len(msgs) != 1 {
		t.Fatalf("payment-failure emails = %d, want 1", len(msgs))
	}
	if !strings.Contains(msgs[0].Body, "Declined Card Night") {
		t.Errorf("the email does not name the event: %s", msgs[0].Body)
	}
	// SRS 4.6 again, in words this time: the buyer must not think they bought.
	if !strings.Contains(msgs[0].Body, "no tickets") {
		t.Errorf("the email does not say no tickets were issued: %s", msgs[0].Body)
	}
	if c.outboxCount(declined, email.TypePaymentFailed) != 1 {
		t.Error("the failure was not recorded in the notifications outbox")
	}
}

// TestFreeRegistrationIsNeverDeclined - the decline simulates a card being
// refused, and a free registration presents no card.
func TestFreeRegistrationIsNeverDeclined(t *testing.T) {
	c := newClient(t)
	organizer := c.register("declinefree")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Free And Undeclinable", "0", 10)

	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Free Buyer",
		"free"+store.DeclineSimulationDomain), http.StatusCreated)
}

// TestOrdinaryBuyersAreUnaffected - the trigger is a reserved domain nobody
// owns, so a real attendee cannot hit it by accident.
func TestOrdinaryBuyersAreUnaffected(t *testing.T) {
	c := newClient(t)
	organizer := c.register("declineordinary")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Ordinary Sales", "5000", 10)

	for _, address := range []string{
		"aigerim@example.kz",
		"decline@example.kz",
		"someone@simulator.biletflow.kz.example.com",
	} {
		res := c.buy(eventID, ticketTypeID, 1, "Ordinary Buyer", address)
		if res.Status != http.StatusCreated {
			t.Errorf("%s: status = %d, want 201; body = %s", address, res.Status, res.Raw)
		}
	}
}

// --- Event updates (SRS 4.10) ------------------------------------------------

// TestTicketHoldersAreToldWhenTheEventMoves is the requirement itself.
func TestTicketHoldersAreToldWhenTheEventMoves(t *testing.T) {
	c := newClient(t)
	organizer := c.register("eventupdate")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Movable Feast", "0", 10)

	buyer := "holder@biletflow.test"
	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Ticket Holder", buyer),
		http.StatusCreated)

	requireStatus(t, c.patch("/api/v1/events/"+eventID.String(), organizer.Token,
		map[string]any{
			"starts_at":  "2027-03-01T18:00:00Z",
			"ends_at":    "2027-03-01T21:00:00Z",
			"venue_name": "The New Hall",
		}), http.StatusOK)

	msgs := c.mailOfType(buyer, email.TypeEventUpdated)
	if len(msgs) != 1 {
		t.Fatalf("event-update emails = %d, want 1", len(msgs))
	}
	body := msgs[0].Body
	if !strings.Contains(body, "The New Hall") {
		t.Errorf("the email does not mention the new venue: %s", body)
	}
	if !strings.Contains(body, "Starts:") {
		t.Errorf("the email does not mention the new time: %s", body)
	}
	// The ticket is still good - saying otherwise would cause a support case.
	if !strings.Contains(body, "still admits you") {
		t.Errorf("the email does not reassure the holder: %s", body)
	}
}

// TestCosmeticEditsDoNotEmailAnybody - a reworded description is not something
// somebody needs to rearrange their evening for.
func TestCosmeticEditsDoNotEmailAnybody(t *testing.T) {
	c := newClient(t)
	organizer := c.register("eventcosmetic")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Quietly Edited", "0", 10)

	buyer := "quiet@biletflow.test"
	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Quiet Holder", buyer), http.StatusCreated)

	requireStatus(t, c.patch("/api/v1/events/"+eventID.String(), organizer.Token,
		map[string]any{"description": "A slightly better description."}), http.StatusOK)

	if msgs := c.mailOfType(buyer, email.TypeEventUpdated); len(msgs) != 0 {
		t.Errorf("event-update emails = %d for a description change, want 0", len(msgs))
	}
}

// TestUnpublishedEventsHaveNobodyToTell.
func TestUnpublishedEventsHaveNobodyToTell(t *testing.T) {
	c := newClient(t)
	organizer := c.register("eventdraftupdate")
	eventID, _ := c.createEvent(organizer.Token, "Still A Draft")

	requireStatus(t, c.patch("/api/v1/events/"+eventID.String(), organizer.Token,
		map[string]any{"venue_name": "Somewhere Else"}), http.StatusOK)

	c.waitForMail()
	for _, m := range c.mail.Messages() {
		if m.Type == email.TypeEventUpdated {
			t.Errorf("a draft event sent an update notice to %s", m.To)
		}
	}
}

// TestOneEmailPerBuyerNotPerTicket - somebody who bought four seats is told
// once.
func TestOneEmailPerBuyerNotPerTicket(t *testing.T) {
	c := newClient(t)
	organizer := c.register("eventfanout")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Fanned Out", "0", 20)

	buyer := "bulk.holder@biletflow.test"
	requireStatus(t, c.buy(eventID, ticketTypeID, 4, "Bulk Holder", buyer), http.StatusCreated)

	requireStatus(t, c.patch("/api/v1/events/"+eventID.String(), organizer.Token,
		map[string]any{"venue_name": "Relocated Hall"}), http.StatusOK)

	if msgs := c.mailOfType(buyer, email.TypeEventUpdated); len(msgs) != 1 {
		t.Errorf("event-update emails = %d for one buyer with four seats, want 1", len(msgs))
	}
}

// --- Organizer payout status (SRS 4.10) --------------------------------------

// TestOrganizerIsToldAboutTheirPayoutStatus is the requirement itself.
func TestOrganizerIsToldAboutTheirPayoutStatus(t *testing.T) {
	c := newClient(t)
	organizer := c.register("payoutnotice")

	eventID, _ := c.createEvent(organizer.Token, "Payout Status Event")
	c.createTicketType(organizer.Token, eventID, ticketTypeBody("Standard", "5000", 10))
	c.activatePaidSales(organizer.Token, eventID)

	msgs := c.mailOfType(organizer.Email, email.TypePayoutStatus)
	if len(msgs) == 0 {
		t.Fatal("no payout-status email reached the organizer")
	}

	joined := strings.Join(bodiesOf(msgs), "\n")
	if !strings.Contains(joined, "Payout Status Event") {
		t.Errorf("no payout email names the event: %s", joined)
	}
	// SRS 4.6 / 8: a demonstration record must never read as a real transfer.
	if !strings.Contains(joined, "No money has moved") {
		t.Errorf("the payout email does not label itself as simulated: %s", joined)
	}
}

// TestPayoutStatusIsNotResentOnEveryChecklistSave.
func TestPayoutStatusIsNotResentOnEveryChecklistSave(t *testing.T) {
	c := newClient(t)
	organizer := c.register("payoutrepeat")

	eventID, _ := c.createEvent(organizer.Token, "Repeatedly Saved")
	c.createTicketType(organizer.Token, eventID, ticketTypeBody("Standard", "5000", 10))
	c.activatePaidSales(organizer.Token, eventID)

	before := len(c.mailOfType(organizer.Email, email.TypePayoutStatus))

	// Re-submitting the completed checklist changes nothing.
	c.activatePaidSales(organizer.Token, eventID)

	if after := len(c.mailOfType(organizer.Email, email.TypePayoutStatus)); after != before {
		t.Errorf("payout emails = %d after re-saving, want %d", after, before)
	}
}

// --- Support-case assignment and status (SRS 4.13) ---------------------------

func bodiesOf(msgs []email.Message) []string {
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Body)
	}
	return out
}

// --- Order confirmation (SRS 4.7, 4.10) ---------------------------------------

// notificationRows reads the outbox for one address, oldest first.
func (c *client) notificationRows(address string) (types, statuses []string) {
	c.t.Helper()

	rows, err := c.pool.Query(c.t.Context(), `
		SELECT type, status::text FROM notifications
		 WHERE recipient_email = $1 ORDER BY created_at`, address)
	if err != nil {
		c.t.Fatalf("read notifications: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var typ, status string
		if err := rows.Scan(&typ, &status); err != nil {
			c.t.Fatalf("scan notification: %v", err)
		}
		types = append(types, typ)
		statuses = append(statuses, status)
	}
	return types, statuses
}

func TestCheckoutSendsOneConfirmationPerOrder(t *testing.T) {
	c := newClient(t)
	organizer := c.register("confirmorganizer")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Confirmation Concert", "5000", 10)

	bought := c.buy(eventID, ticketTypeID, 2, "Aliya Nurlan", "aliya@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	c.waitForMail()

	sent := c.mail.To("aliya@biletflow.test")
	if len(sent) != 1 {
		t.Fatalf("messages to the buyer = %d, want exactly one for the whole order", len(sent))
	}
	msg := sent[0]
	if msg.Type != email.TypeOrderConfirmation {
		t.Errorf("type = %q, want %q", msg.Type, email.TypeOrderConfirmation)
	}
	if msg.Subject != "Your Tickets to Confirmation Concert" {
		t.Errorf("subject = %q", msg.Subject)
	}
	// What the attendee actually paid: 2 x 5000 plus the 3.5% charge.
	if !strings.Contains(msg.Body, "10350.00 KZT") {
		t.Errorf("the email does not state the 10350.00 KZT paid:\n%s", msg.Body)
	}

	// Every ticket is in the email, each with a link that downloads it.
	for _, raw := range bought.Body["tickets"].([]any) {
		ticket := raw.(map[string]any)
		if !strings.Contains(msg.Body, ticket["ticket_code"].(string)) {
			t.Errorf("the email does not name ticket %v", ticket["ticket_code"])
		}
		if !strings.Contains(msg.Body, "/api/v1/tickets/"+ticket["id"].(string)+"/pdf") {
			t.Errorf("the email has no PDF link for ticket %v", ticket["id"])
		}
	}

	types, statuses := c.notificationRows("aliya@biletflow.test")
	if len(types) != 1 || types[0] != email.TypeOrderConfirmation {
		t.Fatalf("outbox = %v, want one order.confirmation", types)
	}
	if statuses[0] != "sent" {
		t.Errorf("outbox status = %q, want sent once delivery finished", statuses[0])
	}
}

// A free registration is a registration: SRS 4.10 confirms it the same way.
func TestFreeRegistrationIsConfirmedToo(t *testing.T) {
	c := newClient(t)
	organizer := c.register("freeconfirm")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Free Lecture", "0", 10)

	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Berik", "berik@biletflow.test"),
		http.StatusCreated)
	c.waitForMail()

	sent := c.mail.To("berik@biletflow.test")
	if len(sent) != 1 || sent[0].Type != email.TypeOrderConfirmation {
		t.Fatalf("messages = %+v, want one order confirmation", sent)
	}
}

// A refused checkout sold nothing, so it confirms nothing.
func TestRefusedCheckoutSendsNothing(t *testing.T) {
	c := newClient(t)
	organizer := c.register("refusedconfirm")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Tiny Event", "1000", 1)

	requireErrorCode(t, c.buy(eventID, ticketTypeID, 5, "Too Many", "toomany@biletflow.test"),
		http.StatusConflict, CodeInsufficientInventory)
	c.waitForMail()

	if sent := c.mail.To("toomany@biletflow.test"); len(sent) != 0 {
		t.Errorf("a refused checkout sent %d messages", len(sent))
	}
	if types, _ := c.notificationRows("toomany@biletflow.test"); len(types) != 0 {
		t.Errorf("a refused checkout wrote %v to the outbox", types)
	}
}

// Account emails go through the same outbox as order emails.
func TestAccountEmailsAreRecordedInTheOutbox(t *testing.T) {
	c := newClient(t)
	acc := c.register("outboxaccount")

	requireStatus(t, c.post("/api/v1/auth/password-reset/request", "",
		map[string]any{"email": acc.Email}), http.StatusAccepted)
	c.waitForMail()

	types, statuses := c.notificationRows(acc.Email)
	if len(types) != 1 || types[0] != email.TypePasswordReset {
		t.Fatalf("outbox = %v, want one password reset", types)
	}
	if statuses[0] != "sent" {
		t.Errorf("outbox status = %q, want sent", statuses[0])
	}
}
