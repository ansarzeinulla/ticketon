package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestCreateEventWithToken is Phase 2 success criterion 3: an authenticated
// POST to /events returns 201 and the row is physically in PostgreSQL.
func TestCreateEventWithToken(t *testing.T) {
	c := newClient(t)
	acc := c.register("organizer")

	body := validEventBody("Almaty Winter Jazz Night")
	res := c.post("/api/v1/events", acc.Token, body)
	requireStatus(t, res, http.StatusCreated)

	if loc := res.Header.Get("Location"); loc == "" {
		t.Error("a 201 response should carry a Location header")
	}

	event := res.event()
	if event["title"] != "Almaty Winter Jazz Night" {
		t.Errorf("title = %v, want the submitted title", event["title"])
	}
	if event["status"] != "draft" {
		t.Errorf("status = %v, want draft - publishing is a separate step", event["status"])
	}
	if event["slug"] != "almaty-winter-jazz-night" {
		t.Errorf("slug = %v, want almaty-winter-jazz-night", event["slug"])
	}
	if event["organizer_id"] != acc.ID.String() {
		t.Errorf("organizer_id = %v, want %v", event["organizer_id"], acc.ID)
	}

	// The row must be readable straight from the database.
	eventID := uuid.MustParse(res.eventString("id"))
	var (
		title       string
		slug        string
		status      string
		visibility  string
		seating     string
		timezone    string
		capacity    *int
		organizerID uuid.UUID
		startsAt    time.Time
		endsAt      time.Time
	)
	err := c.pool.QueryRow(t.Context(), `
		SELECT title, slug, status::text, visibility::text, seating_mode::text,
		       timezone, capacity, organizer_id, starts_at, ends_at
		  FROM events WHERE id = $1`, eventID).
		Scan(&title, &slug, &status, &visibility, &seating, &timezone,
			&capacity, &organizerID, &startsAt, &endsAt)
	if err != nil {
		t.Fatalf("the created event is not in the database: %v", err)
	}

	if title != "Almaty Winter Jazz Night" {
		t.Errorf("db title = %q, want the submitted title", title)
	}
	if status != "draft" {
		t.Errorf("db status = %q, want draft", status)
	}
	if visibility != "public" {
		t.Errorf("db visibility = %q, want the public default", visibility)
	}
	if seating != "general_admission" {
		t.Errorf("db seating_mode = %q, want the general_admission default", seating)
	}
	if timezone != "Asia/Almaty" {
		t.Errorf("db timezone = %q, want Asia/Almaty", timezone)
	}
	if capacity == nil || *capacity != 200 {
		t.Errorf("db capacity = %v, want 200", capacity)
	}
	if organizerID != acc.ID {
		t.Errorf("db organizer_id = %v, want %v", organizerID, acc.ID)
	}
	if !endsAt.After(startsAt) {
		t.Errorf("db ends_at %v is not after starts_at %v", endsAt, startsAt)
	}
}

// Creating an event is what makes someone an organizer, so a freshly
// registered account can go straight from register to POST /events.
func TestCreateEventGrantsOrganizerRole(t *testing.T) {
	c := newClient(t)
	acc := c.register("newroles")

	me := c.get("/api/v1/auth/me", acc.Token)
	requireStatus(t, me, http.StatusOK)
	if roles := me.Body["user"].(map[string]any)["roles"]; fmt.Sprint(roles) != "[attendee]" {
		t.Fatalf("roles before creating an event = %v, want [attendee]", roles)
	}

	c.createEvent(acc.Token, "Role Granting Event")

	var hasOrganizer bool
	if err := c.pool.QueryRow(t.Context(),
		`SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role = 'organizer')`,
		acc.ID).Scan(&hasOrganizer); err != nil {
		t.Fatalf("read roles: %v", err)
	}
	if !hasOrganizer {
		t.Error("creating an event did not grant the organizer role")
	}

	me = c.get("/api/v1/auth/me", acc.Token)
	roles := fmt.Sprint(me.Body["user"].(map[string]any)["roles"])
	if roles != "[attendee organizer]" {
		t.Errorf("roles after creating an event = %v, want [attendee organizer]", roles)
	}
}

