package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// maxPageSize caps how many events one listing request can return.
const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// createEventRequest is the body of POST /api/v1/events.
type createEventRequest struct {
	Title                string     `json:"title"`
	Slug                 *string    `json:"slug"`
	Description          *string    `json:"description"`
	Category             *string    `json:"category"`
	CoverImageURL        *string    `json:"cover_image_url"`
	VenueID              *string    `json:"venue_id"`
	VenueName            *string    `json:"venue_name"`
	VenueAddress         *string    `json:"venue_address"`
	StartsAt             *time.Time `json:"starts_at"`
	EndsAt               *time.Time `json:"ends_at"`
	Timezone             *string    `json:"timezone"`
	Visibility           *string    `json:"visibility"`
	SeatingMode          *string    `json:"seating_mode"`
	Capacity             *int       `json:"capacity"`
	RegistrationOpensAt  *time.Time `json:"registration_opens_at"`
	RegistrationClosesAt *time.Time `json:"registration_closes_at"`
	RefundPolicy         *string    `json:"refund_policy"`
}

type eventResponse struct {
	Event store.Event `json:"event"`
}

type eventListResponse struct {
	Events []store.Event `json:"events"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// validateOptionalText applies the length and character rules to an optional
// event field on a patch, but only when the patch actually sets it to a
// non-blank value - clearing a field (an explicit null, or blank) is a
// different intent the store handles.
func validateOptionalText(errs fieldErrors, field, label string, value store.Optional[string], max int, multiline bool) {
	if !value.Set || !value.Valid || blank(value.Value) {
		return
	}
	var msg string
	if multiline {
		msg = validateMultiline(label, value.Value, 1, max)
	} else {
		msg = validateLine(label, value.Value, 1, max)
	}
	if msg != "" {
		errs.add(field, msg)
	}
}

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}

	if msg := validateLine("Title", req.Title, minTitleLength, maxTitleLength); msg != "" {
		errs.add("title", msg)
	}
	validateEventText(errs, req.Description, req.Category, req.VenueName,
		req.VenueAddress, req.RefundPolicy, req.CoverImageURL)

	if req.StartsAt == nil {
		errs.add("starts_at", "Start time is required, as an RFC 3339 timestamp.")
	}
	if req.EndsAt == nil {
		errs.add("ends_at", "End time is required, as an RFC 3339 timestamp.")
	}
	if req.StartsAt != nil && req.EndsAt != nil && !req.EndsAt.After(*req.StartsAt) {
		errs.add("ends_at", "End time must be after the start time.")
	}

	timezone := "Asia/Almaty"
	if req.Timezone != nil {
		if msg := validateTimezone(*req.Timezone); msg != "" {
			errs.add("timezone", msg)
		} else {
			timezone = *req.Timezone
		}
	}

	visibility := store.VisibilityPublic
	if req.Visibility != nil {
		if !validVisibilities[*req.Visibility] {
			errs.add("visibility", "Visibility must be one of public, unlisted or private.")
		} else {
			visibility = *req.Visibility
		}
	}

	seating := store.SeatingGeneralAdmission
	if req.SeatingMode != nil {
		if !validSeatingModes[*req.SeatingMode] {
			errs.add("seating_mode", "Seating mode must be general_admission or assigned_seating.")
		} else {
			seating = *req.SeatingMode
		}
	}

	var venueID *uuid.UUID
	if req.VenueID != nil {
		parsed, err := uuid.Parse(*req.VenueID)
		if err != nil {
			errs.add("venue_id", "Venue id must be a UUID.")
		} else {
			venueID = &parsed
		}
	}
	if seating == store.SeatingAssigned && venueID == nil {
		errs.add("venue_id", "An assigned-seating event requires a venue_id with a seat layout.")
	}

	if req.Capacity != nil && *req.Capacity <= 0 {
		errs.add("capacity", "Capacity must be greater than zero.")
	}
	if req.RegistrationOpensAt != nil && req.RegistrationClosesAt != nil &&
		!req.RegistrationClosesAt.After(*req.RegistrationOpensAt) {
		errs.add("registration_closes_at", "Registration must close after it opens.")
	}

	slug := ""
	if req.Slug != nil {
		slug = store.Slugify(*req.Slug)
		if slug == "" {
			errs.add("slug", "Slug must contain at least one letter or digit.")
		} else if msg := validateSlug(slug); msg != "" {
			errs.add("slug", msg)
		}
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	userID := mustUserID(r.Context())
	event, err := s.events.Create(r.Context(), store.CreateEventParams{
		OrganizerID:          userID,
		VenueID:              venueID,
		Title:                req.Title,
		Slug:                 slug,
		Description:          req.Description,
		Category:             req.Category,
		CoverImageURL:        req.CoverImageURL,
		VenueName:            req.VenueName,
		VenueAddress:         req.VenueAddress,
		StartsAt:             *req.StartsAt,
		EndsAt:               *req.EndsAt,
		Timezone:             timezone,
		Visibility:           visibility,
		SeatingMode:          seating,
		Capacity:             req.Capacity,
		RegistrationOpensAt:  req.RegistrationOpensAt,
		RegistrationClosesAt: req.RegistrationClosesAt,
		RefundPolicy:         req.RefundPolicy,
	})
	switch {
	case errors.Is(err, store.ErrSlugTaken):
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"An event with this slug already exists. Choose a different slug.")
		return
	case err != nil:
		var constraintErr *store.ConstraintError
		if errors.As(err, &constraintErr) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeValidationFailed,
				"The event violates a database rule: "+constraintErr.Constraint+".")
			return
		}
		httpx.WriteInternalError(w, r, err)
		return
	}

	// Creating an event makes the user an organizer (SRS 3.1: publishing an
	// event and issuing free tickets costs nothing and needs no approval).
	if err := s.users.GrantRole(r.Context(), userID, store.RoleOrganizer); err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, event.ID, userID, "event.created", "event", event.ID.String(),
		"Created event "+event.Title)

	w.Header().Set("Location", "/api/v1/events/"+event.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, eventResponse{Event: event})
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadEvent(w, r)
	if !ok {
		return
	}

	// A draft, unpublished or private event is only visible to its organizer.
	if !s.canView(r, event) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, eventResponse{Event: event})
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	limit, offset, errs := pagination(r)
	q := r.URL.Query()

	filter := store.ListEventsFilter{
		// The public catalogue shows published, publicly visible events only.
		Statuses:     []string{store.EventStatusPublished},
		Visibilities: []string{store.VisibilityPublic},
		Limit:        limit,
		Offset:       offset,
	}

	if v := q.Get("category"); v != "" {
		filter.Category = &v
	}
	if v := q.Get("q"); v != "" {
		filter.Search = &v
	}
	if v := q.Get("starts_after"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			errs.add("starts_after", "Must be an RFC 3339 timestamp.")
		} else {
			filter.StartsAfter = &t
		}
	}
	if v := q.Get("starts_before"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			errs.add("starts_before", "Must be an RFC 3339 timestamp.")
		} else {
			filter.StartsBefore = &t
		}
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	events, total, err := s.events.List(r.Context(), filter)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, eventListResponse{
		Events: events, Total: total, Limit: limit, Offset: offset,
	})
}

func (s *Server) handleListMyEvents(w http.ResponseWriter, r *http.Request) {
	limit, offset, errs := pagination(r)

	userID := mustUserID(r.Context())
	filter := store.ListEventsFilter{OrganizerID: &userID, Limit: limit, Offset: offset}

	if v := r.URL.Query().Get("status"); v != "" {
		switch v {
		case store.EventStatusDraft, store.EventStatusPublished, store.EventStatusUnpublished,
			store.EventStatusCancelled, store.EventStatusCompleted, store.EventStatusSuspended:
			filter.Statuses = []string{v}
		default:
			errs.add("status", "Unknown status filter.")
		}
	}

	// SRS 4.16 groups an organizer's history as Upcoming, Active, Completed and
	// Cancelled. Those are derived from the event's dates rather than stored, so
	// the filter is applied to the rows that come back rather than in SQL.
	lifecycle := r.URL.Query().Get("lifecycle")
	if lifecycle != "" && !validLifecycles[lifecycle] {
		errs.add("lifecycle", "Must be upcoming, active, completed, cancelled, draft or suspended.")
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	if lifecycle != "" {
		// The narrowing happens after the query, so the page has to be fetched
		// whole first. An organizer's own event list is small enough that this
		// is honest rather than clever.
		filter.Limit = maxPageSize
		filter.Offset = 0
	}

	events, total, err := s.events.List(r.Context(), filter)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	if lifecycle != "" {
		matching := make([]store.Event, 0, len(events))
		for _, event := range events {
			if event.LifecycleStage == lifecycle {
				matching = append(matching, event)
			}
		}
		events = matching
		total = len(events)
	}

	httpx.WriteJSON(w, http.StatusOK, eventListResponse{
		Events: events, Total: total, Limit: limit, Offset: offset,
	})
}

// validLifecycles are the groupings the organizer's history view offers.
var validLifecycles = map[string]bool{
	"upcoming": true, "active": true, "completed": true,
	"cancelled": true, "draft": true, "suspended": true,
}

// updateEventRequest carries PATCH fields. Absent keys are left unchanged;
// an explicit null clears a nullable column.
type updateEventRequest struct {
	Title                store.Optional[string]    `json:"title"`
	Slug                 store.Optional[string]    `json:"slug"`
	Description          store.Optional[string]    `json:"description"`
	Category             store.Optional[string]    `json:"category"`
	CoverImageURL        store.Optional[string]    `json:"cover_image_url"`
	VenueID              store.Optional[uuid.UUID] `json:"venue_id"`
	VenueName            store.Optional[string]    `json:"venue_name"`
	VenueAddress         store.Optional[string]    `json:"venue_address"`
	StartsAt             store.Optional[time.Time] `json:"starts_at"`
	EndsAt               store.Optional[time.Time] `json:"ends_at"`
	Timezone             store.Optional[string]    `json:"timezone"`
	Visibility           store.Optional[string]    `json:"visibility"`
	SeatingMode          store.Optional[string]    `json:"seating_mode"`
	Capacity             store.Optional[int]       `json:"capacity"`
	RegistrationOpensAt  store.Optional[time.Time] `json:"registration_opens_at"`
	RegistrationClosesAt store.Optional[time.Time] `json:"registration_closes_at"`
	RefundPolicy         store.Optional[string]    `json:"refund_policy"`
}

// toParams maps the request onto the store's update parameters. Written out
// field by field rather than as a struct conversion, so adding a field to
// either side fails to compile in an obvious place.
func (req updateEventRequest) toParams() store.UpdateEventParams {
	return store.UpdateEventParams{
		Title:                req.Title,
		Slug:                 req.Slug,
		Description:          req.Description,
		Category:             req.Category,
		CoverImageURL:        req.CoverImageURL,
		VenueID:              req.VenueID,
		VenueName:            req.VenueName,
		VenueAddress:         req.VenueAddress,
		StartsAt:             req.StartsAt,
		EndsAt:               req.EndsAt,
		Timezone:             req.Timezone,
		Visibility:           req.Visibility,
		SeatingMode:          req.SeatingMode,
		Capacity:             req.Capacity,
		RegistrationOpensAt:  req.RegistrationOpensAt,
		RegistrationClosesAt: req.RegistrationClosesAt,
		RefundPolicy:         req.RefundPolicy,
	}
}

func (s *Server) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	var req updateEventRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}

	if req.Title.Set {
		if !req.Title.Valid {
			errs.add("title", "Title must not be blank.")
		} else if msg := validateLine("Title", req.Title.Value, minTitleLength, maxTitleLength); msg != "" {
			errs.add("title", msg)
		}
	}

	// Validate the resulting time window, mixing new values with stored ones.
	startsAt, endsAt := event.StartsAt, event.EndsAt
	if req.StartsAt.Set {
		if !req.StartsAt.Valid {
			errs.add("starts_at", "Start time cannot be removed.")
		} else {
			startsAt = req.StartsAt.Value
		}
	}
	if req.EndsAt.Set {
		if !req.EndsAt.Valid {
			errs.add("ends_at", "End time cannot be removed.")
		} else {
			endsAt = req.EndsAt.Value
		}
	}
	if !endsAt.After(startsAt) {
		errs.add("ends_at", "End time must be after the start time.")
	}

	if req.Timezone.Set {
		if !req.Timezone.Valid {
			errs.add("timezone", "Timezone cannot be removed.")
		} else if msg := validateTimezone(req.Timezone.Value); msg != "" {
			errs.add("timezone", msg)
		}
	}
	if req.Visibility.Set && (!req.Visibility.Valid || !validVisibilities[req.Visibility.Value]) {
		errs.add("visibility", "Visibility must be one of public, unlisted or private.")
	}
	if req.SeatingMode.Set && (!req.SeatingMode.Valid || !validSeatingModes[req.SeatingMode.Value]) {
		errs.add("seating_mode", "Seating mode must be general_admission or assigned_seating.")
	}
	if req.Capacity.Set && req.Capacity.Valid && req.Capacity.Value <= 0 {
		errs.add("capacity", "Capacity must be greater than zero.")
	}

	// Optional free-text fields, when the patch touches them.
	validateOptionalText(errs, "description", "Description", req.Description, maxDescriptionLength, true)
	validateOptionalText(errs, "category", "Category", req.Category, maxCategoryLength, false)
	validateOptionalText(errs, "venue_name", "Venue name", req.VenueName, maxVenueNameLength, false)
	validateOptionalText(errs, "venue_address", "Venue address", req.VenueAddress, maxVenueAddressLength, true)
	validateOptionalText(errs, "refund_policy", "Refund policy", req.RefundPolicy, maxRefundPolicyLength, true)
	if req.CoverImageURL.Set && req.CoverImageURL.Valid && !blank(req.CoverImageURL.Value) {
		if msg := validateURL("Cover image URL", req.CoverImageURL.Value, maxURLLength); msg != "" {
			errs.add("cover_image_url", msg)
		}
	}

	// The resulting seating mode must still have a venue behind it.
	seating := event.SeatingMode
	if req.SeatingMode.Set && req.SeatingMode.Valid {
		seating = req.SeatingMode.Value
	}
	venueSet := event.VenueID != nil
	if req.VenueID.Set {
		venueSet = req.VenueID.Valid
	}
	if seating == store.SeatingAssigned && !venueSet {
		errs.add("venue_id", "An assigned-seating event requires a venue_id with a seat layout.")
	}

	if req.Slug.Set {
		if !req.Slug.Valid {
			errs.add("slug", "Slug cannot be removed.")
		} else {
			normalized := store.Slugify(req.Slug.Value)
			if normalized == "" {
				errs.add("slug", "Slug must contain at least one letter or digit.")
			} else if msg := validateSlug(normalized); msg != "" {
				errs.add("slug", msg)
			} else {
				req.Slug.Value = normalized
			}
		}
	}

	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	updated, err := s.events.Update(r.Context(), event.ID, req.toParams())
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return
	case errors.Is(err, store.ErrSlugTaken):
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"An event with this slug already exists.")
		return
	case err != nil:
		var constraintErr *store.ConstraintError
		if errors.As(err, &constraintErr) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeValidationFailed,
				"The change violates a database rule: "+constraintErr.Constraint+".")
			return
		}
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, updated.ID, mustUserID(r.Context()), "event.updated", "event",
		updated.ID.String(), "Updated event "+updated.Title)

	httpx.WriteJSON(w, http.StatusOK, eventResponse{Event: updated})

	// SRS 4.10: ticket holders are told when something they planned around
	// moves. Only a published event has anybody to tell, and only changes to
	// when or where it happens are worth an email - a reworded description is
	// not something somebody needs to rearrange their evening for.
	if event.Status == store.EventStatusPublished {
		s.sendEventUpdate(updated, materialChanges(event, updated))
	}
}

// materialChanges describes the differences a ticket holder would want to know
// about, already phrased as before/after lines.
//
// It deliberately ignores everything else on the event. Deciding what counts
// as material is a product judgement, not a diff, so it lives here rather than
// in the email template.
func materialChanges(before, after store.Event) []string {
	var changes []string

	if !before.StartsAt.Equal(after.StartsAt) {
		changes = append(changes, "Starts: "+localTime(before, before.StartsAt)+
			"  ->  "+localTime(after, after.StartsAt))
	}
	if !before.EndsAt.Equal(after.EndsAt) {
		changes = append(changes, "Ends: "+localTime(before, before.EndsAt)+
			"  ->  "+localTime(after, after.EndsAt))
	}
	if before.Timezone != after.Timezone {
		changes = append(changes, "Time zone: "+before.Timezone+"  ->  "+after.Timezone)
	}
	if changed := stringChange("Venue", before.VenueName, after.VenueName); changed != "" {
		changes = append(changes, changed)
	}
	if changed := stringChange("Address", before.VenueAddress, after.VenueAddress); changed != "" {
		changes = append(changes, changed)
	}
	if before.Title != after.Title {
		changes = append(changes, "Name: "+before.Title+"  ->  "+after.Title)
	}
	return changes
}

// stringChange renders an optional field's change, treating absent and empty
// as the same thing so clearing a blank field is not reported as news.
func stringChange(label string, before, after *string) string {
	b, a := "", ""
	if before != nil {
		b = *before
	}
	if after != nil {
		a = *after
	}
	if b == a {
		return ""
	}
	if b == "" {
		return label + ": " + a
	}
	if a == "" {
		return label + ": " + b + "  ->  (removed)"
	}
	return label + ": " + b + "  ->  " + a
}

func (s *Server) handleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	// A published event may have been seen by attendees, and one with orders
	// carries ticket history, so neither is deletable - they are cancelled.
	if event.Status != store.EventStatusDraft {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"Only a draft event can be deleted. Cancel this event instead.")
		return
	}

	hasOrders, err := s.events.HasOrders(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	if hasOrders {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"This event has orders and cannot be deleted. Cancel it instead.")
		return
	}

	if err := s.events.Delete(r.Context(), event.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
			return
		}
		httpx.WriteInternalError(w, r, err)
		return
	}

	// The audit entry outlives the event: audit_logs holds no foreign keys.
	s.appendAudit(r, event.ID, mustUserID(r.Context()), "event.deleted", "event",
		event.ID.String(), "Deleted draft event "+event.Title)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePublishEvent(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	switch event.Status {
	case store.EventStatusPublished:
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict, "This event is already published.")
		return
	case store.EventStatusCancelled, store.EventStatusCompleted:
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"A "+event.Status+" event cannot be published.")
		return
	}

	// An event whose end time has already passed cannot be put on sale: there
	// is nothing left to sell a ticket to (2.md LIFE-ERR-03). A published event
	// still reaches the "completed" bucket the ordinary way - by ending after
	// it was published, not by being published after it ended.
	if !event.EndsAt.After(time.Now()) {
		httpx.WriteValidationError(w, fieldErrors{
			"ends_at": "This event has already ended and cannot be published.",
		})
		return
	}

	published, err := s.events.Publish(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, published.ID, mustUserID(r.Context()), "event.published", "event",
		published.ID.String(), "Published event "+published.Title)

	httpx.WriteJSON(w, http.StatusOK, eventResponse{Event: published})
}

func (s *Server) handleUnpublishEvent(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	if event.Status != store.EventStatusPublished {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"Only a published event can be unpublished.")
		return
	}

	updated, err := s.events.Unpublish(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, updated.ID, mustUserID(r.Context()), "event.unpublished", "event",
		updated.ID.String(), "Unpublished event "+updated.Title)

	httpx.WriteJSON(w, http.StatusOK, eventResponse{Event: updated})
}

func (s *Server) handleCancelEvent(w http.ResponseWriter, r *http.Request) {
	event, ok := s.loadOwnedEvent(w, r)
	if !ok {
		return
	}

	if event.Status == store.EventStatusCancelled {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict, "This event is already cancelled.")
		return
	}

	cancelled, err := s.events.Cancel(r.Context(), event.ID)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	s.appendAudit(r, cancelled.ID, mustUserID(r.Context()), "event.cancelled", "event",
		cancelled.ID.String(), "Cancelled event "+cancelled.Title)

	httpx.WriteJSON(w, http.StatusOK, eventResponse{Event: cancelled})

	// SRS 4.10: everybody holding a ticket is told, after the cancellation has
	// committed and the response has gone out.
	s.sendEventCancelled(cancelled)
}

// --- shared helpers ----------------------------------------------------------

// loadEvent resolves the {id} path value and fetches the event.
func (s *Server) loadEvent(w http.ResponseWriter, r *http.Request) (store.Event, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The event id must be a UUID.")
		return store.Event{}, false
	}

	event, err := s.events.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return store.Event{}, false
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return store.Event{}, false
	}
	return event, true
}

// loadOwnedEvent additionally requires the caller to be the organizer.
func (s *Server) loadOwnedEvent(w http.ResponseWriter, r *http.Request) (store.Event, bool) {
	event, ok := s.loadEvent(w, r)
	if !ok {
		return store.Event{}, false
	}

	claims, _ := claimsFromContext(r.Context())
	userID := mustUserID(r.Context())

	// Platform admins may act on any event for moderation (SRS 4.12).
	if event.OrganizerID != userID && !(claims != nil && claims.HasRole(store.RolePlatformAdmin)) {
		httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden,
			"Only the organizer of this event can perform this action.")
		return store.Event{}, false
	}
	return event, true
}

// canView reports whether the requester may see this event. Published public
// and unlisted events are visible to anyone with the link; anything else is
// restricted to the organizer and platform admins.
func (s *Server) canView(r *http.Request, event store.Event) bool {
	if event.Status == store.EventStatusPublished && event.Visibility != store.VisibilityPrivate {
		return true
	}

	claims, ok := claimsFromContext(r.Context())
	if !ok {
		return false
	}
	if claims.HasRole(store.RolePlatformAdmin) {
		return true
	}
	userID, err := claims.UserID()
	return err == nil && userID == event.OrganizerID
}

// pagination reads limit and offset from the query string.
func pagination(r *http.Request) (limit, offset int, errs fieldErrors) {
	errs = fieldErrors{}
	limit, offset = defaultPageSize, 0

	if v := r.URL.Query().Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		switch {
		case err != nil:
			errs.add("limit", "Limit must be an integer.")
		case parsed < 1 || parsed > maxPageSize:
			errs.add("limit", "Limit must be between 1 and 100.")
		default:
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		parsed, err := strconv.Atoi(v)
		switch {
		case err != nil:
			errs.add("offset", "Offset must be an integer.")
		case parsed < 0:
			errs.add("offset", "Offset must not be negative.")
		default:
			offset = parsed
		}
	}
	return limit, offset, errs
}

// appendAudit records a timeline entry. A failure here must not fail the
// request that already succeeded, so it is logged rather than returned.
func (s *Server) appendAudit(r *http.Request, eventID, actorID uuid.UUID, action, entityType, entityID, description string) {
	err := s.audit.Append(r.Context(), store.AuditEntry{
		EventID:     &eventID,
		ActorUserID: &actorID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Description: description,
		Metadata:    map[string]any{"request_id": httpx.RequestIDFromContext(r.Context())},
	})
	if err != nil {
		httpx.LogAuditFailure(r.Context(), action, err)
	}
}
