package email

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func sampleOrder() OrderDetails {
	return OrderDetails{
		BuyerName:   "Aliya Toleuova",
		BuyerEmail:  "aliya@biletflow.test",
		OrderNumber: "BF-ABC123XYZ0",
		EventTitle:  "Almaty Autumn Fest 2026",
		EventVenue:  "Gorky Park Stage, Almaty",
		EventWhen:   "Sat 17 Oct 2026, 19:00 (+06)",
		TotalKZT:    "10000.00",
		OrderURL:    "http://localhost:3000/orders/11111111-1111-1111-1111-111111111111",
		Tickets: []TicketLine{
			{TicketCode: "BF-TKT-AAAA", TypeName: "General Admission",
				PDFURL: "http://localhost:8080/api/v1/tickets/aaaa/pdf"},
			{TicketCode: "BF-TKT-BBBB", TypeName: "General Admission",
				PDFURL: "http://localhost:8080/api/v1/tickets/bbbb/pdf"},
		},
	}
}

func TestOrderConfirmationCarriesWhatTheAttendeeNeeds(t *testing.T) {
	msg := OrderConfirmation(sampleOrder())

	if msg.To != "aliya@biletflow.test" {
		t.Errorf("To = %q", msg.To)
	}
	// The subject is specified by the phase brief: "Your Tickets to [Event]".
	if msg.Subject != "Your Tickets to Almaty Autumn Fest 2026" {
		t.Errorf("Subject = %q", msg.Subject)
	}
	if msg.Type != TypeOrderConfirmation {
		t.Errorf("Type = %q", msg.Type)
	}

	for _, want := range []string{
		"Hi Aliya,",
		"BF-ABC123XYZ0",
		"Gorky Park Stage, Almaty",
		"Sat 17 Oct 2026, 19:00 (+06)",
		"10000.00 KZT (simulated payment)",
		"http://localhost:8080/api/v1/tickets/aaaa/pdf",
		"http://localhost:8080/api/v1/tickets/bbbb/pdf",
		"http://localhost:3000/orders/11111111-1111-1111-1111-111111111111",
	} {
		if !strings.Contains(msg.Body, want) {
			t.Errorf("body is missing %q", want)
		}
	}
}

// TestOrderConfirmationDegradesGracefully covers the fields a guest checkout
// can legitimately leave empty. An email is not the place to discover that a
// venue was optional.
func TestOrderConfirmationDegradesGracefully(t *testing.T) {
	d := sampleOrder()
	d.BuyerName = ""
	d.EventVenue = ""
	d.EventWhen = ""
	d.OrderURL = ""
	d.Tickets = nil

	body := OrderConfirmation(d).Body

	if !strings.Contains(body, "Hi there,") {
		t.Errorf("a missing name should greet the reader anyway; got:\n%s", body)
	}
	for _, unwanted := range []string{"When:", "Where:", "Download your tickets"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("body still has an empty %q section:\n%s", unwanted, body)
		}
	}
}

func TestOrderConfirmationCountsTickets(t *testing.T) {
	one := sampleOrder()
	one.Tickets = one.Tickets[:1]
	if !strings.Contains(OrderConfirmation(one).Body, "Your ticket is attached") {
		t.Error("a single ticket should not be described in the plural")
	}
	if !strings.Contains(OrderConfirmation(sampleOrder()).Body, "Your 2 tickets are attached") {
		t.Error("two tickets should be counted")
	}
}

func TestRefundCompletedExplainsTheConsequence(t *testing.T) {
	msg := RefundCompleted(RefundDetails{
		BuyerName:   "Dana Kim",
		BuyerEmail:  "dana@biletflow.test",
		OrderNumber: "BF-REFUND01",
		EventTitle:  "Almaty Autumn Fest 2026",
		AmountKZT:   "10000.00",
		TicketCount: 2,
		Reason:      "Event rescheduled",
	})

	if msg.Subject != "Refund for Almaty Autumn Fest 2026" {
		t.Errorf("Subject = %q", msg.Subject)
	}
	if msg.Type != TypeRefundCompleted {
		t.Errorf("Type = %q", msg.Type)
	}
	for _, want := range []string{
		"Hi Dana,", "BF-REFUND01", "10000.00 KZT (simulated)",
		"2, now void", "Event rescheduled", "no longer be admitted",
	} {
		if !strings.Contains(msg.Body, want) {
			t.Errorf("body is missing %q", want)
		}
	}
}

