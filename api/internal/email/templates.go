package email

import (
	"fmt"
	"strings"
)

// Notification types, mirrored into notifications.type.
const (
	TypeOrderConfirmation = "order.confirmation"
	TypeRefundCompleted   = "refund.completed"
	// TypeRegistrationCancelled is the free-registration counterpart of
	// TypeRefundCompleted: no money moves, but the ticket stops working, and
	// the attendee has to be told before they travel to the venue (SRS 4.9).
	TypeRegistrationCancelled = "registration.cancelled"
)

// TicketLine is one ticket on a confirmation, with the link that downloads it.
type TicketLine struct {
	TicketCode string
	TypeName   string
	PDFURL     string
}

// OrderDetails is everything the templates need. It is a plain struct rather
// than a store type so the templates can be unit-tested without a database,
// and so a change to the store's shape cannot silently reword an email.
type OrderDetails struct {
	BuyerName   string
	BuyerEmail  string
	OrderNumber string
	EventTitle  string
	EventVenue  string
	EventWhen   string
	TotalKZT    string
	Tickets     []TicketLine
	// OrderURL is the attendee's order page, where every ticket can be
	// downloaded at once.
	OrderURL string
}

// OrderConfirmation builds the message sent when a checkout completes.
func OrderConfirmation(d OrderDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.BuyerName))
	fmt.Fprintf(&b, "Your order %s is confirmed. %s attached below.\n\n",
		d.OrderNumber, plural(len(d.Tickets), "ticket is", "tickets are"))

	fmt.Fprintf(&b, "  Event:  %s\n", d.EventTitle)
	if d.EventWhen != "" {
		fmt.Fprintf(&b, "  When:   %s\n", d.EventWhen)
	}
	if d.EventVenue != "" {
		fmt.Fprintf(&b, "  Where:  %s\n", d.EventVenue)
	}
	fmt.Fprintf(&b, "  Paid:   %s KZT (simulated payment)\n", d.TotalKZT)
	b.WriteString("\n")

	if len(d.Tickets) > 0 {
		b.WriteString("Download your tickets:\n\n")
		for _, t := range d.Tickets {
			fmt.Fprintf(&b, "  %s  %s\n", t.TicketCode, t.TypeName)
			fmt.Fprintf(&b, "    %s\n", t.PDFURL)
		}
		b.WriteString("\n")
	}

	if d.OrderURL != "" {
		fmt.Fprintf(&b, "All of them on one page: %s\n\n", d.OrderURL)
	}

	b.WriteString("Show the QR code on your phone or on paper at the entrance.\n\n")
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypeOrderConfirmation,
		To:      d.BuyerEmail,
		Subject: "Your Tickets to " + d.EventTitle,
		Body:    b.String(),
	}
}

// RefundDetails is what a refund notification needs.
type RefundDetails struct {
	BuyerName   string
	BuyerEmail  string
	OrderNumber string
	EventTitle  string
	AmountKZT   string
	TicketCount int
	Reason      string
}

// RefundCompleted builds the message sent when an organizer refunds an order
// (SRS 4.10, "refund completion").
func RefundCompleted(d RefundDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.BuyerName))
	fmt.Fprintf(&b, "Your order %s for %s has been refunded.\n\n", d.OrderNumber, d.EventTitle)

	fmt.Fprintf(&b, "  Refunded: %s KZT (simulated)\n", d.AmountKZT)
	fmt.Fprintf(&b, "  Tickets:  %d, now void\n", d.TicketCount)
	if d.Reason != "" {
		fmt.Fprintf(&b, "  Reason:   %s\n", d.Reason)
	}
	b.WriteString("\n")

	b.WriteString("The QR codes on these tickets will no longer be admitted at the\n")
	b.WriteString("entrance. There is nothing else you need to do.\n\n")
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypeRefundCompleted,
		To:      d.BuyerEmail,
		Subject: "Refund for " + d.EventTitle,
		Body:    b.String(),
	}
}

// CancellationDetails is what a free-registration cancellation needs. It
// deliberately carries no amount: a free registration has nothing to refund,
// and printing "0.00 KZT refunded" would invite a question that has no answer.
type CancellationDetails struct {
	BuyerName   string
	BuyerEmail  string
	OrderNumber string
	EventTitle  string
	TicketCount int
	Reason      string
}

