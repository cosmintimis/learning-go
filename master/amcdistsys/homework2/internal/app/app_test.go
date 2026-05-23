package app

import (
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

func TestOnBroadcastReplyAndTriggersRB(t *testing.T) {
	fs := &fakeSender{}
	a := New(fs, discardLog{})
	a.SetSelf("n1")

	out := a.OnBroadcast(event.AppBroadcast{
		Msg:   event.IncomingMsg{Src: "c1", MsgID: 42},
		Value: 7,
	})

	if out.Value != 7 {
		t.Errorf("rb broadcast value = %d want 7", out.Value)
	}
	if len(fs.sent) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(fs.sent))
	}
	if fs.sent[0].dest != "c1" {
		t.Errorf("dest = %s want c1", fs.sent[0].dest)
	}
	if fs.sent[0].body["type"] != "broadcast_ok" {
		t.Errorf("type = %v want broadcast_ok", fs.sent[0].body["type"])
	}
	if fs.sent[0].body["in_reply_to"] != 42 {
		t.Errorf("in_reply_to = %v want 42", fs.sent[0].body["in_reply_to"])
	}
}

func TestOnReadReturnsSortedDelivered(t *testing.T) {
	fs := &fakeSender{}
	a := New(fs, discardLog{})
	a.SetSelf("n1")

	a.OnRBDeliver(event.RBDeliver{From: "n2", Value: 9})
	a.OnRBDeliver(event.RBDeliver{From: "n3", Value: 3})
	a.OnRBDeliver(event.RBDeliver{From: "n2", Value: 7})

	a.OnRead(event.AppRead{Msg: event.IncomingMsg{Src: "c1", MsgID: 5}})

	if len(fs.sent) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(fs.sent))
	}
	body := fs.sent[0].body
	if body["type"] != "read_ok" {
		t.Errorf("type = %v want read_ok", body["type"])
	}
	if body["in_reply_to"] != 5 {
		t.Errorf("in_reply_to = %v want 5", body["in_reply_to"])
	}
	msgs, ok := body["messages"].([]int)
	if !ok {
		t.Fatalf("messages type %T", body["messages"])
	}
	want := []int{3, 7, 9}
	if len(msgs) != len(want) {
		t.Fatalf("messages = %v want %v", msgs, want)
	}
	for i, v := range want {
		if msgs[i] != v {
			t.Errorf("messages[%d] = %d want %d", i, msgs[i], v)
		}
	}
}
