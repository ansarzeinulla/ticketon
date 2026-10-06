package api

import (
	"net/http"
	"testing"
)

// SRS 4.6: "Organizers shall be able to view gross sales, fees, refunds, and
// estimated payouts." The processing fee is charged inside the order total, so
// without these two figures an organizer cannot tell what BiletFlow kept or
// what they would actually be paid.
func TestAnalyticsReportsFeesAndEstimatedPayout(t *testing.T) {
	c := newClient(t)
	organizer := c.register("payoutfigures")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Payout Figures Event", "10000", 10)
	path := "/api/v1/events/" + eventID.String() + "/analytics"

	// Two tickets at 10000 = 20000 of ticket money, plus the 3.5% fee = 700.
	requireStatus(t, c.buy(eventID, ticketTypeID, 2, "Payer", "payer@biletflow.test"),
		http.StatusCreated)

	res := c.get(path, organizer.Token)
	requireStatus(t, res, http.StatusOK)
	a := analytics(res)

	if a["gross_revenue_kzt"] != "20700.00" {
		t.Errorf("gross_revenue_kzt = %v, want 20700.00 (20000 + 3.5%% fee)", a["gross_revenue_kzt"])
	}
	if a["fees_kzt"] != "700.00" {
		t.Errorf("fees_kzt = %v, want 700.00", a["fees_kzt"])
	}
	// The organizer keeps the ticket money, not the fee.
	if a["estimated_payout_kzt"] != "20000.00" {
		t.Errorf("estimated_payout_kzt = %v, want 20000.00 (gross minus the fee)",
			a["estimated_payout_kzt"])
	}

	// A refunded order pays out nothing, while gross stays historical.
	orderID := c.get("/api/v1/events/"+eventID.String()+"/orders", organizer.Token)
	requireStatus(t, orderID, http.StatusOK)
	orders, _ := orderID.Body["orders"].([]any)
	if len(orders) == 0 {
		t.Fatal("no orders listed for the event")
	}
	first, _ := orders[0].(map[string]any)
	id, _ := first["id"].(string)
	requireStatus(t, c.post("/api/v1/orders/"+id+"/refund", organizer.Token,
		map[string]any{"reason": "testing payout figures"}), http.StatusOK)

	after := analytics(c.get(path, organizer.Token))
	if after["gross_revenue_kzt"] != "20700.00" {
		t.Errorf("gross after refund = %v, want it to stay historical at 20700.00",
			after["gross_revenue_kzt"])
	}
	if after["estimated_payout_kzt"] != "0.00" {
		t.Errorf("estimated_payout_kzt after a full refund = %v, want 0.00",
			after["estimated_payout_kzt"])
	}
}