func TestCreateEventRequiresAuthentication(t *testing.T) {
	c := newClient(t)

	res := c.post("/api/v1/events", "", validEventBody("Anonymous Event"))
	requireErrorCode(t, res, http.StatusUnauthorized, "unauthorized")

	var count int
	if err := c.pool.QueryRow(t.Context(), `SELECT count(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 0 {
		t.Errorf("%d events were created without a token, want 0", count)
	}
}

func TestCreateEventValidation(t *testing.T) {
	c := newClient(t)
	acc := c.register("validation")

	start := time.Now().Add(24 * time.Hour).UTC()

	tests := []struct {
		name      string
		mutate    func(map[string]any)
		wantField string
	}{
		{"missing title", func(b map[string]any) { delete(b, "title") }, "title"},
		{"blank title", func(b map[string]any) { b["title"] = "   " }, "title"},
		{"missing start", func(b map[string]any) { delete(b, "starts_at") }, "starts_at"},
		{"missing end", func(b map[string]any) { delete(b, "ends_at") }, "ends_at"},
		{"end before start", func(b map[string]any) {
			b["starts_at"] = start.Format(time.RFC3339)
			b["ends_at"] = start.Add(-time.Hour).Format(time.RFC3339)
		}, "ends_at"},
		{"end equals start", func(b map[string]any) {
			b["starts_at"] = start.Format(time.RFC3339)
			b["ends_at"] = start.Format(time.RFC3339)
		}, "ends_at"},
		{"zero capacity", func(b map[string]any) { b["capacity"] = 0 }, "capacity"},
		{"negative capacity", func(b map[string]any) { b["capacity"] = -5 }, "capacity"},
		{"unknown timezone", func(b map[string]any) { b["timezone"] = "Mars/Olympus" }, "timezone"},
		{"unknown visibility", func(b map[string]any) { b["visibility"] = "semi-secret" }, "visibility"},
		{"unknown seating mode", func(b map[string]any) { b["seating_mode"] = "standing" }, "seating_mode"},
		{"assigned seating without venue", func(b map[string]any) { b["seating_mode"] = "assigned_seating" }, "venue_id"},
		{"malformed venue id", func(b map[string]any) { b["venue_id"] = "not-a-uuid" }, "venue_id"},
		{"unusable slug", func(b map[string]any) { b["slug"] = "!!!" }, "slug"},
		{"registration closes before it opens", func(b map[string]any) {
			b["registration_opens_at"] = start.Format(time.RFC3339)
			b["registration_closes_at"] = start.Add(-time.Hour).Format(time.RFC3339)
		}, "registration_closes_at"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := validEventBody("Validation Case")
			tt.mutate(body)

			res := c.post("/api/v1/events", acc.Token, body)
			requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")

			if _, ok := res.errorFields()[tt.wantField]; !ok {
				t.Errorf("error fields = %v, want an entry for %q", res.errorFields(), tt.wantField)
			}
		})
	}

	var count int
	if err := c.pool.QueryRow(t.Context(), `SELECT count(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 0 {
		t.Errorf("%d invalid events reached the database, want 0", count)
	}
}

// Two events may legitimately share a title, so the slug is made unique
// automatically rather than rejecting the second one.
func TestCreateEventDerivesUniqueSlugs(t *testing.T) {
	c := newClient(t)
	acc := c.register("slugs")

	_, first := c.createEvent(acc.Token, "Repeated Title")
	_, second := c.createEvent(acc.Token, "Repeated Title")
	_, third := c.createEvent(acc.Token, "Repeated Title")

	if first.eventString("slug") != "repeated-title" {
		t.Errorf("first slug = %q, want repeated-title", first.eventString("slug"))
	}
	if second.eventString("slug") != "repeated-title-2" {
		t.Errorf("second slug = %q, want repeated-title-2", second.eventString("slug"))
	}
	if third.eventString("slug") != "repeated-title-3" {
		t.Errorf("third slug = %q, want repeated-title-3", third.eventString("slug"))
	}
}

// An explicit slug is the client's choice, so a clash is reported rather than
// silently renamed.
func TestCreateEventRejectsDuplicateExplicitSlug(t *testing.T) {
	c := newClient(t)
	acc := c.register("explicitslug")

	body := validEventBody("First Event")
	body["slug"] = "my-chosen-slug"
	requireStatus(t, c.post("/api/v1/events", acc.Token, body), http.StatusCreated)

	body = validEventBody("Second Event")
	body["slug"] = "my-chosen-slug"
	requireErrorCode(t, c.post("/api/v1/events", acc.Token, body), http.StatusConflict, "conflict")
}

func TestCreateEventWritesAuditEntry(t *testing.T) {
	c := newClient(t)
	acc := c.register("audit")
	eventID, _ := c.createEvent(acc.Token, "Audited Event")

	var action, entityType string
	var actor uuid.UUID
	err := c.pool.QueryRow(t.Context(), `
		SELECT action, entity_type, actor_user_id FROM audit_logs
		 WHERE event_id = $1 ORDER BY created_at DESC LIMIT 1`, eventID).
		Scan(&action, &entityType, &actor)
	if err != nil {
		t.Fatalf("no audit entry for the created event: %v", err)
	}

	if action != "event.created" {
		t.Errorf("action = %q, want event.created", action)
	}
	if entityType != "event" {
		t.Errorf("entity_type = %q, want event", entityType)
	}
	if actor != acc.ID {
		t.Errorf("actor_user_id = %v, want %v", actor, acc.ID)
	}
}

// --- reading -----------------------------------------------------------------

func TestGetEventVisibilityRules(t *testing.T) {
	c := newClient(t)
	owner := c.register("owner")
	other := c.register("other")

	eventID, _ := c.createEvent(owner.Token, "Draft Event")
	path := "/api/v1/events/" + eventID.String()

	// A draft is only visible to its organizer.
	requireStatus(t, c.get(path, owner.Token), http.StatusOK)
	requireErrorCode(t, c.get(path, ""), http.StatusNotFound, "not_found")
	requireErrorCode(t, c.get(path, other.Token), http.StatusNotFound, "not_found")

	// Publishing makes it public.
	requireStatus(t, c.post(path+"/publish", owner.Token, nil), http.StatusOK)
	requireStatus(t, c.get(path, ""), http.StatusOK)
	requireStatus(t, c.get(path, other.Token), http.StatusOK)
}

func TestGetPrivateEventStaysHiddenAfterPublishing(t *testing.T) {
	c := newClient(t)
	owner := c.register("privateowner")
	other := c.register("privateother")

	body := validEventBody("Private Party")
	body["visibility"] = "private"
	res := c.post("/api/v1/events", owner.Token, body)
	requireStatus(t, res, http.StatusCreated)

	path := "/api/v1/events/" + res.eventString("id")
	requireStatus(t, c.post(path+"/publish", owner.Token, nil), http.StatusOK)

	requireStatus(t, c.get(path, owner.Token), http.StatusOK)
	requireErrorCode(t, c.get(path, ""), http.StatusNotFound, "not_found")
	requireErrorCode(t, c.get(path, other.Token), http.StatusNotFound, "not_found")
}

func TestGetEventNotFoundAndMalformedID(t *testing.T) {
	c := newClient(t)

	requireErrorCode(t, c.get("/api/v1/events/"+uuid.NewString(), ""),
		http.StatusNotFound, "not_found")
	requireErrorCode(t, c.get("/api/v1/events/not-a-uuid", ""),
		http.StatusBadRequest, "validation_failed")
}

func TestListEventsShowsOnlyPublishedPublicEvents(t *testing.T) {
	c := newClient(t)
	owner := c.register("listing")

	draftID, _ := c.createEvent(owner.Token, "Still A Draft")

	publishedID, _ := c.createEvent(owner.Token, "Published Event")
	requireStatus(t, c.post("/api/v1/events/"+publishedID.String()+"/publish", owner.Token, nil), http.StatusOK)

	privateBody := validEventBody("Published But Private")
	privateBody["visibility"] = "private"
	privateRes := c.post("/api/v1/events", owner.Token, privateBody)
	requireStatus(t, privateRes, http.StatusCreated)
	requireStatus(t, c.post("/api/v1/events/"+privateRes.eventString("id")+"/publish", owner.Token, nil), http.StatusOK)

	res := c.get("/api/v1/events", "")
	requireStatus(t, res, http.StatusOK)

	events, _ := res.Body["events"].([]any)
	if len(events) != 1 {
		t.Fatalf("listing returned %d events, want only the published public one: %s", len(events), res.Raw)
	}
	if id := events[0].(map[string]any)["id"]; id != publishedID.String() {
		t.Errorf("listed event id = %v, want %v", id, publishedID)
	}
	if total, _ := res.Body["total"].(float64); int(total) != 1 {
		t.Errorf("total = %v, want 1", res.Body["total"])
	}

	_ = draftID
}

func TestListEventsFiltersAndPages(t *testing.T) {
	c := newClient(t)
	owner := c.register("filters")

	for i := 0; i < 5; i++ {
		body := validEventBody(fmt.Sprintf("Concert Number %d", i))
		body["category"] = "music"
		start := time.Now().Add(time.Duration(i+1) * 24 * time.Hour).UTC().Truncate(time.Second)
		body["starts_at"] = start.Format(time.RFC3339)
		body["ends_at"] = start.Add(2 * time.Hour).Format(time.RFC3339)

		res := c.post("/api/v1/events", owner.Token, body)
		requireStatus(t, res, http.StatusCreated)
		requireStatus(t, c.post("/api/v1/events/"+res.eventString("id")+"/publish", owner.Token, nil), http.StatusOK)
	}

	lecture := validEventBody("Open Lecture")
	lecture["category"] = "education"
	res := c.post("/api/v1/events", owner.Token, lecture)
	requireStatus(t, res, http.StatusCreated)
	requireStatus(t, c.post("/api/v1/events/"+res.eventString("id")+"/publish", owner.Token, nil), http.StatusOK)

	t.Run("category filter", func(t *testing.T) {
		res := c.get("/api/v1/events?category=education", "")
		requireStatus(t, res, http.StatusOK)
		events := res.Body["events"].([]any)
		if len(events) != 1 {
			t.Fatalf("got %d events, want 1", len(events))
		}
		if events[0].(map[string]any)["title"] != "Open Lecture" {
			t.Errorf("filtered event = %v, want Open Lecture", events[0])
		}
	})

	t.Run("text search", func(t *testing.T) {
		res := c.get("/api/v1/events?q=Concert", "")
		requireStatus(t, res, http.StatusOK)
		if got := len(res.Body["events"].([]any)); got != 5 {
			t.Errorf("got %d events, want 5", got)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		res := c.get("/api/v1/events?limit=2&offset=0", "")
		requireStatus(t, res, http.StatusOK)
		if got := len(res.Body["events"].([]any)); got != 2 {
			t.Errorf("page size = %d, want 2", got)
		}
		if total, _ := res.Body["total"].(float64); int(total) != 6 {
			t.Errorf("total = %v, want 6 regardless of the page size", res.Body["total"])
		}

		page2 := c.get("/api/v1/events?limit=2&offset=2", "")
		requireStatus(t, page2, http.StatusOK)

		firstID := res.Body["events"].([]any)[0].(map[string]any)["id"]
		secondPageID := page2.Body["events"].([]any)[0].(map[string]any)["id"]
		if firstID == secondPageID {
			t.Error("offset had no effect: both pages start with the same event")
		}
	})

	t.Run("invalid pagination", func(t *testing.T) {
		for _, query := range []string{"?limit=0", "?limit=1000", "?limit=abc", "?offset=-1"} {
			res := c.get("/api/v1/events"+query, "")
			requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")
		}
	})
}

func TestListMyEventsShowsOwnDraftsOnly(t *testing.T) {
	c := newClient(t)
	owner := c.register("mine")
	other := c.register("notmine")

	c.createEvent(owner.Token, "My Draft One")
	c.createEvent(owner.Token, "My Draft Two")
	c.createEvent(other.Token, "Someone Else's Event")

	res := c.get("/api/v1/events/mine", owner.Token)
	requireStatus(t, res, http.StatusOK)

	events := res.Body["events"].([]any)
	if len(events) != 2 {
		t.Fatalf("got %d events, want the 2 owned by this organizer: %s", len(events), res.Raw)
	}
	for _, e := range events {
		if got := e.(map[string]any)["organizer_id"]; got != owner.ID.String() {
			t.Errorf("listing leaked an event owned by %v", got)
		}
	}

	requireErrorCode(t, c.get("/api/v1/events/mine", ""), http.StatusUnauthorized, "unauthorized")
}

// --- updating ----------------------------------------------------------------

func TestUpdateEvent(t *testing.T) {
	c := newClient(t)
	owner := c.register("updater")
	eventID, created := c.createEvent(owner.Token, "Original Title")
	path := "/api/v1/events/" + eventID.String()

	res := c.patch(path, owner.Token, map[string]any{
		"title":    "Updated Title",
		"capacity": 500,
	})
	requireStatus(t, res, http.StatusOK)

	if res.eventString("title") != "Updated Title" {
		t.Errorf("title = %v, want Updated Title", res.event()["title"])
	}
	if capacity, _ := res.event()["capacity"].(float64); int(capacity) != 500 {
		t.Errorf("capacity = %v, want 500", res.event()["capacity"])
	}
	// Fields that were not sent must be untouched.
	if res.eventString("category") != created.eventString("category") {
		t.Errorf("category changed to %v without being sent", res.event()["category"])
	}

	var title string
	var capacity *int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT title, capacity FROM events WHERE id = $1`, eventID).Scan(&title, &capacity); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if title != "Updated Title" || capacity == nil || *capacity != 500 {
		t.Errorf("db state = (%q, %v), want (\"Updated Title\", 500)", title, capacity)
	}
}

// A PATCH must tell "field absent" from "field set to null".
func TestUpdateEventClearsFieldWithExplicitNull(t *testing.T) {
	c := newClient(t)
	owner := c.register("nulls")
	eventID, _ := c.createEvent(owner.Token, "Has A Description")
	path := "/api/v1/events/" + eventID.String()

	res := c.patch(path, owner.Token, map[string]any{"description": nil})
	requireStatus(t, res, http.StatusOK)

	var description *string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT description FROM events WHERE id = $1`, eventID).Scan(&description); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if description != nil {
		t.Errorf("description = %v, want NULL after an explicit null", *description)
	}

	// Capacity was never mentioned, so it must survive.
	var capacity *int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT capacity FROM events WHERE id = $1`, eventID).Scan(&capacity); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if capacity == nil || *capacity != 200 {
		t.Errorf("capacity = %v, want the original 200", capacity)
	}
}

func TestUpdateEventAuthorization(t *testing.T) {
	c := newClient(t)
	owner := c.register("patchowner")
	other := c.register("patchother")

	eventID, _ := c.createEvent(owner.Token, "Protected Event")
	path := "/api/v1/events/" + eventID.String()
	body := map[string]any{"title": "Hijacked"}

	requireErrorCode(t, c.patch(path, "", body), http.StatusUnauthorized, "unauthorized")
	requireErrorCode(t, c.patch(path, other.Token, body), http.StatusForbidden, "forbidden")

	var title string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT title FROM events WHERE id = $1`, eventID).Scan(&title); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if title != "Protected Event" {
		t.Errorf("title = %q, want it unchanged", title)
	}
}

func TestUpdateEventValidation(t *testing.T) {
	c := newClient(t)
	owner := c.register("patchvalidation")
	eventID, created := c.createEvent(owner.Token, "Validated Event")
	path := "/api/v1/events/" + eventID.String()

	// The stored start time, so "end before start" can be built from it.
	storedStart, err := time.Parse(time.RFC3339, created.eventString("starts_at"))
	if err != nil {
		t.Fatalf("parse starts_at: %v", err)
	}

	tests := []struct {
		name      string
		body      map[string]any
		wantField string
	}{
		{"blank title", map[string]any{"title": "   "}, "title"},
		{"null title", map[string]any{"title": nil}, "title"},
		{"end before stored start", map[string]any{
			"ends_at": storedStart.Add(-time.Hour).Format(time.RFC3339)}, "ends_at"},
		{"zero capacity", map[string]any{"capacity": 0}, "capacity"},
		{"bad timezone", map[string]any{"timezone": "Nowhere/Special"}, "timezone"},
		{"bad visibility", map[string]any{"visibility": "invisible"}, "visibility"},
		{"assigned seating without venue", map[string]any{"seating_mode": "assigned_seating"}, "venue_id"},
		{"unusable slug", map[string]any{"slug": "***"}, "slug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.patch(path, owner.Token, tt.body)
			requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")

			if _, ok := res.errorFields()[tt.wantField]; !ok {
				t.Errorf("error fields = %v, want an entry for %q", res.errorFields(), tt.wantField)
			}
		})
	}
}

