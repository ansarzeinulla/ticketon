package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/biletflow/api/internal/store"
)

// --- processing fees (SRS 3.3) -----------------------------------------------

// TestProcessingFeeIsChargedOnWhatIsActuallyPaid pins the arithmetic: the fee
// is a percentage of the discounted subtotal, not of the list price.
func TestProcessingFeeIsChargedOnWhatIsActuallyPaid(t *testing.T) {
	c := newClient(t)
	organizer := c.register("feearithmetic")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Fee Arithmetic Fest", "5000", 20)

	// 20% off 10 000 leaves 8 000; 3.5% of 8 000 is 280.
	c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "Fee Promo", "code": "FEE20", "discount_type": "percentage",
		"discount_value": "20", "max_redemptions": 5,
	})

	bought := c.buyWithPromo(eventID, ticketTypeID, 2, "fee@biletflow.test",
		map[string]any{"promo_code": "FEE20"})
	requireStatus(t, bought, http.StatusCreated)

	order := bought.Body["order"].(map[string]any)
	for field, want := range map[string]string{
		"subtotal_kzt":       "10000.00",
		"discount_kzt":       "2000.00",
		"processing_fee_kzt": "280.00",
		"total_kzt":          "8280.00",
	} {
		if order[field] != want {
			t.Errorf("%s = %v, want %v", field, order[field], want)
		}
	}

	// The payment is for what the attendee actually owes, fee included.
	if payment := bought.Body["payment"].(map[string]any); payment["amount_kzt"] != "8280.00" {
		t.Errorf("payment amount = %v, want 8280.00", payment["amount_kzt"])
	}

	// And the database agrees, because orders_total_math_chk would have
	// refused the row otherwise.
	var subtotal, discount, fee, total string
	if err := c.pool.QueryRow(t.Context(), `
		SELECT subtotal_kzt::text, discount_kzt::text,
		       processing_fee_kzt::text, total_kzt::text
		  FROM orders WHERE id = $1`, order["id"]).Scan(&subtotal, &discount, &fee, &total); err != nil {
		t.Fatalf("read the order: %v", err)
	}
	if subtotal != "10000.00" || discount != "2000.00" || fee != "280.00" || total != "8280.00" {
		t.Errorf("db row = %s/%s/%s/%s", subtotal, discount, fee, total)
	}
}

