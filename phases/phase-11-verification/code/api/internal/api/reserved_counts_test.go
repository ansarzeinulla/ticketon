package api

import (
	"net/http"
	"testing"
)

// SRS 4.3 asks the organizer to see "available, reserved, sold, refunded and
// checked-in" tickets. Reserved is stock a cart is holding but has not paid
// for, and it used to be summed in SQL and thrown away.
func TestAnalyticsReportsReservedStock(t *testing.T) {
	c := newClient(t)
	organizer := c.register("reservedcounts")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Reserved Counts Event", "1000", 10)
	path := "/api/v1/events/" + eventID.String() + "/analytics"

	before := c.get(path, organizer.Token)
	requireStatus(t, before, http.StatusOK)
	if got := number(analytics(before), "tickets_reserved"); got != 0 {
		t.Fatalf("tickets_reserved = %d before any hold, want 0", got)
	}

	// Open a cart without paying: three seats are now held, not sold.
	held := c.post("/api/v1/events/"+eventID.String()+"/holds", "", map[string]any{
		"items": []map[string]any{{"ticket_type_id": ticketTypeID.String(), "quantity": 3}},
	})
	requireStatus(t, held, http.StatusCreated)

	after := c.get(path, organizer.Token)
	requireStatus(t, after, http.StatusOK)

	if got := number(analytics(after), "tickets_reserved"); got != 3 {
		t.Errorf("tickets_reserved = %d while a cart holds 3, want 3", got)
	}
	if got := number(analytics(after), "tickets_sold"); got != 0 {
		t.Errorf("tickets_sold = %d, want 0 - a held seat is not a sale", got)
	}

	// The per-tier breakdown carries it too, so the organizer can see which
	// tier the held stock is in.
	tiers, _ := analytics(after)["by_ticket_type"].([]any)
	if len(tiers) != 1 {
		t.Fatalf("%d ticket-type rows, want 1", len(tiers))
	}
	tier, _ := tiers[0].(map[string]any)
	if got, _ := tier["reserved"].(float64); int(got) != 3 {
		t.Errorf("per-tier reserved = %v, want 3", tier["reserved"])
	}
}
