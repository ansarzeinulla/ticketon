package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/biletflow/api/internal/email"
)

// SRS 4.10: "Registration or purchase confirmation" and "Ticket delivery" are
// notifications. Every one is written to the notifications table before it is
// handed to the sender, so the table is an outbox - a notification that was
// attempted is visible even when delivery then failed.

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
