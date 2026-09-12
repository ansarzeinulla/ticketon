package api

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// createCampaign posts a campaign and returns its id and the response.
func (c *client) createCampaign(token string, eventID uuid.UUID, body map[string]any) (uuid.UUID, response) {
	c.t.Helper()

	res := c.post("/api/v1/events/"+eventID.String()+"/campaigns", token, body)
	if res.Status != http.StatusCreated {
		c.t.Fatalf("create campaign: status = %d, body = %s", res.Status, res.Raw)
	}

	campaign, _ := res.Body["campaign"].(map[string]any)
	id, err := uuid.Parse(campaign["id"].(string))
	if err != nil {
		c.t.Fatalf("campaign id is not a uuid: %v", err)
	}
	return id, res
}

func campaign(res response) map[string]any {
	c, _ := res.Body["campaign"].(map[string]any)
	return c
}

// redemptionCount reads the counter straight from PostgreSQL.
func (c *client) redemptionCount(campaignID uuid.UUID) int {
	c.t.Helper()
	var n int
	if err := c.pool.QueryRow(c.t.Context(),
		`SELECT redemption_count FROM campaigns WHERE id = $1`, campaignID).Scan(&n); err != nil {
		c.t.Fatalf("read redemption_count: %v", err)
	}
	return n
}

// buyWithPromo runs a checkout carrying a promo code or campaign token.
func (c *client) buyWithPromo(
	eventID, ticketTypeID uuid.UUID, quantity int, email string, promo map[string]any,
) response {
	body := map[string]any{
		"buyer_name":  "Promo Buyer",
		"buyer_email": email,
		"items":       []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": quantity}},
	}
	for k, v := range promo {
		body[k] = v
	}
	return c.post("/api/v1/events/"+eventID.String()+"/checkout", "", body)
}

