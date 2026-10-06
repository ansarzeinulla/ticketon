package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// SRS 4.12 asks platform administrators to "review reported events" and to
// "configure activation fees and platform settings". Neither had anywhere to
// put its data before this, so these tests cover the queue and the settings
// together.

// reportableEvent publishes an organizer's event and returns its id.
func (c *client) reportableEvent(token, title string) uuid.UUID {
	c.t.Helper()
	eventID, _, _ := c.sellableEvent(token, title, "0", 10)
	return eventID
}

// --- The moderation queue ----------------------------------------------------

// TestAttendeeCanReportAnEventAndAdminSeesIt is the requirement itself.
func TestAttendeeCanReportAnEventAndAdminSeesIt(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("reportadmin")
	organizer := c.register("reportorganizer")
	reporter := c.register("reportreporter")

	eventID := c.reportableEvent(organizer.Token, "Reported Event")

	filed := c.post("/api/v1/events/"+eventID.String()+"/report", reporter.Token,
		map[string]any{"reason": "misleading", "details": "The venue does not exist."})
	requireStatus(t, filed, http.StatusCreated)

	report := filed.Body["report"].(map[string]any)
	if report["status"] != store.ReportStatusOpen {
		t.Errorf("status = %v, want open", report["status"])
	}
	// The queue is only useful if it carries the event's context with it.
	if report["event_title"] != "Reported Event" {
		t.Errorf("event_title = %v, want the event title", report["event_title"])
	}
	if report["organizer_email"] != organizer.Email {
		t.Errorf("organizer_email = %v, want %s", report["organizer_email"], organizer.Email)
	}

	queue := c.get("/api/v1/admin/event-reports?status=open", admin.Token)
	requireStatus(t, queue, http.StatusOK)
	if queue.Body["total"] != float64(1) {
		t.Fatalf("queue total = %v, want 1; body = %s", queue.Body["total"], queue.Raw)
	}
}

// TestModerationQueueIsPlatformAdminOnly - a queue of complaints about other
// organizers is not something an organizer may read.
func TestModerationQueueIsPlatformAdminOnly(t *testing.T) {
	c := newClient(t)
	organizer := c.register("queueauthz")

	requireStatus(t, c.get("/api/v1/admin/event-reports", ""), http.StatusUnauthorized)
	requireStatus(t, c.get("/api/v1/admin/event-reports", organizer.Token), http.StatusForbidden)
}

// TestReportingRequiresAnAccount keeps the queue from becoming an anonymous
// denial-of-service tool aimed at organizers.
func TestReportingRequiresAnAccount(t *testing.T) {
	c := newClient(t)
	organizer := c.register("reportanon")
	eventID := c.reportableEvent(organizer.Token, "Anonymously Reported")

	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/report", "",
		map[string]any{"reason": "spam"}), http.StatusUnauthorized)
}

// TestReportValidation refuses a reason outside the closed set, so the queue
// cannot fill with free text nobody can filter on.
func TestReportValidation(t *testing.T) {
	c := newClient(t)
	organizer := c.register("reportvalid")
	reporter := c.register("reportvalidreporter")
	eventID := c.reportableEvent(organizer.Token, "Validated Reports")

	res := c.post("/api/v1/events/"+eventID.String()+"/report", reporter.Token,
		map[string]any{"reason": "i just do not like it"})
	requireStatus(t, res, http.StatusUnprocessableEntity)
	if _, ok := res.errorFields()["reason"]; !ok {
		t.Errorf("no field error on reason: %s", res.Raw)
	}

	// Over-long details are refused here rather than by the check constraint.
	long := make([]byte, maxReportDetails+1)
	for i := range long {
		long[i] = 'x'
	}
	res = c.post("/api/v1/events/"+eventID.String()+"/report", reporter.Token,
		map[string]any{"reason": "spam", "details": string(long)})
	requireStatus(t, res, http.StatusUnprocessableEntity)
	if _, ok := res.errorFields()["details"]; !ok {
		t.Errorf("no field error on details: %s", res.Raw)
	}
}

