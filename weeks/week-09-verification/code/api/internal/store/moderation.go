package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Moderation queue and platform settings (SRS 4.12).
//
// SRS 4.12 asks administrators to "review reported events" and to "configure
// activation fees and platform settings". Both were listed requirements with
// nowhere to put the data.

// Report reasons and statuses, mirroring the event_report_* enums.
const (
	ReportStatusOpen      = "open"
	ReportStatusReviewing = "reviewing"
	ReportStatusUpheld    = "upheld"
	ReportStatusDismissed = "dismissed"
)

// ReportReasons is the closed set an attendee may pick from.
var ReportReasons = []string{"fraud", "misleading", "inappropriate", "spam", "copyright", "other"}

// ReportStatuses is the closed set a moderator may move a report to.
var ReportStatuses = []string{
	ReportStatusOpen, ReportStatusReviewing, ReportStatusUpheld, ReportStatusDismissed,
}

// ErrAlreadyReported reports a second open complaint from the same person
// about the same event.
var ErrAlreadyReported = errors.New("this event is already reported by this user")

// EventReport is one complaint about an event.
type EventReport struct {
	ID             uuid.UUID  `json:"id"`
	EventID        uuid.UUID  `json:"event_id"`
	EventTitle     string     `json:"event_title"`
	EventSlug      string     `json:"event_slug"`
	EventStatus    string     `json:"event_status"`
	OrganizerID    uuid.UUID  `json:"organizer_id"`
	OrganizerEmail string     `json:"organizer_email"`
	ReporterID     *uuid.UUID `json:"reporter_user_id,omitempty"`
	ReporterEmail  *string    `json:"reporter_email,omitempty"`
	Reason         string     `json:"reason"`
	Details        *string    `json:"details,omitempty"`
	Status         string     `json:"status"`
	ReviewedBy     *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	Resolution     *string    `json:"resolution,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ModerationStore reads and writes the report queue and platform settings.
type ModerationStore struct {
	pool *pgxpool.Pool
}

// NewModerationStore builds a ModerationStore.
func NewModerationStore(pool *pgxpool.Pool) *ModerationStore {
	return &ModerationStore{pool: pool}
}

const reportColumns = `
	r.id, r.event_id, e.title, e.slug, e.status::text, e.organizer_id, u.email,
	r.reporter_user_id, reporter.email, r.reason::text, r.details, r.status::text,
	r.reviewed_by, r.reviewed_at, r.resolution, r.created_at`

const reportJoins = `
	  FROM event_reports r
	  JOIN events e ON e.id = r.event_id
	  JOIN users  u ON u.id = e.organizer_id
	  LEFT JOIN users reporter ON reporter.id = r.reporter_user_id`

func scanReport(row pgx.Row) (EventReport, error) {
	var rep EventReport
	err := row.Scan(&rep.ID, &rep.EventID, &rep.EventTitle, &rep.EventSlug, &rep.EventStatus,
		&rep.OrganizerID, &rep.OrganizerEmail, &rep.ReporterID, &rep.ReporterEmail,
		&rep.Reason, &rep.Details, &rep.Status, &rep.ReviewedBy, &rep.ReviewedAt,
		&rep.Resolution, &rep.CreatedAt)
	return rep, err
}

// ReportParams is a new complaint.
type ReportParams struct {
	EventID    uuid.UUID
	ReporterID uuid.UUID
	Reason     string
	Details    string
}

// Report files a complaint about an event (SRS 4.12).
//
// The partial unique index event_reports_one_open_per_reporter_uidx makes a
// second open report from the same person impossible at the database level, so
// a double click is a conflict rather than two rows in the queue.
func (s *ModerationStore) Report(ctx context.Context, p ReportParams) (EventReport, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO event_reports (event_id, reporter_user_id, reason, details)
		VALUES ($1, $2, $3::event_report_reason, $4)
		RETURNING id`,
		p.EventID, p.ReporterID, p.Reason, nullableString(p.Details)).Scan(&id)
	if err != nil {
		if isUniqueViolation(err, "event_reports_one_open_per_reporter_uidx") {
			return EventReport{}, ErrAlreadyReported
		}
		return EventReport{}, mapError(err)
	}
	return s.GetReport(ctx, id)
}

