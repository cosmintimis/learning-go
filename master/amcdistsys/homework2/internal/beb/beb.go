package beb

import (
	"encoding/json"
	"fmt"

	"amcdistsys/homework2/internal/event"
	"amcdistsys/homework2/internal/pl"
)

type Sender interface {
	Send(dest string, body any) error
}

type Logger interface {
	Printf(format string, args ...any)
}

type BEB struct {
	nodes []string
	pl    Sender
	log   Logger
}

func New(nodes []string, sender Sender, log Logger) *BEB {
	return &BEB{nodes: nodes, pl: sender, log: log}
}

func (b *BEB) SetNodes(nodes []string) {
	b.nodes = nodes
}

func (b *BEB) Broadcast(rbData event.RBData) {
	body := map[string]any{
		"type":   "rb_data",
		"origin": rbData.Origin,
		"value":  rbData.Value,
	}
	for _, q := range b.nodes {
		if err := b.pl.Send(q, body); err != nil {
			b.log.Printf("beb: send to %s failed: %v", q, err)
		}
	}
}

func (b *BEB) OnPLDeliver(ev event.PLDeliver) (event.Event, bool) {
	head, err := pl.PeekType(ev.Body)
	if err != nil {
		b.log.Printf("beb: peek type: %v", err)
		return nil, false
	}
	if head.Type != "rb_data" {
		return nil, false
	}
	data, err := ParseDataBody(ev.Body)
	if err != nil {
		b.log.Printf("beb: parse rb_data: %v", err)
		return nil, false
	}
	return event.BEBDeliver{From: ev.From, Inner: data}, true
}

func ParseDataBody(body json.RawMessage) (event.RBData, error) {
	var raw struct {
		Type   string `json:"type"`
		Origin string `json:"origin"`
		Value  int    `json:"value"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return event.RBData{}, fmt.Errorf("beb parse rb_data: %w", err)
	}
	return event.RBData{Origin: raw.Origin, Value: raw.Value}, nil
}
