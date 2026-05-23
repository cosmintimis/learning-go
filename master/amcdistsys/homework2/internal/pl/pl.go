// Package pl is the Perfect Links layer, simplified per req.txt to plain
// STDIN/STDOUT JSON I/O. Maelstrom is the transport, so no retries/acks.
package pl

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"amcdistsys/homework2/internal/event"
)

// Maelstrom's outer wire shape. Body stays raw so the upper layer can
// dispatch on body["type"].
type envelope struct {
	Src  string          `json:"src"`
	Dest string          `json:"dest"`
	Body json.RawMessage `json:"body"`
}

type PL struct {
	self string
	out  io.Writer
	q    *event.Queue

	// Parked: single-writer invariant (event-processor goroutine only)
	// makes this lock unnecessary. Kept for documentation.
	mu sync.Mutex
}

func New(self string, out io.Writer, q *event.Queue) *PL {
	return &PL{self: self, out: out, q: q}
}

func (p *PL) SetSelf(self string) { p.self = self }

// Send: local-loopback when dest == self (enqueues PLDeliver directly so
// BEB can iterate "forall q in Pi" including self); otherwise a newline-
// terminated JSON envelope to STDOUT.
func (p *PL) Send(dest string, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("pl marshal body: %w", err)
	}

	if dest == p.self {
		p.q.Enqueue(event.PLDeliver{From: p.self, Body: raw})
		return nil
	}

	env := envelope{Src: p.self, Dest: dest, Body: raw}
	line, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("pl marshal envelope: %w", err)
	}

	// Single writer (event processor) -> no race; lock not needed.
	// p.mu.Lock()
	// defer p.mu.Unlock()
	if _, err := p.out.Write(line); err != nil {
		return err
	}
	if _, err := p.out.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

// ParseLine decodes one envelope from STDIN; body stays raw for the
// dispatch loop to peek at via PeekType.
func ParseLine(line []byte) (event.PLDeliver, string, error) {
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return event.PLDeliver{}, "", fmt.Errorf("pl parse envelope: %w", err)
	}
	return event.PLDeliver{From: env.Src, Body: env.Body}, env.Dest, nil
}

// Maelstrom-reserved body fields (doc/protocol.md).
type BodyHead struct {
	Type      string `json:"type"`
	MsgID     int    `json:"msg_id,omitempty"`
	InReplyTo int    `json:"in_reply_to,omitempty"`
}

// PeekType decodes only the reserved fields so dispatch can route by type.
func PeekType(body json.RawMessage) (BodyHead, error) {
	var h BodyHead
	if err := json.Unmarshal(body, &h); err != nil {
		return h, fmt.Errorf("pl peek type: %w", err)
	}
	return h, nil
}
