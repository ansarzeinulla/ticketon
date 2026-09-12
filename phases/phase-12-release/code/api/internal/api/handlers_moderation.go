package api

import (
	"errors"
	"net/http"
	"slices"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

// Moderation-queue and platform-settings codes (SRS 4.12).
const (
	CodeAlreadyReported = "already_reported"
	// maxReportDetails matches the event_reports_details_length_chk constraint,
	// so an over-long complaint is a 422 naming the field rather than a 500
	// from the database.
	maxReportDetails = 2000
)

type reportRequest struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

type reportResponse struct {
	Report store.EventReport `json:"report"`
}

type reportsResponse struct {
	Reports []store.EventReport `json:"reports"`
	Total   int                 `json:"total"`
}

// handleReportEvent lets any signed-in user flag an event for review
// (SRS 4.12: "Review reported events" needs something to review).
//
// An account is required rather than allowing anonymous reports: a queue that
// anybody can fill without identifying themselves is a denial-of-service tool
// aimed at organizers, not a moderation aid.
func (s *Server) handleReportEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The event id must be a UUID.")
		return
	}

	var req reportRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}
	if !slices.Contains(store.ReportReasons, req.Reason) {
		errs.add("reason", "Choose one of: "+joinWords(store.ReportReasons)+".")
	}
	if runeLen(req.Details) > maxReportDetails {
		errs.add("details", "Keep the details under 2000 characters.")
	}
	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	// The event has to exist, and a report about an event nobody can see is
	// not useful, so the same visibility rule as a public read applies.
	event, err := s.events.GetByID(r.Context(), eventID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No event with this id.")
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	reporterID := mustUserID(r.Context())
	if event.OrganizerID == reporterID {
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict,
			"You cannot report your own event.")
		return
	}

	report, err := s.moderation.Report(r.Context(), store.ReportParams{
		EventID:    eventID,
		ReporterID: reporterID,
		Reason:     req.Reason,
		Details:    req.Details,
	})
	switch {
	case errors.Is(err, store.ErrAlreadyReported):
		httpx.WriteError(w, http.StatusConflict, CodeAlreadyReported,
			"You have already reported this event; it is in the moderation queue.")
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, reportResponse{Report: report})
}

// handleListReports is the administrator's moderation queue (SRS 4.12).
func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && !slices.Contains(store.ReportStatuses, status) {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"Unknown report status.")
		return
	}

	reports, err := s.moderation.ListReports(r.Context(), status, 0)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, reportsResponse{Reports: reports, Total: len(reports)})
}

type reviewRequest struct {
	Status     string `json:"status"`
	Resolution string `json:"resolution"`
}

// handleReviewReport records a moderator's decision (SRS 4.12).
//
// The decision is deliberately separate from suspending the event: upholding a
// report says "this complaint was justified", and whether that warrants
// suspension is a second, explicit choice the administrator makes.
func (s *Server) handleReviewReport(w http.ResponseWriter, r *http.Request) {
	reportID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"The report id must be a UUID.")
		return
	}

	var req reviewRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}

	errs := fieldErrors{}
	if !slices.Contains(store.ReportStatuses, req.Status) {
		errs.add("status", "Choose one of: "+joinWords(store.ReportStatuses)+".")
	}
	if runeLen(req.Resolution) > maxReportDetails {
		errs.add("resolution", "Keep the resolution under 2000 characters.")
	}
	if errs.any() {
		httpx.WriteValidationError(w, errs)
		return
	}

	actorID := mustUserID(r.Context())
	report, err := s.moderation.Review(r.Context(), store.ReviewParams{
		ReportID:   reportID,
		ReviewerID: actorID,
		Status:     req.Status,
		Resolution: req.Resolution,
	})
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No report with this id.")
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}

	eventID := report.EventID
	if err := s.audit.Append(r.Context(), store.AuditEntry{
		EventID:     &eventID,
		ActorUserID: &actorID,
		Action:      "event.report_reviewed",
		EntityType:  "event_report",
		EntityID:    report.ID.String(),
		Description: "Report marked " + report.Status + " by a platform administrator",
		Metadata: map[string]any{
			"request_id": httpx.RequestIDFromContext(r.Context()),
			"reason":     report.Reason,
		},
	}); err != nil {
		httpx.LogAuditFailure(r.Context(), "event.report_reviewed", err)
	}

	httpx.WriteJSON(w, http.StatusOK, reportResponse{Report: report})
}

// --- Platform settings (SRS 4.12) --------------------------------------------

type settingsResponse struct {
	Settings []store.Setting `json:"settings"`
}

type settingResponse struct {
	Setting store.Setting `json:"setting"`
}

// handleListSettings shows the configurable platform values.
func (s *Server) handleListSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.moderation.ListSettings(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, settingsResponse{Settings: settings})
}

type settingRequest struct {
	Value any `json:"value"`
}

// handleUpdateSetting changes one platform setting (SRS 4.12: "Configure
// activation fees and platform settings").
//
// Only settings that already exist can be changed. Creating one from a request
// body would let a typo add a key nothing reads, which looks like a successful
// configuration change and is not one.
func (s *Server) handleUpdateSetting(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var req settingRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidJSON, err.Error())
		return
	}
	if req.Value == nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeValidationFailed,
			"A value is required.")
		return
	}

	// The activation fee reaches a numeric column, so a value that is not a
	// decimal amount has to be refused here rather than at the constraint.
	if key == store.SettingActivationFee {
		fee, ok := req.Value.(string)
		if amount, valid := parseMoney(fee); !ok || !valid || amount.Sign() < 0 {
			httpx.WriteValidationError(w, fieldErrors{
				"value": "The fee must be a decimal amount in KZT, as a string, " +
					"for example \"7500.00\".",
			})
			return
		}
	}

	actorID := mustUserID(r.Context())
	setting, err := s.moderation.SetSetting(r.Context(), key, req.Value, actorID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound,
			"No platform setting with this key.")
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}

	if err := s.audit.Append(r.Context(), store.AuditEntry{
		ActorUserID: &actorID,
		Action:      "platform.setting_changed",
		EntityType:  "platform_setting",
		EntityID:    key,
		Description: "Platform setting " + key + " changed by an administrator",
		Metadata: map[string]any{
			"request_id": httpx.RequestIDFromContext(r.Context()),
			"value":      setting.Value,
		},
	}); err != nil {
		httpx.LogAuditFailure(r.Context(), "platform.setting_changed", err)
	}

	httpx.WriteJSON(w, http.StatusOK, settingResponse{Setting: setting})
}

// joinWords renders a closed set for an error message.
func joinWords(words []string) string {
	out := ""
	for i, w := range words {
		switch {
		case i == 0:
		case i == len(words)-1:
			out += " or "
		default:
			out += ", "
		}
		out += w
	}
	return out
}
