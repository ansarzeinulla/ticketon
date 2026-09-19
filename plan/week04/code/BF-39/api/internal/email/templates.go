package email

import (
	"fmt"
	"strings"
)

// Notification types, mirrored into notifications.type.
const (
	TypeOrderConfirmation = "order.confirmation"
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

// Notification types for account emails (SRS 4.1).
const (
	TypeEmailVerification = "account.email_verification"
	TypePasswordReset     = "account.password_reset"
)

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
