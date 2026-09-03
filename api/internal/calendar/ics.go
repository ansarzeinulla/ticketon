// Package calendar writes iCalendar files (SRS 4.11).
//
// RFC 5545 by hand rather than through a library: the whole of what BiletFlow
// needs is one VEVENT, and the fiddly parts - line folding, escaping, the
// UTC-only DTSTAMP - are a few dozen lines that are easier to get right than to
// audit in a dependency.
package calendar

import (
	"fmt"
	"strings"
	"time"
)

// Event is what goes into the file.
type Event struct {
	// UID is the stable identifier SRS 4.11 requires, so an updated file
	// replaces the earlier entry in the attendee's calendar rather than
	// creating a second one. The event's own id is exactly that.
	UID string
	// Sequence increments when the details change. A calendar client ignores
	// an update whose sequence is not higher than the copy it already has.
	Sequence int

	Summary     string
	Description string
	Location    string
	URL         string

	Starts   time.Time
	Ends     time.Time
	Timezone string

	// Cancelled writes STATUS:CANCELLED, which is how a calendar is told an
	// event is off without the attendee deleting anything (SRS 4.11).
	Cancelled bool

	// Stamp is the file's creation time. Injectable so a test can pin it.
	Stamp time.Time
}

// The product identifier that appears in every file.
const prodID = "-//BiletFlow//Ticketing//EN"

// Render returns the .ics document.
func Render(e Event) string {
	stamp := e.Stamp
	if stamp.IsZero() {
		stamp = time.Now()
	}

	location, err := time.LoadLocation(e.Timezone)
	if err != nil || e.Timezone == "" {
		location = time.UTC
	}

	var b strings.Builder

	// CRLF throughout: RFC 5545 requires it, and some clients - Outlook in
	// particular - reject a file that uses bare newlines.
	write := func(line string) { b.WriteString(fold(line) + "\r\n") }

	write("BEGIN:VCALENDAR")
	write("VERSION:2.0")
	write("PRODID:" + prodID)
	write("CALSCALE:GREGORIAN")
	// PUBLISH, not REQUEST: this is information about an event, not an
	// invitation the attendee has to respond to. SRS 4.11 is explicit that
	// export "shall not require the platform to request write access".
	write("METHOD:PUBLISH")

	// The timezone definition travels with the file so a client that has never
	// heard of Asia/Almaty still shows the right local time.
	writeTimezone(&b, location, e.Starts)

	write("BEGIN:VEVENT")
	write("UID:" + e.UID)
	write("DTSTAMP:" + stamp.UTC().Format("20060102T150405Z"))
	write(fmt.Sprintf("SEQUENCE:%d", e.Sequence))

	// Local times with a TZID, not UTC: an attendee who travels should still
	// see the event at the hour it happens at the venue.
	write("DTSTART;TZID=" + location.String() + ":" + e.Starts.In(location).Format("20060102T150405"))
	write("DTEND;TZID=" + location.String() + ":" + e.Ends.In(location).Format("20060102T150405"))

	write("SUMMARY:" + escape(e.Summary))
	if e.Location != "" {
		write("LOCATION:" + escape(e.Location))
	}
	if e.Description != "" {
		write("DESCRIPTION:" + escape(e.Description))
	}
	if e.URL != "" {
		write("URL:" + escape(e.URL))
	}

	if e.Cancelled {
		write("STATUS:CANCELLED")
	} else {
		write("STATUS:CONFIRMED")
	}
	write("TRANSP:OPAQUE")

	write("END:VEVENT")
	write("END:VCALENDAR")

	return b.String()
}

// writeTimezone emits a VTIMEZONE for the event's zone.
//
// Only the offsets actually in force around the event are described, which is
// all a client needs to place it. A full historical zone definition would be
// hundreds of lines of no use to anybody.
func writeTimezone(b *strings.Builder, location *time.Location, at time.Time) {
	if location == time.UTC {
		return
	}

	name, offset := at.In(location).Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	utcOffset := fmt.Sprintf("%s%02d%02d", sign, offset/3600, (offset%3600)/60)

	for _, line := range []string{
		"BEGIN:VTIMEZONE",
		"TZID:" + location.String(),
		"BEGIN:STANDARD",
		// A date in the past, so the rule is already in force for any event.
		"DTSTART:19700101T000000",
		"TZOFFSETFROM:" + utcOffset,
		"TZOFFSETTO:" + utcOffset,
		"TZNAME:" + name,
		"END:STANDARD",
		"END:VTIMEZONE",
	} {
		b.WriteString(line + "\r\n")
	}
}

// escape applies RFC 5545 text escaping.
//
// A comma or a semicolon in a venue name is a property separator unless it is
// escaped, so "Almaty Arena, Momyshuly 2" would otherwise silently become two
// values and the address would vanish.
func escape(text string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		"\r\n", `\n`,
		"\n", `\n`,
		"\r", `\n`,
		`;`, `\;`,
		`,`, `\,`,
	)
	return replacer.Replace(text)
}

// fold wraps a long line the way RFC 5545 requires: at most 75 octets, with
// continuations starting with a single space.
//
// It counts bytes, not runes, but never splits a multi-byte character - a
// Cyrillic event title cut down the middle of a letter would arrive as
// mojibake in somebody's calendar.
func fold(line string) string {
	const limit = 75

	if len(line) <= limit {
		return line
	}

	var b strings.Builder
	written := 0

	for i, r := range line {
		size := len(string(r))
		if written+size > limit {
			b.WriteString("\r\n ")
			// A continuation line has one leading space, which counts towards
			// its own limit.
			written = 1
		}
		b.WriteString(line[i : i+size])
		written += size
	}
	return b.String()
}