// RegistrationCancelled builds the message sent when an organizer cancels a
// free registration (SRS 4.9, and SRS 4.10's notification list).
func RegistrationCancelled(d CancellationDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.BuyerName))
	fmt.Fprintf(&b, "Your registration %s for %s has been cancelled by the organizer.\n\n",
		d.OrderNumber, d.EventTitle)

	noun := "tickets"
	if d.TicketCount == 1 {
		noun = "ticket"
	}
	fmt.Fprintf(&b, "  Cancelled: %d %s, now void\n", d.TicketCount, noun)
	if d.Reason != "" {
		fmt.Fprintf(&b, "  Reason: %s\n", d.Reason)
	}
	b.WriteString("\n")

	b.WriteString("The QR code on this registration will no longer be admitted at the\n")
	b.WriteString("entrance. Nothing was charged, so there is no refund to wait for.\n\n")
	b.WriteString("If you think this was a mistake, reply to the organizer through the\n")
	b.WriteString("support link on your order page.\n\n")
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypeRegistrationCancelled,
		To:      d.BuyerEmail,
		Subject: "Your registration for " + d.EventTitle + " was cancelled",
		Body:    b.String(),
	}
}

// firstName keeps a greeting from reading like a database row. An empty or
// single-word name degrades gracefully rather than producing "Hi ,".
func firstName(full string) string {
	fields := strings.Fields(full)
	if len(fields) == 0 {
		return "there"
	}
	return fields[0]
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "Your " + one
	}
	return fmt.Sprintf("Your %d %s", n, many)
}

// Notification types added in Phase 12 (SRS 4.1, 4.10).
const (
	TypeEmailVerification = "account.email_verification"
	TypePasswordReset     = "account.password_reset"
	TypeEventCancelled    = "event.cancelled"
	TypeSupportMessage    = "support.new_message"
)

// The remaining notification triggers SRS 4.10 lists. Six of the nine existed;
// these are the rest.
const (
	TypePaymentFailed        = "payment.failed"
	TypeEventUpdated         = "event.updated"
	TypePayoutStatus         = "payout.status"
	TypeSupportAssigned      = "support.case_assigned"
	TypeSupportStatusChanged = "support.status_changed"
)

// PaymentFailureDetails is what a failed payment notification needs
// (SRS 4.10, "payment failure").
type PaymentFailureDetails struct {
	BuyerName   string
	BuyerEmail  string
	OrderNumber string
	EventTitle  string
	AmountKZT   string
	Reason      string
	RetryURL    string
}

// PaymentFailed tells a buyer their payment did not go through and that no
// tickets were issued (SRS 4.6: "Failed or abandoned transactions shall not
// create valid tickets").
func PaymentFailed(d PaymentFailureDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.BuyerName))
	fmt.Fprintf(&b, "The payment for order %s did not go through, so no tickets\n", d.OrderNumber)
	fmt.Fprintf(&b, "were issued for %s.\n\n", d.EventTitle)

	// The amount is omitted rather than guessed: the decline happens before the
	// order is priced in SQL, and re-deriving a total in Go would be a second,
	// divergent copy of the pricing rules.
	if d.AmountKZT != "" {
		fmt.Fprintf(&b, "  Attempted: %s KZT (simulated)\n", d.AmountKZT)
	}
	if d.Reason != "" {
		fmt.Fprintf(&b, "  Reason:    %s\n", d.Reason)
	}
	if d.AmountKZT != "" || d.Reason != "" {
		b.WriteString("\n")
	}

	b.WriteString("Nothing was charged. Your seats were released back to the event, so\n")
	b.WriteString("they may be gone if you wait.\n\n")
	if d.RetryURL != "" {
		fmt.Fprintf(&b, "Try again: %s\n\n", d.RetryURL)
	}
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypePaymentFailed,
		To:      d.BuyerEmail,
		Subject: "Payment failed for " + d.EventTitle,
		Body:    b.String(),
	}
}

// EventUpdateDetails carries a change an attendee needs to know about.
type EventUpdateDetails struct {
	AttendeeName  string
	AttendeeEmail string
	EventTitle    string
	// Changes are human-readable lines, already phrased as before/after by the
	// caller. The template does not compare rows: what counts as a change
	// worth an email is a decision for the handler that made it.
	Changes  []string
	EventURL string
}

