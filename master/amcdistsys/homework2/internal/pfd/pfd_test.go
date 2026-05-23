package pfd

import (
	"testing"

	"amcdistsys/homework2/internal/event"
)

type fakeSender struct {
	sent []send
}

type send struct {
	dest string
	body map[string]any
}

func (f *fakeSender) Send(dest string, body any) error {
	m, _ := body.(map[string]any)
	f.sent = append(f.sent, send{dest: dest, body: m})
	return nil
}

type discardLog struct{}

func (discardLog) Printf(format string, args ...any) {}

func TestMissedHeartbeatProducesCrash(t *testing.T) {
	p := New("n1", []string{"n1", "n2", "n3"}, &fakeSender{}, discardLog{})

	_ = p.OnTimeout()
	out := p.OnTimeout()

	gotCrashed := map[string]bool{}
	for _, e := range out {
		c, ok := e.(event.Crash)
		if !ok {
			t.Fatalf("unexpected event %T", e)
		}
		gotCrashed[c.Who] = true
	}
	for _, want := range []string{"n2", "n3"} {
		if !gotCrashed[want] {
			t.Errorf("expected Crash(%s) on second timeout", want)
		}
	}
}

func TestHeartbeatOkPreventsCrash(t *testing.T) {
	p := New("n1", []string{"n1", "n2"}, &fakeSender{}, discardLog{})

	_ = p.OnTimeout()
	p.OnHeartbeatOk(event.HeartbeatOk{From: "n2"})
	out := p.OnTimeout()

	for _, e := range out {
		if _, ok := e.(event.Crash); ok {
			t.Fatalf("unexpected Crash event: %+v", e)
		}
	}
}

func TestCrashEmittedOnce(t *testing.T) {
	p := New("n1", []string{"n1", "n2"}, &fakeSender{}, discardLog{})

	_ = p.OnTimeout()
	first := p.OnTimeout()
	if len(first) != 1 {
		t.Fatalf("expected 1 crash, got %d", len(first))
	}
	second := p.OnTimeout()
	if len(second) != 0 {
		t.Fatalf("expected no further crash, got %d", len(second))
	}
}

func TestOnHeartbeatReqSendsOk(t *testing.T) {
	fs := &fakeSender{}
	p := New("n1", []string{"n1", "n2"}, fs, discardLog{})

	p.OnHeartbeatReq(event.HeartbeatReq{From: "n2", MsgID: 42})

	if len(fs.sent) != 1 {
		t.Fatalf("expected one reply, got %d", len(fs.sent))
	}
	if fs.sent[0].dest != "n2" {
		t.Errorf("dest = %s want n2", fs.sent[0].dest)
	}
	if fs.sent[0].body["type"] != "heartbeat_ok" {
		t.Errorf("type = %v want heartbeat_ok", fs.sent[0].body["type"])
	}
	if fs.sent[0].body["in_reply_to"] != 42 {
		t.Errorf("in_reply_to = %v want 42", fs.sent[0].body["in_reply_to"])
	}
}
