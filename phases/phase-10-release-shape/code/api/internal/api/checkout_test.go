package api

import (
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// sellableEvent creates a published event with one ticket type on sale, and
// returns the event id, its slug and the ticket type id.
func (c *client) sellableEvent(token, title, price string, quantity int) (uuid.UUID, string, uuid.UUID) {
	c.t.Helper()

	eventID, created := c.createEvent(token, title)
	ticketTypeID, _ := c.createTicketType(token, eventID, ticketTypeBody("General Admission", price, quantity))

	// A paid event cannot sell anything until its activation checklist is
	// done (SRS 4.5), so "sellable" now includes being cleared to take money.
	// Free events need no activation and are deliberately left without one.
	if price != "0" && price != "0.00" && price != "" {
		c.activatePaidSales(token, eventID)
	}

	requireStatus(c.t, c.post("/api/v1/events/"+eventID.String()+"/publish", token, nil), http.StatusOK)
	return eventID, created.eventString("slug"), ticketTypeID
}

// buy posts a checkout as an anonymous attendee.
func (c *client) buy(eventID uuid.UUID, ticketTypeID uuid.UUID, quantity int, name, email string) response {
	return c.post("/api/v1/events/"+eventID.String()+"/checkout", "", map[string]any{
		"buyer_name":  name,
		"buyer_email": email,
		"items":       []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": quantity}},
	})
}

// soldFor reads quantity_sold straight from PostgreSQL.
func (c *client) soldFor(ticketTypeID uuid.UUID) (sold, total int) {
	c.t.Helper()
	err := c.pool.QueryRow(c.t.Context(),
		`SELECT quantity_sold, quantity_total FROM ticket_types WHERE id = $1`,
		ticketTypeID).Scan(&sold, &total)
	if err != nil {
		c.t.Fatalf("read inventory: %v", err)
	}
	return sold, total
}

// TestPhase4SuccessCriteria walks the five Phase 4 acceptance criteria in order.
func TestPhase4SuccessCriteria(t *testing.T) {
	c := newClient(t)
	organizer := c.register("phase4organizer")

	// --- 1. A ticket type of exactly 5 tickets at 5,000 KZT -----------------
	eventID, slug, ticketTypeID := c.sellableEvent(organizer.Token, "Phase 4 Concert", "5000", 5)

	var (
		price string
		total int
	)
	err := c.pool.QueryRow(t.Context(),
		`SELECT price_kzt::text, quantity_total FROM ticket_types WHERE id = $1`, ticketTypeID).
		Scan(&price, &total)
	if err != nil {
		t.Fatalf("criterion 1: ticket type not in the database: %v", err)
	}
	if price != "5000.00" || total != 5 {
		t.Fatalf("criterion 1: db has price %s and quantity %d, want 5000.00 and 5", price, total)
	}
	t.Logf("criterion 1 OK: 5 tickets at %s KZT", price)

	// --- 2. The attendee sees the event publicly and buys 2 -----------------
	public := c.get("/api/v1/public/events/"+slug, "")
	requireStatus(t, public, http.StatusOK)

	types, _ := public.Body["ticket_types"].([]any)
	if len(types) != 1 {
		t.Fatalf("criterion 2: public page shows %d ticket types, want 1: %s", len(types), public.Raw)
	}
	if remaining, _ := types[0].(map[string]any)["quantity_remaining"].(float64); int(remaining) != 5 {
		t.Fatalf("criterion 2: public remaining = %v, want 5", remaining)
	}

	bought := c.buy(eventID, ticketTypeID, 2, "Nurlan Attendee", "nurlan@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	t.Log("criterion 2 OK: 2 tickets purchased through the public checkout")

	// --- 3. An order with status paid and a 10,000 KZT total ----------------
	order, _ := bought.Body["order"].(map[string]any)
	if order["status"] != "paid" {
		t.Errorf("criterion 3: order status = %v, want paid", order["status"])
	}
	// 2 x 5000 = 10000, plus the 3.5 percent processing charge SRS 3.3 adds
	// to each transaction.
	if order["total_kzt"] != "10350.00" {
		t.Errorf("criterion 3: total_kzt = %v, want 10350.00", order["total_kzt"])
	}
	if order["processing_fee_kzt"] != "350.00" {
		t.Errorf("criterion 3: processing_fee_kzt = %v, want 350.00", order["processing_fee_kzt"])
	}
	if order["currency"] != "KZT" {
		t.Errorf("criterion 3: currency = %v, want KZT", order["currency"])
	}

	orderID := uuid.MustParse(order["id"].(string))
	var (
		dbStatus    string
		dbTotal     string
		dbCurrency  string
		dbSimulated bool
	)
	err = c.pool.QueryRow(t.Context(), `
		SELECT o.status::text, o.total_kzt::text, o.currency, p.is_simulated
		  FROM orders o JOIN payments p ON p.order_id = o.id
		 WHERE o.id = $1`, orderID).Scan(&dbStatus, &dbTotal, &dbCurrency, &dbSimulated)
	if err != nil {
		t.Fatalf("criterion 3: order not in the database: %v", err)
	}
	if dbStatus != "paid" || dbTotal != "10350.00" || dbCurrency != "KZT" || !dbSimulated {
		t.Fatalf("criterion 3: db order = (%s, %s, %s, simulated=%v), want (paid, 10350.00, KZT, true)",
			dbStatus, dbTotal, dbCurrency, dbSimulated)
	}
	t.Logf("criterion 3 OK: order %s is %s for %s KZT (simulated)", order["order_number"], dbStatus, dbTotal)

	// Two tickets were actually issued.
	tickets, _ := bought.Body["tickets"].([]any)
	if len(tickets) != 2 {
		t.Errorf("criterion 3: %d tickets issued, want 2", len(tickets))
	}
	var dbTickets int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM tickets WHERE order_id = $1 AND status = 'valid'`, orderID).
		Scan(&dbTickets); err != nil {
		t.Fatalf("count tickets: %v", err)
	}
	if dbTickets != 2 {
		t.Errorf("criterion 3: %d valid tickets in the database, want 2", dbTickets)
	}

	// --- 4. quantity_sold is 2, remaining is 3 ------------------------------
	sold, totalNow := c.soldFor(ticketTypeID)
	if sold != 2 {
		t.Errorf("criterion 4: quantity_sold = %d, want 2", sold)
	}
	if totalNow-sold != 3 {
		t.Errorf("criterion 4: remaining = %d, want 3", totalNow-sold)
	}
	t.Logf("criterion 4 OK: quantity_sold = %d, remaining = %d", sold, totalNow-sold)

	// --- 5. A second attendee asking for 4 is rejected ----------------------
	rejected := c.buy(eventID, ticketTypeID, 4, "Aigerim Attendee", "aigerim@biletflow.test")
	if rejected.Status != http.StatusConflict {
		t.Fatalf("criterion 5: status = %d, want 409; body = %s", rejected.Status, rejected.Raw)
	}
	if rejected.errorCode() != CodeInsufficientInventory {
		t.Errorf("criterion 5: error code = %q, want %q", rejected.errorCode(), CodeInsufficientInventory)
	}

	errObj, _ := rejected.Body["error"].(map[string]any)
	if remaining, _ := errObj["remaining"].(float64); int(remaining) != 3 {
		t.Errorf("criterion 5: error remaining = %v, want 3", errObj["remaining"])
	}

	// The rejection changed nothing.
	soldAfter, _ := c.soldFor(ticketTypeID)
	if soldAfter != 2 {
		t.Errorf("criterion 5: quantity_sold = %d after the rejected order, want it unchanged at 2", soldAfter)
	}
	var orderCount int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM orders WHERE event_id = $1`, eventID).Scan(&orderCount); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != 1 {
		t.Errorf("criterion 5: %d orders exist, want only the successful one", orderCount)
	}
	t.Logf("criterion 5 OK: rejected with %q, inventory untouched", rejected.errorCode())
}

// TestCheckoutDoesNotOversellUnderConcurrency is the real test of the atomic
// path: many buyers race for the last few tickets at the same instant.
func TestCheckoutDoesNotOversellUnderConcurrency(t *testing.T) {
	c := newClient(t)
	organizer := c.register("raceorganizer")

	const stock = 10
	const buyers = 30
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Race Condition Concert", "1000", stock)

	type outcome struct {
		status int
		code   string
	}
	results := make([]outcome, buyers)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := range buyers {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start // release everyone at once

			res := c.post("/api/v1/events/"+eventID.String()+"/checkout", "", map[string]any{
				"buyer_name":  "Racer",
				"buyer_email": "racer@biletflow.test",
				"items": []map[string]any{
					{"ticket_type_id": ticketTypeID.String(), "quantity": 1},
				},
			})
			results[index] = outcome{status: res.Status, code: res.errorCode()}
		}(i)
	}

	close(start)
	wg.Wait()

	succeeded, rejected := 0, 0
	for i, r := range results {
		switch {
		case r.status == http.StatusCreated:
			succeeded++
		case r.status == http.StatusConflict && r.code == CodeInsufficientInventory:
			rejected++
		default:
			t.Errorf("buyer %d got status %d with code %q, want 201 or a 409 sold-out", i, r.status, r.code)
		}
	}

	if succeeded != stock {
		t.Errorf("%d checkouts succeeded, want exactly the %d in stock", succeeded, stock)
	}
	if rejected != buyers-stock {
		t.Errorf("%d checkouts were rejected, want %d", rejected, buyers-stock)
	}

	// The database is the final word: never oversold.
	sold, total := c.soldFor(ticketTypeID)
	if sold != stock {
		t.Errorf("quantity_sold = %d, want exactly %d", sold, stock)
	}
	if sold > total {
		t.Fatalf("OVERSOLD: quantity_sold = %d exceeds quantity_total = %d", sold, total)
	}

	var ticketCount int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM tickets WHERE event_id = $1`, eventID).Scan(&ticketCount); err != nil {
		t.Fatalf("count tickets: %v", err)
	}
	if ticketCount != stock {
		t.Errorf("%d tickets were issued, want %d", ticketCount, stock)
	}
}

func TestCheckoutRejectsMoreThanRemaining(t *testing.T) {
	c := newClient(t)
	owner := c.register("oversell")
	eventID, _, ticketTypeID := c.sellableEvent(owner.Token, "Small Event", "5000", 3)

	res := c.buy(eventID, ticketTypeID, 4, "Greedy Buyer", "greedy@biletflow.test")
	if res.Status != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", res.Status, res.Raw)
	}
	if res.errorCode() != CodeInsufficientInventory {
		t.Errorf("error code = %q, want %q", res.errorCode(), CodeInsufficientInventory)
	}

	sold, _ := c.soldFor(ticketTypeID)
	if sold != 0 {
		t.Errorf("quantity_sold = %d after a rejected order, want 0", sold)
	}
}

func TestCheckoutSellsOutExactly(t *testing.T) {
	c := newClient(t)
	owner := c.register("selloutexact")
	eventID, slug, ticketTypeID := c.sellableEvent(owner.Token, "Sell Out Event", "2500", 2)

	requireStatus(t, c.buy(eventID, ticketTypeID, 2, "Last Buyer", "last@biletflow.test"),
		http.StatusCreated)

	sold, total := c.soldFor(ticketTypeID)
	if sold != total {
		t.Errorf("quantity_sold = %d of %d, want a complete sell-out", sold, total)
	}

	// Even a single extra ticket is refused once stock is gone.
	res := c.buy(eventID, ticketTypeID, 1, "Too Late", "late@biletflow.test")
	if res.Status != http.StatusConflict {
		t.Fatalf("status = %d, want 409 once sold out; body = %s", res.Status, res.Raw)
	}

	public := c.get("/api/v1/public/events/"+slug, "")
	requireStatus(t, public, http.StatusOK)
	if public.Body["sold_out"] != true {
		t.Errorf("sold_out = %v, want true", public.Body["sold_out"])
	}
}

func TestCheckoutEnforcesMaxPerOrder(t *testing.T) {
	c := newClient(t)
	owner := c.register("maxperorder")
	eventID, _ := c.createEvent(owner.Token, "Limited Event")

	body := ticketTypeBody("Standard", "1000", 100)
	body["max_per_order"] = 2
	ticketTypeID, _ := c.createTicketType(owner.Token, eventID, body)
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/publish", owner.Token, nil), http.StatusOK)

	res := c.buy(eventID, ticketTypeID, 3, "Bulk Buyer", "bulk@biletflow.test")
	requireErrorCode(t, res, http.StatusConflict, "conflict")

	// Splitting the request across two lines must not dodge the limit.
	split := c.post("/api/v1/events/"+eventID.String()+"/checkout", "", map[string]any{
		"buyer_name":  "Sneaky Buyer",
		"buyer_email": "sneaky@biletflow.test",
		"items": []map[string]any{
			{"ticket_type_id": ticketTypeID.String(), "quantity": 2},
			{"ticket_type_id": ticketTypeID.String(), "quantity": 2},
		},
	})
	requireErrorCode(t, split, http.StatusConflict, "conflict")

	sold, _ := c.soldFor(ticketTypeID)
	if sold != 0 {
		t.Errorf("quantity_sold = %d, want 0 after both attempts were refused", sold)
	}
}

func TestCheckoutValidation(t *testing.T) {
	c := newClient(t)
	owner := c.register("checkoutvalidation")
	eventID, _, ticketTypeID := c.sellableEvent(owner.Token, "Validation Checkout", "1000", 10)
	path := "/api/v1/events/" + eventID.String() + "/checkout"

	validItems := []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 1}}

	tests := []struct {
		name      string
		body      map[string]any
		wantField string
	}{
		{"missing name", map[string]any{"buyer_email": "a@b.kz", "items": validItems}, "buyer_name"},
		{"blank name", map[string]any{"buyer_name": "  ", "buyer_email": "a@b.kz", "items": validItems}, "buyer_name"},
		{"missing email", map[string]any{"buyer_name": "A", "items": validItems}, "buyer_email"},
		{"malformed email", map[string]any{"buyer_name": "A", "buyer_email": "nope", "items": validItems}, "buyer_email"},
		{"no items", map[string]any{"buyer_name": "A", "buyer_email": "a@b.kz", "items": []any{}}, "items"},
		{"zero quantity", map[string]any{"buyer_name": "A", "buyer_email": "a@b.kz",
			"items": []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 0}}}, "items"},
		{"negative quantity", map[string]any{"buyer_name": "A", "buyer_email": "a@b.kz",
			"items": []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": -2}}}, "items"},
		{"malformed ticket type id", map[string]any{"buyer_name": "A", "buyer_email": "a@b.kz",
			"items": []map[string]any{{"ticket_type_id": "nope", "quantity": 1}}}, "items"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.post(path, "", tt.body)
			requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")
			if _, ok := res.errorFields()[tt.wantField]; !ok {
				t.Errorf("error fields = %v, want an entry for %q", res.errorFields(), tt.wantField)
			}
		})
	}

	sold, _ := c.soldFor(ticketTypeID)
	if sold != 0 {
		t.Errorf("quantity_sold = %d, want 0 after only invalid requests", sold)
	}
}

func TestCheckoutRejectsUnpublishedAndCancelledEvents(t *testing.T) {
	c := newClient(t)
	owner := c.register("checkoutstate")

	// A draft is not on sale.
	draftID, _ := c.createEvent(owner.Token, "Draft Event")
	draftTypeID, _ := c.createTicketType(owner.Token, draftID, ticketTypeBody("Standard", "1000", 10))
	res := c.buy(draftID, draftTypeID, 1, "Early Bird", "early@biletflow.test")
	requireErrorCode(t, res, http.StatusConflict, CodeSalesClosed)

	// A cancelled event stops selling.
	eventID, _, ticketTypeID := c.sellableEvent(owner.Token, "Doomed Event", "1000", 10)
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/cancel", owner.Token, nil), http.StatusOK)
	res = c.buy(eventID, ticketTypeID, 1, "Hopeful", "hopeful@biletflow.test")
	requireErrorCode(t, res, http.StatusConflict, CodeSalesClosed)
}

func TestCheckoutRejectsHiddenTicketType(t *testing.T) {
	c := newClient(t)
	owner := c.register("hiddentype")
	eventID, slug, ticketTypeID := c.sellableEvent(owner.Token, "Hidden Type Event", "1000", 10)

	requireStatus(t, c.patch("/api/v1/ticket-types/"+ticketTypeID.String(), owner.Token,
		map[string]any{"is_hidden": true}), http.StatusOK)

	// It disappears from the public page...
	public := c.get("/api/v1/public/events/"+slug, "")
	requireStatus(t, public, http.StatusOK)
	if types, _ := public.Body["ticket_types"].([]any); len(types) != 0 {
		t.Errorf("public page shows %d ticket types, want the hidden one excluded", len(types))
	}

	// ...and cannot be bought by anyone who kept its id.
	res := c.buy(eventID, ticketTypeID, 1, "Insider", "insider@biletflow.test")
	requireErrorCode(t, res, http.StatusConflict, CodeNotOnSale)
}

func TestCheckoutRejectsTicketTypeFromAnotherEvent(t *testing.T) {
	c := newClient(t)
	owner := c.register("crossevent")

	eventA, _, _ := c.sellableEvent(owner.Token, "Event A", "1000", 10)
	_, _, typeB := c.sellableEvent(owner.Token, "Event B", "1000", 10)

	res := c.buy(eventA, typeB, 1, "Confused Buyer", "confused@biletflow.test")
	requireErrorCode(t, res, http.StatusNotFound, "not_found")
}

func TestCheckoutLinksOrderToSignedInBuyer(t *testing.T) {
	c := newClient(t)
	owner := c.register("linkorganizer")
	buyer := c.register("linkbuyer")
	eventID, _, ticketTypeID := c.sellableEvent(owner.Token, "Linked Event", "1000", 10)

	res := c.post("/api/v1/events/"+eventID.String()+"/checkout", buyer.Token, map[string]any{
		"buyer_name":  "Signed In Buyer",
		"buyer_email": buyer.Email,
		"items":       []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 1}},
	})
	requireStatus(t, res, http.StatusCreated)

	order, _ := res.Body["order"].(map[string]any)
	if order["buyer_user_id"] != buyer.ID.String() {
		t.Errorf("buyer_user_id = %v, want %v", order["buyer_user_id"], buyer.ID)
	}
}

func TestCheckoutIssuesDistinctAdmissionTokens(t *testing.T) {
	c := newClient(t)
	owner := c.register("tokens")
	eventID, _, ticketTypeID := c.sellableEvent(owner.Token, "Token Event", "1000", 10)

	res := c.buy(eventID, ticketTypeID, 3, "Token Buyer", "tokens@biletflow.test")
	requireStatus(t, res, http.StatusCreated)

	tickets, _ := res.Body["tickets"].([]any)
	seenCodes := map[string]bool{}
	seenTokens := map[string]bool{}

	for _, raw := range tickets {
		ticket := raw.(map[string]any)
		code := ticket["ticket_code"].(string)
		token := ticket["qr_token"].(string)

		if seenCodes[code] {
			t.Errorf("duplicate ticket_code %q", code)
		}
		if seenTokens[token] {
			t.Errorf("duplicate qr_token %q", token)
		}
		seenCodes[code], seenTokens[token] = true, true

		// SRS 4.14: an admission token must be distinguishable from a campaign one.
		if len(token) < 12 || token[:4] != "TKT_" {
			t.Errorf("qr_token = %q, want the TKT_ admission prefix", token)
		}
	}
}

func TestGetOrderAfterCheckout(t *testing.T) {
	c := newClient(t)
	owner := c.register("getorder")
	eventID, _, ticketTypeID := c.sellableEvent(owner.Token, "Confirmation Event", "5000", 10)

	bought := c.buy(eventID, ticketTypeID, 2, "Confirmed Buyer", "confirmed@biletflow.test")
	requireStatus(t, bought, http.StatusCreated)
	orderID := bought.Body["order"].(map[string]any)["id"].(string)

	res := c.get("/api/v1/orders/"+orderID, "")
	requireStatus(t, res, http.StatusOK)

	order, _ := res.Body["order"].(map[string]any)
	if order["status"] != "paid" || order["total_kzt"] != "10350.00" {
		t.Errorf("order = %v, want a paid 10350.00 order", order)
	}
	if tickets, _ := res.Body["tickets"].([]any); len(tickets) != 2 {
		t.Errorf("%d tickets on the fetched order, want 2", len(tickets))
	}
	if items, _ := res.Body["items"].([]any); len(items) != 1 {
		t.Errorf("%d order items, want 1", len(items))
	}

	requireErrorCode(t, c.get("/api/v1/orders/"+uuid.NewString(), ""),
		http.StatusNotFound, "not_found")
}

func TestPublicEventPageHidesUnpublishedAndPrivate(t *testing.T) {
	c := newClient(t)
	owner := c.register("publicvisibility")

	// A draft is not publicly readable.
	_, draft := c.createEvent(owner.Token, "Unpublished Event")
	requireErrorCode(t, c.get("/api/v1/public/events/"+draft.eventString("slug"), ""),
		http.StatusNotFound, "not_found")

	// Nor is a published-but-private one, even for its organizer.
	body := validEventBody("Private Event")
	body["visibility"] = "private"
	privateRes := c.post("/api/v1/events", owner.Token, body)
	requireStatus(t, privateRes, http.StatusCreated)
	requireStatus(t, c.post("/api/v1/events/"+privateRes.eventString("id")+"/publish", owner.Token, nil),
		http.StatusOK)
	requireErrorCode(t, c.get("/api/v1/public/events/"+privateRes.eventString("slug"), owner.Token),
		http.StatusNotFound, "not_found")

	requireErrorCode(t, c.get("/api/v1/public/events/no-such-event", ""),
		http.StatusNotFound, "not_found")
}

func TestPublicEventPageReflectsRemainingStock(t *testing.T) {
	c := newClient(t)
	owner := c.register("publicstock")
	eventID, slug, ticketTypeID := c.sellableEvent(owner.Token, "Stock Event", "1000", 10)

	requireStatus(t, c.buy(eventID, ticketTypeID, 4, "Buyer", "buyer@biletflow.test"),
		http.StatusCreated)

	res := c.get("/api/v1/public/events/"+slug, "")
	requireStatus(t, res, http.StatusOK)

	types, _ := res.Body["ticket_types"].([]any)
	tt := types[0].(map[string]any)
	if remaining, _ := tt["quantity_remaining"].(float64); int(remaining) != 6 {
		t.Errorf("quantity_remaining = %v, want 6 after selling 4 of 10", tt["quantity_remaining"])
	}
	if res.Body["on_sale"] != true {
		t.Errorf("on_sale = %v, want true", res.Body["on_sale"])
	}
	if res.Body["sold_out"] != false {
		t.Errorf("sold_out = %v, want false", res.Body["sold_out"])
	}
}