// TestFreeRegistrationPaysNoFee is SRS 3.3's "free events: no platform fee".
func TestFreeRegistrationPaysNoFee(t *testing.T) {
	c := newClient(t)
	organizer := c.register("freenofee")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Free No Fee Fest", "0", 20)

	bought := c.buy(eventID, ticketTypeID, 2, "Free Rider", "freefee@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)

	order := bought.Body["order"].(map[string]any)
	if order["processing_fee_kzt"] != "0.00" || order["total_kzt"] != "0.00" {
		t.Errorf("a free registration was charged: fee=%v total=%v",
			order["processing_fee_kzt"], order["total_kzt"])
	}
}

// TestFeeIsWaivedWhenADiscountClearsTheBasket: 100% off leaves nothing to
// process, so there is nothing to charge for processing it.
func TestFeeIsWaivedWhenADiscountClearsTheBasket(t *testing.T) {
	c := newClient(t)
	organizer := c.register("fullyfree")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Comped Fest", "5000", 20)

	c.createCampaign(organizer.Token, eventID, map[string]any{
		"name": "On the house", "code": "COMPED", "discount_type": "percentage",
		"discount_value": "100", "max_redemptions": 5,
	})

	bought := c.buyWithPromo(eventID, ticketTypeID, 1, "comped@biletflow.test",
		map[string]any{"promo_code": "COMPED"})
	requireStatus(t, bought, http.StatusCreated)

	order := bought.Body["order"].(map[string]any)
	if order["processing_fee_kzt"] != "0.00" || order["total_kzt"] != "0.00" {
		t.Errorf("a fully discounted basket was charged a fee: %v", order)
	}
}

// --- cart holds (SRS 4.6) ----------------------------------------------------

// holdBasket reserves tickets and returns the decoded hold.
func (c *client) holdBasket(eventID, ticketTypeID uuid.UUID, quantity int) response {
	c.t.Helper()
	return c.post("/api/v1/events/"+eventID.String()+"/holds", "", map[string]any{
		"items": []map[string]any{
			{"ticket_type_id": ticketTypeID.String(), "quantity": quantity},
		},
	})
}

// reservedFor reads quantity_reserved straight from PostgreSQL.
func (c *client) reservedFor(ticketTypeID uuid.UUID) int {
	c.t.Helper()
	var reserved int
	if err := c.pool.QueryRow(c.t.Context(),
		`SELECT quantity_reserved FROM ticket_types WHERE id = $1`,
		ticketTypeID).Scan(&reserved); err != nil {
		c.t.Fatalf("read quantity_reserved: %v", err)
	}
	return reserved
}

// TestHoldReservesWithoutSelling is the heart of SRS 4.6: picking tickets takes
// them off the shelf without recording a sale.
func TestHoldReservesWithoutSelling(t *testing.T) {
	c := newClient(t)
	organizer := c.register("holdreserve")
	eventID, slug, ticketTypeID := c.sellableEvent(organizer.Token, "Hold Fest", "5000", 10)

	held := c.holdBasket(eventID, ticketTypeID, 3)
	requireStatus(t, held, http.StatusCreated)

	hold := held.Body["hold"].(map[string]any)
	if hold["subtotal_kzt"] != "15000.00" {
		t.Errorf("subtotal = %v, want 15000.00", hold["subtotal_kzt"])
	}
	// The attendee is quoted the fee before committing.
	if hold["estimated_processing_fee_kzt"] != "525.00" ||
		hold["estimated_total_kzt"] != "15525.00" {
		t.Errorf("quote = %v / %v, want 525.00 and 15525.00",
			hold["estimated_processing_fee_kzt"], hold["estimated_total_kzt"])
	}
	if seconds, _ := hold["seconds_remaining"].(float64); seconds < 600 || seconds > 900 {
		t.Errorf("seconds_remaining = %v, want about 15 minutes", seconds)
	}

	// Reserved, not sold.
	if got := c.reservedFor(ticketTypeID); got != 3 {
		t.Errorf("quantity_reserved = %d, want 3", got)
	}
	if sold, _ := c.soldFor(ticketTypeID); sold != 0 {
		t.Errorf("quantity_sold = %d, want 0 - nothing has been paid for", sold)
	}

	var status string
	var reservedUntil *time.Time
	if err := c.pool.QueryRow(t.Context(),
		`SELECT status::text, reserved_until FROM orders WHERE id = $1`,
		hold["order_id"]).Scan(&status, &reservedUntil); err != nil {
		t.Fatalf("read the order: %v", err)
	}
	if status != "pending" || reservedUntil == nil {
		t.Errorf("order is %s with reserved_until %v, want pending with a deadline",
			status, reservedUntil)
	}

	// The public page counts a held ticket as unavailable, which is the whole
	// point: somebody else must not be sold it.
	page := c.get("/api/v1/public/events/"+slug, "")
	remaining := page.Body["ticket_types"].([]any)[0].(map[string]any)["quantity_remaining"]
	if remaining != float64(7) {
		t.Errorf("public remaining = %v, want 7 with 3 held", remaining)
	}
}

// TestHoldBlocksOverselling: held stock is not available to anybody else.
func TestHoldBlocksOverselling(t *testing.T) {
	c := newClient(t)
	organizer := c.register("holdblocks")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Scarce Fest", "5000", 3)

	requireStatus(t, c.holdBasket(eventID, ticketTypeID, 3), http.StatusCreated)

	// Every seat is spoken for, so a straight purchase is refused.
	requireErrorCode(t, c.buy(eventID, ticketTypeID, 1, "Too Slow", "slow@biletflow.test"),
		http.StatusConflict, CodeInsufficientInventory)

	// And so is a second hold.
	requireErrorCode(t, c.holdBasket(eventID, ticketTypeID, 1),
		http.StatusConflict, CodeInsufficientInventory)
}

// TestConfirmTurnsAHoldIntoASale walks the two-step flow end to end.
func TestConfirmTurnsAHoldIntoASale(t *testing.T) {
	c := newClient(t)
	organizer := c.register("holdconfirm")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Confirm Fest", "5000", 10)

	held := c.holdBasket(eventID, ticketTypeID, 2)
	requireStatus(t, held, http.StatusCreated)
	orderID := held.Body["hold"].(map[string]any)["order_id"].(string)

	confirmed := c.post("/api/v1/orders/"+orderID+"/confirm", "", map[string]any{
		"buyer_name":  "Aigerim Serikova",
		"buyer_email": "aigerim.hold@biletflow.test",
	})
	requireStatus(t, confirmed, http.StatusOK)

	order := confirmed.Body["order"].(map[string]any)
	if order["status"] != "paid" || order["total_kzt"] != "10350.00" {
		t.Errorf("confirmed order = %v", order)
	}
	if tickets := confirmed.Body["tickets"].([]any); len(tickets) != 2 {
		t.Errorf("tickets issued = %d, want 2", len(tickets))
	}

	// The reservation became a sale rather than adding to it.
	if got := c.reservedFor(ticketTypeID); got != 0 {
		t.Errorf("quantity_reserved = %d after confirming, want 0", got)
	}
	if sold, _ := c.soldFor(ticketTypeID); sold != 2 {
		t.Errorf("quantity_sold = %d, want 2", sold)
	}

	// Paying twice for the same basket is refused.
	requireErrorCode(t, c.post("/api/v1/orders/"+orderID+"/confirm", "", map[string]any{
		"buyer_name": "Again", "buyer_email": "again@biletflow.test",
	}), http.StatusConflict, CodeHoldNotPending)
}

// TestReleasingAHoldPutsTheTicketsBack covers the attendee closing the tab.
func TestReleasingAHoldPutsTheTicketsBack(t *testing.T) {
	c := newClient(t)
	organizer := c.register("holdrelease")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Release Fest", "5000", 2)

	held := c.holdBasket(eventID, ticketTypeID, 2)
	requireStatus(t, held, http.StatusCreated)
	orderID := held.Body["hold"].(map[string]any)["order_id"].(string)

	requireErrorCode(t, c.buy(eventID, ticketTypeID, 1, "Waiting", "waiting@biletflow.test"),
		http.StatusConflict, CodeInsufficientInventory)

	requireStatus(t, c.delete("/api/v1/orders/"+orderID+"/hold", ""), http.StatusOK)

	if got := c.reservedFor(ticketTypeID); got != 0 {
		t.Errorf("quantity_reserved = %d after releasing, want 0", got)
	}
	// The seats really are back on sale.
	requireStatus(t, c.buy(eventID, ticketTypeID, 2, "Second Chance", "second@biletflow.test"),
		http.StatusCreated)
}

// TestExpiredHoldReleasesItsInventory is the fifteen-minute timer (SRS 4.6).
//
// The clock is moved rather than waited on: a test that sleeps for a quarter of
// an hour is a test nobody runs.
func TestExpiredHoldReleasesItsInventory(t *testing.T) {
	c := newClient(t)
	organizer := c.register("holdexpiry")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Expiry Fest", "5000", 2)

	held := c.holdBasket(eventID, ticketTypeID, 2)
	requireStatus(t, held, http.StatusCreated)
	orderID := held.Body["hold"].(map[string]any)["order_id"].(string)

	if got := c.reservedFor(ticketTypeID); got != 2 {
		t.Fatalf("quantity_reserved = %d, want 2", got)
	}

	// Age the basket past its deadline.
	if _, err := c.pool.Exec(t.Context(),
		`UPDATE orders SET reserved_until = now() - interval '1 minute' WHERE id = $1`,
		orderID); err != nil {
		t.Fatalf("age the hold: %v", err)
	}

	// Somebody else arriving is what triggers the opportunistic sweep, so the
	// stale hold cannot be the reason a real sale is refused.
	requireStatus(t, c.buy(eventID, ticketTypeID, 2, "Punctual", "punctual@biletflow.test"),
		http.StatusCreated)

	var status string
	if err := c.pool.QueryRow(t.Context(),
		`SELECT status::text FROM orders WHERE id = $1`, orderID).Scan(&status); err != nil {
		t.Fatalf("read the expired order: %v", err)
	}
	if status != "expired" {
		t.Errorf("the abandoned basket is %s, want expired", status)
	}

	// Confirming it now is refused, and says why.
	requireErrorCode(t, c.post("/api/v1/orders/"+orderID+"/confirm", "", map[string]any{
		"buyer_name": "Too Late", "buyer_email": "late@biletflow.test",
	}), http.StatusConflict, CodeHoldExpired)
}

// TestSweeperReleasesAbandonedHolds covers the background timer, for the case
// where nobody arrives to trigger the opportunistic sweep.
func TestSweeperReleasesAbandonedHolds(t *testing.T) {
	c := newClient(t)
	organizer := c.register("sweeper")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Sweeper Fest", "5000", 5)

	held := c.holdBasket(eventID, ticketTypeID, 4)
	requireStatus(t, held, http.StatusCreated)

	if _, err := c.pool.Exec(t.Context(),
		`UPDATE orders SET reserved_until = now() - interval '1 minute'
		  WHERE id = $1`, held.Body["hold"].(map[string]any)["order_id"]); err != nil {
		t.Fatalf("age the hold: %v", err)
	}

	released, err := store.NewCheckoutStore(c.pool).ReleaseExpired(t.Context())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if released == 0 {
		t.Error("the sweeper released nothing")
	}
	if got := c.reservedFor(ticketTypeID); got != 0 {
		t.Errorf("quantity_reserved = %d after the sweep, want 0", got)
	}
}

// TestHoldCanBeReadBack: a page reloaded mid-checkout picks up its basket.
func TestHoldCanBeReadBack(t *testing.T) {
	c := newClient(t)
	organizer := c.register("holdreload")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Reload Fest", "5000", 10)

	held := c.holdBasket(eventID, ticketTypeID, 1)
	requireStatus(t, held, http.StatusCreated)
	orderID := held.Body["hold"].(map[string]any)["order_id"].(string)

	again := c.get("/api/v1/orders/"+orderID+"/hold", "")
	requireStatus(t, again, http.StatusOK)

	hold := again.Body["hold"].(map[string]any)
	if hold["subtotal_kzt"] != "5000.00" {
		t.Errorf("reloaded subtotal = %v, want 5000.00", hold["subtotal_kzt"])
	}
	if items := hold["items"].([]any); len(items) != 1 {
		t.Errorf("reloaded items = %d, want 1", len(items))
	}
}

// TestOneShotCheckoutStillLeavesNoReservation guards the composition: the
// one-step purchase reserves and sells inside one transaction, and must not
// leave a stray reservation behind.
func TestOneShotCheckoutStillLeavesNoReservation(t *testing.T) {
	c := newClient(t)
	organizer := c.register("oneshot")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "One Shot Fest", "5000", 10)

	requireStatus(t, c.buy(eventID, ticketTypeID, 3, "Straight Through", "straight@biletflow.test"),
		http.StatusCreated)

	if got := c.reservedFor(ticketTypeID); got != 0 {
		t.Errorf("quantity_reserved = %d after a one-shot purchase, want 0", got)
	}
	if sold, _ := c.soldFor(ticketTypeID); sold != 3 {
		t.Errorf("quantity_sold = %d, want 3", sold)
	}
}

