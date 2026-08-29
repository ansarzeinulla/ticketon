package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Event mirrors a row of the events table.
type Event struct {
	ID                    uuid.UUID  `json:"id"`
	OrganizerID           uuid.UUID  `json:"organizer_id"`
	VenueID               *uuid.UUID `json:"venue_id,omitempty"`
	Title                 string     `json:"title"`
	Slug                  string     `json:"slug"`
	Description           *string    `json:"description,omitempty"`
	Category              *string    `json:"category,omitempty"`
	CoverImageURL         *string    `json:"cover_image_url,omitempty"`
	VenueName             *string    `json:"venue_name,omitempty"`
	VenueAddress          *string    `json:"venue_address,omitempty"`
	StartsAt              time.Time  `json:"starts_at"`
	EndsAt                time.Time  `json:"ends_at"`
	Timezone              string     `json:"timezone"`
	Status                string     `json:"status"`
	Visibility            string     `json:"visibility"`
	SeatingMode           string     `json:"seating_mode"`
	Capacity              *int       `json:"capacity,omitempty"`
	RegistrationOpensAt   *time.Time `json:"registration_opens_at,omitempty"`
	RegistrationClosesAt  *time.Time `json:"registration_closes_at,omitempty"`
	PaidSalesEnabled      bool       `json:"paid_sales_enabled"`
	RefundPolicy          *string    `json:"refund_policy,omitempty"`
	PublishedAt           *time.Time `json:"published_at,omitempty"`
	CancelledAt           *time.Time `json:"cancelled_at,omitempty"`
	DuplicatedFromEventID *uuid.UUID `json:"duplicated_from_event_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`

	// Lifecycle is derived, not stored: Upcoming / Active / Completed /
	// Cancelled is how SRS 4.16 asks the organizer's history to be grouped, and
	// "completed" is simply an event whose end time has passed.
	LifecycleStage string `json:"lifecycle"`
}

// Values of the event_status enum.
const (
	EventStatusDraft       = "draft"
	EventStatusPublished   = "published"
	EventStatusUnpublished = "unpublished"
	EventStatusCancelled   = "cancelled"
	EventStatusCompleted   = "completed"
	// EventStatusSuspended is set only by a platform administrator (SRS 4.12).
	// The organizer cannot set or clear it.
	EventStatusSuspended = "suspended"
)

// Values of the event_visibility and seating_mode enums.
const (
	VisibilityPublic   = "public"
	VisibilityUnlisted = "unlisted"
	VisibilityPrivate  = "private"

	SeatingGeneralAdmission = "general_admission"
	SeatingAssigned         = "assigned_seating"
)

// EventStore reads and writes events.
type EventStore struct {
	pool *pgxpool.Pool
}

// NewEventStore builds an EventStore.
func NewEventStore(pool *pgxpool.Pool) *EventStore { return &EventStore{pool: pool} }

const eventColumns = `id, organizer_id, venue_id, title, slug, description, category,
	cover_image_url, venue_name, venue_address, starts_at, ends_at, timezone,
	status::text, visibility::text, seating_mode::text, capacity,
	registration_opens_at, registration_closes_at, paid_sales_enabled, refund_policy,
	published_at, cancelled_at, duplicated_from_event_id, created_at, updated_at`

func scanEvent(row pgx.Row) (Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.OrganizerID, &e.VenueID, &e.Title, &e.Slug, &e.Description,
		&e.Category, &e.CoverImageURL, &e.VenueName, &e.VenueAddress, &e.StartsAt, &e.EndsAt,
		&e.Timezone, &e.Status, &e.Visibility, &e.SeatingMode, &e.Capacity,
		&e.RegistrationOpensAt, &e.RegistrationClosesAt, &e.PaidSalesEnabled, &e.RefundPolicy,
		&e.PublishedAt, &e.CancelledAt, &e.DuplicatedFromEventID, &e.CreatedAt, &e.UpdatedAt)
	if err == nil {
		e.LifecycleStage = Lifecycle(e, time.Now())
	}
	return e, err
}