func TestUpdateEventWithEmptyBodyChangesNothing(t *testing.T) {
	c := newClient(t)
	owner := c.register("emptypatch")
	eventID, created := c.createEvent(owner.Token, "Untouched Event")

	res := c.patch("/api/v1/events/"+eventID.String(), owner.Token, map[string]any{})
	requireStatus(t, res, http.StatusOK)

	if res.eventString("title") != created.eventString("title") {
		t.Errorf("title changed to %q on an empty patch", res.eventString("title"))
	}
}

// --- lifecycle ---------------------------------------------------------------

func TestPublishAndCancelLifecycle(t *testing.T) {
	c := newClient(t)
	owner := c.register("lifecycle")
	eventID, _ := c.createEvent(owner.Token, "Lifecycle Event")
	path := "/api/v1/events/" + eventID.String()

	published := c.post(path+"/publish", owner.Token, nil)
	requireStatus(t, published, http.StatusOK)
	if published.event()["status"] != "published" {
		t.Errorf("status = %v, want published", published.event()["status"])
	}
	if published.event()["published_at"] == nil {
		t.Error("published_at was not recorded")
	}

	// Publishing twice is a conflict, not a silent success.
	requireErrorCode(t, c.post(path+"/publish", owner.Token, nil), http.StatusConflict, "conflict")

	unpublished := c.post(path+"/unpublish", owner.Token, nil)
	requireStatus(t, unpublished, http.StatusOK)
	if unpublished.event()["status"] != "unpublished" {
		t.Errorf("status = %v, want unpublished", unpublished.event()["status"])
	}

	requireStatus(t, c.post(path+"/publish", owner.Token, nil), http.StatusOK)

	cancelled := c.post(path+"/cancel", owner.Token, nil)
	requireStatus(t, cancelled, http.StatusOK)
	if cancelled.event()["status"] != "cancelled" {
		t.Errorf("status = %v, want cancelled", cancelled.event()["status"])
	}
	if cancelled.event()["cancelled_at"] == nil {
		t.Error("cancelled_at was not recorded")
	}

	// A cancelled event cannot be brought back by publishing it again.
	requireErrorCode(t, c.post(path+"/publish", owner.Token, nil), http.StatusConflict, "conflict")
	requireErrorCode(t, c.post(path+"/cancel", owner.Token, nil), http.StatusConflict, "conflict")

	var status string
	var cancelledAt *time.Time
	if err := c.pool.QueryRow(t.Context(),
		`SELECT status::text, cancelled_at FROM events WHERE id = $1`, eventID).
		Scan(&status, &cancelledAt); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if status != "cancelled" || cancelledAt == nil {
		t.Errorf("db state = (%q, %v), want cancelled with a timestamp", status, cancelledAt)
	}
}

