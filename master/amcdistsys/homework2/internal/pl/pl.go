package pl

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"amcdistsys/homework2/internal/event"
)

type envelope struct {
	Src  string          `json:"src"`
	Dest string          `json:"dest"`
	Body json.RawMessage `json:"body"`
}

type PL struct {
	self string
	out  io.Writer
	q    *event.Queue

	mu sync.Mutex
}

func New(self string, out io.Writer, q *event.Queue) *PL {
	return &PL{self: self, out: out, q: q}
}

func (p *PL) SetSelf(self string) {
	p.self = self
}

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

	p.mu.Lock()
	defer p.mu.Unlock()
	if _, err := p.out.Write(line); err != nil {
		return err
	}
	if _, err := p.out.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func ParseLine(line []byte) (event.PLDeliver, string, error) {
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return event.PLDeliver{}, "", fmt.Errorf("pl parse envelope: %w", err)
	}
	return event.PLDeliver{From: env.Src, Body: env.Body}, env.Dest, nil
}

type BodyHead struct {
	Type      string `json:"type"`
	MsgID     int    `json:"msg_id,omitempty"`
	InReplyTo int    `json:"in_reply_to,omitempty"`
}

func PeekType(body json.RawMessage) (BodyHead, error) {
	var h BodyHead
	if err := json.Unmarshal(body, &h); err != nil {
		return h, fmt.Errorf("pl peek type: %w", err)
	}
	return h, nil
}
