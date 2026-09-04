package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func analytics(res response) map[string]any {
	a, _ := res.Body["analytics"].(map[string]any)
	return a
}

func number(m map[string]any, key string) int {
	v, _ := m[key].(float64)
	return int(v)
}

func decimal(m map[string]any, key string) float64 {
	v, _ := m[key].(float64)
	return v
}

// TestPhase9SuccessCriteria walks the four Phase 9 acceptance criteria.
func TestPhase9SuccessCriteria(t *testing.T) {
	c := newClient(t)
	organizer := c.register("phase9organizer")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Phase 9 Retrospective", "5000", 20)

	// Three orders: 4 tickets sold in total, one of them checked in.
	first := c.buy(eventID, ticketTypeID, 2, "Buyer One", "one@biletflow.test")
	requireStatus(t, first, http.StatusCreated)
	second := c.buy(eventID, ticketTypeID, 1, "Buyer Two", "two@biletflow.test")
	requireStatus(t, second, http.StatusCreated)
	third := c.buy(eventID, ticketTypeID, 1, "Buyer Three", "three@biletflow.test")
	requireStatus(t, third, http.StatusCreated)

	checkedInToken := first.Body["tickets"].([]any)[0].(map[string]any)["qr_token"].(string)
	requireStatus(t, c.scan(organizer.Token, eventID, checkedInToken), http.StatusOK)

	// --- 1. The figures match the rows in PostgreSQL ------------------------
	res := c.get("/api/v1/events/"+eventID.String()+"/analytics", organizer.Token)
	requireStatus(t, res, http.StatusOK)
	a := analytics(res)

	// What the database itself says.
	var (
		dbSold      int
		dbCheckedIn int
		dbCapacity  int
		dbRevenue   string
		dbDiscounts string
		dbOrders    int
	)
	err := c.pool.QueryRow(t.Context(), `
		SELECT (SELECT count(*) FROM tickets WHERE event_id = $1 AND status IN ('valid','checked_in')),
		       (SELECT count(*) FROM tickets WHERE event_id = $1 AND status = 'checked_in'),
		       (SELECT COALESCE(sum(quantity_total), 0) FROM ticket_types WHERE event_id = $1),
		       (SELECT COALESCE(sum(total_kzt), 0)::text FROM orders
		         WHERE event_id = $1 AND status IN ('paid','completed','refunded','partially_refunded')),
		       (SELECT COALESCE(sum(discount_kzt), 0)::text FROM orders
		         WHERE event_id = $1 AND status IN ('paid','completed','refunded','partially_refunded')),
		       (SELECT count(*) FROM orders
		         WHERE event_id = $1 AND status IN ('paid','completed','refunded','partially_refunded'))`,
		eventID).Scan(&dbSold, &dbCheckedIn, &dbCapacity, &dbRevenue, &dbDiscounts, &dbOrders)
	if err != nil {
		t.Fatalf("criterion 1: read the database: %v", err)
	}

	if got := number(a, "tickets_sold"); got != dbSold {
		t.Errorf("criterion 1: tickets_sold = %d, database says %d", got, dbSold)
	}
	if got := number(a, "total_capacity"); got != dbCapacity {
		t.Errorf("criterion 1: total_capacity = %d, database says %d", got, dbCapacity)
	}
	if got := number(a, "tickets_remaining"); got != dbCapacity-dbSold {
		t.Errorf("criterion 1: tickets_remaining = %d, want %d", got, dbCapacity-dbSold)
	}
	if got, _ := a["gross_revenue_kzt"].(string); got != dbRevenue {
		t.Errorf("criterion 1: gross_revenue_kzt = %q, database says %q", got, dbRevenue)
	}
	if got, _ := a["discounts_kzt"].(string); got != dbDiscounts {
		t.Errorf("criterion 1: discounts_kzt = %q, database says %q", got, dbDiscounts)
	}
	if got := number(a, "orders_count"); got != dbOrders {
		t.Errorf("criterion 1: orders_count = %d, database says %d", got, dbOrders)
	}
	if got := number(a, "checked_in"); got != dbCheckedIn {
		t.Errorf("criterion 1: checked_in = %d, database says %d", got, dbCheckedIn)
	}
	if got := number(a, "absent"); got != dbSold-dbCheckedIn {
		t.Errorf("criterion 1: absent = %d, want %d", got, dbSold-dbCheckedIn)
	}

	// The percentages the criteria call out, checked against the same rows.
	wantSoldPct := float64(int64(float64(dbSold)/float64(dbCapacity)*1000+0.5)) / 10
	if got := decimal(a, "percentage_sold"); got != wantSoldPct {
		t.Errorf("criterion 1: percentage_sold = %v, want %v", got, wantSoldPct)
	}
	wantCheckInPct := float64(int64(float64(dbCheckedIn)/float64(dbSold)*1000+0.5)) / 10
	if got := decimal(a, "check_in_percentage"); got != wantCheckInPct {
		t.Errorf("criterion 1: check_in_percentage = %v, want %v", got, wantCheckInPct)
	}

	// 4 tickets at 5000 = 20000 KZT, plus the 3.5% processing charge that
	// SRS 3.3 adds to each transaction = 20700. 1 of 4 checked in = 25%.
	if a["gross_revenue_kzt"] != "20700.00" {
		t.Errorf("criterion 1: gross revenue = %v, want 20700.00", a["gross_revenue_kzt"])
	}
	if decimal(a, "check_in_percentage") != 25 {
		t.Errorf("criterion 1: check-in = %v%%, want 25", a["check_in_percentage"])
	}
	t.Logf("criterion 1 OK: %d sold of %d (%v%%), %s KZT, %d checked in (%v%%)",
		number(a, "tickets_sold"), number(a, "total_capacity"), a["percentage_sold"],
		a["gross_revenue_kzt"], number(a, "checked_in"), a["check_in_percentage"])

	// The by-ticket-type breakdown agrees with the totals.
	byType, _ := a["by_ticket_type"].([]any)
	if len(byType) != 1 {
		t.Fatalf("criterion 1: %d ticket types in the breakdown, want 1", len(byType))
	}
	line := byType[0].(map[string]any)
	if number(line, "sold") != dbSold || line["revenue_kzt"] != "20000.00" {
		t.Errorf("criterion 1: breakdown says %v sold for %v", line["sold"], line["revenue_kzt"])
	}

	// --- 2 & 3. Duplicating a past event with orders ------------------------
	duplicated := c.post("/api/v1/events/"+eventID.String()+"/duplicate", organizer.Token, nil)
	requireStatus(t, duplicated, http.StatusCreated)

	copyID := uuid.MustParse(duplicated.eventString("id"))
	if copyID == eventID {
		t.Fatal("criterion 2: the duplicate is the same event")
	}
	if duplicated.event()["status"] != "draft" {
		t.Fatalf("criterion 3: the copy is %v, want draft", duplicated.event()["status"])
	}
	t.Logf("criterion 2 OK: duplicated into draft %s", copyID)

	// The configuration carried over.
	var (
		copyTitle, sourceTitle       string
		copyVenue, sourceVenue       *string
		copyAddress, sourceAddress   *string
		copyTimezone, sourceTimezone string
		copyCapacity, sourceCapacity *int
		copyStatus                   string
		duplicatedFrom               *uuid.UUID
	)
	err = c.pool.QueryRow(t.Context(), `
		SELECT copy.title, src.title, copy.venue_name, src.venue_name,
		       copy.venue_address, src.venue_address, copy.timezone, src.timezone,
		       copy.capacity, src.capacity, copy.status::text, copy.duplicated_from_event_id
		  FROM events copy, events src
		 WHERE copy.id = $1 AND src.id = $2`, copyID, eventID).
		Scan(&copyTitle, &sourceTitle, &copyVenue, &sourceVenue, &copyAddress, &sourceAddress,
			&copyTimezone, &sourceTimezone, &copyCapacity, &sourceCapacity,
			&copyStatus, &duplicatedFrom)
	if err != nil {
		t.Fatalf("criterion 3: read the copy: %v", err)
	}

	if copyTitle != sourceTitle+" (copy)" {
		t.Errorf("criterion 3: title = %q, want %q", copyTitle, sourceTitle+" (copy)")
	}
	if derefString(copyVenue) != derefString(sourceVenue) {
		t.Errorf("criterion 3: venue = %q, want %q", derefString(copyVenue), derefString(sourceVenue))
	}
	if derefString(copyAddress) != derefString(sourceAddress) {
		t.Errorf("criterion 3: venue address did not carry over")
	}
	if copyTimezone != sourceTimezone {
		t.Errorf("criterion 3: timezone = %q, want %q", copyTimezone, sourceTimezone)
	}
	if (copyCapacity == nil) != (sourceCapacity == nil) ||
		(copyCapacity != nil && *copyCapacity != *sourceCapacity) {
		t.Errorf("criterion 3: capacity did not carry over")
	}
	if duplicatedFrom == nil || *duplicatedFrom != eventID {
		t.Errorf("criterion 3: duplicated_from_event_id = %v, want %v", duplicatedFrom, eventID)
	}

	// The ticket type definitions came too, with counters at zero.
	var copyTypes, copySold int
	if err := c.pool.QueryRow(t.Context(), `
		SELECT count(*), COALESCE(sum(quantity_sold), 0)
		  FROM ticket_types WHERE event_id = $1`, copyID).Scan(&copyTypes, &copySold); err != nil {
		t.Fatalf("criterion 3: read ticket types: %v", err)
	}
	if copyTypes != 1 {
		t.Errorf("criterion 3: %d ticket types on the copy, want 1", copyTypes)
	}
	if copySold != 0 {
		t.Errorf("criterion 3: the copy's ticket type already claims %d sold", copySold)
	}

	// It shows up in the organizer's dashboard.
	mine := c.get("/api/v1/events/mine", organizer.Token)
	requireStatus(t, mine, http.StatusOK)
	found := false
	for _, raw := range mine.Body["events"].([]any) {
		if raw.(map[string]any)["id"] == copyID.String() {
			found = true
		}
	}
	if !found {
		t.Error("criterion 3: the duplicate is missing from the dashboard")
	}
	t.Logf("criterion 3 OK: %q with the same venue, timezone and capacity", copyTitle)

	// --- 4. Nothing transactional came with it ------------------------------
	var orders, tickets, checkIns, cases, payments int
	err = c.pool.QueryRow(t.Context(), `
		SELECT (SELECT count(*) FROM orders WHERE event_id = $1),
		       (SELECT count(*) FROM tickets WHERE event_id = $1),
		       (SELECT count(*) FROM check_in_records WHERE event_id = $1),
		       (SELECT count(*) FROM support_cases WHERE event_id = $1),
		       (SELECT count(*) FROM payments WHERE event_id = $1)`, copyID).
		Scan(&orders, &tickets, &checkIns, &cases, &payments)
	if err != nil {
		t.Fatalf("criterion 4: count the copy's rows: %v", err)
	}

	if copyStatus != "draft" {
		t.Errorf("criterion 4: status = %q, want draft", copyStatus)
	}
	if orders != 0 || tickets != 0 || checkIns != 0 || cases != 0 || payments != 0 {
		t.Fatalf("criterion 4: the copy carried %d orders, %d tickets, %d check-ins, %d cases, %d payments — want all zero",
			orders, tickets, checkIns, cases, payments)
	}

	// And the original kept everything.
	var sourceOrders, sourceTickets, sourceCheckIns int
	if err := c.pool.QueryRow(t.Context(), `
		SELECT (SELECT count(*) FROM orders WHERE event_id = $1),
		       (SELECT count(*) FROM tickets WHERE event_id = $1),
		       (SELECT count(*) FROM check_in_records WHERE event_id = $1)`, eventID).
		Scan(&sourceOrders, &sourceTickets, &sourceCheckIns); err != nil {
		t.Fatalf("criterion 4: count the original: %v", err)
	}
	if sourceOrders != 3 || sourceTickets != 4 || sourceCheckIns != 1 {
		t.Errorf("criterion 4: the original now has %d orders, %d tickets, %d check-ins — duplication changed it",
			sourceOrders, sourceTickets, sourceCheckIns)
	}
	t.Logf("criterion 4 OK: copy is %s with 0 orders, 0 tickets, 0 check-ins; original untouched", copyStatus)

	// The copy's own analytics are all zeros.
	copyAnalytics := c.get("/api/v1/events/"+copyID.String()+"/analytics", organizer.Token)
	requireStatus(t, copyAnalytics, http.StatusOK)
	ca := analytics(copyAnalytics)
	if number(ca, "tickets_sold") != 0 || ca["gross_revenue_kzt"] != "0.00" {
		t.Errorf("criterion 4: the copy reports %v sold for %v", ca["tickets_sold"], ca["gross_revenue_kzt"])
	}
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// Refunds must not inflate revenue or the sold count.
func TestAnalyticsExcludesRefundedTickets(t *testing.T) {
	c := newClient(t)
	organizer := c.register("analyticsrefund")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Refund Analytics Event", "5000", 10)

	bought := c.buy(eventID, ticketTypeID, 2, "Refund Buyer", "refund@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)

	ticketID := bought.Body["tickets"].([]any)[0].(map[string]any)["id"].(string)
	if _, err := c.pool.Exec(t.Context(),
		`UPDATE tickets SET status = 'refunded' WHERE id = $1`, uuid.MustParse(ticketID)); err != nil {
		t.Fatalf("refund the ticket: %v", err)
	}

	res := c.get("/api/v1/events/"+eventID.String()+"/analytics", organizer.Token)
	requireStatus(t, res, http.StatusOK)
	a := analytics(res)

	if got := number(a, "tickets_sold"); got != 1 {
		t.Errorf("tickets_sold = %d, want 1 after one of two was refunded", got)
	}
	if got := number(a, "tickets_refunded"); got != 1 {
		t.Errorf("tickets_refunded = %d, want 1", got)
	}
}

func TestAnalyticsWithNoSales(t *testing.T) {
	c := newClient(t)
	organizer := c.register("analyticsempty")
	eventID, _, _ := c.sellableEvent(organizer.Token, "Quiet Event", "5000", 10)

	res := c.get("/api/v1/events/"+eventID.String()+"/analytics", organizer.Token)
	requireStatus(t, res, http.StatusOK)
	a := analytics(res)

	// Zero sales must read as zeros, not as a division by zero.
	if number(a, "tickets_sold") != 0 || number(a, "checked_in") != 0 {
		t.Errorf("a quiet event reports %v sold and %v checked in", a["tickets_sold"], a["checked_in"])
	}
	if decimal(a, "percentage_sold") != 0 || decimal(a, "check_in_percentage") != 0 {
		t.Errorf("percentages are %v and %v, want 0", a["percentage_sold"], a["check_in_percentage"])
	}
	if a["gross_revenue_kzt"] != "0.00" || a["net_revenue_kzt"] != "0.00" {
		t.Errorf("revenue is %v gross / %v net, want 0.00", a["gross_revenue_kzt"], a["net_revenue_kzt"])
	}
	if points, _ := a["sales_over_time"].([]any); len(points) != 0 {
		t.Errorf("%d sales points on an event with no sales", len(points))
	}
}

func TestAnalyticsCountsDiscountsAndCampaigns(t *testing.T) {
	c := newClient(t)
	organizer := c.register("analyticspromo")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Promo Analytics Event", "5000", 20)

	_, created := c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Analytics Promo", "code": "ANALYTICS20",
		"discount_type": "percentage", "discount_value": "20",
	})
	qrToken := campaign(created)["qr_token"].(string)

	requireStatus(t, c.buyWithPromo(eventID, ticketTypeID, 2, "promo@biletflow.test",
		map[string]any{"campaign_token": qrToken}), http.StatusCreated)
	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Plain", "plain@biletflow.test"),
		http.StatusCreated)

	res := c.get("/api/v1/events/"+eventID.String()+"/analytics", organizer.Token)
	requireStatus(t, res, http.StatusOK)
	a := analytics(res)

	// 8000 discounted + 5000 plain = 13000 gross, 2000 discounted.
	if a["gross_revenue_kzt"] != "13455.00" {
		t.Errorf("gross_revenue_kzt = %v, want 13455.00 (13000 plus the 3.5 percent fee)", a["gross_revenue_kzt"])
	}
	if a["discounts_kzt"] != "2000.00" {
		t.Errorf("discounts_kzt = %v, want 2000.00", a["discounts_kzt"])
	}

	byCampaign, _ := a["by_campaign"].([]any)
	if len(byCampaign) != 1 {
		t.Fatalf("%d campaigns in the breakdown, want 1", len(byCampaign))
	}
	cs := byCampaign[0].(map[string]any)
	if cs["code"] != "ANALYTICS20" || number(cs, "tickets_sold") != 2 ||
		cs["revenue_kzt"] != "8280.00" {
		t.Errorf("campaign line = %v", cs)
	}
}