// TestPhase7SuccessCriteria walks the four Phase 7 acceptance criteria.
func TestPhase7SuccessCriteria(t *testing.T) {
	c := newClient(t)
	organizer := c.register("phase7organizer")

	eventID, slug, ticketTypeID := c.sellableEvent(organizer.Token, "Phase 7 Spring Gala", "5000", 20)

	// --- 1. SPRING20: 20% off, max 1 redemption -----------------------------
	campaignID, created := c.createCampaign(organizer.Token, eventID, map[string]any{
		"name":            "Spring Promotion",
		"code":            "SPRING20",
		"discount_type":   "percentage",
		"discount_value":  "20",
		"max_redemptions": 1,
	})

	view := campaign(created)
	if view["code"] != "SPRING20" {
		t.Errorf("criterion 1: code = %v, want SPRING20", view["code"])
	}
	if view["discount_value"] != "20.00" {
		t.Errorf("criterion 1: discount_value = %v, want 20.00", view["discount_value"])
	}
	if max, _ := view["max_redemptions"].(float64); int(max) != 1 {
		t.Errorf("criterion 1: max_redemptions = %v, want 1", view["max_redemptions"])
	}

	qrToken, _ := view["qr_token"].(string)
	if len(qrToken) < 12 || qrToken[:4] != "CMP_" {
		t.Fatalf("criterion 1: qr_token = %q, want the CMP_ campaign prefix", qrToken)
	}

	// The QR encodes a trackable link carrying only the opaque token.
	campaignURL, _ := view["campaign_url"].(string)
	if !contains(campaignURL, "/events/"+slug) || !contains(campaignURL, "c="+qrToken) {
		t.Errorf("criterion 1: campaign_url = %q, want an event link carrying the token", campaignURL)
	}
	if contains(campaignURL, "20") && contains(campaignURL, "discount") {
		t.Error("criterion 1: the campaign link must not carry the discount value")
	}
	t.Logf("criterion 1 OK: SPRING20 at 20%%, max 1, token %s", qrToken)

	// --- 2. The campaign link prices 10 000 down to 8 000 -------------------
	preview := c.post("/api/v1/events/"+eventID.String()+"/promo/preview", "", map[string]any{
		"campaign_token": qrToken,
		"items":          []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 2}},
	})
	requireStatus(t, preview, http.StatusOK)

	if preview.Body["subtotal_kzt"] != "10000.00" {
		t.Errorf("criterion 2: subtotal = %v, want 10000.00", preview.Body["subtotal_kzt"])
	}
	if preview.Body["discount_kzt"] != "2000.00" {
		t.Errorf("criterion 2: discount = %v, want 2000.00", preview.Body["discount_kzt"])
	}
	if preview.Body["total_kzt"] != "8000.00" {
		t.Errorf("criterion 2: total = %v, want 8000.00", preview.Body["total_kzt"])
	}
	t.Logf("criterion 2 OK: %v - %v = %v KZT",
		preview.Body["subtotal_kzt"], preview.Body["discount_kzt"], preview.Body["total_kzt"])

	// --- 3. Paying redeems it; a second attempt is refused ------------------
	bought := c.buyWithPromo(eventID, ticketTypeID, 2, "spring@biletflow.test",
		map[string]any{"campaign_token": qrToken})
	requireStatus(t, bought, http.StatusCreated)

	order, _ := bought.Body["order"].(map[string]any)
	if order["subtotal_kzt"] != "10000.00" || order["discount_kzt"] != "2000.00" ||
		order["total_kzt"] != "8280.00" {
		t.Fatalf("criterion 3: order totals = %v/%v/%v, want 10000.00/2000.00/8280.00 (8000 plus the 3.5 percent fee)",
			order["subtotal_kzt"], order["discount_kzt"], order["total_kzt"])
	}

	var (
		dbCount     int
		dbDiscount  string
		dbTotal     string
		redemptions int
	)
	err := c.pool.QueryRow(t.Context(), `
		SELECT ca.redemption_count, o.discount_kzt::text, o.total_kzt::text,
		       (SELECT count(*) FROM promo_redemptions WHERE campaign_id = ca.id)
		  FROM campaigns ca
		  JOIN orders o ON o.campaign_id = ca.id
		 WHERE ca.id = $1`, campaignID).
		Scan(&dbCount, &dbDiscount, &dbTotal, &redemptions)
	if err != nil {
		t.Fatalf("criterion 3: read redemption: %v", err)
	}
	if dbCount != 1 {
		t.Errorf("criterion 3: redemption_count = %d, want 1", dbCount)
	}
	if redemptions != 1 {
		t.Errorf("criterion 3: %d promo_redemptions rows, want 1", redemptions)
	}
	if dbDiscount != "2000.00" || dbTotal != "8280.00" {
		t.Errorf("criterion 3: db order = %s discount / %s total, want 2000.00 / 8280.00",
			dbDiscount, dbTotal)
	}
	t.Logf("criterion 3 OK: redemption_count = %d, order stored at %s KZT", dbCount, dbTotal)

	// The code is now exhausted for everyone.
	second := c.buyWithPromo(eventID, ticketTypeID, 1, "second@biletflow.test",
		map[string]any{"promo_code": "SPRING20"})
	requireErrorCode(t, second, http.StatusConflict, CodePromoExhausted)

	if got := c.redemptionCount(campaignID); got != 1 {
		t.Errorf("criterion 3: redemption_count = %d after the refused attempt, want still 1", got)
	}
	t.Logf("criterion 3 OK: a second use is refused with %q", second.errorCode())

	// --- 4. The gate refuses the campaign QR --------------------------------
	// Exactly what a scanner reads off the poster: the link, not the raw token.
	scanned := c.scan(organizer.Token, eventID, campaignURL)
	if scanned.Status != http.StatusBadRequest {
		t.Fatalf("criterion 4: status = %d, want 400; body = %s", scanned.Status, scanned.Raw)
	}
	if scanned.errorCode() != CodeCampaignToken {
		t.Errorf("criterion 4: code = %q, want %q", scanned.errorCode(), CodeCampaignToken)
	}

	// And the bare token, in case a scanner strips the URL.
	bare := c.scan(organizer.Token, eventID, qrToken)
	requireErrorCode(t, bare, http.StatusBadRequest, CodeCampaignToken)

	var gateRecords int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM check_in_records WHERE event_id = $1`, eventID).Scan(&gateRecords); err != nil {
		t.Fatalf("criterion 4: count check-ins: %v", err)
	}
	if gateRecords != 0 {
		t.Errorf("criterion 4: %d check-ins from a campaign code, want 0", gateRecords)
	}
	t.Logf("criterion 4 OK: the gate refuses the campaign QR with %q", scanned.errorCode())
}

// A limit of one means one, even when several buyers pay at the same instant.
func TestPromoRedemptionLimitIsAtomic(t *testing.T) {
	c := newClient(t)
	organizer := c.register("promorace")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Promo Race Event", "1000", 50)

	campaignID, created := c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "One Only", "code": "ONLYONE", "discount_type": "percentage",
		"discount_value": "50", "max_redemptions": 1,
	})
	qrToken := campaign(created)["qr_token"].(string)

	const buyers = 12
	results := make([]response, buyers)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := range buyers {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			results[index] = c.buyWithPromo(eventID, ticketTypeID, 1,
				"racer@biletflow.test", map[string]any{"campaign_token": qrToken})
		}(i)
	}
	close(start)
	wg.Wait()

	discounted, refused := 0, 0
	for i, res := range results {
		switch {
		case res.Status == http.StatusCreated:
			discounted++
		case res.Status == http.StatusConflict && res.errorCode() == CodePromoExhausted:
			refused++
		default:
			t.Errorf("buyer %d got %d with code %q", i, res.Status, res.errorCode())
		}
	}

	if discounted != 1 {
		t.Errorf("%d buyers redeemed a single-use code, want exactly 1", discounted)
	}
	if refused != buyers-1 {
		t.Errorf("%d buyers were refused, want %d", refused, buyers-1)
	}
	if got := c.redemptionCount(campaignID); got != 1 {
		t.Fatalf("OVER-REDEEMED: redemption_count = %d, want 1", got)
	}

	var rows int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM promo_redemptions WHERE campaign_id = $1`, campaignID).Scan(&rows); err != nil {
		t.Fatalf("count redemptions: %v", err)
	}
	if rows != 1 {
		t.Errorf("%d promo_redemptions rows, want 1", rows)
	}
}

