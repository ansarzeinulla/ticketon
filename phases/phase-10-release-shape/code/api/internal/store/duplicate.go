package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DuplicateParams describes how to copy an event into a new draft.
type DuplicateParams struct {
	SourceID  uuid.UUID
	CreatedBy uuid.UUID

	// Optional overrides. Left unset, the copy keeps the original's values -
	// predictable beats clever, and it is a draft the organizer must edit
	// before publishing anyway.
	Title    *string
	StartsAt *time.Time
	EndsAt   *time.Time
}

// Duplicate copies an event's configuration into a new draft.
//
// SRS 4.16: the copy carries the event's setup and its ticket type definitions,
// and excludes "the original event's orders, tickets, payments, check-ins, and
// support cases". That exclusion is not filtering - none of those rows are
// touched at all. Only two INSERTs run: one event and its ticket types.
//
// Campaigns are also left behind, though the SRS does not name them: a promo
// code is globally unique, so copying one would collide with the original the
// moment two drafts existed. The organizer creates fresh codes for a new run.
func (s *EventStore) Duplicate(ctx context.Context, p DuplicateParams) (Event, error) {
	source, err := s.GetByID(ctx, p.SourceID)
	if err != nil {
		return Event{}, err
	}

	title := source.Title + " (copy)"
	if p.Title != nil && *p.Title != "" {
		title = *p.Title
	}

	startsAt, endsAt := source.StartsAt, source.EndsAt
	if p.StartsAt != nil {
		startsAt = *p.StartsAt
	}
	if p.EndsAt != nil {
		endsAt = *p.EndsAt
	}
	if !endsAt.After(startsAt) {
		// Keep the original's duration rather than rejecting a lone start time.
		endsAt = startsAt.Add(source.EndsAt.Sub(source.StartsAt))
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Event{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	base := Slugify(title)
	if base == "" {
		base = "event"
	}

	var created Event
	inserted := false

	for attempt := 1; attempt <= slugAttempts; attempt++ {
		slug := base
		if attempt > 1 {
			slug = fmt.Sprintf("%s-%d", base, attempt)
		}

		// A savepoint so a slug collision does not poison the transaction.
		savepoint, spErr := tx.Begin(ctx)
		if spErr != nil {
			return Event{}, mapError(spErr)
		}

		created, err = scanEvent(savepoint.QueryRow(ctx, `
			INSERT INTO events (
				organizer_id, venue_id, title, slug, description, category, cover_image_url,
				venue_name, venue_address, starts_at, ends_at, timezone,
				status, visibility, seating_mode, capacity, refund_policy,
				duplicated_from_event_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
				'draft', $13::event_visibility, $14::seating_mode, $15, $16, $17)
			RETURNING `+eventColumns,
			source.OrganizerID, source.VenueID, title, slug, source.Description,
			source.Category, source.CoverImageURL, source.VenueName, source.VenueAddress,
			startsAt, endsAt, source.Timezone,
			source.Visibility, source.SeatingMode, source.Capacity, source.RefundPolicy,
			source.ID))

		if err == nil {
			if commitErr := savepoint.Commit(ctx); commitErr != nil {
				return Event{}, mapError(commitErr)
			}
			inserted = true
			break
		}

		_ = savepoint.Rollback(ctx)
		if !isUniqueViolation(err, "events_slug_key") {
			return Event{}, mapError(err)
		}
	}

	if !inserted {
		return Event{}, ErrSlugTaken
	}

	// Ticket type definitions only: every counter starts at zero, because the
	// copy has sold nothing.
	if _, err := tx.Exec(ctx, `
		INSERT INTO ticket_types (event_id, name, description, price_kzt, quantity_total,
		                          max_per_order, is_hidden, price_category, display_order)
		SELECT $2, name, description, price_kzt, quantity_total,
		       max_per_order, is_hidden, price_category, display_order
		  FROM ticket_types
		 WHERE event_id = $1`, source.ID, created.ID); err != nil {
		return Event{}, mapError(err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_user_id, action, entity_type, entity_id, description, metadata)
		VALUES ($1, $2, 'event.duplicated', 'event', $3, $4, jsonb_build_object('source_event_id', $5::text))`,
		created.ID, nullableUUID(p.CreatedBy), created.ID.String(),
		"Duplicated from "+source.Title, source.ID.String()); err != nil {
		return Event{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Event{}, mapError(err)
	}
	return created, nil
}

// Lifecycle classifies an event the way the organizer's history view groups
// them (SRS 4.16: Upcoming, Active, Completed, Cancelled).
//
// It is derived rather than stored: "completed" is simply an event whose end
// time has passed, and nothing has to run to make that true.
func Lifecycle(e Event, now time.Time) string {
	switch e.Status {
	case EventStatusCancelled:
		return "cancelled"
	case EventStatusSuspended:
		return "suspended"
	case EventStatusDraft, EventStatusUnpublished:
		return "draft"
	}

	switch {
	case e.Status == EventStatusCompleted, !now.Before(e.EndsAt):
		return "completed"
	case !now.Before(e.StartsAt):
		return "active"
	default:
		return "upcoming"
	}
}

// TimelineEntry is one line of an event's activity history (SRS 4.16).
type TimelineEntry struct {
	ID          int64      `json:"id"`
	EventID     *uuid.UUID `json:"event_id,omitempty"`
	ActorID     *uuid.UUID `json:"actor_user_id,omitempty"`
	ActorName   *string    `json:"actor_name,omitempty"`
	Action      string     `json:"action"`
	EntityType  string     `json:"entity_type"`
	EntityID    string     `json:"entity_id,omitempty"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TimelineFilter narrows a timeline by date and activity type (SRS 4.16).
type TimelineFilter struct {
	From   *time.Time
	To     *time.Time
	Prefix string // e.g. "ticket." to see only ticket activity
	Limit  int
}

// Timeline returns an event's activity history, newest first.
//
// audit_logs holds no foreign keys - it is append-only, so a cascading delete
// would be blocked by its own trigger - which is why the actor's name is joined
// on rather than stored, and why an entry survives whatever it describes.
func (s *AuditStore) Timeline(
	ctx context.Context, eventID uuid.UUID, f TimelineFilter,
) ([]TimelineEntry, error) {
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var from, to, prefix any
	if f.From != nil {
		from = *f.From
	}
	if f.To != nil {
		to = *f.To
	}
	if f.Prefix != "" {
		prefix = f.Prefix + "%"
	}

	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.event_id, a.actor_user_id, u.full_name,
		       a.action, a.entity_type, COALESCE(a.entity_id, ''),
		       COALESCE(a.description, ''), a.created_at
		  FROM audit_logs a
		  LEFT JOIN users u ON u.id = a.actor_user_id
		 WHERE a.event_id = $1
		   AND ($2::timestamptz IS NULL OR a.created_at >= $2)
		   AND ($3::timestamptz IS NULL OR a.created_at < $3)
		   AND ($4::text IS NULL OR a.action LIKE $4)
		 ORDER BY a.created_at DESC, a.id DESC
		 LIMIT $5`, eventID, from, to, prefix, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	entries := []TimelineEntry{}
	for rows.Next() {
		var e TimelineEntry
		if err := rows.Scan(&e.ID, &e.EventID, &e.ActorID, &e.ActorName,
			&e.Action, &e.EntityType, &e.EntityID, &e.Description, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
