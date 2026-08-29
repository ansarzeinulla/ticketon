package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// activationOf reads the activation object out of a response.
func activationOf(t *testing.T, res response) map[string]any {
	t.Helper()
	a, ok := res.Body["activation"].(map[string]any)
	if !ok {
		t.Fatalf("no activation in response: %s", res.Raw)
	}
	return a
}

// TestFreeEventsNeedNoActivation is the other half of the gate: activation
// exists to clear an organizer to take money, so an event that takes none must
// not be blocked by it.
func TestFreeEventsNeedNoActivation(t *testing.T) {
	c := newClient(t)
	organizer := c.register("freeevent")
	eventID, _, ticketTypeID := c.sellableEvent(organizer.Token, "Free Community Meetup", "0", 10)

	requireStatus(t, c.buy(eventID, ticketTypeID, 1, "Free Rider", "free@biletflow.test"),
		http.StatusCreated)

	state := c.get("/api/v1/events/"+eventID.String()+"/activation", organizer.Token)
	requireStatus(t, state, http.StatusOK)
	activation := activationOf(t, state)

	if activation["required_for_sales"] != false {
		t.Errorf("a free event reports activation as required: %s", state.Raw)
	}
	if activation["is_active"] != false {
		t.Errorf("a free event was activated without anybody asking: %s", state.Raw)
	}
}