func TestAnalyticsFilters(t *testing.T) {
	c := newClient(t)
	organizer := c.register("analyticsfilter")
	eventID, _ := c.createEvent(organizer.Token, "Filtered Analytics Event")

	vipID, _ := c.createTicketType(organizer.Token, eventID, ticketTypeBody("VIP", "10000", 10))
	stdID, _ := c.createTicketType(organizer.Token, eventID, ticketTypeBody("Standard", "5000", 10))
	c.activatePaidSales(organizer.Token, eventID)
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/publish", organizer.Token, nil),
		http.StatusOK)

	requireStatus(t, c.buy(eventID, vipID, 1, "VIP Buyer", "vip@biletflow.test"), http.StatusCreated)
	requireStatus(t, c.buy(eventID, stdID, 2, "Std Buyer", "std@biletflow.test"), http.StatusCreated)

	path := "/api/v1/events/" + eventID.String() + "/analytics"

	all := c.get(path, organizer.Token)
	requireStatus(t, all, http.StatusOK)
	if got := number(analytics(all), "tickets_sold"); got != 3 {
		t.Errorf("unfiltered tickets_sold = %d, want 3", got)
	}

	vipOnly := c.get(path+"?ticket_type_id="+vipID.String(), organizer.Token)
	requireStatus(t, vipOnly, http.StatusOK)
	if got := number(analytics(vipOnly), "tickets_sold"); got != 1 {
		t.Errorf("VIP-only tickets_sold = %d, want 1", got)
	}
	if byType, _ := analytics(vipOnly)["by_ticket_type"].([]any); len(byType) != 1 {
		t.Errorf("VIP-only breakdown has %d rows, want 1", len(byType))
	}

	// A window before the sales happened contains none of them.
	past := time.Now().AddDate(0, 0, -30).UTC().Format("2006-01-02")
	pastEnd := time.Now().AddDate(0, 0, -29).UTC().Format("2006-01-02")
	empty := c.get(path+"?from="+past+"&to="+pastEnd, organizer.Token)
	requireStatus(t, empty, http.StatusOK)
	if got := analytics(empty)["gross_revenue_kzt"]; got != "0.00" {
		t.Errorf("revenue in a past window = %v, want 0.00", got)
	}

	// Today's window contains all of them.
	today := time.Now().UTC().Format("2006-01-02")
	current := c.get(path+"?from="+today+"&to="+today, organizer.Token)
	requireStatus(t, current, http.StatusOK)
	if got := analytics(current)["gross_revenue_kzt"]; got != "20700.00" {
		t.Errorf("today's revenue = %v, want 20700.00 including the processing fee", got)
	}

	requireErrorCode(t, c.get(path+"?from=not-a-date", organizer.Token),
		http.StatusUnprocessableEntity, "validation_failed")
	requireErrorCode(t, c.get(path+"?ticket_type_id=nope", organizer.Token),
		http.StatusUnprocessableEntity, "validation_failed")
}

