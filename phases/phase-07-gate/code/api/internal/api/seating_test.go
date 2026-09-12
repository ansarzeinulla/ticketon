package api

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/google/uuid"
)

// seedVenueLayout inserts a predefined venue layout (SRS 4.3.1).
//
// Built here rather than leaned on from `make seed`: the test harness truncates
// every table, so a fixture that depends on demo data is a fixture that depends
// on the order somebody ran two commands in.
//
// The shape mirrors the real seeded hall - a premium section, a standard one,
// and an accessible box sharing the standard category - because that shared
// category is exactly what makes seat pricing non-trivial.
func (c *client) seedVenueLayout() uuid.UUID {
	c.t.Helper()

	ctx := c.t.Context()

	var venueID uuid.UUID
	if err := c.pool.QueryRow(ctx, `
		INSERT INTO venues (name, address_line, city, is_predefined_layout)
		VALUES ('Almaty Test Hall', 'Abay Avenue 1', 'Almaty', true)
		RETURNING id`).Scan(&venueID); err != nil {
		c.t.Fatalf("create the venue: %v", err)
	}

	sections := []struct {
		name       string
		category   string
		order      int
		rows       int
		perRow     int
		accessible bool
	}{
		{"Orchestra", "premium", 1, 3, 4, false},
		{"Balcony", "standard", 2, 2, 4, false},
		{"Accessible Box", "standard", 3, 1, 2, true},
	}

	y := 30.0
	for _, section := range sections {
		var sectionID uuid.UUID
		if err := c.pool.QueryRow(ctx, `
			INSERT INTO venue_sections (venue_id, name, price_category, display_order)
			VALUES ($1, $2, $3, $4) RETURNING id`,
			venueID, section.name, section.category, section.order).Scan(&sectionID); err != nil {
			c.t.Fatalf("create section %s: %v", section.name, err)
		}

		for r := 1; r <= section.rows; r++ {
			var rowID uuid.UUID
			label := string(rune('A' + r - 1))
			if err := c.pool.QueryRow(ctx, `
				INSERT INTO seat_rows (section_id, label, display_order)
				VALUES ($1, $2, $3) RETURNING id`,
				sectionID, label, r).Scan(&rowID); err != nil {
				c.t.Fatalf("create row %s: %v", label, err)
			}

			for n := 1; n <= section.perRow; n++ {
				if _, err := c.pool.Exec(ctx, `
					INSERT INTO seats (row_id, seat_number, is_accessible, map_x, map_y)
					VALUES ($1, $2, $3, $4, $5)`,
					rowID, strconv.Itoa(n), section.accessible,
					30.0+float64(n)*30, y); err != nil {
					c.t.Fatalf("create seat %d: %v", n, err)
				}
			}
			y += 30
		}
	}
	return venueID
}

// Totals for the fixture above: 3*4 + 2*4 + 1*2.
const (
	fixtureSeats           = 22
	fixtureAccessibleSeats = 2
)

// seatedEvent builds an assigned-seating event on the predefined layout.
//
// It attaches a layout rather than inventing one per event, which is the point
// of SRS 4.3.1's "predefined venue layout": an organizer selects one, they do
// not draw it.
func (c *client) seatedEvent(token, title string) (uuid.UUID, string) {
	c.t.Helper()

	venueID := c.seedVenueLayout()
	eventID, created := c.createEvent(token, title)

	// Orchestra is premium; Balcony and Accessible Box are both standard -
	// matching the sections by name, which is how a seat is priced.
	for _, tier := range []struct{ name, price, category string }{
		{"Orchestra", "12000", "premium"},
		{"Balcony", "7000", "standard"},
		{"Accessible Box", "7000", "standard"},
	} {
		body := ticketTypeBody(tier.name, tier.price, 60)
		body["price_category"] = tier.category
		c.createTicketType(token, eventID, body)
	}

	if _, err := c.pool.Exec(c.t.Context(), `
		UPDATE events SET venue_id = $2, seating_mode = 'assigned_seating'
		 WHERE id = $1`, eventID, venueID); err != nil {
		c.t.Fatalf("attach the seating plan: %v", err)
	}

	c.activatePaidSales(token, eventID)
	requireStatus(c.t, c.post("/api/v1/events/"+eventID.String()+"/publish", token, nil),
		http.StatusOK)

	return eventID, created.eventString("slug")
}

// seatsOf flattens the map into one list.
func seatsOf(t *testing.T, res response) []map[string]any {
	t.Helper()

	plan, ok := res.Body["seat_map"].(map[string]any)
	if !ok {
		t.Fatalf("no seat_map in the response: %s", res.Raw)
	}

	var all []map[string]any
	for _, section := range plan["sections"].([]any) {
		for _, row := range section.(map[string]any)["rows"].([]any) {
			for _, seat := range row.(map[string]any)["seats"].([]any) {
				all = append(all, seat.(map[string]any))
			}
		}
	}
	return all
}