func TestFixedAmountDiscount(t *testing.T) {
	c := newClient(t)
	organizer := c.register("fixedpromo")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Fixed Discount Event", "5000", 20)

	c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "1500 Off", "code": "TAKE1500",
		"discount_type": "fixed_amount", "discount_value": "1500",
	})

	res := c.buyWithPromo(eventID, ticketTypeID, 2, "fixed@biletflow.test",
		map[string]any{"promo_code": "TAKE1500"})
	requireStatus(t, res, http.StatusCreated)

	order, _ := res.Body["order"].(map[string]any)
	if order["discount_kzt"] != "1500.00" || order["total_kzt"] != "8797.50" {
		t.Errorf("order = %v discount / %v total, want 1500.00 / 8797.50 (8500 plus the 3.5 percent fee)",
			order["discount_kzt"], order["total_kzt"])
	}
}

// A fixed discount larger than the basket must not produce a negative total -
// orders_total_math_chk and the amounts check would both reject that.
func TestFixedDiscountNeverExceedsTheBasket(t *testing.T) {
	c := newClient(t)
	organizer := c.register("bigdiscount")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Big Discount Event", "1000", 10)

	c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Huge", "code": "HUGE99999",
		"discount_type": "fixed_amount", "discount_value": "99999",
	})

	res := c.buyWithPromo(eventID, ticketTypeID, 1, "huge@biletflow.test",
		map[string]any{"promo_code": "HUGE99999"})
	requireStatus(t, res, http.StatusCreated)

	order, _ := res.Body["order"].(map[string]any)
	if order["discount_kzt"] != "1000.00" || order["total_kzt"] != "0.00" {
		t.Errorf("order = %v discount / %v total, want the discount capped at 1000.00 and a 0.00 total",
			order["discount_kzt"], order["total_kzt"])
	}
}

