package ticketpdf

import "strings"

// truncate shortens text to at most max characters, ending with an ellipsis.
//
// It counts runes, not bytes. Now that tickets carry Cyrillic natively, a
// byte-based cut would slice a two-byte letter in half and put a replacement
// character on somebody's ticket.
func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return strings.TrimRight(string(runes[:max-1]), " ") + "..."
}