// TestOrganizerCannotReportTheirOwnEvent.
func TestOrganizerCannotReportTheirOwnEvent(t *testing.T) {
	c := newClient(t)
	organizer := c.register("selfreport")
	eventID := c.reportableEvent(organizer.Token, "Self Reported")

	requireErrorCode(t, c.post("/api/v1/events/"+eventID.String()+"/report", organizer.Token,
		map[string]any{"reason": "spam"}), http.StatusConflict, httpx.CodeConflict)
}

// TestDoubleReportIsOneComplaint - a reporter clicking twice is the same
// complaint, not two rows in the queue. The partial unique index enforces it.
func TestDoubleReportIsOneComplaint(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("doublereportadmin")
	organizer := c.register("doublereportorganizer")
	reporter := c.register("doublereportreporter")
	eventID := c.reportableEvent(organizer.Token, "Twice Reported")

	body := map[string]any{"reason": "fraud"}
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/report", reporter.Token, body),
		http.StatusCreated)
	requireErrorCode(t, c.post("/api/v1/events/"+eventID.String()+"/report", reporter.Token, body),
		http.StatusConflict, CodeAlreadyReported)

	queue := c.get("/api/v1/admin/event-reports", admin.Token)
	if queue.Body["total"] != float64(1) {
		t.Errorf("queue total = %v, want exactly 1", queue.Body["total"])
	}

	// A different person reporting the same event is a separate complaint.
	other := c.register("doublereportother")
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/report", other.Token, body),
		http.StatusCreated)
	queue = c.get("/api/v1/admin/event-reports", admin.Token)
	if queue.Body["total"] != float64(2) {
		t.Errorf("queue total = %v, want 2", queue.Body["total"])
	}
}

// TestAdminReviewsAReport walks a complaint to a decision.
func TestAdminReviewsAReport(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("reviewadmin")
	organizer := c.register("revieworganizer")
	reporter := c.register("reviewreporter")
	eventID := c.reportableEvent(organizer.Token, "Reviewed Event")

	filed := c.post("/api/v1/events/"+eventID.String()+"/report", reporter.Token,
		map[string]any{"reason": "fraud"})
	requireStatus(t, filed, http.StatusCreated)
	reportID := filed.Body["report"].(map[string]any)["id"].(string)

	res := c.patch("/api/v1/admin/event-reports/"+reportID, admin.Token,
		map[string]any{"status": "upheld", "resolution": "Event suspended pending evidence."})
	requireStatus(t, res, http.StatusOK)

	report := res.Body["report"].(map[string]any)
	if report["status"] != store.ReportStatusUpheld {
		t.Errorf("status = %v, want upheld", report["status"])
	}
	// event_reports_reviewed_chk holds that a decided report names its
	// reviewer, so the queue cannot contain a verdict nobody is accountable for.
	if report["reviewed_by"] != admin.ID.String() {
		t.Errorf("reviewed_by = %v, want the admin", report["reviewed_by"])
	}
	if report["reviewed_at"] == nil {
		t.Error("reviewed_at is null on a decided report")
	}

	// It leaves the open queue.
	open := c.get("/api/v1/admin/event-reports?status=open", admin.Token)
	if open.Body["total"] != float64(0) {
		t.Errorf("open queue total = %v, want 0", open.Body["total"])
	}
}

// TestReviewIsAudited - SRS 7 requires administrator actions to be auditable.
func TestReviewIsAudited(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("reviewauditadmin")
	organizer := c.register("reviewauditorganizer")
	reporter := c.register("reviewauditreporter")
	eventID := c.reportableEvent(organizer.Token, "Audited Review")

	filed := c.post("/api/v1/events/"+eventID.String()+"/report", reporter.Token,
		map[string]any{"reason": "copyright"})
	reportID := filed.Body["report"].(map[string]any)["id"].(string)

	requireStatus(t, c.patch("/api/v1/admin/event-reports/"+reportID, admin.Token,
		map[string]any{"status": "dismissed"}), http.StatusOK)

	var n int
	if err := c.pool.QueryRow(t.Context(), `
		SELECT count(*) FROM audit_logs
		 WHERE action = 'event.report_reviewed' AND entity_id = $1`, reportID).Scan(&n); err != nil {
		t.Fatalf("count audit entries: %v", err)
	}
	if n != 1 {
		t.Errorf("audit entries = %d, want 1", n)
	}
}