func TestPromoPreviewRejectsUnusableCodes(t *testing.T) {
	c := newClient(t)
	organizer := c.register("unusable")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Unusable Codes Event", "1000", 20)
	items := []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 1}}
	path := "/api/v1/events/" + eventID.String() + "/promo/preview"

	past := time.Now().Add(-48 * time.Hour).UTC()
	future := time.Now().Add(48 * time.Hour).UTC()

	c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Expired", "code": "EXPIRED1", "discount_type": "percentage",
		"discount_value": "10",
		"starts_at":      past.Format(time.RFC3339),
		"ends_at":        past.Add(time.Hour).Format(time.RFC3339),
	})
	c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Future", "code": "FUTURE1", "discount_type": "percentage",
		"discount_value": "10",
		"starts_at":      future.Format(time.RFC3339),
	})
	disabledID, _ := c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Disabled", "code": "DISABLED1", "discount_type": "percentage",
		"discount_value": "10", "active": false,
	})

	tests := []struct {
		name     string
		code     string
		wantCode string
	}{
		{"unknown code", "NOSUCHCODE", CodePromoNotFound},
		{"expired", "EXPIRED1", CodePromoExpired},
		{"not started", "FUTURE1", CodePromoNotStarted},
		{"disabled", "DISABLED1", CodePromoNotActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.post(path, "", map[string]any{"code": tt.code, "items": items})
			if res.errorCode() != tt.wantCode {
				t.Errorf("code = %q, want %q; body = %s", res.errorCode(), tt.wantCode, res.Raw)
			}
		})
	}

	// Checkout must refuse them for the same reasons, not just the preview.
	for _, code := range []string{"EXPIRED1", "FUTURE1", "DISABLED1", "NOSUCHCODE"} {
		res := c.buyWithPromo(eventID, ticketTypeID, 1, "x@biletflow.test",
			map[string]any{"promo_code": code})
		if res.Status == http.StatusCreated {
			t.Errorf("checkout accepted the unusable code %q", code)
		}
	}

	// Re-enabling a disabled campaign works when it has redemptions left.
	res := c.patch("/api/v1/campaigns/"+disabledID.String(), organizer.Token,
		map[string]any{"active": true})
	requireStatus(t, res, http.StatusOK)
	if campaign(res)["status"] != "active" {
		t.Errorf("status = %v, want active", campaign(res)["status"])
	}
}

func TestPromoRestrictedToTicketTypes(t *testing.T) {
	c := newClient(t)
	organizer := c.register("restricted")
	eventID, _ := c.createEvent(organizer.Token, "Restricted Promo Event")

	vipID, _ := c.createTicketType(organizer.Token, eventID, ticketTypeBody("VIP", "10000", 10))
	standardID, _ := c.createTicketType(organizer.Token, eventID, ticketTypeBody("Standard", "5000", 10))
	c.activatePaidSales(organizer.Token, eventID)
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/publish", organizer.Token, nil),
		http.StatusOK)

	c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "VIP Only", "code": "VIPONLY", "discount_type": "percentage",
		"discount_value": "10", "ticket_type_ids": []string{vipID.String()},
	})

	// A standard-only basket is not covered at all.
	res := c.buyWithPromo(eventID, standardID, 1, "std@biletflow.test",
		map[string]any{"promo_code": "VIPONLY"})
	requireErrorCode(t, res, http.StatusConflict, CodePromoNotApplicable)

	// A mixed basket discounts only the VIP line: 10% of 10000 = 1000.
	mixed := c.post("/api/v1/events/"+eventID.String()+"/checkout", "", map[string]any{
		"buyer_name": "Mixed Buyer", "buyer_email": "mixed@biletflow.test",
		"promo_code": "VIPONLY",
		"items": []map[string]any{
			{"ticket_type_id": vipID.String(), "quantity": 1},
			{"ticket_type_id": standardID.String(), "quantity": 1},
		},
	})
	requireStatus(t, mixed, http.StatusCreated)

	order, _ := mixed.Body["order"].(map[string]any)
	if order["subtotal_kzt"] != "15000.00" || order["discount_kzt"] != "1000.00" ||
		order["total_kzt"] != "14490.00" {
		t.Errorf("order = %v/%v/%v, want 15000.00/1000.00/14490.00 (14000 plus the 3.5 percent fee)",
			order["subtotal_kzt"], order["discount_kzt"], order["total_kzt"])
	}
}