// EventUpdated tells ticket holders that something they planned around has
// moved (SRS 4.10, "event updates").
func EventUpdated(d EventUpdateDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.AttendeeName))
	fmt.Fprintf(&b, "%s has been updated since you booked.\n\n", d.EventTitle)

	for _, line := range d.Changes {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	if len(d.Changes) > 0 {
		b.WriteString("\n")
	}

	b.WriteString("Your ticket is unchanged and still admits you.\n\n")
	if d.EventURL != "" {
		fmt.Fprintf(&b, "Event page: %s\n\n", d.EventURL)
	}
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypeEventUpdated,
		To:      d.AttendeeEmail,
		Subject: "Update to " + d.EventTitle,
		Body:    b.String(),
	}
}

// PayoutStatusDetails is an organizer-facing payout update.
type PayoutStatusDetails struct {
	OrganizerName  string
	OrganizerEmail string
	EventTitle     string
	Status         string
	AmountKZT      string
	MaskedAccount  string
	Note           string
}

// PayoutStatus tells an organizer where their money is (SRS 4.10, "organizer
// payout status"). Nothing moves: SRS 8 excludes real payouts from the MVP,
// so the message says so rather than implying a transfer.
func PayoutStatus(d PayoutStatusDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.OrganizerName))
	fmt.Fprintf(&b, "Payout status for %s: %s.\n\n", d.EventTitle, d.Status)

	if d.AmountKZT != "" {
		fmt.Fprintf(&b, "  Amount:      %s KZT (simulated)\n", d.AmountKZT)
	}
	if d.MaskedAccount != "" {
		fmt.Fprintf(&b, "  Destination: %s\n", d.MaskedAccount)
	}
	if d.Note != "" {
		fmt.Fprintf(&b, "  Note:        %s\n", d.Note)
	}
	b.WriteString("\n")

	b.WriteString("This is a demonstration record. No money has moved and none will:\n")
	b.WriteString("real payouts are outside the scope of this release.\n\n")
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypePayoutStatus,
		To:      d.OrganizerEmail,
		Subject: "Payout " + d.Status + " for " + d.EventTitle,
		Body:    b.String(),
	}
}

// SupportCaseDetails carries an assignment or a status change.
type SupportCaseDetails struct {
	RecipientName  string
	RecipientEmail string
	CaseNumber     string
	Subject        string
	// AssigneeName is set for an assignment, Status for a status change.
	AssigneeName string
	Status       string
	CaseURL      string
}

// SupportCaseAssigned tells the requester who picked their case up
// (SRS 4.13: "support-case assignment").
func SupportCaseAssigned(d SupportCaseDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.RecipientName))
	fmt.Fprintf(&b, "%s is now with %s.\n\n", d.CaseNumber, d.AssigneeName)
	fmt.Fprintf(&b, "  Subject: %s\n\n", d.Subject)
	if d.CaseURL != "" {
		fmt.Fprintf(&b, "Read the conversation: %s\n\n", d.CaseURL)
	}
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypeSupportAssigned,
		To:      d.RecipientEmail,
		Subject: d.CaseNumber + " assigned to " + d.AssigneeName,
		Body:    b.String(),
	}
}

// SupportCaseStatusChanged tells the requester their case moved
// (SRS 4.13: "support-case status changes").
func SupportCaseStatusChanged(d SupportCaseDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.RecipientName))
	fmt.Fprintf(&b, "%s is now %s.\n\n", d.CaseNumber, readableStatus(d.Status))
	fmt.Fprintf(&b, "  Subject: %s\n\n", d.Subject)

	if d.Status == "waiting_for_customer" {
		b.WriteString("Support is waiting on a reply from you before they can go further.\n\n")
	}
	if d.CaseURL != "" {
		fmt.Fprintf(&b, "Read the conversation: %s\n\n", d.CaseURL)
	}
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypeSupportStatusChanged,
		To:      d.RecipientEmail,
		Subject: d.CaseNumber + " is now " + readableStatus(d.Status),
		Body:    b.String(),
	}
}

// readableStatus turns an enum label into something an attendee reads without
// wondering what an underscore means.
func readableStatus(status string) string {
	switch status {
	case "in_progress":
		return "in progress"
	case "waiting_for_customer":
		return "waiting for your reply"
	default:
		return status
	}
}