// GetReport returns one report with its event and organizer context.
func (s *ModerationStore) GetReport(ctx context.Context, id uuid.UUID) (EventReport, error) {
	rep, err := scanReport(s.pool.QueryRow(ctx,
		`SELECT `+reportColumns+reportJoins+` WHERE r.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return EventReport{}, ErrNotFound
	}
	if err != nil {
		return EventReport{}, mapError(err)
	}
	return rep, nil
}

// ListReports returns the moderation queue, newest first. An empty status
// filter returns everything; "open" is what an administrator wants by default.
func (s *ModerationStore) ListReports(ctx context.Context, status string, limit int) ([]EventReport, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := s.pool.Query(ctx, `
		SELECT `+reportColumns+reportJoins+`
		 WHERE ($1::text = '' OR r.status::text = $1)
		 ORDER BY r.created_at DESC
		 LIMIT $2`, status, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	reports := []EventReport{}
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

// ReviewParams is a moderator's decision on a report.
type ReviewParams struct {
	ReportID   uuid.UUID
	ReviewerID uuid.UUID
	Status     string
	Resolution string
}

// Review records a decision. Moving a report out of 'open'/'reviewing'
// requires a reviewer and a timestamp - event_reports_reviewed_chk enforces
// that, so a decided report always says who decided it.
func (s *ModerationStore) Review(ctx context.Context, p ReviewParams) (EventReport, error) {
	decided := p.Status == ReportStatusUpheld || p.Status == ReportStatusDismissed

	tag, err := s.pool.Exec(ctx, `
		UPDATE event_reports
		   SET status      = $2::event_report_status,
		       resolution  = $3,
		       reviewed_by = CASE WHEN $4::boolean THEN $5::uuid ELSE reviewed_by END,
		       reviewed_at = CASE WHEN $4::boolean THEN now()    ELSE reviewed_at END
		 WHERE id = $1`,
		p.ReportID, p.Status, nullableString(p.Resolution), decided, p.ReviewerID)
	if err != nil {
		return EventReport{}, mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return EventReport{}, ErrNotFound
	}
	return s.GetReport(ctx, p.ReportID)
}

// CountOpenReports is the "needs attention" figure on the admin dashboard.
func (s *ModerationStore) CountOpenReports(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM event_reports WHERE status IN ('open', 'reviewing')`).Scan(&n)
	return n, mapError(err)
}

// --- Platform settings (SRS 4.12) --------------------------------------------

// SettingActivationFee is the key holding the paid-sales activation fee. It
// was a Go constant until now, so changing it needed a rebuild.
const SettingActivationFee = "activation_fee_kzt"

// Setting is one configurable value.
type Setting struct {
	Key         string     `json:"key"`
	Value       any        `json:"value"`
	Description *string    `json:"description,omitempty"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListSettings returns every setting, alphabetically.
func (s *ModerationStore) ListSettings(ctx context.Context) ([]Setting, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT key, value, description, updated_by, updated_at
		   FROM platform_settings ORDER BY key`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	settings := []Setting{}
	for rows.Next() {
		var (
			set Setting
			raw []byte
		)
		if err := rows.Scan(&set.Key, &raw, &set.Description, &set.UpdatedBy, &set.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &set.Value); err != nil {
			return nil, err
		}
		settings = append(settings, set)
	}
	return settings, rows.Err()
}

// SetSetting updates an existing setting. It deliberately will not create one:
// a typo in a key would otherwise silently add a setting nothing reads.
func (s *ModerationStore) SetSetting(
	ctx context.Context, key string, value any, actorID uuid.UUID,
) (Setting, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return Setting{}, err
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE platform_settings
		   SET value = $2::jsonb, updated_by = $3, updated_at = now()
		 WHERE key = $1`, key, string(raw), actorID)
	if err != nil {
		return Setting{}, mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return Setting{}, ErrNotFound
	}

	settings, err := s.ListSettings(ctx)
	if err != nil {
		return Setting{}, err
	}
	for _, set := range settings {
		if set.Key == key {
			return set, nil
		}
	}
	return Setting{}, ErrNotFound
}

// ActivationFeeKZT reads the configured fee, falling back to the historical
// constant if the row is missing or malformed. A misconfigured setting must
// not stop an organizer activating paid sales.
func (s *ModerationStore) ActivationFeeKZT(ctx context.Context) string {
	var raw []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT value FROM platform_settings WHERE key = $1`, SettingActivationFee,
	).Scan(&raw); err != nil {
		return SimulatedActivationFeeKZT
	}

	var fee string
	if err := json.Unmarshal(raw, &fee); err != nil || blankAmount(fee) {
		return SimulatedActivationFeeKZT
	}
	return fee
}

// blankAmount rejects an empty or non-numeric fee before it reaches a numeric
// column, where it would surface as a 500 rather than a fallback.
func blankAmount(v string) bool {
	if v == "" {
		return true
	}
	digits := false
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9':
			digits = true
		case r == '.':
		default:
			return true
		}
	}
	return !digits
}
