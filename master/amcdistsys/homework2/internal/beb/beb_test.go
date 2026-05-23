package beb

import (
	"encoding/json"
	"testing"

	"amcdistsys/homework2/internal/event"
)

type fakeSender struct {
	sent []sentMsg
}

type sentMsg struct {
	dest string
	body map[string]any
}

func (f *fakeSender) Send(dest string, body any) error {
	m, _ := body.(map[string]any)
	f.sent = append(f.sent, sentMsg{dest: dest, body: m})
	return nil
}

type discardLog struct{}

func (discardLog) Printf(format string, args ...any) {}

func TestBroadcastSendsToAllIncludingSelf(t *testing.T) {
	fs := &fakeSender{}
	b := New([]string{"n1", "n2", "n3"}, fs, discardLog{})

	b.Broadcast(event.RBData{Origin: "n1", Value: 7})

	if len(fs.sent) != 3 {
		t.Fatalf("expected 3 sends (forall q in Pi, including self), got %d", len(fs.sent))
	}
	dests := map[string]bool{}
	for _, s := range fs.sent {
		dests[s.dest] = true
		if s.body["type"] != "rb_data" || s.body["origin"] != "n1" || s.body["value"] != 7 {
			t.Errorf("unexpected body: %+v", s.body)
		}
	}
	for _, want := range []string{"n1", "n2", "n3"} {
		if !dests[want] {
			t.Errorf("missing destination %s", want)
		}
	}
}

func TestOnPLDeliverRoutesRBData(t *testing.T) {
	b := New([]string{"n1", "n2"}, &fakeSender{}, discardLog{})

	body, _ := json.Marshal(map[string]any{"type": "rb_data", "origin": "n2", "value": 42})
	ev, ok := b.OnPLDeliver(event.PLDeliver{From: "n2", Body: body})
	if !ok {
		t.Fatal("expected route")
	}
	bd, ok := ev.(event.BEBDeliver)
	if !ok {
		t.Fatalf("expected BEBDeliver, got %T", ev)
	}
	if bd.From != "n2" {
		t.Errorf("From = %q want n2", bd.From)
	}
	inner, ok := bd.Inner.(event.RBData)
	if !ok {
		t.Fatalf("Inner = %T want RBData", bd.Inner)
	}
	if inner.Origin != "n2" || inner.Value != 42 {
		t.Errorf("Inner = %+v", inner)
	}
}

func TestOnPLDeliverIgnoresOtherTypes(t *testing.T) {
	b := New([]string{"n1"}, &fakeSender{}, discardLog{})
	body, _ := json.Marshal(map[string]any{"type": "heartbeat"})
	if _, ok := b.OnPLDeliver(event.PLDeliver{From: "n1", Body: body}); ok {
		t.Fatal("expected ignore")
	}
}