func TestSeatMapDescribesThePredefinedLayout(t *testing.T) {
	c := newClient(t)
	organizer := c.register("seatmap")
	eventID, _ := c.seatedEvent(organizer.Token, "Seat Map Fest")

	res := c.get("/api/v1/events/"+eventID.String()+"/seats", "")
	requireStatus(t, res, http.StatusOK)

	plan := res.Body["seat_map"].(map[string]any)
	if plan["venue_name"] != "Almaty Test Hall" {
		t.Errorf("venue = %v", plan["venue_name"])
	}
	if plan["total_seats"] != float64(fixtureSeats) {
		t.Errorf("total_seats = %v, want %d", plan["total_seats"], fixtureSeats)
	}
	if plan["available_seats"] != float64(fixtureSeats) {
		t.Errorf("available_seats = %v, want all of them free", plan["available_seats"])
	}

	// The bounding box lets a client set a viewBox without measuring.
	for _, key := range []string{"min_x", "max_x", "min_y", "max_y"} {
		if _, ok := plan[key].(float64); !ok {
			t.Errorf("no %s on the map", key)
		}
	}

	sections := plan["sections"].([]any)
	if len(sections) != 3 {
		t.Fatalf("sections = %d, want 3", len(sections))
	}

	// Every section is priced, or an attendee could not see what a seat costs
	// before choosing it (SRS 4.3.1).
	byName := map[string]map[string]any{}
	for _, s := range sections {
		section := s.(map[string]any)
		byName[section["name"].(string)] = section
	}
	if got := byName["Orchestra"]["price_kzt"]; got != "12000.00" {
		t.Errorf("Orchestra price = %v, want 12000.00", got)
	}
	// Balcony and Accessible Box share a price category, so matching on
	// category alone would price one as the other.
	if got := byName["Balcony"]["ticket_type_name"]; got != "Balcony" {
		t.Errorf("Balcony resolved to the %v tier", got)
	}
	if got := byName["Accessible Box"]["ticket_type_name"]; got != "Accessible Box" {
		t.Errorf("Accessible Box resolved to the %v tier", got)
	}

	// Accessible seats are flagged, because SRS 4.3.1 requires them to be
	// distinguishable rather than merely bookable.
	accessible := 0
	for _, seat := range seatsOf(t, res) {
		if seat["accessible"] == true {
			accessible++
		}
		if seat["status"] != "available" {
			t.Errorf("seat %v is %v before anything is sold", seat["number"], seat["status"])
		}
	}
	if accessible != fixtureAccessibleSeats {
		t.Errorf("accessible seats = %d, want %d", accessible, fixtureAccessibleSeats)
	}
}

// TestSeatMapShowsHeldAndSoldSeats is the state machine SRS 4.3.1 asks the map
// to distinguish.
func TestSeatMapShowsHeldAndSoldSeats(t *testing.T) {
	c := newClient(t)
	organizer := c.register("seatstates")
	eventID, _ := c.seatedEvent(organizer.Token, "Seat States Fest")

	all := seatsOf(t, c.get("/api/v1/events/"+eventID.String()+"/seats", ""))
	holdSeat := all[0]["id"].(string)
	buySeat := all[1]["id"].(string)

	// One seat held by an open basket.
	held := c.post("/api/v1/events/"+eventID.String()+"/holds", "",
		map[string]any{"seat_ids": []string{holdSeat}})
	requireStatus(t, held, http.StatusCreated)

	// One seat bought outright.
	bought := c.post("/api/v1/events/"+eventID.String()+"/checkout", "", map[string]any{
		"buyer_name": "Seat Buyer", "buyer_email": "seat@biletflow.test",
		"seat_ids": []string{buySeat},
	})
	requireStatus(t, bought, http.StatusCreated)

	states := map[string]string{}
	for _, seat := range seatsOf(t, c.get("/api/v1/events/"+eventID.String()+"/seats", "")) {
		states[seat["id"].(string)] = seat["status"].(string)
	}

	if states[holdSeat] != "held" {
		t.Errorf("the held seat reads %q, want held", states[holdSeat])
	}
	if states[buySeat] != "sold" {
		t.Errorf("the sold seat reads %q, want sold", states[buySeat])
	}
}

// TestSeatPriceComesFromTheSeat: a crafted basket must not buy an Orchestra
// seat at Balcony prices.
func TestSeatPriceComesFromTheSeat(t *testing.T) {
	c := newClient(t)
	organizer := c.register("seatprice")
	eventID, _ := c.seatedEvent(organizer.Token, "Seat Price Fest")

	res := c.get("/api/v1/events/"+eventID.String()+"/seats", "")
	plan := res.Body["seat_map"].(map[string]any)

	var orchestraSeat string
	for _, s := range plan["sections"].([]any) {
		section := s.(map[string]any)
		if section["name"] != "Orchestra" {
			continue
		}
		row := section["rows"].([]any)[0].(map[string]any)
		orchestraSeat = row["seats"].([]any)[0].(map[string]any)["id"].(string)
	}
	if orchestraSeat == "" {
		t.Fatal("no Orchestra seat on the map")
	}

	// The request names only the seat. The tier - and therefore the price - is
	// decided by the server from where the seat is.
	held := c.post("/api/v1/events/"+eventID.String()+"/holds", "",
		map[string]any{"seat_ids": []string{orchestraSeat}})
	requireStatus(t, held, http.StatusCreated)

	hold := held.Body["hold"].(map[string]any)
	if hold["subtotal_kzt"] != "12000.00" {
		t.Errorf("subtotal = %v, want the Orchestra price of 12000.00", hold["subtotal_kzt"])
	}
}

