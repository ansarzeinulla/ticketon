package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

type analyticsResponse struct {
	Analytics store.EventAnalytics `json:"analytics"`
	// The event is included so the dashboard can render titles and capacity
	// without a second request.
	Event store.Event `json:"event"`
}

// handleEventAnalytics returns the organizer's dashboard figures.
//
// Filters follow SRS 4.15: date range and ticket type. Everything is computed
// from order, ticket and check-in rows, so the numbers on screen are the same
// ones a SQL query against the database would produce.
func (s *Server) handleEventAnalytics(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	filter, errs := analyticsFilterFrom(r)
	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	analytics, err := s.analytics.ForEvent(r.Context(), event.ID, filter)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, analyticsResponse{Analytics: analytics, Event: event})
}

func analyticsFilterFrom(r *http.Request) (store.AnalyticsFilter, fieldErrors) {
	errs := fieldErrors{}
	filter := store.AnalyticsFilter{}
	query := r.URL.Query()

	if v := query.Get("from"); v != "" {
		parsed, err := parseFilterTime(v)
		if err != nil {
			errs.add("from", "Must be a date (2026-01-31) or an RFC 3339 timestamp.")
		} else {
			filter.From = &parsed
		}
	}
	if v := query.Get("to"); v != "" {
		parsed, err := parseFilterTime(v)
		if err != nil {
			errs.add("to", "Must be a date (2026-01-31) or an RFC 3339 timestamp.")
		} else {
			// A plain date means "through the end of that day", which is what
			// someone picking a range in a date field expects.
			if len(v) == len("2006-01-02") {
				parsed = parsed.AddDate(0, 0, 1)
			}
			filter.To = &parsed
		}
	}
	if v := query.Get("ticket_type_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			errs.add("ticket_type_id", "Must be a UUID.")
		} else {
			filter.TicketTypeID = &id
		}
	}

	return filter, errs
}

// parseFilterTime accepts either a plain date or a full timestamp, because a
// date input sends the former and a chart drill-down the latter.
func parseFilterTime(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02", value)
}

type timelineResponse struct {
	Entries []store.TimelineEntry `json:"entries"`
}

// handleEventTimeline returns the event's chronological activity history.
func (s *Server) handleEventTimeline(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	errs := fieldErrors{}
	filter := store.TimelineFilter{}
	query := r.URL.Query()

	if v := query.Get("from"); v != "" {
		parsed, err := parseFilterTime(v)
		if err != nil {
			errs.add("from", "Must be a date or an RFC 3339 timestamp.")
		} else {
			filter.From = &parsed
		}
	}
	if v := query.Get("to"); v != "" {
		parsed, err := parseFilterTime(v)
		if err != nil {
			errs.add("to", "Must be a date or an RFC 3339 timestamp.")
		} else {
			if len(v) == len("2006-01-02") {
				parsed = parsed.AddDate(0, 0, 1)
			}
			filter.To = &parsed
		}
	}
	// "type" filters by action prefix: ticket, order, event, campaign, support.
	if v := query.Get("type"); v != "" {
		filter.Prefix = v
	}
	if v := query.Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 1 || parsed > 500 {
			errs.add("limit", "Limit must be between 1 and 500.")
		} else {
			filter.Limit = parsed
		}
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	entries, err := s.audit.Timeline(r.Context(), event.ID, filter)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, timelineResponse{Entries: entries})
}

type duplicateRequest struct {
	Title    *string    `json:"title"`
	StartsAt *time.Time `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`
}

// handleDuplicateEvent copies an event's setup into a new draft (SRS 4.16).
func (s *Server) handleDuplicateEvent(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	var req duplicateRequest
	if r.ContentLength > 0 {
		if err := httpx.DecodeJSON(w, r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
			return
		}
	}

	errs := fieldErrors{}
	if req.Title != nil && blank(*req.Title) {
		errs.add("title", "Title must not be blank.")
	}
	if req.StartsAt != nil && req.EndsAt != nil && !req.EndsAt.After(*req.StartsAt) {
		errs.add("ends_at", "End time must be after the start time.")
	}
	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	created, err := s.events.Duplicate(r.Context(), store.DuplicateParams{
		SourceID:  event.ID,
		CreatedBy: mustUserID(r.Context()),
		Title:     req.Title,
		StartsAt:  req.StartsAt,
		EndsAt:    req.EndsAt,
	})
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	// The source event's own timeline records that a copy was taken.
	s.appendAudit(r, event.ID, mustUserID(r.Context()), "event.duplicated_from", "event",
		created.ID.String(), "Duplicated into draft "+created.Title)

	w.Header().Set("Location", "/api/v1/events/"+created.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, eventResponse{Event: created})
}
