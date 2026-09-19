package email

import (
	"context"
	"io"
	"os"
	"sync"
)

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
