package api

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

// fieldErrors collects per-field validation messages.
type fieldErrors map[string]string

func (f fieldErrors) add(field, message string) {
	if _, exists := f[field]; !exists {
		f[field] = message
	}
}

func (f fieldErrors) any() bool { return len(f) > 0 }

// Password bounds. The upper bound is bcrypt's input limit.
const (
	minPasswordLength = 8
	maxPasswordBytes  = 72
	maxTitleLength    = 200
	maxNameLength     = 200
	maxSlugLength     = 80
	// maxReasonLength bounds the free text on a refund or a suspension. Long
	// enough for an explanation, short enough that it cannot be used to store
	// a document in an audit row.
	maxReasonLength = 500
)

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
	if len(email) > 254 {
		return "Email must not exceed 254 characters."
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
	if utf8.RuneCountInString(password) < minPasswordLength {
		return fmt.Sprintf("Password must be at least %d characters.", minPasswordLength)
	}
	if len(password) > maxPasswordBytes {
		return fmt.Sprintf("Password must not exceed %d bytes.", maxPasswordBytes)
	}
	return ""
}

// validLocales matches the users.locale check constraint.
var validLocales = map[string]bool{"kk": true, "ru": true, "en": true}

// validVisibilities matches the event_visibility enum.
var validVisibilities = map[string]bool{"public": true, "unlisted": true, "private": true}

// validSeatingModes matches the seating_mode enum.
var validSeatingModes = map[string]bool{"general_admission": true, "assigned_seating": true}

// validateTimezone checks the value is a loadable IANA zone, which SRS 4.11
// requires so calendar exports keep the configured zone.
func validateTimezone(tz string) string {
	if tz == "" {
		return "Timezone is required."
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "Timezone must be an IANA name such as Asia/Almaty."
	}
	return ""
}

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