// CreateEventParams describes a new event. Slug may be empty, in which case one
// is derived from the title.
type CreateEventParams struct {
	OrganizerID          uuid.UUID
	VenueID              *uuid.UUID
	Title                string
	Slug                 string
	Description          *string
	Category             *string
	CoverImageURL        *string
	VenueName            *string
	VenueAddress         *string
	StartsAt             time.Time
	EndsAt               time.Time
	Timezone             string
	Visibility           string
	SeatingMode          string
	Capacity             *int
	RegistrationOpensAt  *time.Time
	RegistrationClosesAt *time.Time
	RefundPolicy         *string
}

// slugAttempts caps how many times Create retries a colliding slug before
// giving up, so a pathological title cannot spin forever.
const slugAttempts = 12

// Create inserts an event as a draft. When the derived slug is taken it retries
// with a numeric suffix, so two events with the same title both succeed.
func (s *EventStore) Create(ctx context.Context, p CreateEventParams) (Event, error) {
	base := p.Slug
	if base == "" {
		base = Slugify(p.Title)
	}
	if base == "" {
		base = "event"
	}

	for attempt := 1; attempt <= slugAttempts; attempt++ {
		slug := base
		if attempt > 1 {
			slug = fmt.Sprintf("%s-%d", base, attempt)
		}

		event, err := s.insert(ctx, p, slug)
		if errors.Is(err, ErrSlugTaken) {
			// An explicit slug from the client must not be silently changed.
			if p.Slug != "" {
				return Event{}, ErrSlugTaken
			}
			continue
		}
		if err != nil {
			return Event{}, err
		}
		return event, nil
	}
	return Event{}, ErrSlugTaken
}

func (s *EventStore) insert(ctx context.Context, p CreateEventParams, slug string) (Event, error) {
	event, err := scanEvent(s.pool.QueryRow(ctx, `
		INSERT INTO events (
			organizer_id, venue_id, title, slug, description, category, cover_image_url,
			venue_name, venue_address, starts_at, ends_at, timezone,
			status, visibility, seating_mode, capacity,
			registration_opens_at, registration_closes_at, refund_policy)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			'draft', $13::event_visibility, $14::seating_mode, $15, $16, $17, $18)
		RETURNING `+eventColumns,
		p.OrganizerID, p.VenueID, p.Title, slug, p.Description, p.Category, p.CoverImageURL,
		p.VenueName, p.VenueAddress, p.StartsAt, p.EndsAt, p.Timezone,
		p.Visibility, p.SeatingMode, p.Capacity,
		p.RegistrationOpensAt, p.RegistrationClosesAt, p.RefundPolicy))
	if err != nil {
		return Event{}, mapError(err)
	}
	return event, nil
}

