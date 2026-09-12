package api

import (
	"net/http"
	"testing"
)

// SRS 4.9 ("Attendees shall be able to view their orders") and SRS 4.7
// ("through the attendee's account"): an order page reachable only by its
// unguessable link is not enough once somebody has an account.
func TestMyOrdersListsTheBuyersOwnOrders(t *testing.T) {
	c := newClient(t)
	organizer := c.register("myordersorg")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "My Orders Event", "0", 20)
	path := "/api/v1/events/" + eventID.String() + "/checkout"

	buyer := c.register("myordersbuyer")
	requireStatus(t, c.post(path, buyer.Token, map[string]any{
		"buyer_name":  "Buyer One",
		"buyer_email": buyer.Email,
		"items":       []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 2}},
	}), http.StatusCreated)

	res := c.get("/api/v1/orders", buyer.Token)
	requireStatus(t, res, http.StatusOK)

	orders, _ := res.Body["orders"].([]any)
	if len(orders) != 1 {
		t.Fatalf("%d orders for the buyer, want 1; body = %s", len(orders), res.Raw)
	}
	order, _ := orders[0].(map[string]any)
	if order["event_title"] != "My Orders Event" {
		t.Errorf("event_title = %v, want the event the order was placed against", order["event_title"])
	}
	if count, _ := order["ticket_count"].(float64); int(count) != 2 {
		t.Errorf("ticket_count = %v, want 2", order["ticket_count"])
	}
	if order["order_number"] == nil || order["order_number"] == "" {
		t.Error("the listing carries no order number to identify the order by")
	}

	// Somebody else's account sees none of it.
	other := c.register("myordersother")
	otherRes := c.get("/api/v1/orders", other.Token)
	requireStatus(t, otherRes, http.StatusOK)
	if got, _ := otherRes.Body["orders"].([]any); len(got) != 0 {
		t.Errorf("a different account sees %d of the buyer's orders, want 0", len(got))
	}

	requireErrorCode(t, c.get("/api/v1/orders", ""), http.StatusUnauthorized, "unauthorized")
}

// A guest checkout is claimed by whoever later registers that address - the
// same bridge the support desk uses, so tickets bought before signing up are
// not stranded.
func TestMyOrdersClaimsGuestOrdersByEmail(t *testing.T) {
	c := newClient(t)
	organizer := c.register("guestbridgeorg")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Guest Bridge Event", "0", 20)

	guestEmail := nextEmail("guestbridgebuyer")
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/checkout", "", map[string]any{
		"buyer_name":  "Guest Buyer",
		"buyer_email": guestEmail,
		"items":       []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 1}},
	}), http.StatusCreated)

	// Register with the address the guest checkout used.
	res := c.post("/api/v1/auth/register", "", map[string]any{
		"email": guestEmail, "password": "correct horse battery",
	})
	requireStatus(t, res, http.StatusCreated)
	token, _ := res.Body["access_token"].(string)

	listed := c.get("/api/v1/orders", token)
	requireStatus(t, listed, http.StatusOK)
	if got, _ := listed.Body["orders"].([]any); len(got) != 1 {
		t.Errorf("%d orders after claiming the guest checkout, want 1; body = %s",
			len(got), listed.Raw)
	}
}