// TestTwoBasketsCannotHoldTheSameSeat is SRS 4.3.1's "the system shall prevent
// two orders from purchasing the same seat".
func TestTwoBasketsCannotHoldTheSameSeat(t *testing.T) {
	c := newClient(t)
	organizer := c.register("seatrace")
	eventID, _ := c.seatedEvent(organizer.Token, "Seat Race Fest")

	seat := seatsOf(t, c.get("/api/v1/events/"+eventID.String()+"/seats", ""))[0]["id"].(string)

	first := c.post("/api/v1/events/"+eventID.String()+"/holds", "",
		map[string]any{"seat_ids": []string{seat}})
	requireStatus(t, first, http.StatusCreated)

	second := c.post("/api/v1/events/"+eventID.String()+"/holds", "",
		map[string]any{"seat_ids": []string{seat}})
	if second.Status != http.StatusConflict {
		t.Fatalf("a second basket took the same seat: %d %s", second.Status, second.Raw)
	}

	// Releasing the first puts the seat back.
	orderID := first.Body["hold"].(map[string]any)["order_id"].(string)
	requireStatus(t, c.delete("/api/v1/orders/"+orderID+"/hold", ""), http.StatusOK)

	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/holds", "",
		map[string]any{"seat_ids": []string{seat}}), http.StatusCreated)
}

// TestSeatsAreRecordedOnTheTicket is SRS 4.3.1's "the assigned section, row and
// seat number shall be stored on the ticket and order item".
func TestSeatsAreRecordedOnTheTicket(t *testing.T) {
	c := newClient(t)
	organizer := c.register("seatticket")
	eventID, _ := c.seatedEvent(organizer.Token, "Seat Ticket Fest")

	all := seatsOf(t, c.get("/api/v1/events/"+eventID.String()+"/seats", ""))
	seats := []string{all[0]["id"].(string), all[1]["id"].(string)}

	bought := c.post("/api/v1/events/"+eventID.String()+"/checkout", "", map[string]any{
		"buyer_name": "Seated Buyer", "buyer_email": "seated@biletflow.test",
		"seat_ids": seats,
	})
	requireStatus(t, bought, http.StatusCreated)

	if tickets := bought.Body["tickets"].([]any); len(tickets) != 2 {
		t.Fatalf("tickets = %d, want one per seat", len(tickets))
	}

	// Every ticket carries its own seat: with quantity folded into one line,
	// two tickets would have shared a single seat.
	var distinct, withSection int
	if err := c.pool.QueryRow(c.t.Context(), `
		SELECT count(DISTINCT seat_id),
		       count(*) FILTER (WHERE seat_section IS NOT NULL
		                          AND seat_row IS NOT NULL
		                          AND seat_number IS NOT NULL)
		  FROM tickets WHERE order_id = $1`,
		bought.Body["order"].(map[string]any)["id"]).Scan(&distinct, &withSection); err != nil {
		t.Fatalf("read the tickets: %v", err)
	}
	if distinct != 2 {
		t.Errorf("distinct seats on the tickets = %d, want 2", distinct)
	}
	if withSection != 2 {
		t.Errorf("%d of 2 tickets carry section/row/number", withSection)
	}

	// And the order item records it too.
	var itemsWithSeats int
	if err := c.pool.QueryRow(c.t.Context(),
		`SELECT count(*) FROM order_items WHERE order_id = $1 AND seat_id IS NOT NULL`,
		bought.Body["order"].(map[string]any)["id"]).Scan(&itemsWithSeats); err != nil {
		t.Fatalf("read the order items: %v", err)
	}
	if itemsWithSeats != 2 {
		t.Errorf("order items with a seat = %d, want 2", itemsWithSeats)
	}
}

// TestSeatMapRefusedForGeneralAdmission: the map is meaningless without a plan.
func TestSeatMapRefusedForGeneralAdmission(t *testing.T) {
	c := newClient(t)
	organizer := c.register("seatnone")
	eventID, _, _ := c.sellableEvent(organizer.Token, "General Admission Fest", "5000", 10)

	requireErrorCode(t, c.get("/api/v1/events/"+eventID.String()+"/seats", ""),
		http.StatusConflict, CodeNoSeatingPlan)
}

// TestSeatFromAnotherEventIsRefused guards the resolver.
func TestSeatFromAnotherEventIsRefused(t *testing.T) {
	c := newClient(t)
	organizer := c.register("seatforeign")
	eventID, _, _ := c.sellableEvent(organizer.Token, "No Seats Here", "5000", 10)

	res := c.post("/api/v1/events/"+eventID.String()+"/holds", "",
		map[string]any{"seat_ids": []string{uuid.NewString()}})
	if res.Status != http.StatusConflict && res.Status != http.StatusNotFound {
		t.Errorf("a foreign seat was accepted: %d %s", res.Status, res.Raw)
	}
}
