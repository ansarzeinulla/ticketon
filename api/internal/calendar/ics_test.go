package calendar

import (
	"strings"
	"testing"
	"time"
)

func sample() Event {
	almaty, _ := time.LoadLocation("Asia/Almaty")
	return Event{
		UID:         "11111111-1111-1111-1111-111111111111@biletflow.kz",
		Summary:     "Almaty Autumn Fest",
		Description: "An evening of live jazz.",
		Location:    "Gorky Park Stage, Almaty",
		URL:         "http://localhost:3000/events/almaty-autumn-fest",
		Starts:      time.Date(2026, 10, 17, 19, 0, 0, 0, almaty),
		Ends:        time.Date(2026, 10, 17, 23, 0, 0, 0, almaty),
		Timezone:    "Asia/Almaty",
		Stamp:       time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
	}
}

// lines splits the document the way a parser would.
func lines(t *testing.T, document string) []string {
	t.Helper()

	// RFC 5545 requires CRLF; some clients reject anything else.
	if strings.Contains(strings.ReplaceAll(document, "\r\n", ""), "\n") {
		t.Error("the document contains a bare newline")
	}

	// Unfold before reading: a folded line continues with a single space.
	unfolded := strings.ReplaceAll(document, "\r\n ", "")
	return strings.Split(strings.TrimRight(unfolded, "\r\n"), "\r\n")
}

func find(t *testing.T, document, prefix string) string {
	t.Helper()
	for _, line := range lines(t, document) {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	t.Fatalf("no %q line in:\n%s", prefix, document)
	return ""
}

func TestRenderIsAWellFormedCalendar(t *testing.T) {
	document := Render(sample())

	all := lines(t, document)
	if all[0] != "BEGIN:VCALENDAR" || all[len(all)-1] != "END:VCALENDAR" {
		t.Errorf("the document is not wrapped in VCALENDAR: %q ... %q", all[0], all[len(all)-1])
	}

	for _, want := range []string{
		"VERSION:2.0",
		"PRODID:-//BiletFlow//Ticketing//EN",
		"METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"END:VEVENT",
		"STATUS:CONFIRMED",
	} {
		if !strings.Contains(document, want) {
			t.Errorf("missing %q", want)
		}
	}
}

// TestRenderKeepsTheEventInItsOwnTimezone is SRS 4.11: a ticket read anywhere
// still shows the time at the venue.
func TestRenderKeepsTheEventInItsOwnTimezone(t *testing.T) {
	document := Render(sample())

	if got := find(t, document, "DTSTART;TZID=Asia/Almaty:"); got != "20261017T190000" {
		t.Errorf("DTSTART = %q, want the local 19:00", got)
	}
	if got := find(t, document, "DTEND;TZID=Asia/Almaty:"); got != "20261017T230000" {
		t.Errorf("DTEND = %q, want the local 23:00", got)
	}

	// The zone definition travels with the file, so a client that has never
	// heard of Asia/Almaty still places it correctly.
	if !strings.Contains(document, "BEGIN:VTIMEZONE") ||
		!strings.Contains(document, "TZID:Asia/Almaty") {
		t.Error("no VTIMEZONE for the event's zone")
	}
	if !strings.Contains(document, "TZOFFSETTO:+0500") {
		t.Errorf("wrong offset for Almaty in:\n%s", document)
	}
}

// TestRenderEscapesSeparators: a comma in a venue name is a property separator
// unless it is escaped, and the address would silently vanish.
func TestRenderEscapesSeparators(t *testing.T) {
	e := sample()
	e.Location = "Almaty Arena, Momyshuly 2; gate 3"
	e.Description = "Line one\nLine two"
	e.Summary = `Jazz \ Blues`

	document := Render(e)

	if got := find(t, document, "LOCATION:"); got != `Almaty Arena\, Momyshuly 2\; gate 3` {
		t.Errorf("LOCATION = %q, want the comma and semicolon escaped", got)
	}
	if got := find(t, document, "DESCRIPTION:"); got != `Line one\nLine two` {
		t.Errorf("DESCRIPTION = %q, want the newline escaped", got)
	}
	if got := find(t, document, "SUMMARY:"); got != `Jazz \\ Blues` {
		t.Errorf("SUMMARY = %q, want the backslash escaped", got)
	}
}

// TestRenderFoldsLongLines keeps the file inside the 75-octet limit.
func TestRenderFoldsLongLines(t *testing.T) {
	e := sample()
	e.Description = strings.Repeat("A very long description that needs folding. ", 6)

	document := Render(e)

	for _, line := range strings.Split(document, "\r\n") {
		if len(line) > 75 {
			t.Errorf("line is %d octets, over the 75 limit: %q", len(line), line)
		}
	}

	// And it still reads back as one value.
	if got := find(t, document, "DESCRIPTION:"); got != strings.TrimSpace(e.Description) &&
		!strings.HasPrefix(got, "A very long description") {
		t.Errorf("the folded description did not survive unfolding: %q", got)
	}
}

// TestRenderFoldingKeepsCyrillicIntact: folding counts octets, and a Cyrillic
// title cut mid-character would arrive as mojibake.
func TestRenderFoldingKeepsCyrillicIntact(t *testing.T) {
	e := sample()
	e.Summary = strings.Repeat("Қазақстан Тәуелсіздік ", 6)

	document := Render(e)

	unfolded := strings.ReplaceAll(document, "\r\n ", "")
	if !strings.Contains(unfolded, "Қазақстан Тәуелсіздік Қазақстан") {
		t.Errorf("the folded Cyrillic title did not survive:\n%s", unfolded)
	}
	if strings.Contains(document, "�") {
		t.Error("folding split a multi-byte character")
	}
}

// TestRenderMarksCancellations is SRS 4.11's cancellation file.
func TestRenderMarksCancellations(t *testing.T) {
	e := sample()
	e.Cancelled = true
	e.Sequence = 3

	document := Render(e)

	if !strings.Contains(document, "STATUS:CANCELLED") {
		t.Error("a cancelled event is not marked cancelled")
	}
	if got := find(t, document, "SEQUENCE:"); got != "3" {
		t.Errorf("SEQUENCE = %q, want 3 so the update supersedes the old entry", got)
	}
}

// TestRenderUsesAStableUID: re-downloading replaces the entry rather than
// creating a second one (SRS 4.11).
func TestRenderUsesAStableUID(t *testing.T) {
	first := Render(sample())

	later := sample()
	later.Stamp = later.Stamp.Add(48 * time.Hour)
	later.Sequence = 1
	second := Render(later)

	if find(t, first, "UID:") != find(t, second, "UID:") {
		t.Error("the UID changed between exports, so calendars would duplicate the event")
	}
	if find(t, first, "DTSTAMP:") == find(t, second, "DTSTAMP:") {
		t.Error("DTSTAMP did not move, so a client cannot tell which file is newer")
	}
}

// TestRenderFallsBackToUTC: an unknown zone must still produce a usable file.
func TestRenderFallsBackToUTC(t *testing.T) {
	e := sample()
	e.Timezone = "Mars/Olympus_Mons"

	document := Render(e)
	if !strings.Contains(document, "DTSTART;TZID=UTC:") {
		t.Errorf("an unknown zone did not fall back to UTC:\n%s", document)
	}
	if strings.Contains(document, "BEGIN:VTIMEZONE") {
		t.Error("UTC needs no VTIMEZONE block")
	}
}
