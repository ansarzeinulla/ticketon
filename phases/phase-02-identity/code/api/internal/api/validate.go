package api

import (
	"fmt"
	"net/mail"
	neturl "net/url"
	"strings"
	"time"
	"unicode"
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

// Length bounds, all counted in characters (runes), never bytes.
//
// Byte counting is wrong for a bilingual platform: "Асан" is four letters but
// eight bytes, so a byte cap silently gives Kazakh and Russian half the room it
// gives English. Every limit here is a rune count, so a field holds the same
// number of letters whatever alphabet they are written in.
const (
	minPasswordLength = 8
	// Passwords are no longer capped by bytes - the hasher pre-hashes, so
	// bcrypt's 72-byte window no longer leaks into the product. This is a
	// sanity ceiling against a multi-megabyte request body, not a security
	// limit.
	maxPasswordLength = 128

	minTitleLength = 3
	maxTitleLength = 200

	minNameLength = 1
	maxNameLength = 200

	minTicketTypeNameLength = 1
	maxTicketTypeNameLength = 120

	minSlugLength = 3
	maxSlugLength = 80

	maxEmailLength = 254

	// maxReasonLength bounds the free text on a refund or a suspension. Long
	// enough for an explanation, short enough that it cannot be used to store
	// a document in an audit row.
	maxReasonLength = 500

	maxDescriptionLength  = 5000
	maxCategoryLength     = 60
	maxVenueNameLength    = 200
	maxVenueAddressLength = 300
	maxRefundPolicyLength = 2000
	maxURLLength          = 500
)

// runeLen counts characters, not bytes.
func runeLen(s string) int { return utf8.RuneCountInString(s) }

// hasControlChars reports whether s carries any control character. Tab, newline
// and carriage return are permitted only when multiline is true, so a name or a
// title cannot smuggle in a line break while a description still can.
func hasControlChars(s string, multiline bool) bool {
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' {
			if multiline {
				continue
			}
			return true
		}
		// Rejects C0/C1 controls and other non-printing code points, which have
		// no place in a display string and are a common injection vector.
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// validateLine validates a single-line human text field: it is trimmed, counted
// in runes against [min,max], and refused if it carries control characters or a
// line break. It returns "" when the value is acceptable.
func validateLine(label, value string, min, max int) string {
	return validateText(label, value, min, max, false)
}

// validateMultiline is validateLine for fields that may span several lines
// (descriptions, messages, reasons): line breaks and tabs are allowed, other
// control characters are not.
func validateMultiline(label, value string, min, max int) string {
	return validateText(label, value, min, max, true)
}

func validateText(label, value string, min, max int, multiline bool) string {
	v := strings.TrimSpace(value)
	n := runeLen(v)
	if n == 0 {
		// An empty required field reads as "required", not "too short" - the
		// user has not typed too little, they have typed nothing.
		return label + " is required."
	}
	if n < min {
		return fmt.Sprintf("%s must be at least %d characters.", label, min)
	}
	if n > max {
		return fmt.Sprintf("%s must not exceed %d characters.", label, max)
	}
	if hasControlChars(v, multiline) {
		return label + " contains characters that are not allowed."
	}
	return ""
}

// checkOptionalLine validates an optional single-line field, skipping it when
// absent or blank (blank means "leave unset", which the store already handles).
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

// validateEventText applies the length and character rules to an event's
// optional free-text fields. Shared by create and (via pointers) patch so the
// two paths cannot drift apart.
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

// validateURL checks a stored URL: bounded length, and either an absolute
// http(s) URL or a same-origin path. The cover image URL is echoed back from an
// upload, which this API may serve at a relative path (/uploads/<name>), so a
// root-relative path is accepted; a protocol-relative "//host" one is not,
// because it points off-site while looking local.
func validateURL(label, value string, max int) string {
	v := strings.TrimSpace(value)
	if runeLen(v) > max {
		return fmt.Sprintf("%s must not exceed %d characters.", label, max)
	}
	if hasControlChars(v, false) || strings.ContainsAny(v, " \t") {
		return label + " is not a valid URL."
	}
	if strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "//") {
		return "" // Same-origin path such as /uploads/<name>.
	}
	u, err := neturl.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return label + " must be a valid http(s) URL."
	}
	return ""
}

// isIdentifierChar reports whether r may appear in a machine identifier - a
// slug or a promo code. Deliberately ASCII: these travel in URLs and are typed
// by hand, so they are Latin letters, digits and a hyphen only. This restriction
// is right for identifiers and wrong for names, which stay Unicode so Kazakh and
// Russian are first-class.
func isIdentifierChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-'
}

// validateSlug checks a URL slug: Latin letters, digits and hyphens, within the
// length bounds. Returns "" when acceptable.
func validateSlug(slug string) string {
	if slug == "" {
		return "" // Optional: the server derives one from the title when blank.
	}
	if n := runeLen(slug); n < minSlugLength {
		return fmt.Sprintf("The slug must be at least %d characters.", minSlugLength)
	} else if n > maxSlugLength {
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
	if runeLen(email) > maxEmailLength {
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

// validatePassword enforces the length bounds, counted in characters.
func validatePassword(password string) string {
	if password == "" {
		return "Password is required."
	}
	if n := runeLen(password); n < minPasswordLength {
		return fmt.Sprintf("Password must be at least %d characters.", minPasswordLength)
	} else if n > maxPasswordLength {
		return fmt.Sprintf("Password must not exceed %d characters.", maxPasswordLength)
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
