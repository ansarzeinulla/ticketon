package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

type adminSearchResponse struct {
	Results store.AdminSearchResult `json:"results"`
	Stats   store.PlatformStats     `json:"stats"`
}

// handleAdminSearch backs the administrative portal's search across users,
// events, orders and payments (SRS 2.1, 4.12).
//
// One endpoint rather than four: an admin given a name, an email or an order
// number rarely knows which kind of thing it is, and making them choose the
// right tab first is a worse tool than showing every match.
func (s *Server) handleAdminSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			"That search term is too long.")
		return
	}

	results, err := s.admin.Search(r.Context(), query)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	stats, err := s.admin.Stats(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, adminSearchResponse{Results: results, Stats: stats})
}

// handleAdminReport exports the operational report as CSV (SRS 4.12).
//
// CSV rather than a bespoke format because the person asking for an
// "operational report" is going to open it in a spreadsheet. It is streamed
// straight to the response: the report is one row per event, and buffering the
// whole thing to count it first would buy nothing.
func (s *Server) handleAdminReport(w http.ResponseWriter, r *http.Request) {
	rows, err := s.admin.Report(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	filename := fmt.Sprintf("biletflow-events-%s.csv", s.now().UTC().Format("2006-01-02"))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	// The browser must not cache an operational report: the next download is
	// meant to show what has changed since this one.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	out := csv.NewWriter(w)
	defer out.Flush()

	header := []string{
		"event_id", "title", "status", "lifecycle", "organizer_email",
		"starts_at_utc", "timezone", "capacity", "tickets_sold", "checked_in",
		"orders", "gross_kzt", "discounts_kzt", "refunded_kzt", "net_kzt",
		"paid_sales_activation",
	}
	if err := out.Write(header); err != nil {
		return
	}

	for _, row := range rows {
		capacity := ""
		if row.Capacity != nil {
			capacity = strconv.Itoa(*row.Capacity)
		}
		record := []string{
			row.EventID.String(),
			row.Title,
			row.Status,
			row.Lifecycle,
			row.OrganizerEmail,
			row.StartsAt.UTC().Format(time.RFC3339),
			row.Timezone,
			capacity,
			strconv.Itoa(row.TicketsSold),
			strconv.Itoa(row.CheckedIn),
			strconv.Itoa(row.Orders),
			row.GrossKZT,
			row.DiscountsKZT,
			row.RefundedKZT,
			row.NetKZT,
			row.Activation,
		}
		if err := out.Write(record); err != nil {
			return
		}
	}
}
