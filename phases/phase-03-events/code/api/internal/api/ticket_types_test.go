package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ticketTypeBody is a valid create-ticket-type payload.
func ticketTypeBody(name string, price string, quantity int) map[string]any {
	return map[string]any{
		"name":           name,
		"price_kzt":      price,
		"quantity_total": quantity,
	}
}

// createTicketType posts a ticket type and returns its id.
func (c *client) createTicketType(token string, eventID uuid.UUID, body map[string]any) (uuid.UUID, response) {
	c.t.Helper()

	res := c.post("/api/v1/events/"+eventID.String()+"/ticket-types", token, body)
	if res.Status != http.StatusCreated {
		c.t.Fatalf("create ticket type: status = %d, body = %s", res.Status, res.Raw)
	}

	tt, _ := res.Body["ticket_type"].(map[string]any)
	id, err := uuid.Parse(tt["id"].(string))
	if err != nil {
		c.t.Fatalf("ticket type id is not a uuid: %v", err)
	}
	return id, res
}

func ticketType(res response) map[string]any {
	tt, _ := res.Body["ticket_type"].(map[string]any)
	return tt
}

func TestCreateTicketType(t *testing.T) {
	c := newClient(t)
	owner := c.register("ttowner")
	eventID, _ := c.createEvent(owner.Token, "Ticket Type Event")

	res := c.post("/api/v1/events/"+eventID.String()+"/ticket-types", owner.Token, map[string]any{
		"name":           "General Admission",
		"description":    "Standing room",
		"price_kzt":      "5000",
		"quantity_total": 5,
		"max_per_order":  4,
	})
	requireStatus(t, res, http.StatusCreated)

	tt := ticketType(res)
	if tt["name"] != "General Admission" {
		t.Errorf("name = %v, want General Admission", tt["name"])
	}
	if tt["price_kzt"] != "5000.00" {
		t.Errorf("price_kzt = %v, want the numeric(14,2) form 5000.00", tt["price_kzt"])
	}
	if qty, _ := tt["quantity_total"].(float64); int(qty) != 5 {
		t.Errorf("quantity_total = %v, want 5", tt["quantity_total"])
	}
	if sold, _ := tt["quantity_sold"].(float64); int(sold) != 0 {
		t.Errorf("quantity_sold = %v, want 0 on a new type", tt["quantity_sold"])
	}
	if remaining, _ := tt["quantity_remaining"].(float64); int(remaining) != 5 {
		t.Errorf("quantity_remaining = %v, want 5", tt["quantity_remaining"])
	}
	if tt["is_free"] != false {
		t.Errorf("is_free = %v, want false for a priced type", tt["is_free"])
	}

	// And it is really in PostgreSQL.
	var (
		name     string
		price    string
		total    int
		sold     int
		maxOrder int
	)
	err := c.pool.QueryRow(t.Context(), `
		SELECT name, price_kzt::text, quantity_total, quantity_sold, max_per_order
		  FROM ticket_types WHERE event_id = $1`, eventID).
		Scan(&name, &price, &total, &sold, &maxOrder)
	if err != nil {
		t.Fatalf("the ticket type is not in the database: %v", err)
	}
	if name != "General Admission" || price != "5000.00" || total != 5 || sold != 0 || maxOrder != 4 {
		t.Errorf("db row = (%q, %q, %d, %d, %d), want (General Admission, 5000.00, 5, 0, 4)",
			name, price, total, sold, maxOrder)
	}
}

func TestCreateFreeTicketType(t *testing.T) {
	c := newClient(t)
	owner := c.register("ttfree")
	eventID, _ := c.createEvent(owner.Token, "Free Event")

	res := c.post("/api/v1/events/"+eventID.String()+"/ticket-types", owner.Token, map[string]any{
		"name":           "Free Entry",
		"quantity_total": 100,
	})
	requireStatus(t, res, http.StatusCreated)

	tt := ticketType(res)
	if tt["price_kzt"] != "0.00" {
		t.Errorf("price_kzt = %v, want 0.00 by default", tt["price_kzt"])
	}
	if tt["is_free"] != true {
		t.Errorf("is_free = %v, want true for a zero-price type", tt["is_free"])
	}
}