// TestPublishRejectsAnEndedEvent covers 2.md LIFE-ERR-03: an event whose end
// time is already in the past cannot be put on sale.
func TestPublishRejectsAnEndedEvent(t *testing.T) {
	c := newClient(t)
	owner := c.register("endedpublish")
	eventID, _ := c.createEvent(owner.Token, "Already Over Event")

	// Age it into the past directly - the create endpoint forbids past dates,
	// but time passing on an unpublished draft is a real situation.
	if _, err := c.pool.Exec(t.Context(), `
		UPDATE events SET starts_at = now() - interval '2 days',
		                  ends_at = now() - interval '1 day'
		 WHERE id = $1`, eventID); err != nil {
		t.Fatalf("age the event: %v", err)
	}

	res := c.post("/api/v1/events/"+eventID.String()+"/publish", owner.Token, nil)
	requireStatus(t, res, http.StatusUnprocessableEntity)
	if _, ok := res.errorFields()["ends_at"]; !ok {
		t.Errorf("error fields = %v, want an entry for ends_at", res.errorFields())
	}

	var status string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT status::text FROM events WHERE id = $1`, eventID).Scan(&status); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if status != "draft" {
		t.Errorf("status = %q, want draft (the publish must not have taken effect)", status)
	}
}

func TestLifecycleActionsRequireOwnership(t *testing.T) {
	c := newClient(t)
	owner := c.register("lifecycleowner")
	other := c.register("lifecycleother")

	eventID, _ := c.createEvent(owner.Token, "Owned Event")
	path := "/api/v1/events/" + eventID.String()

	for _, action := range []string{"/publish", "/unpublish", "/cancel"} {
		requireErrorCode(t, c.post(path+action, "", nil), http.StatusUnauthorized, "unauthorized")
		requireErrorCode(t, c.post(path+action, other.Token, nil), http.StatusForbidden, "forbidden")
	}
}

func TestUnpublishRequiresPublishedEvent(t *testing.T) {
	c := newClient(t)
	owner := c.register("unpublishdraft")
	eventID, _ := c.createEvent(owner.Token, "Draft Event")

	requireErrorCode(t, c.post("/api/v1/events/"+eventID.String()+"/unpublish", owner.Token, nil),
		http.StatusConflict, "conflict")
}

func TestLifecycleWritesTimeline(t *testing.T) {
	c := newClient(t)
	owner := c.register("timeline")
	eventID, _ := c.createEvent(owner.Token, "Timeline Event")
	path := "/api/v1/events/" + eventID.String()

	requireStatus(t, c.patch(path, owner.Token, map[string]any{"title": "Renamed"}), http.StatusOK)
	requireStatus(t, c.post(path+"/publish", owner.Token, nil), http.StatusOK)
	requireStatus(t, c.post(path+"/cancel", owner.Token, nil), http.StatusOK)

	res := c.get(path+"/timeline", owner.Token)
	requireStatus(t, res, http.StatusOK)

	entries, _ := res.Body["entries"].([]any)
	actions := map[string]bool{}
	for _, e := range entries {
		actions[e.(map[string]any)["action"].(string)] = true
	}

	for _, want := range []string{"event.created", "event.updated", "event.published", "event.cancelled"} {
		if !actions[want] {
			t.Errorf("timeline is missing %q; got %v", want, actions)
		}
	}

	requireErrorCode(t, c.get(path+"/timeline", ""), http.StatusUnauthorized, "unauthorized")
}

// --- deleting ----------------------------------------------------------------

func TestDeleteDraftEvent(t *testing.T) {
	c := newClient(t)
	owner := c.register("deleter")
	eventID, _ := c.createEvent(owner.Token, "Disposable Draft")

	res := c.delete("/api/v1/events/"+eventID.String(), owner.Token)
	requireStatus(t, res, http.StatusNoContent)

	var count int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM events WHERE id = $1`, eventID).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 0 {
		t.Error("the event is still in the database after a successful delete")
	}

	requireErrorCode(t, c.get("/api/v1/events/"+eventID.String(), owner.Token),
		http.StatusNotFound, "not_found")
}

