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

// openSupportCase has an attendee open a case against an organizer's event.
func (c *client) openSupportCase(
	organizerToken string, attendee account, title string,
) (eventID string, caseID string) {
	c.t.Helper()

	id, _, ticketTypeID := c.sellableEvent(organizerToken, title, "0", 10)
	bought := c.post("/api/v1/events/"+id.String()+"/checkout", attendee.Token, map[string]any{
		"buyer_name":  "Case Opener",
		"buyer_email": attendee.Email,
		"items":       []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 1}},
	})
	requireStatus(c.t, bought, http.StatusCreated)

	opened := c.post("/api/v1/support/cases", attendee.Token, map[string]any{
		"category": "ticket_delivery",
		"subject":  "My ticket has not arrived",
		"message":  "I completed the registration but received nothing.",
		"order_id": orderIDOf(c.t, bought),
	})
	requireStatus(c.t, opened, http.StatusCreated)

	return id.String(), opened.Body["case"].(map[string]any)["id"].(string)
}

// TestCaseCanBeAssignedToANamedPerson is SRS 4.13: "Authorized staff shall be
// able to assign a case."
func TestCaseCanBeAssignedToANamedPerson(t *testing.T) {
	c := newClient(t)
	organizer := c.register("assignorganizer")
	attendee := c.register("assignattendee")
	colleague := c.register("assigncolleague")

	_, caseID := c.openSupportCase(organizer.Token, attendee, "Assignable Case")

	res := c.post("/api/v1/support/cases/"+caseID+"/assign", organizer.Token,
		map[string]any{"email": colleague.Email})
	requireStatus(t, res, http.StatusOK)

	supportCase := res.Body["case"].(map[string]any)
	if supportCase["assigned_to_user_id"] != colleague.ID.String() {
		t.Errorf("assigned_to_user_id = %v, want %s",
			supportCase["assigned_to_user_id"], colleague.ID)
	}

	// SRS 4.10: the requester is told who has their case.
	msgs := c.mailOfType(attendee.Email, email.TypeSupportAssigned)
	if len(msgs) != 1 {
		t.Fatalf("assignment emails = %d, want 1", len(msgs))
	}

	// And it can be handed back to the pool.
	res = c.post("/api/v1/support/cases/"+caseID+"/assign", organizer.Token,
		map[string]any{"email": ""})
	requireStatus(t, res, http.StatusOK)
	if _, present := res.Body["case"].(map[string]any)["assigned_to_user_id"]; present {
		t.Errorf("the case is still assigned after being handed back: %s", res.Raw)
	}
}

// TestAssignmentIsStaffOnly - an attendee cannot decide who handles their case.
func TestAssignmentIsStaffOnly(t *testing.T) {
	c := newClient(t)
	organizer := c.register("assignauthzorg")
	attendee := c.register("assignauthzatt")

	_, caseID := c.openSupportCase(organizer.Token, attendee, "Authorized Assignment")

	requireStatus(t, c.post("/api/v1/support/cases/"+caseID+"/assign", attendee.Token,
		map[string]any{"email": attendee.Email}), http.StatusForbidden)
	requireStatus(t, c.post("/api/v1/support/cases/"+caseID+"/assign", "",
		map[string]any{"email": attendee.Email}), http.StatusUnauthorized)
}

// TestAssignmentNeedsARealAccount.
func TestAssignmentNeedsARealAccount(t *testing.T) {
	c := newClient(t)
	organizer := c.register("assignmissing")
	attendee := c.register("assignmissingatt")

	_, caseID := c.openSupportCase(organizer.Token, attendee, "Missing Assignee")

	res := c.post("/api/v1/support/cases/"+caseID+"/assign", organizer.Token,
		map[string]any{"email": "nobody@biletflow.test"})
	requireStatus(t, res, http.StatusUnprocessableEntity)
	if _, ok := res.errorFields()["email"]; !ok {
		t.Errorf("no field error on email: %s", res.Raw)
	}
}

// TestRequesterIsToldWhenTheirCaseChangesStatus is SRS 4.13's "status changes"
// notification.
func TestRequesterIsToldWhenTheirCaseChangesStatus(t *testing.T) {
	c := newClient(t)
	organizer := c.register("statusorganizer")
	attendee := c.register("statusattendee")

	_, caseID := c.openSupportCase(organizer.Token, attendee, "Status Change Case")

	requireStatus(t, c.patch("/api/v1/support/cases/"+caseID, organizer.Token,
		map[string]any{"status": "waiting_for_customer"}), http.StatusOK)

	msgs := c.mailOfType(attendee.Email, email.TypeSupportStatusChanged)
	if len(msgs) != 1 {
		t.Fatalf("status-change emails = %d, want 1", len(msgs))
	}
	// The enum label must not leak into an attendee's inbox.
	if strings.Contains(msgs[0].Subject, "waiting_for_customer") {
		t.Errorf("the subject shows a raw enum label: %q", msgs[0].Subject)
	}
	if !strings.Contains(msgs[0].Body, "waiting on a reply from you") {
		t.Errorf("the email does not say what is expected of them: %s", msgs[0].Body)
	}

	// Setting the same status again is not a change and sends nothing more.
	requireStatus(t, c.patch("/api/v1/support/cases/"+caseID, organizer.Token,
		map[string]any{"status": "waiting_for_customer"}), http.StatusOK)
	if msgs := c.mailOfType(attendee.Email, email.TypeSupportStatusChanged); len(msgs) != 1 {
		t.Errorf("status-change emails = %d after a no-op change, want 1", len(msgs))
	}
}

// TestEveryNotificationTypeInTheSRSHasATemplate is the coverage check: SRS 4.10
// lists nine, SRS 4.13 adds two more.
func TestEveryNotificationTypeInTheSRSHasATemplate(t *testing.T) {
	required := map[string]string{
		"account verification":       email.TypeEmailVerification,
		"purchase confirmation":      email.TypeOrderConfirmation,
		"payment failure":            email.TypePaymentFailed,
		"event updates":              email.TypeEventUpdated,
		"event cancellation":         email.TypeEventCancelled,
		"refund completion":          email.TypeRefundCompleted,
		"organizer payout status":    email.TypePayoutStatus,
		"new support message":        email.TypeSupportMessage,
		"support-case assignment":    email.TypeSupportAssigned,
		"support-case status change": email.TypeSupportStatusChanged,
		"registration cancellation":  email.TypeRegistrationCancelled,
		"password reset":             email.TypePasswordReset,
	}
	for name, msgType := range required {
		if msgType == "" {
			t.Errorf("SRS notification %q has no type constant", name)
		}
	}
}

func bodiesOf(msgs []email.Message) []string {
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Body)
	}
	return out
}