func TestCreateTicketTypeValidation(t *testing.T) {
	c := newClient(t)
	owner := c.register("ttvalidation")
	eventID, _ := c.createEvent(owner.Token, "Validation Event")
	path := "/api/v1/events/" + eventID.String() + "/ticket-types"

	start := time.Now().Add(24 * time.Hour).UTC()

	tests := []struct {
		name      string
		body      map[string]any
		wantField string
	}{
		{"missing name", map[string]any{"quantity_total": 10}, "name"},
		{"blank name", map[string]any{"name": "  ", "quantity_total": 10}, "name"},
		{"missing quantity", map[string]any{"name": "No Quantity"}, "quantity_total"},
		{"negative quantity", map[string]any{"name": "Negative", "quantity_total": -1}, "quantity_total"},
		{"negative price", map[string]any{"name": "Cheap", "price_kzt": "-100", "quantity_total": 5}, "price_kzt"},
		{"price not a number", map[string]any{"name": "Words", "price_kzt": "five thousand", "quantity_total": 5}, "price_kzt"},
		{"price with too many decimals", map[string]any{"name": "Precise", "price_kzt": "10.005", "quantity_total": 5}, "price_kzt"},
		{"zero per-order limit", map[string]any{"name": "Zero", "quantity_total": 5, "max_per_order": 0}, "max_per_order"},
		{"sales end before start", map[string]any{
			"name": "Backwards", "quantity_total": 5,
			"sales_start_at": start.Format(time.RFC3339),
			"sales_end_at":   start.Add(-time.Hour).Format(time.RFC3339),
		}, "sales_end_at"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.post(path, owner.Token, tt.body)
			requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")
			if _, ok := res.errorFields()[tt.wantField]; !ok {
				t.Errorf("error fields = %v, want an entry for %q", res.errorFields(), tt.wantField)
			}
		})
	}

	var count int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM ticket_types WHERE event_id = $1`, eventID).Scan(&count); err != nil {
		t.Fatalf("count ticket types: %v", err)
	}
	if count != 0 {
		t.Errorf("%d invalid ticket types reached the database, want 0", count)
	}
}

func TestTicketTypeNamesAreUniquePerEvent(t *testing.T) {
	c := newClient(t)
	owner := c.register("ttunique")
	eventID, _ := c.createEvent(owner.Token, "Unique Names Event")

	c.createTicketType(owner.Token, eventID, ticketTypeBody("Standard", "5000", 10))

	res := c.post("/api/v1/events/"+eventID.String()+"/ticket-types", owner.Token,
		ticketTypeBody("Standard", "7000", 10))
	requireErrorCode(t, res, http.StatusUnprocessableEntity, "validation_failed")
	if _, ok := res.errorFields()["name"]; !ok {
		t.Errorf("error fields = %v, want an entry for name", res.errorFields())
	}

	// The same name on a different event is fine.
	otherID, _ := c.createEvent(owner.Token, "Another Event")
	c.createTicketType(owner.Token, otherID, ticketTypeBody("Standard", "5000", 10))
}

func TestTicketTypeAuthorization(t *testing.T) {
	c := newClient(t)
	owner := c.register("ttauthowner")
	other := c.register("ttauthother")
	eventID, _ := c.createEvent(owner.Token, "Protected Ticket Types")
	path := "/api/v1/events/" + eventID.String() + "/ticket-types"

	body := ticketTypeBody("Sneaky", "1000", 5)

	requireErrorCode(t, c.post(path, "", body), http.StatusUnauthorized, "unauthorized")
	requireErrorCode(t, c.post(path, other.Token, body), http.StatusForbidden, "forbidden")
	requireErrorCode(t, c.get(path, ""), http.StatusUnauthorized, "unauthorized")
	requireErrorCode(t, c.get(path, other.Token), http.StatusForbidden, "forbidden")

	ticketTypeID, _ := c.createTicketType(owner.Token, eventID, ticketTypeBody("Standard", "5000", 5))
	ttPath := "/api/v1/ticket-types/" + ticketTypeID.String()

	requireErrorCode(t, c.patch(ttPath, other.Token, map[string]any{"price_kzt": "1"}),
		http.StatusForbidden, "forbidden")
	requireErrorCode(t, c.delete(ttPath, other.Token), http.StatusForbidden, "forbidden")
}

func TestListTicketTypesIncludesHiddenForOrganizer(t *testing.T) {
	c := newClient(t)
	owner := c.register("ttlist")
	eventID, _ := c.createEvent(owner.Token, "Listing Event")

	c.createTicketType(owner.Token, eventID, ticketTypeBody("Visible", "5000", 10))

	hidden := ticketTypeBody("Backstage", "50000", 2)
	hidden["is_hidden"] = true
	c.createTicketType(owner.Token, eventID, hidden)

	res := c.get("/api/v1/events/"+eventID.String()+"/ticket-types", owner.Token)
	requireStatus(t, res, http.StatusOK)

	types, _ := res.Body["ticket_types"].([]any)
	if len(types) != 2 {
		t.Fatalf("organizer sees %d ticket types, want both including the hidden one", len(types))
	}
}

func TestUpdateTicketType(t *testing.T) {
	c := newClient(t)
	owner := c.register("ttupdate")
	eventID, _ := c.createEvent(owner.Token, "Update Event")
	ticketTypeID, _ := c.createTicketType(owner.Token, eventID, ticketTypeBody("Early Bird", "3000", 20))
	path := "/api/v1/ticket-types/" + ticketTypeID.String()

	res := c.patch(path, owner.Token, map[string]any{"price_kzt": "4500", "quantity_total": 30})
	requireStatus(t, res, http.StatusOK)

	tt := ticketType(res)
	if tt["price_kzt"] != "4500.00" {
		t.Errorf("price_kzt = %v, want 4500.00", tt["price_kzt"])
	}
	if qty, _ := tt["quantity_total"].(float64); int(qty) != 30 {
		t.Errorf("quantity_total = %v, want 30", tt["quantity_total"])
	}
	if tt["name"] != "Early Bird" {
		t.Errorf("name changed to %v without being sent", tt["name"])
	}

	// Hiding a type is how an organizer withdraws it from sale.
	res = c.patch(path, owner.Token, map[string]any{"is_hidden": true})
	requireStatus(t, res, http.StatusOK)
	if ticketType(res)["is_hidden"] != true {
		t.Error("is_hidden was not applied")
	}
}

func TestDeleteTicketType(t *testing.T) {
	c := newClient(t)
	owner := c.register("ttdelete")
	eventID, _ := c.createEvent(owner.Token, "Delete Event")
	ticketTypeID, _ := c.createTicketType(owner.Token, eventID, ticketTypeBody("Removable", "1000", 5))

	requireStatus(t, c.delete("/api/v1/ticket-types/"+ticketTypeID.String(), owner.Token),
		http.StatusNoContent)

	var count int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM ticket_types WHERE id = $1`, ticketTypeID).Scan(&count); err != nil {
		t.Fatalf("count ticket types: %v", err)
	}
	if count != 0 {
		t.Error("the ticket type is still in the database after a successful delete")
	}
}