// AccountTokenDetails carries a single-use token to its owner.
type AccountTokenDetails struct {
	FullName string
	Email    string
	// Token is the plaintext secret. It appears in the message and nowhere
	// else - the database stores only its hash.
	Token string
	// Link is the page that consumes the token. Included alongside the bare
	// token so the console output is usable either by clicking or by pasting.
	Link      string
	ExpiresIn string
}

// EmailVerification asks a new account to confirm its address (SRS 4.1).
func EmailVerification(d AccountTokenDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.FullName))
	b.WriteString("Confirm this address to finish setting up your BiletFlow account.\n\n")
	fmt.Fprintf(&b, "  %s\n\n", d.Link)
	fmt.Fprintf(&b, "Or paste this code into the app:\n\n  %s\n\n", d.Token)
	fmt.Fprintf(&b, "The code stops working in %s.\n\n", d.ExpiresIn)
	b.WriteString("If you did not create an account, ignore this message.\n\n")
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypeEmailVerification,
		To:      d.Email,
		Subject: "Confirm your BiletFlow email address",
		Body:    b.String(),
	}
}

// PasswordReset carries a reset token (SRS 4.1, 4.10).
func PasswordReset(d AccountTokenDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.FullName))
	b.WriteString("Somebody asked to reset the password on this BiletFlow account.\n\n")
	fmt.Fprintf(&b, "  %s\n\n", d.Link)
	fmt.Fprintf(&b, "Or paste this code into the reset page:\n\n  %s\n\n", d.Token)
	fmt.Fprintf(&b, "The code stops working in %s, and only once.\n\n", d.ExpiresIn)
	b.WriteString("If this was not you, nothing has changed and you can ignore this\n")
	b.WriteString("message. Your current password still works.\n\n")
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypePasswordReset,
		To:      d.Email,
		Subject: "Reset your BiletFlow password",
		Body:    b.String(),
	}
}

// EventCancelledDetails is what a cancellation notice needs.
type EventCancelledDetails struct {
	AttendeeName string
	Email        string
	EventTitle   string
	EventWhen    string
	OrderNumber  string
	TicketCount  int
	RefundPolicy string
}

// EventCancelled tells a ticket holder their event is off (SRS 4.10).
func EventCancelled(d EventCancelledDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.AttendeeName))
	fmt.Fprintf(&b, "%s has been cancelled by the organizer.\n\n", d.EventTitle)

	if d.EventWhen != "" {
		fmt.Fprintf(&b, "  Was due:  %s\n", d.EventWhen)
	}
	fmt.Fprintf(&b, "  Order:    %s\n", d.OrderNumber)
	fmt.Fprintf(&b, "  Tickets:  %d, now void\n", d.TicketCount)
	b.WriteString("\n")

	if d.RefundPolicy != "" {
		b.WriteString("The organizer's refund policy:\n\n")
		for _, line := range strings.Split(strings.TrimSpace(d.RefundPolicy), "\n") {
			fmt.Fprintf(&b, "  %s\n", line)
		}
		b.WriteString("\n")
	} else {
		b.WriteString("Contact the organizer from your order page about a refund.\n\n")
	}

	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypeEventCancelled,
		To:      d.Email,
		Subject: d.EventTitle + " has been cancelled",
		Body:    b.String(),
	}
}

// SupportMessageDetails is what a new-message notice needs.
type SupportMessageDetails struct {
	RecipientName  string
	Email          string
	EventTitle     string
	CaseSubject    string
	SenderName     string
	MessagePreview string
	CaseURL        string
}

// NewSupportMessage tells the other side of a conversation that a reply has
// arrived (SRS 4.10, 4.13).
func NewSupportMessage(d SupportMessageDetails) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "Hi %s,\n\n", firstName(d.RecipientName))
	fmt.Fprintf(&b, "%s replied about %s.\n\n", d.SenderName, d.EventTitle)

	fmt.Fprintf(&b, "  Case: %s\n\n", d.CaseSubject)
	for _, line := range strings.Split(strings.TrimSpace(d.MessagePreview), "\n") {
		fmt.Fprintf(&b, "  > %s\n", line)
	}
	b.WriteString("\n")

	if d.CaseURL != "" {
		fmt.Fprintf(&b, "Reply here: %s\n\n", d.CaseURL)
	}
	b.WriteString("- BiletFlow\n")

	return Message{
		Type:    TypeSupportMessage,
		To:      d.Email,
		Subject: "New message about " + d.EventTitle,
		Body:    b.String(),
	}
}