func TestCampaignValidation(t *testing.T) {
	c := newClient(t)
	organizer := c.register("campaignvalidation")
	eventID, _ := c.createEvent(organizer.Token, "Campaign Validation Event")
	path := "/api/v1/events/" + eventID.String() + "/campaigns"

	base := func() map[string]any {
		return map[string]any{
			"name": "Test", "code": "VALIDCODE",
			"discount_type": "percentage", "discount_value": "10",
		}
	}

	tests := []struct {
		name      string
		mutate    func(map[string]any)
		wantField string
	}{
		{"missing name", func(b map[string]any) { delete(b, "name") }, "name"},
		{"missing code", func(b map[string]any) { delete(b, "code") }, "code"},
		{"code with spaces", func(b map[string]any) { b["code"] = "TWO WORDS" }, "code"},
		{"code too short", func(b map[string]any) { b["code"] = "AB" }, "code"},
		{"unknown discount type", func(b map[string]any) { b["discount_type"] = "buy_one_get_one" }, "discount_type"},
		{"zero discount", func(b map[string]any) { b["discount_value"] = "0" }, "discount_value"},
		{"negative discount", func(b map[string]any) { b["discount_value"] = "-5" }, "discount_value"},
		{"percentage over 100", func(b map[string]any) { b["discount_value"] = "150" }, "discount_value"},
		{"zero redemption limit", func(b map[string]any) { b["max_redemptions"] = 0 }, "max_redemptions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := base()
			tt.mutate(body)

			res := c.post(path, organizer.Token, body)
			requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")
			if _, ok := res.errorFields()[tt.wantField]; !ok {
				t.Errorf("error fields = %v, want an entry for %q", res.errorFields(), tt.wantField)
			}
		})
	}

	// Codes are globally unique and case-insensitive.
	c.createCampaign(organizer.Token, eventID, base())
	duplicate := base()
	duplicate["code"] = "validcode"
	res := c.post(path, organizer.Token, duplicate)
	requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")

	var count int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM campaigns WHERE event_id = $1`, eventID).Scan(&count); err != nil {
		t.Fatalf("count campaigns: %v", err)
	}
	if count != 1 {
		t.Errorf("%d campaigns reached the database, want only the valid one", count)
	}
}

func TestCampaignAuthorization(t *testing.T) {
	c := newClient(t)
	organizer := c.register("campaignowner")
	outsider := c.register("campaignoutsider")
	eventID, _ := c.createEvent(organizer.Token, "Protected Campaign Event")
	path := "/api/v1/events/" + eventID.String() + "/campaigns"

	body := map[string]any{
		"name": "Sneaky", "code": "SNEAKY1",
		"discount_type": "percentage", "discount_value": "90",
	}

	requireErrorCode(t, c.post(path, "", body), http.StatusUnauthorized, "unauthorized")
	requireErrorCode(t, c.post(path, outsider.Token, body), http.StatusForbidden, "forbidden")
	requireErrorCode(t, c.get(path, outsider.Token), http.StatusForbidden, "forbidden")

	campaignID, _ := c.createCampaign(organizer.Token, eventID, body)
	requireErrorCode(t, c.patch("/api/v1/campaigns/"+campaignID.String(), outsider.Token,
		map[string]any{"active": false}), http.StatusForbidden, "forbidden")
	requireErrorCode(t, c.delete("/api/v1/campaigns/"+campaignID.String(), outsider.Token),
		http.StatusForbidden, "forbidden")

	// The QR image itself is deliberately public - it is printed on posters,
	// and an <img> tag cannot carry a bearer token. Everything that manages the
	// campaign stays behind the organizer's credentials.
	qr := c.getBinary("/api/v1/campaigns/"+campaignID.String()+"/qr.png", "")
	if qr.Status != http.StatusOK {
		t.Errorf("public campaign QR = %d, want 200", qr.Status)
	}
}

func TestCampaignQRImageEncodesTheLink(t *testing.T) {
	c := newClient(t)
	organizer := c.register("campaignqr")
	eventID, slug, _ := c.sellableEvent(organizer.Token, "QR Campaign Event", "1000", 10)

	campaignID, created := c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Poster", "code": "POSTER10",
		"discount_type": "percentage", "discount_value": "10",
	})
	qrToken := campaign(created)["qr_token"].(string)

	res := c.getBinary("/api/v1/campaigns/"+campaignID.String()+"/qr.png", "")
	requireBinaryStatus(t, res, http.StatusOK)
	if ct := res.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}

	// What a phone camera would read off the poster: the campaign link.
	link := campaign(created)["campaign_url"].(string)

	if !contains(link, "/events/"+slug) {
		t.Errorf("the QR link is %q, want a link to the event page", link)
	}
	if !contains(link, qrToken) {
		t.Errorf("the QR link is %q, want it to carry the campaign token", link)
	}
	// SRS 4.14: the link must not carry a discount the client could tamper with.
	if contains(link, "discount") || contains(link, "percent") {
		t.Errorf("the QR link leaks a discount value: %q", link)
	}

	// And the served image is exactly the QR for that link.
	assertQRImageEncodes(t, res.Body, link)
}

// SRS 4.14: the admission endpoint must never accept a campaign QR.
func TestGateRejectsEveryCampaignForm(t *testing.T) {
	c := newClient(t)
	organizer := c.register("gatepromo")
	eventID, slug, _ := c.sellableEvent(organizer.Token, "Gate Promo Event", "1000", 10)

	_, created := c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Gate Test", "code": "GATETEST",
		"discount_type": "percentage", "discount_value": "10",
	})
	view := campaign(created)
	qrToken := view["qr_token"].(string)
	campaignURL := view["campaign_url"].(string)

	forms := map[string]string{
		"the bare token":           qrToken,
		"the campaign link":        campaignURL,
		"an https campaign link":   "https://biletflow.kz/events/" + slug + "?c=" + qrToken,
		"a link with extra params": campaignURL + "&utm_source=poster",
	}

	for name, code := range forms {
		t.Run(name, func(t *testing.T) {
			res := c.scan(organizer.Token, eventID, code)
			if res.Status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", res.Status, res.Raw)
			}
			if res.errorCode() != CodeCampaignToken {
				t.Errorf("code = %q, want %q", res.errorCode(), CodeCampaignToken)
			}
		})
	}

	var records int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM check_in_records WHERE event_id = $1`, eventID).Scan(&records); err != nil {
		t.Fatalf("count check-ins: %v", err)
	}
	if records != 0 {
		t.Errorf("%d check-ins from campaign codes, want 0", records)
	}
}