// --- Platform settings -------------------------------------------------------

// TestActivationFeeIsConfigurable is the second half of SRS 4.12: the fee was
// a Go constant, so changing it needed a rebuild.
func TestActivationFeeIsConfigurable(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("feeadmin")
	organizer := c.register("feeorganizer")

	// The seeded default is what the constant used to hold.
	settings := c.get("/api/v1/admin/settings", admin.Token)
	requireStatus(t, settings, http.StatusOK)
	if !containsSetting(settings, store.SettingActivationFee, store.SimulatedActivationFeeKZT) {
		t.Fatalf("activation fee is not the seeded default: %s", settings.Raw)
	}

	// Change it.
	res := c.patch("/api/v1/admin/settings/"+store.SettingActivationFee, admin.Token,
		map[string]any{"value": "7500.00"})
	requireStatus(t, res, http.StatusOK)

	// A newly activated event is charged the new fee, without a rebuild.
	eventID, _ := c.createEvent(organizer.Token, "Fee Follows The Setting")
	c.createTicketType(organizer.Token, eventID, ticketTypeBody("Standard", "5000", 10))
	c.activatePaidSales(organizer.Token, eventID)

	var charged string
	if err := c.pool.QueryRow(t.Context(), `
		SELECT activation_fee_kzt::text FROM paid_sales_activations WHERE event_id = $1`,
		eventID).Scan(&charged); err != nil {
		t.Fatalf("read activation fee: %v", err)
	}
	if charged != "7500.00" {
		t.Errorf("activation fee charged = %q, want 7500.00", charged)
	}
}

// TestActivationFeeRejectsNonsense keeps a bad value away from the numeric
// column, where it would be a 500 rather than a validation error.
func TestActivationFeeRejectsNonsense(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("feevalidadmin")

	for _, bad := range []any{"free", "", 7500, true, "-100.00"} {
		res := c.patch("/api/v1/admin/settings/"+store.SettingActivationFee, admin.Token,
			map[string]any{"value": bad})
		if res.Status != http.StatusUnprocessableEntity {
			t.Errorf("value %v: status = %d, want 422; body = %s", bad, res.Status, res.Raw)
		}
	}

	// And the stored value is untouched.
	settings := c.get("/api/v1/admin/settings", admin.Token)
	if !containsSetting(settings, store.SettingActivationFee, store.SimulatedActivationFeeKZT) {
		t.Errorf("the fee changed despite the refusals: %s", settings.Raw)
	}
}

// TestUnknownSettingIsNotCreated - a typo in a key must not look like a
// successful configuration change.
func TestUnknownSettingIsNotCreated(t *testing.T) {
	c := newClient(t)
	admin := c.adminAccount("settingtypoadmin")

	requireStatus(t, c.patch("/api/v1/admin/settings/activation_fee_kzr", admin.Token,
		map[string]any{"value": "1.00"}), http.StatusNotFound)
}

// TestSettingsArePlatformAdminOnly.
func TestSettingsArePlatformAdminOnly(t *testing.T) {
	c := newClient(t)
	organizer := c.register("settingauthz")

	requireStatus(t, c.get("/api/v1/admin/settings", ""), http.StatusUnauthorized)
	requireStatus(t, c.get("/api/v1/admin/settings", organizer.Token), http.StatusForbidden)
	requireStatus(t, c.patch("/api/v1/admin/settings/"+store.SettingActivationFee,
		organizer.Token, map[string]any{"value": "1.00"}), http.StatusForbidden)
}

func containsSetting(res response, key, want string) bool {
	list, _ := res.Body["settings"].([]any)
	for _, raw := range list {
		set, _ := raw.(map[string]any)
		if set["key"] == key {
			return set["value"] == want
		}
	}
	return false
}