func TestAnalyticsAuthorization(t *testing.T) {
	c := newClient(t)
	organizer := c.register("analyticsowner")
	outsider := c.register("analyticsoutsider")
	eventID, _, _ := c.sellableEvent(organizer.Token, "Private Figures Event", "1000", 10)

	path := "/api/v1/events/" + eventID.String() + "/analytics"
	requireErrorCode(t, c.get(path, ""), http.StatusUnauthorized, "unauthorized")
	requireErrorCode(t, c.get(path, outsider.Token), http.StatusForbidden, "forbidden")

	requireErrorCode(t, c.post("/api/v1/events/"+eventID.String()+"/duplicate", outsider.Token, nil),
		http.StatusForbidden, "forbidden")
	requireErrorCode(t, c.get("/api/v1/events/"+eventID.String()+"/timeline", outsider.Token),
		http.StatusForbidden, "forbidden")
}

// SRS 4.16: the timeline records the important actions, with who and when.
func TestTimelineRecordsTheEventHistory(t *testing.T) {
	c := newClient(t)
	organizer := c.register("timelineowner")

	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Timeline Event", "1000", 10)
	bought := c.buy(eventID, ticketTypeID, 1, "Timeline Buyer", "tl@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)

	token := bought.Body["tickets"].([]any)[0].(map[string]any)["qr_token"].(string)
	requireStatus(t, c.scan(organizer.Token, eventID, token), http.StatusOK)

	res := c.get("/api/v1/events/"+eventID.String()+"/timeline", organizer.Token)
	requireStatus(t, res, http.StatusOK)

	entries, _ := res.Body["entries"].([]any)
	actions := map[string]bool{}
	for _, raw := range entries {
		entry := raw.(map[string]any)
		actions[entry["action"].(string)] = true

		if entry["created_at"] == nil {
			t.Error("a timeline entry has no timestamp")
		}
	}

	for _, want := range []string{
		"event.created", "ticket_type.created", "event.published",
		"order.created", "ticket.checked_in",
	} {
		if !actions[want] {
			t.Errorf("the timeline is missing %q; got %v", want, actions)
		}
	}

	// Newest first, so the most recent activity is at the top.
	if len(entries) >= 2 {
		first := entries[0].(map[string]any)["created_at"].(string)
		last := entries[len(entries)-1].(map[string]any)["created_at"].(string)
		if first < last {
			t.Errorf("the timeline is oldest-first (%s before %s)", first, last)
		}
	}

	// Filtering by activity type (SRS 4.16).
	ticketsOnly := c.get("/api/v1/events/"+eventID.String()+"/timeline?type=ticket.", organizer.Token)
	requireStatus(t, ticketsOnly, http.StatusOK)
	filtered, _ := ticketsOnly.Body["entries"].([]any)
	if len(filtered) == 0 {
		t.Fatal("filtering by ticket. returned nothing")
	}
	for _, raw := range filtered {
		if action := raw.(map[string]any)["action"].(string); action[:7] != "ticket." {
			t.Errorf("the ticket. filter returned %q", action)
		}
	}

	// Filtering by date range.
	future := time.Now().AddDate(0, 0, 1).UTC().Format("2006-01-02")
	afterEverything := c.get(
		"/api/v1/events/"+eventID.String()+"/timeline?from="+future, organizer.Token)
	requireStatus(t, afterEverything, http.StatusOK)
	if got, _ := afterEverything.Body["entries"].([]any); len(got) != 0 {
		t.Errorf("%d entries in a future window, want 0", len(got))
	}
}

// SRS 4.16 groups an organizer's events as Upcoming, Active, Completed and
// Cancelled.
func TestEventLifecycleFiltering(t *testing.T) {
	c := newClient(t)
	organizer := c.register("lifecycleowner")

	// Upcoming: the default helper places events a month out.
	upcomingID, _ := c.createEvent(organizer.Token, "Upcoming Show")
	requireStatus(t, c.post("/api/v1/events/"+upcomingID.String()+"/publish", organizer.Token, nil),
		http.StatusOK)

	// Active: started an hour ago, ends in an hour.
	activeID, _ := c.createEvent(organizer.Token, "Active Show")
	if _, err := c.pool.Exec(t.Context(), `
		UPDATE events SET starts_at = now() - interval '1 hour',
		                  ends_at = now() + interval '1 hour'
		 WHERE id = $1`, activeID); err != nil {
		t.Fatalf("age the active event: %v", err)
	}
	requireStatus(t, c.post("/api/v1/events/"+activeID.String()+"/publish", organizer.Token, nil),
		http.StatusOK)

	// Completed: published while still upcoming, then aged so it finished
	// yesterday. Publishing happens first because an event that has already
	// ended can no longer be published (LIFE-ERR-03); a completed event is one
	// that ended after being put on sale.
	completedID, _ := c.createEvent(organizer.Token, "Completed Show")
	requireStatus(t, c.post("/api/v1/events/"+completedID.String()+"/publish", organizer.Token, nil),
		http.StatusOK)
	if _, err := c.pool.Exec(t.Context(), `
		UPDATE events SET starts_at = now() - interval '2 days',
		                  ends_at = now() - interval '1 day'
		 WHERE id = $1`, completedID); err != nil {
		t.Fatalf("age the completed event: %v", err)
	}

	// Cancelled.
	cancelledID, _ := c.createEvent(organizer.Token, "Cancelled Show")
	requireStatus(t, c.post("/api/v1/events/"+cancelledID.String()+"/cancel", organizer.Token, nil),
		http.StatusOK)

	// A draft, which is none of the above.
	draftID, _ := c.createEvent(organizer.Token, "Draft Show")

	expected := map[string]uuid.UUID{
		"upcoming":  upcomingID,
		"active":    activeID,
		"completed": completedID,
		"cancelled": cancelledID,
		"draft":     draftID,
	}

	for stage, wantID := range expected {
		t.Run(stage, func(t *testing.T) {
			res := c.get("/api/v1/events/mine?lifecycle="+stage, organizer.Token)
			requireStatus(t, res, http.StatusOK)

			events, _ := res.Body["events"].([]any)
			if len(events) != 1 {
				t.Fatalf("%d events in %s, want 1: %s", len(events), stage, res.Raw)
			}
			got := events[0].(map[string]any)
			if got["id"] != wantID.String() {
				t.Errorf("%s returned %v, want %v", stage, got["title"], wantID)
			}
			if got["lifecycle"] != stage {
				t.Errorf("lifecycle = %v, want %v", got["lifecycle"], stage)
			}
		})
	}

	requireErrorCode(t, c.get("/api/v1/events/mine?lifecycle=someday", organizer.Token),
		http.StatusUnprocessableEntity, "validation_failed")
}

func TestDuplicateAcceptsOverrides(t *testing.T) {
	c := newClient(t)
	organizer := c.register("duplicateoverride")
	eventID, _, _ := c.sellableEvent(organizer.Token, "Override Source", "1000", 10)

	start := time.Now().AddDate(1, 0, 0).UTC().Truncate(time.Second)
	res := c.post("/api/v1/events/"+eventID.String()+"/duplicate", organizer.Token, map[string]any{
		"title":     "Override Source 2027",
		"starts_at": start.Format(time.RFC3339),
		"ends_at":   start.Add(4 * time.Hour).Format(time.RFC3339),
	})
	requireStatus(t, res, http.StatusCreated)

	if res.eventString("title") != "Override Source 2027" {
		t.Errorf("title = %v, want the override", res.event()["title"])
	}
	if res.eventString("slug") == "" || res.eventString("slug") == "override-source" {
		t.Errorf("slug = %q, want a distinct slug for the copy", res.eventString("slug"))
	}

	requireErrorCode(t, c.post("/api/v1/events/"+eventID.String()+"/duplicate", organizer.Token,
		map[string]any{"title": "   "}), http.StatusUnprocessableEntity, "validation_failed")
}

// Duplicating twice must not collide on the generated slug.
func TestDuplicateTwiceProducesDistinctEvents(t *testing.T) {
	c := newClient(t)
	organizer := c.register("duplicatetwice")
	eventID, _, _ := c.sellableEvent(organizer.Token, "Popular Source", "1000", 10)

	first := c.post("/api/v1/events/"+eventID.String()+"/duplicate", organizer.Token, nil)
	requireStatus(t, first, http.StatusCreated)
	second := c.post("/api/v1/events/"+eventID.String()+"/duplicate", organizer.Token, nil)
	requireStatus(t, second, http.StatusCreated)

	if first.eventString("id") == second.eventString("id") {
		t.Fatal("the two duplicates are the same event")
	}
	if first.eventString("slug") == second.eventString("slug") {
		t.Errorf("both duplicates got the slug %q", first.eventString("slug"))
	}

	var copies int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM events WHERE duplicated_from_event_id = $1`, eventID).
		Scan(&copies); err != nil {
		t.Fatalf("count copies: %v", err)
	}
	if copies != 2 {
		t.Errorf("%d copies recorded, want 2", copies)
	}
}

// A duplicate must not inherit campaigns: promo codes are globally unique.
func TestDuplicateLeavesCampaignsBehind(t *testing.T) {
	c := newClient(t)
	organizer := c.register("duplicatecampaign")
	eventID, _, _ := c.sellableEvent(organizer.Token, "Campaign Source", "1000", 10)

	c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Source Promo", "code": "SOURCEPROMO",
		"discount_type": "percentage", "discount_value": "10",
	})

	res := c.post("/api/v1/events/"+eventID.String()+"/duplicate", organizer.Token, nil)
	requireStatus(t, res, http.StatusCreated)

	var campaigns int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM campaigns WHERE event_id = $1`,
		uuid.MustParse(res.eventString("id"))).Scan(&campaigns); err != nil {
		t.Fatalf("count campaigns: %v", err)
	}
	if campaigns != 0 {
		t.Errorf("the copy inherited %d campaigns, want 0", campaigns)
	}
}