// A published event may already have been seen by attendees, so it is
// cancelled rather than deleted.
func TestDeletePublishedEventIsRejected(t *testing.T) {
	c := newClient(t)
	owner := c.register("deletepublished")
	eventID, _ := c.createEvent(owner.Token, "Live Event")
	path := "/api/v1/events/" + eventID.String()

	requireStatus(t, c.post(path+"/publish", owner.Token, nil), http.StatusOK)

	res := c.delete(path, owner.Token)
	requireErrorCode(t, res, http.StatusConflict, "conflict")

	var count int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM events WHERE id = $1`, eventID).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 1 {
		t.Error("the published event was deleted despite the conflict response")
	}
}

func TestDeleteEventAuthorization(t *testing.T) {
	c := newClient(t)
	owner := c.register("deleteowner")
	other := c.register("deleteother")

	eventID, _ := c.createEvent(owner.Token, "Not Yours")
	path := "/api/v1/events/" + eventID.String()

	requireErrorCode(t, c.delete(path, ""), http.StatusUnauthorized, "unauthorized")
	requireErrorCode(t, c.delete(path, other.Token), http.StatusForbidden, "forbidden")

	var count int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM events WHERE id = $1`, eventID).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 1 {
		t.Error("an unauthorised caller deleted the event")
	}
}