func TestCampaignReportingFigures(t *testing.T) {
	c := newClient(t)
	organizer := c.register("campaignreport")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Reporting Event", "5000", 50)

	campaignID, created := c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Reported", "code": "REPORT10",
		"discount_type": "percentage", "discount_value": "10",
	})
	qrToken := campaign(created)["qr_token"].(string)

	// Two orders through the campaign: 2 tickets then 1.
	requireStatus(t, c.buyWithPromo(eventID, ticketTypeID, 2, "r1@biletflow.test",
		map[string]any{"campaign_token": qrToken}), http.StatusCreated)
	requireStatus(t, c.buyWithPromo(eventID, ticketTypeID, 1, "r2@biletflow.test",
		map[string]any{"campaign_token": qrToken}), http.StatusCreated)
	// And one without it, which must not be attributed.
	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "plain@biletflow.test", "plain@biletflow.test"),
		http.StatusCreated)

	res := c.get("/api/v1/events/"+eventID.String()+"/campaigns", organizer.Token)
	requireStatus(t, res, http.StatusOK)

	list, _ := res.Body["campaigns"].([]any)
	if len(list) != 1 {
		t.Fatalf("%d campaigns listed, want 1", len(list))
	}
	view := list[0].(map[string]any)

	if count, _ := view["redemption_count"].(float64); int(count) != 2 {
		t.Errorf("redemption_count = %v, want 2", view["redemption_count"])
	}
	if orders, _ := view["orders_count"].(float64); int(orders) != 2 {
		t.Errorf("orders_count = %v, want 2", view["orders_count"])
	}
	if tickets, _ := view["tickets_sold"].(float64); int(tickets) != 3 {
		t.Errorf("tickets_sold = %v, want 3", view["tickets_sold"])
	}
	// 9000 + 4500 = 13500 gross, 1000 + 500 = 1500 discount.
	if view["gross_revenue_kzt"] != "13972.50" {
		t.Errorf("gross_revenue_kzt = %v, want 13972.50 (13500 plus the 3.5 percent fee)", view["gross_revenue_kzt"])
	}
	if view["discount_given_kzt"] != "1500.00" {
		t.Errorf("discount_given_kzt = %v, want 1500.00", view["discount_given_kzt"])
	}

	_ = campaignID
}

func TestDeleteCampaignBlockedAfterRedemption(t *testing.T) {
	c := newClient(t)
	organizer := c.register("campaigndelete")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Delete Campaign Event", "1000", 10)

	unusedID, _ := c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Unused", "code": "UNUSED1",
		"discount_type": "percentage", "discount_value": "10",
	})
	requireStatus(t, c.delete("/api/v1/campaigns/"+unusedID.String(), organizer.Token),
		http.StatusNoContent)

	usedID, created := c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Used", "code": "USED1",
		"discount_type": "percentage", "discount_value": "10",
	})
	requireStatus(t, c.buyWithPromo(eventID, ticketTypeID, 1, "used@biletflow.test",
		map[string]any{"campaign_token": campaign(created)["qr_token"].(string)}),
		http.StatusCreated)

	requireErrorCode(t, c.delete("/api/v1/campaigns/"+usedID.String(), organizer.Token),
		http.StatusConflict, "conflict")
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
