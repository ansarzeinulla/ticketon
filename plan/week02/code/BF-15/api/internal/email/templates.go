package email

import (
	"fmt"
	"strings"
)

// Notification types for account emails (SRS 4.1).
const (
	TypeEmailVerification = "account.email_verification"
	TypePasswordReset     = "account.password_reset"
)

func firstName(full string) string {
	fields := strings.Fields(full)
	if len(fields) == 0 {
		return "there"
	}
	return fields[0]
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