// TestMixedEventSellsOnlyItsFreeTierBeforeActivation covers the awkward middle
// case: one event, a free tier and a paid tier, activation outstanding.
func TestMixedEventSellsOnlyItsFreeTierBeforeActivation(t *testing.T) {
	c := newClient(t)
	organizer := c.register("mixedtiers")
	eventID, _ := c.createEvent(organizer.Token, "Mixed Tier Fest")

	freeID, _ := c.createTicketType(organizer.Token, eventID, ticketTypeBody("Free Entry", "0", 10))
	paidID, _ := c.createTicketType(organizer.Token, eventID, ticketTypeBody("VIP", "10000", 10))
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/publish", organizer.Token, nil),
		http.StatusOK)

	// The free tier is unaffected.
	requireStatus(t, c.buy(eventID, freeID, 1, "Free Attendee", "freetier@biletflow.test"),
		http.StatusCreated)

	// The paid tier is refused.
	requireErrorCode(t, c.buy(eventID, paidID, 1, "Paying Attendee", "paidtier@biletflow.test"),
		http.StatusForbidden, CodePaidSalesNotActive)

	// A basket mixing the two is refused as a whole: one paid line is enough,
	// and a partial order would be a surprise nobody asked for.
	mixed := c.post("/api/v1/events/"+eventID.String()+"/checkout", "", map[string]any{
		"buyer_name": "Mixed", "buyer_email": "mixed@biletflow.test",
		"items": []map[string]any{
			{"ticket_type_id": freeID.String(), "quantity": 1},
			{"ticket_type_id": paidID.String(), "quantity": 1},
		},
	})
	requireErrorCode(t, mixed, http.StatusForbidden, CodePaidSalesNotActive)

	// Only the one free ticket exists.
	var issued int
	if err := c.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM tickets WHERE event_id = $1`, eventID).Scan(&issued); err != nil {
		t.Fatalf("count tickets: %v", err)
	}
	if issued != 1 {
		t.Errorf("tickets issued = %d, want only the free one", issued)
	}

	c.activatePaidSales(organizer.Token, eventID)
	requireStatus(t, c.buy(eventID, paidID, 1, "Now Paying", "nowpaying@biletflow.test"),
		http.StatusCreated)
}

func TestActivationIsIdempotent(t *testing.T) {
	c := newClient(t)
	organizer := c.register("activationtwice")
	eventID, _ := c.createEvent(organizer.Token, "Idempotent Activation")
	c.createTicketType(organizer.Token, eventID, ticketTypeBody("GA", "5000", 10))

	c.activatePaidSales(organizer.Token, eventID)
	first := c.get("/api/v1/events/"+eventID.String()+"/activation", organizer.Token)
	activatedAt := activationOf(t, first)["activated_at"]

	// Submitting the whole checklist again must not mint a second fee payment
	// or move the activation timestamp.
	c.activatePaidSales(organizer.Token, eventID)

	second := c.get("/api/v1/events/"+eventID.String()+"/activation", organizer.Token)
	if got := activationOf(t, second)["activated_at"]; got != activatedAt {
		t.Errorf("activated_at moved from %v to %v on a repeat submission", activatedAt, got)
	}

	var payments int
	if err := c.pool.QueryRow(t.Context(), `
		SELECT count(*) FROM payments
		 WHERE event_id = $1 AND purpose = 'paid_sales_activation'`, eventID).Scan(&payments); err != nil {
		t.Fatalf("count activation payments: %v", err)
	}
	if payments != 1 {
		t.Errorf("activation fee payments = %d, want exactly 1", payments)
	}
}

func TestActivationIsOrganizerOnly(t *testing.T) {
	c := newClient(t)
	organizer := c.register("activationowner")
	stranger := c.register("activationstranger")
	eventID, _ := c.createEvent(organizer.Token, "Owner Only Activation")
	c.createTicketType(organizer.Token, eventID, ticketTypeBody("GA", "5000", 10))

	path := "/api/v1/events/" + eventID.String() + "/activation"

	requireStatus(t, c.get(path, ""), http.StatusUnauthorized)
	requireStatus(t, c.get(path, stranger.Token), http.StatusForbidden)
	requireStatus(t, c.post(path, stranger.Token,
		map[string]any{"accept_terms": true}), http.StatusForbidden)

	// An empty submission is a validation error, not a silent no-op.
	requireStatus(t, c.post(path, organizer.Token, map[string]any{}), http.StatusBadRequest)
}

// TestActivationSetsPaidSalesEnabled checks the flag the rest of the schema
// reads stays in step with the activation row.
func TestActivationSetsPaidSalesEnabled(t *testing.T) {
	c := newClient(t)
	organizer := c.register("activationflag")
	eventID, _ := c.createEvent(organizer.Token, "Flag Sync Fest")
	c.createTicketType(organizer.Token, eventID, ticketTypeBody("GA", "5000", 10))

	enabled := func() bool {
		var v bool
		if err := c.pool.QueryRow(t.Context(),
			`SELECT paid_sales_enabled FROM events WHERE id = $1`, eventID).Scan(&v); err != nil {
			t.Fatalf("read paid_sales_enabled: %v", err)
		}
		return v
	}

	if enabled() {
		t.Fatal("a new event already has paid sales enabled")
	}
	c.activatePaidSales(organizer.Token, eventID)
	if !enabled() {
		t.Error("paid_sales_enabled is still false after activation")
	}
}

// TestPlatformAdminCanSuspendPaidSales covers the last bullet of SRS 4.5.
func TestPlatformAdminCanSuspendPaidSales(t *testing.T) {
	c := newClient(t)
	organizer := c.register("paidsuspend")
	admin := c.register("paidsuspendadmin")
	c.makePlatformAdmin(admin.ID)

	eventID, _ := c.createEvent(organizer.Token, "Paid Suspension Fest")
	freeID, _ := c.createTicketType(organizer.Token, eventID, ticketTypeBody("Free", "0", 10))
	paidID, _ := c.createTicketType(organizer.Token, eventID, ticketTypeBody("Paid", "5000", 10))
	c.activatePaidSales(organizer.Token, eventID)
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/publish", organizer.Token, nil),
		http.StatusOK)

	requireStatus(t, c.buy(eventID, paidID, 1, "Before", "before@biletflow.test"),
		http.StatusCreated)

	suspendPath := "/api/v1/admin/events/" + eventID.String() + "/paid-sales/suspend"

	// The organizer cannot suspend their own paid sales, and nor can a stranger.
	requireStatus(t, c.post(suspendPath, organizer.Token, nil), http.StatusForbidden)
	requireStatus(t, c.post(suspendPath, "", nil), http.StatusUnauthorized)

	suspended := c.post(suspendPath, admin.Token, map[string]any{"reason": "Chargeback pattern"})
	requireStatus(t, suspended, http.StatusOK)
	if activationOf(t, suspended)["status"] != "suspended" {
		t.Errorf("status = %v, want suspended", activationOf(t, suspended)["status"])
	}

	// Paid tickets stop; free registration carries on, which is the point of
	// this being narrower than suspending the whole event.
	requireErrorCode(t, c.buy(eventID, paidID, 1, "After", "after@biletflow.test"),
		http.StatusForbidden, CodePaidSalesNotActive)
	requireStatus(t, c.buy(eventID, freeID, 1, "Still Free", "stillfree@biletflow.test"),
		http.StatusCreated)

	// The organizer cannot re-activate their way out of a suspension.
	requireStatus(t, c.post("/api/v1/events/"+eventID.String()+"/activation", organizer.Token,
		map[string]any{"accept_terms": true}), http.StatusForbidden)

	// Only the platform can lift it.
	lifted := c.post("/api/v1/admin/events/"+eventID.String()+"/paid-sales/unsuspend", admin.Token, nil)
	requireStatus(t, lifted, http.StatusOK)
	if activationOf(t, lifted)["is_active"] != true {
		t.Errorf("lifting the suspension did not restore an already-complete checklist: %s", lifted.Raw)
	}
	requireStatus(t, c.buy(eventID, paidID, 1, "Restored", "restored@biletflow.test"),
		http.StatusCreated)
}

func TestSuspendPaidSalesNeedsAnActivation(t *testing.T) {
	c := newClient(t)
	admin := c.register("suspendnothing")
	c.makePlatformAdmin(admin.ID)
	organizer := c.register("suspendnothingorg")
	eventID, _ := c.createEvent(organizer.Token, "Never Activated")

	requireStatus(t, c.post("/api/v1/admin/events/"+eventID.String()+"/paid-sales/suspend",
		admin.Token, nil), http.StatusNotFound)

	requireStatus(t, c.post("/api/v1/admin/events/"+uuid.NewString()+"/paid-sales/suspend",
		admin.Token, nil), http.StatusNotFound)
}