// TestDeclinedPaymentLeavesNoReservation: a refused card must not hold stock.
func TestDeclinedPaymentLeavesNoReservation(t *testing.T) {
	c := newClient(t)
	organizer := c.register("declinehold")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Decline Fest", "5000", 5)

	requireStatus(t, c.buy(eventID, ticketTypeID, 2, "Unlucky",
		"buyer@decline.simulator.biletflow.kz"), http.StatusPaymentRequired)

	if got := c.reservedFor(ticketTypeID); got != 0 {
		t.Errorf("quantity_reserved = %d after a declined payment, want 0", got)
	}
	if sold, _ := c.soldFor(ticketTypeID); sold != 0 {
		t.Errorf("quantity_sold = %d after a declined payment, want 0", sold)
	}
}

// --- calendar export (SRS 4.11) ----------------------------------------------

// TestCalendarExportServesAnICSFile covers the endpoint end to end.
func TestCalendarExportServesAnICSFile(t *testing.T) {
	c := newClient(t)
	organizer := c.register("calendar")

	body := validEventBody("Almaty Calendar Fest")
	body["venue_name"] = "Gorky Park Stage"
	body["venue_address"] = "Gorky Park, Almaty"
	body["description"] = "An evening of live jazz."
	created := c.post("/api/v1/events", organizer.Token, body)
	requireStatus(t, created, http.StatusCreated)

	eventID := created.eventString("id")
	slug := created.eventString("slug")
	c.createTicketType(organizer.Token, uuid.MustParse(eventID), ticketTypeBody("GA", "0", 10))
	requireStatus(t, c.post("/api/v1/events/"+eventID+"/publish", organizer.Token, nil),
		http.StatusOK)

	res := c.getBinary("/api/v1/events/"+eventID+"/calendar.ics", "")
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d", res.Status)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/calendar") {
		t.Errorf("Content-Type = %q, want text/calendar", ct)
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.Contains(cd, ".ics") {
		t.Errorf("Content-Disposition = %q, want an .ics attachment", cd)
	}

	document := string(res.Body)
	for _, want := range []string{
		"BEGIN:VCALENDAR",
		"BEGIN:VEVENT",
		"SUMMARY:Almaty Calendar Fest",
		"DTSTART;TZID=Asia/Almaty:",
		"STATUS:CONFIRMED",
		"END:VCALENDAR",
	} {
		if !strings.Contains(document, want) {
			t.Errorf("the file is missing %q:\n%s", want, document)
		}
	}
	// The venue's comma is escaped rather than splitting the property.
	if !strings.Contains(document, `Gorky Park Stage\, Gorky Park\, Almaty`) {
		t.Errorf("the venue was not escaped:\n%s", document)
	}

	// The same file is reachable by slug, which is what the public page has.
	bySlug := c.getBinary("/api/v1/events/"+slug+"/calendar.ics", "")
	if bySlug.Status != http.StatusOK {
		t.Errorf("by slug: status = %d", bySlug.Status)
	}

	// Cancelling the event exports a cancellation (SRS 4.11).
	requireStatus(t, c.post("/api/v1/events/"+eventID+"/cancel", organizer.Token, nil),
		http.StatusOK)

	cancelled := string(c.getBinary("/api/v1/events/"+eventID+"/calendar.ics", "").Body)
	if !strings.Contains(cancelled, "STATUS:CANCELLED") {
		t.Errorf("a cancelled event still exports as confirmed:\n%s", cancelled)
	}
}

// TestCalendarExportHidesDrafts: a draft is not downloadable by strangers.
func TestCalendarExportHidesDrafts(t *testing.T) {
	c := newClient(t)
	organizer := c.register("calendardraft")
	eventID, _ := c.createEvent(organizer.Token, "Unpublished Fest")

	if res := c.getBinary("/api/v1/events/"+eventID.String()+"/calendar.ics", ""); res.Status != http.StatusNotFound {
		t.Errorf("an anonymous caller got %d for a draft, want 404", res.Status)
	}
	// Its own organizer may still export it, to check it before publishing.
	if res := c.getBinary("/api/v1/events/"+eventID.String()+"/calendar.ics",
		organizer.Token); res.Status != http.StatusOK {
		t.Errorf("the organizer got %d for their own draft, want 200", res.Status)
	}
}
