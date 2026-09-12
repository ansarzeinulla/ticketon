package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/calendar"
	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// handleEventCalendar serves the event as an .ics file (SRS 4.11).
//
// Public and unauthenticated: the attendee-facing event page links straight to
// it, and a calendar file is the same information the page already shows. SRS
// 4.11 is explicit that export must not require write access to anybody's
// calendar account - this is a download, not an integration.
func (s *Server) handleEventCalendar(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadCalendarEvent(w, r)
	if !ok {
		return
	}

	document := calendar.Render(calendar.Event{
		// The event id is the stable identifier SRS 4.11 asks for: downloading
		// the file again after the organizer changes the time replaces the
		// entry rather than adding a second one.
		UID:      event.ID.String() + "@biletflow.kz",
		Sequence: calendarSequence(event),

		Summary:     event.Title,
		Description: calendarDescription(event),
		Location:    venueLine(event),
		URL:         s.cfg.WebBaseURL + "/events/" + event.Slug,

		Starts:   event.StartsAt,
		Ends:     event.EndsAt,
		Timezone: event.Timezone,

		// A cancelled event exports a cancellation, which is how a calendar is
		// told the evening is off (SRS 4.11).
		Cancelled: event.Status == store.EventStatusCancelled,

		Stamp: s.now(),
	})

	filename := calendarFilename(event.Slug)

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	// Not cached: an attendee who downloads it again is doing so precisely
	// because they think something has changed.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(document))
}

// loadCalendarEvent resolves the event by id or slug and decides whether this
// caller may see it.
//
// Both forms are accepted because the two places that link to a calendar file
// know different things: the organizer's dashboard has an id, the public page
// has a slug.
func (s *Server) loadCalendarEvent(w http.ResponseWriter, r *http.Request) (store.Event, bool) {
	raw := r.PathValue("id")

	var (
		event store.Event
		err   error
	)
	if id, parseErr := uuid.Parse(raw); parseErr == nil {
		event, err = s.events.GetByID(r.Context(), id)
	} else {
		event, err = s.events.GetBySlug(r.Context(), raw)
	}

	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return store.Event{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.Event{}, false
	}

	// The same visibility rule as the event page, with one deliberate
	// exception: an event that was published and has since been cancelled
	// stays downloadable. SRS 4.11 requires a cancellation file, and people
	// already hold entries for it - refusing the export would leave a stale
	// entry in their calendar with no way to correct it.
	//
	// A private event is still private, and a draft that was never published
	// is still nobody else's business.
	cancelledButWasPublic := event.Status == store.EventStatusCancelled &&
		event.PublishedAt != nil && event.Visibility != store.VisibilityPrivate

	if !s.canView(r, event) && !cancelledButWasPublic {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return store.Event{}, false
	}
	return event, true
}

// calendarSequence increments when the event changes, so a re-downloaded file
// supersedes the copy already in somebody's calendar.
//
// Seconds since the event row was last updated, which is monotonic per event
// and needs no extra column: a later edit always produces a higher number.
func calendarSequence(event store.Event) int {
	if event.UpdatedAt.IsZero() {
		return 0
	}
	return int(event.UpdatedAt.Unix() - event.CreatedAt.Unix())
}

// calendarDescription is the body of the entry.
func calendarDescription(event store.Event) string {
	var parts []string

	if event.Description != nil && *event.Description != "" {
		parts = append(parts, *event.Description)
	}
	if event.Status == store.EventStatusCancelled {
		parts = append(parts, "This event has been cancelled by the organizer.")
	}
	if event.RefundPolicy != nil && *event.RefundPolicy != "" {
		parts = append(parts, "Refund policy: "+*event.RefundPolicy)
	}

	return strings.Join(parts, "\n\n")
}

// calendarFilename keeps the download to characters every filesystem accepts.
func calendarFilename(slug string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '-'
		}
	}, slug)

	if safe == "" {
		safe = "event"
	}
	return safe + ".ics"
}
