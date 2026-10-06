// Package email is the notification transport for the academic MVP.
//
// SRS 4.10 requires the system to notify attendees on purchase, ticket
// delivery and refund completion. Wiring a real provider (SES, Postmark,
// Mailgun) is a deployment concern, not a product one, so this package sends
// to the console instead and makes that fact impossible to miss in the output.
//
// The seam is the Sender interface: swapping ConsoleSender for an SMTP or API
// client later changes this package and nothing else.
package email

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Message is one outbound notification.
type Message struct {
	// Type is the notification kind, mirrored into notifications.type so a
	// stored row can be traced back to the code that produced it.
	Type    string
	To      string
	Subject string
	Body    string

	// Ref is an opaque caller reference echoed back to the Mailer's
	// completion callback. The API puts the outbox row's id here so a
	// finished send can be matched to the row it belongs to, without this
	// package knowing anything about notifications.
	Ref string
}

// Sender delivers a message. Implementations must be safe for concurrent use.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// rule is the width of the console frame: wide enough for a URL, narrow
// enough to survive a split terminal.
const ruleWidth = 74

// ConsoleSender writes messages to an io.Writer, one framed block each.
type ConsoleSender struct {
	mu  sync.Mutex
	out io.Writer
}

// NewConsoleSender writes to stdout, as the phase brief requires. A nil writer
// means os.Stdout.
func NewConsoleSender(out io.Writer) *ConsoleSender {
	if out == nil {
		out = os.Stdout
	}
	return &ConsoleSender{out: out}
}

// Send prints the message.
//
// The whole block is assembled first and written under a mutex, so two
// concurrent sends cannot interleave their lines into an unreadable mess.
func (c *ConsoleSender) Send(_ context.Context, msg Message) error {
	block := Render(msg)

	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := io.WriteString(c.out, block)
	return err
}

// Render formats a message as the console block. It is exported so tests can
// assert on the exact output a marker would read.
func Render(msg Message) string {
	heavy := strings.Repeat("=", ruleWidth)
	light := strings.Repeat("-", ruleWidth)

	var b strings.Builder
	b.WriteString("\n" + heavy + "\n")
	b.WriteString("  MOCK EMAIL - simulated delivery, nothing left this machine\n")
	b.WriteString(light + "\n")
	fmt.Fprintf(&b, "  To:      %s\n", msg.To)
	fmt.Fprintf(&b, "  Subject: %s\n", msg.Subject)
	if msg.Type != "" {
		fmt.Fprintf(&b, "  Type:    %s\n", msg.Type)
	}
	b.WriteString(light + "\n")

	for _, line := range strings.Split(strings.TrimRight(msg.Body, "\n"), "\n") {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString("  " + line + "\n")
	}

	b.WriteString(heavy + "\n\n")
	return b.String()
}

// Mailer dispatches messages without making the caller wait.
//
// SRS 4.10 asks for notification on purchase; nothing says the buyer should
// watch a mail server while their tickets are held up. Delivery therefore
// happens after the response is written, and a failure to notify never fails
// a checkout that already took the attendee's money.
type Mailer struct {
	sender  Sender
	timeout time.Duration
	onDone  func(msg Message, err error)

	wg sync.WaitGroup
}

// NewMailer wraps a Sender. onDone, when set, is called on the sending
// goroutine once each message finishes - the API uses it to mark the stored
// notification row as sent.
func NewMailer(sender Sender, onDone func(Message, error)) *Mailer {
	return &Mailer{sender: sender, timeout: 10 * time.Second, onDone: onDone}
}

// SendAsync queues a message and returns immediately.
//
// The context is deliberately not inherited from the request: the request is
// finished by the time this runs, and a cancelled context would abort the very
// delivery it was supposed to carry.
func (m *Mailer) SendAsync(msg Message) {
	if m == nil || m.sender == nil {
		return
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()

		err := m.sender.Send(ctx, msg)
		if m.onDone != nil {
			m.onDone(msg, err)
		}
	}()
}

// Wait blocks until every queued message has been dispatched.
//
// Tests use it to make an asynchronous send observable, and shutdown uses it
// so a confirmation is not lost to a process exiting a millisecond too early.
func (m *Mailer) Wait() {
	if m == nil {
		return
	}
	m.wg.Wait()
}

// Recorder is a Sender that keeps messages in memory instead of printing them.
type Recorder struct {
	mu       sync.Mutex
	messages []Message
}

// NewRecorder builds an empty Recorder.
func NewRecorder() *Recorder { return &Recorder{} }

// Send stores the message.
func (r *Recorder) Send(_ context.Context, msg Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, msg)
	return nil
}

// Messages returns a copy of everything sent so far.
func (r *Recorder) Messages() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Message(nil), r.messages...)
}

// To returns every message addressed to one recipient.
func (r *Recorder) To(address string) []Message {
	var out []Message
	for _, msg := range r.Messages() {
		if strings.EqualFold(msg.To, address) {
			out = append(out, msg)
		}
	}
	return out
}

// Reset drops everything recorded.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = nil
}
