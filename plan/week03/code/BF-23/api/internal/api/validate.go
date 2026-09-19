package api

import (
	"fmt"
	"net/mail"
	neturl "net/url"
	"strings"
	"time"
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

	minTitleLength = 3
	maxTitleLength = 200

	minTicketTypeNameLength = 1
	maxTicketTypeNameLength = 120

	minSlugLength = 3
	maxSlugLength = 80

	maxDescriptionLength  = 5000
	maxCategoryLength     = 60
	maxVenueNameLength    = 200
	maxVenueAddressLength = 300
	maxRefundPolicyLength = 2000
	maxURLLength          = 500
)

// validateLine checks a single-line text field is present and within bounds.
func validateLine(label, value string, min, max int) string {
	if msg := validateMultiline(label, value, min, max); msg != "" {
		return msg
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Sprintf("%s must be a single line.", label)
	}
	return ""
}

// validateMultiline is validateLine for fields that may span several lines,
// such as a description.
func validateMultiline(label, value string, min, max int) string {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return fmt.Sprintf("%s is required.", label)
	case len(value) < min:
		return fmt.Sprintf("%s must be at least %d characters.", label, min)
	case len(value) > max:
		return fmt.Sprintf("%s must not exceed %d characters.", label, max)
	}
	return ""
}

// checkOptionalLine validates an optional single-line field, skipping it when
// absent or blank (blank means "leave unset").
func checkOptionalLine(errs fieldErrors, field, label string, value *string, max int) {
	if value == nil || blank(*value) {
		return
	}
	if msg := validateLine(label, *value, 1, max); msg != "" {
		errs.add(field, msg)
	}
}

// checkOptionalMultiline is checkOptionalLine for fields that may span lines.
func checkOptionalMultiline(errs fieldErrors, field, label string, value *string, max int) {
	if value == nil || blank(*value) {
		return
	}
	if msg := validateMultiline(label, *value, 1, max); msg != "" {
		errs.add(field, msg)
	}
}

// validateEventText checks an event's optional free-text fields. Shared by
// create and patch so the two cannot drift apart.
func validateEventText(errs fieldErrors, description, category, venueName, venueAddress, refundPolicy, coverImageURL *string) {
	checkOptionalMultiline(errs, "description", "Description", description, maxDescriptionLength)
	checkOptionalLine(errs, "category", "Category", category, maxCategoryLength)
	checkOptionalLine(errs, "venue_name", "Venue name", venueName, maxVenueNameLength)
	checkOptionalMultiline(errs, "venue_address", "Venue address", venueAddress, maxVenueAddressLength)
	checkOptionalMultiline(errs, "refund_policy", "Refund policy", refundPolicy, maxRefundPolicyLength)
	if coverImageURL != nil && !blank(*coverImageURL) {
		if msg := validateURL("Cover image URL", *coverImageURL, maxURLLength); msg != "" {
			errs.add("cover_image_url", msg)
		}
	}
}

// validateURL accepts an absolute http(s) URL or a same-origin path such as
// /uploads/<name>. A protocol-relative "//host" is refused: it points off-site
// while looking local.
func validateURL(label, value string, max int) string {
	v := strings.TrimSpace(value)
	if len(v) > max {
		return fmt.Sprintf("%s must not exceed %d characters.", label, max)
	}
	if strings.ContainsAny(v, " \t\r\n") {
		return label + " is not a valid URL."
	}
	if strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "//") {
		return ""
	}
	u, err := neturl.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return label + " must be a valid http(s) URL."
	}
	return ""
}

// isIdentifierChar reports whether r may appear in a slug: Latin letters,
// digits and a hyphen, because slugs travel in URLs and are typed by hand.
func isIdentifierChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-'
}

// validateSlug checks a URL slug. Returns "" when acceptable.
func validateSlug(slug string) string {
	if slug == "" {
		return "" // Optional: the server derives one from the title when blank.
	}
	if len(slug) < minSlugLength {
		return fmt.Sprintf("The slug must be at least %d characters.", minSlugLength)
	}
	if len(slug) > maxSlugLength {
		return fmt.Sprintf("The slug must not exceed %d characters.", maxSlugLength)
	}
	for _, r := range slug {
		if !isIdentifierChar(r) {
			return "The slug may use only lowercase letters, digits and hyphens."
		}
	}
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") || strings.Contains(slug, "--") {
		return "The slug cannot start or end with a hyphen or contain a double hyphen."
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

// validVisibilities matches the event_visibility enum.
var validVisibilities = map[string]bool{"public": true, "unlisted": true, "private": true}

// validSeatingModes matches the seating_mode enum.
var validSeatingModes = map[string]bool{"general_admission": true, "assigned_seating": true}

// validateTimezone checks the value is a loadable IANA zone, so calendar
// exports can keep the event's own zone.
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
