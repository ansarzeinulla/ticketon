package email

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func sampleAccount() AccountTokenDetails {
	return AccountTokenDetails{
		FullName:  "Aliya Nurlanovna",
		Email:     "aliya@biletflow.test",
		Token:     "abc123",
		Link:      "http://localhost:3000/verify-email?token=abc123",
		ExpiresIn: "24 hours",
	}
}

func TestEmailVerificationCarriesTheLink(t *testing.T) {
	msg := EmailVerification(sampleAccount())

	if msg.To != "aliya@biletflow.test" {
		t.Errorf("to = %q", msg.To)
	}
	if msg.Type != TypeEmailVerification {
		t.Errorf("type = %q, want %q", msg.Type, TypeEmailVerification)
	}
	for _, want := range []string{"Hi Aliya,", "/verify-email?token=abc123", "24 hours"} {
		if !strings.Contains(msg.Body, want) {
			t.Errorf("body is missing %q:\n%s", want, msg.Body)
		}
	}
}

func TestPasswordResetCarriesTheLink(t *testing.T) {
	d := sampleAccount()
	d.Token = "xyz789"
	d.Link = "http://localhost:3000/reset-password?token=xyz789"
	d.ExpiresIn = "1 hour"
	msg := PasswordReset(d)

	if msg.Type != TypePasswordReset {
		t.Errorf("type = %q, want %q", msg.Type, TypePasswordReset)
	}
	for _, want := range []string{"/reset-password?token=xyz789", "1 hour"} {
		if !strings.Contains(msg.Body, want) {
			t.Errorf("body is missing %q:\n%s", want, msg.Body)
		}
	}
}

// TestGreetingFallsBackWithoutAName: an account created from an email alone
// still gets a greeting rather than "Hi ,".
func TestGreetingFallsBackWithoutAName(t *testing.T) {
	d := sampleAccount()
	d.FullName = ""
	if !strings.HasPrefix(EmailVerification(d).Body, "Hi there,") {
		t.Error("an empty name should fall back to \"Hi there,\"")
	}
}

// TestRenderIsReadable pins the console format the phase brief asks for: To,
// Subject and a body, framed and unmistakably labelled as simulated.
func TestRenderIsReadable(t *testing.T) {
	out := Render(Message{
		Type:    TypeEmailVerification,
		To:      "aliya@biletflow.test",
		Subject: "Confirm your email",
		Body:    "Hi Aliya,\n\nPlease confirm your email.\n",
	})

	for _, want := range []string{
		"MOCK EMAIL - simulated delivery",
		"To:      aliya@biletflow.test",
		"Subject: Confirm your email",
		"Type:    " + TypeEmailVerification,
		"  Hi Aliya,",
		"  Please confirm your email.",
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