// TestRenderIsReadable pins the console format the phase brief asks for: To,
// Subject and a body, framed and unmistakably labelled as simulated.
func TestRenderIsReadable(t *testing.T) {
	out := Render(Message{
		Type:    TypeOrderConfirmation,
		To:      "aliya@biletflow.test",
		Subject: "Your Tickets to Almaty Autumn Fest 2026",
		Body:    "Hi Aliya,\n\nYour order is confirmed.\n",
	})

	for _, want := range []string{
		"MOCK EMAIL - simulated delivery",
		"To:      aliya@biletflow.test",
		"Subject: Your Tickets to Almaty Autumn Fest 2026",
		"Type:    " + TypeOrderConfirmation,
		"  Hi Aliya,",
		"  Your order is confirmed.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered block is missing %q; got:\n%s", want, out)
		}
	}

	// Every line stays inside the frame, so the output survives a narrow
	// terminal without wrapping into nonsense.
	for _, line := range strings.Split(out, "\n") {
		if len(line) > ruleWidth+4 {
			t.Errorf("line is %d chars, wider than the frame: %q", len(line), line)
		}
	}
}

func TestConsoleSenderWritesToItsWriter(t *testing.T) {
	var buf bytes.Buffer
	sender := NewConsoleSender(&buf)

	if err := sender.Send(context.Background(), Message{
		To: "x@biletflow.test", Subject: "One", Body: "first",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := sender.Send(context.Background(), Message{
		To: "y@biletflow.test", Subject: "Two", Body: "second",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Subject: One") || !strings.Contains(out, "Subject: Two") {
		t.Errorf("both messages should appear:\n%s", out)
	}
	if strings.Index(out, "Subject: One") > strings.Index(out, "Subject: Two") {
		t.Error("messages came out in the wrong order")
	}
}

// TestConsoleSenderDoesNotInterleave is why Send builds the whole block before
// writing: two goroutines printing line by line would produce a mess that is
// unreadable exactly when you need to read it.
func TestConsoleSenderDoesNotInterleave(t *testing.T) {
	var buf bytes.Buffer
	sender := NewConsoleSender(&buf)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = sender.Send(context.Background(), Message{
				To:      "concurrent@biletflow.test",
				Subject: "Concurrent",
				Body:    strings.Repeat("line\n", 5),
			})
		}(i)
	}
	wg.Wait()

	// Each message contributes exactly one header pair, and no header may land
	// inside another message's body.
	if got := strings.Count(buf.String(), "Subject: Concurrent"); got != 20 {
		t.Errorf("subject lines = %d, want 20", got)
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "line") && trimmed != "line" {
			t.Errorf("a body line was corrupted by a concurrent write: %q", line)
		}
	}
}

// failingSender always fails, which is how a real transport behaves on a bad
// day.
type failingSender struct{ err error }

func (f failingSender) Send(context.Context, Message) error { return f.err }

func TestMailerReportsFailuresToItsCallback(t *testing.T) {
	wanted := errors.New("smtp is down")

	var (
		mu     sync.Mutex
		gotErr error
		gotRef string
	)
	mailer := NewMailer(failingSender{wanted}, func(msg Message, err error) {
		mu.Lock()
		defer mu.Unlock()
		gotErr, gotRef = err, msg.Ref
	})

	mailer.SendAsync(Message{To: "x@biletflow.test", Ref: "row-1"})
	mailer.Wait()

	mu.Lock()
	defer mu.Unlock()
	if !errors.Is(gotErr, wanted) {
		t.Errorf("callback error = %v, want %v", gotErr, wanted)
	}
	if gotRef != "row-1" {
		t.Errorf("the caller's reference was lost: %q", gotRef)
	}
}

// TestMailerWaitIsWhatMakesAsyncTestable: without Wait, an assertion after
// SendAsync would be a race rather than a test.
func TestMailerWaitDrainsEverything(t *testing.T) {
	recorder := NewRecorder()
	mailer := NewMailer(recorder, nil)

	for i := 0; i < 50; i++ {
		mailer.SendAsync(Message{To: "bulk@biletflow.test", Subject: "Bulk"})
	}
	mailer.Wait()

	if got := len(recorder.Messages()); got != 50 {
		t.Errorf("delivered = %d, want 50", got)
	}
}

func TestNilMailerIsSafe(t *testing.T) {
	var mailer *Mailer
	mailer.SendAsync(Message{To: "nobody@biletflow.test"})
	mailer.Wait()
}
