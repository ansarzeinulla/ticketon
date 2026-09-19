package api

import (
	"fmt"
	"net/mail"
	"strings"
)

// fieldErrors collects per-field validation messages.
type fieldErrors map[string]string

func (f fieldErrors) add(field, message string) {
	if _, exists := f[field]; !exists {
		f[field] = message
	}
}

func (f fieldErrors) any() bool { return len(f) > 0 }

// Length bounds for user input.
const (
	minPasswordLength = 8
	// bcrypt only looks at the first 72 bytes of a password.
	maxPasswordBytes = 72

	minNameLength  = 1
	maxNameLength  = 200
	maxEmailLength = 254
)

// validateLine checks a single-line text field is present and within bounds.
func validateLine(label, value string, min, max int) string {
	value = strings.TrimSpace(value)
	switch {
	case len(value) < min:
		return fmt.Sprintf("%s is required.", label)
	case len(value) > max:
		return fmt.Sprintf("%s must not exceed %d characters.", label, max)
	case strings.ContainsAny(value, "\r\n"):
		return fmt.Sprintf("%s must be a single line.", label)
	}
	return ""
}

// normalizeEmail trims and lowercases an address. users.email is a citext
// column, so this is presentation only - uniqueness is enforced by the database.
func normalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// validateEmail checks the address is syntactically usable.
func validateEmail(email string) string {
	if email == "" {
		return "Email is required."
	}
	if len(email) > maxEmailLength {
		return fmt.Sprintf("Email must not exceed %d characters.", maxEmailLength)
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "Email must be a valid address such as name@example.kz."
	}
	// The database also enforces this shape; rejecting it here turns what would
	// be a 500 from a check constraint into a clear 422.
	at := strings.LastIndex(email, "@")
	if at <= 0 || !strings.Contains(email[at+1:], ".") || strings.ContainsAny(email, " \t") {
		return "Email must be a valid address such as name@example.kz."
	}
	return ""
}

// validatePassword enforces the length bounds.
func validatePassword(password string) string {
	if password == "" {
		return "Password is required."
	}
	if len(password) < minPasswordLength {
		return fmt.Sprintf("Password must be at least %d characters.", minPasswordLength)
	}
	if len(password) > maxPasswordBytes {
		return fmt.Sprintf("Password must not exceed %d bytes.", maxPasswordBytes)
	}
	return ""
}

// validLocales matches the users.locale check constraint.
var validLocales = map[string]bool{"kk": true, "ru": true, "en": true}

// blank reports whether a string is empty once trimmed, matching the
// btrim(...) <> ” check constraints in the schema.
func blank(s string) bool { return strings.TrimSpace(s) == "" }

// nameFromEmail derives a display name from an address, used when a client
// registers with email and password only.
func nameFromEmail(email string) string {
	local, _, found := strings.Cut(email, "@")
	if !found || local == "" {
		return "BiletFlow User"
	}
	local = strings.NewReplacer(".", " ", "_", " ", "-", " ", "+", " ").Replace(local)

	words := strings.Fields(local)
	for i, w := range words {
		r := []rune(w)
		words[i] = strings.ToUpper(string(r[0])) + string(r[1:])
	}
	if len(words) == 0 {
		return "BiletFlow User"
	}
	return strings.Join(words, " ")
}