// GetByID returns one event.
func (s *EventStore) GetByID(ctx context.Context, id uuid.UUID) (Event, error) {
	event, err := scanEvent(s.pool.QueryRow(ctx,
		`SELECT `+eventColumns+` FROM events WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, mapError(err)
	}
	return event, nil
}

// GetBySlug returns one event by its URL slug.
func (s *EventStore) GetBySlug(ctx context.Context, slug string) (Event, error) {
	event, err := scanEvent(s.pool.QueryRow(ctx,
		`SELECT `+eventColumns+` FROM events WHERE slug = $1`, slug))
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, mapError(err)
	}
	return event, nil
}

// ListEventsFilter narrows and pages a listing.
type ListEventsFilter struct {
	OrganizerID  *uuid.UUID
	Statuses     []string
	Visibilities []string
	Category     *string
	Search       *string
	StartsAfter  *time.Time
	StartsBefore *time.Time
	Limit        int
	Offset       int
}

// List returns a page of events plus the total number of matches.
func (s *EventStore) List(ctx context.Context, f ListEventsFilter) ([]Event, int, error) {
	var (
		where []string
		args  []any
	)
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if f.OrganizerID != nil {
		add("organizer_id = $%d", *f.OrganizerID)
	}
	if len(f.Statuses) > 0 {
		add("status::text = ANY($%d)", f.Statuses)
	}
	if len(f.Visibilities) > 0 {
		add("visibility::text = ANY($%d)", f.Visibilities)
	}
	if f.Category != nil {
		add("category = $%d", *f.Category)
	}
	if f.Search != nil {
		args = append(args, *f.Search)
		where = append(where, fmt.Sprintf(
			"(title ILIKE '%%' || $%d || '%%' OR description ILIKE '%%' || $%[1]d || '%%')",
			len(args)))
	}
	if f.StartsAfter != nil {
		add("starts_at >= $%d", *f.StartsAfter)
	}
	if f.StartsBefore != nil {
		add("starts_at <= $%d", *f.StartsBefore)
	}

	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM events`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapError(err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	args = append(args, limit, f.Offset)
	query := `SELECT ` + eventColumns + ` FROM events` + clause +
		fmt.Sprintf(" ORDER BY starts_at ASC, created_at ASC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, mapError(err)
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

// UpdateEventParams carries the fields a PATCH may change. An unset Optional is
// left untouched; an Optional explicitly set to null clears a nullable column.
type UpdateEventParams struct {
	Title                Optional[string]
	Slug                 Optional[string]
	Description          Optional[string]
	Category             Optional[string]
	CoverImageURL        Optional[string]
	VenueName            Optional[string]
	VenueAddress         Optional[string]
	VenueID              Optional[uuid.UUID]
	StartsAt             Optional[time.Time]
	EndsAt               Optional[time.Time]
	Timezone             Optional[string]
	Visibility           Optional[string]
	SeatingMode          Optional[string]
	Capacity             Optional[int]
	RegistrationOpensAt  Optional[time.Time]
	RegistrationClosesAt Optional[time.Time]
	RefundPolicy         Optional[string]
}

// Update applies the supplied fields. It returns the event unchanged when the
// caller supplied no fields at all.
func (s *EventStore) Update(ctx context.Context, id uuid.UUID, p UpdateEventParams) (Event, error) {
	var (
		sets []string
		args []any
	)
	// A nil value clears the column; Ptr() returns nil for an explicit null.
	set := func(column string, present bool, value any) {
		if !present {
			return
		}
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	set("title", p.Title.Set, p.Title.Ptr())
	set("slug", p.Slug.Set, p.Slug.Ptr())
	set("description", p.Description.Set, p.Description.Ptr())
	set("category", p.Category.Set, p.Category.Ptr())
	set("cover_image_url", p.CoverImageURL.Set, p.CoverImageURL.Ptr())
	set("venue_name", p.VenueName.Set, p.VenueName.Ptr())
	set("venue_address", p.VenueAddress.Set, p.VenueAddress.Ptr())
	set("venue_id", p.VenueID.Set, p.VenueID.Ptr())
	set("starts_at", p.StartsAt.Set, p.StartsAt.Ptr())
	set("ends_at", p.EndsAt.Set, p.EndsAt.Ptr())
	set("timezone", p.Timezone.Set, p.Timezone.Ptr())
	set("capacity", p.Capacity.Set, p.Capacity.Ptr())
	set("registration_opens_at", p.RegistrationOpensAt.Set, p.RegistrationOpensAt.Ptr())
	set("registration_closes_at", p.RegistrationClosesAt.Set, p.RegistrationClosesAt.Ptr())
	set("refund_policy", p.RefundPolicy.Set, p.RefundPolicy.Ptr())

	// Enum columns need an explicit cast from text.
	if p.Visibility.Set && p.Visibility.Valid {
		args = append(args, p.Visibility.Value)
		sets = append(sets, fmt.Sprintf("visibility = $%d::event_visibility", len(args)))
	}
	if p.SeatingMode.Set && p.SeatingMode.Valid {
		args = append(args, p.SeatingMode.Value)
		sets = append(sets, fmt.Sprintf("seating_mode = $%d::seating_mode", len(args)))
	}

	if len(sets) == 0 {
		return s.GetByID(ctx, id)
	}

	args = append(args, id)
	query := `UPDATE events SET ` + strings.Join(sets, ", ") +
		fmt.Sprintf(" WHERE id = $%d RETURNING ", len(args)) + eventColumns

	event, err := scanEvent(s.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, mapError(err)
	}
	return event, nil
}

// Publish moves a draft or unpublished event to published.
func (s *EventStore) Publish(ctx context.Context, id uuid.UUID) (Event, error) {
	event, err := scanEvent(s.pool.QueryRow(ctx, `
		UPDATE events
		   SET status = 'published', published_at = COALESCE(published_at, now())
		 WHERE id = $1
		RETURNING `+eventColumns, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, mapError(err)
	}
	return event, nil
}

// Unpublish hides a published event without cancelling it.
func (s *EventStore) Unpublish(ctx context.Context, id uuid.UUID) (Event, error) {
	event, err := scanEvent(s.pool.QueryRow(ctx, `
		UPDATE events SET status = 'unpublished' WHERE id = $1
		RETURNING `+eventColumns, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, mapError(err)
	}
	return event, nil
}

// Cancel marks an event cancelled and records when.
func (s *EventStore) Cancel(ctx context.Context, id uuid.UUID) (Event, error) {
	event, err := scanEvent(s.pool.QueryRow(ctx, `
		UPDATE events
		   SET status = 'cancelled', cancelled_at = COALESCE(cancelled_at, now())
		 WHERE id = $1
		RETURNING `+eventColumns, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, mapError(err)
	}
	return event, nil
}

// SetStatus moves an event to an arbitrary status. Used by platform moderation;
// organizer-facing transitions go through Publish, Unpublish and Cancel, which
// also maintain their timestamps.
func (s *EventStore) SetStatus(ctx context.Context, id uuid.UUID, status string) (Event, error) {
	event, err := scanEvent(s.pool.QueryRow(ctx, `
		UPDATE events SET status = $2::event_status WHERE id = $1
		RETURNING `+eventColumns, id, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, mapError(err)
	}
	return event, nil
}

// Delete removes an event permanently.
func (s *EventStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM events WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// HasOrders reports whether any order references the event. An event with
// orders must be cancelled rather than deleted, so ticket history survives.
func (s *EventStore) HasOrders(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM orders WHERE event_id = $1)`, id).Scan(&exists)
	return exists, mapError(err)
}

// AffectedTicketHolder is one order that a cancellation invalidates.
type AffectedTicketHolder struct {
	UserID      *uuid.UUID
	OrderID     uuid.UUID
	OrderNumber string
	BuyerName   string
	BuyerEmail  string
	TicketCount int
}

// TicketHolders lists the people holding live tickets for an event, one entry
// per order, so a cancellation can tell each of them once (SRS 4.10).
func (s *EventStore) TicketHolders(
	ctx context.Context, eventID uuid.UUID,
) ([]AffectedTicketHolder, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.buyer_user_id, o.id, o.order_number, o.buyer_name, o.buyer_email::text,
		       count(t.id)
		  FROM orders o
		  JOIN tickets t ON t.order_id = o.id
		 WHERE o.event_id = $1
		   AND t.status IN ('valid', 'checked_in')
		 GROUP BY o.id
		 ORDER BY o.created_at`, eventID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []AffectedTicketHolder{}
	for rows.Next() {
		var h AffectedTicketHolder
		if err := rows.Scan(&h.UserID, &h.OrderID, &h.OrderNumber,
			&h.BuyerName, &h.BuyerEmail, &h.TicketCount); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
