package rb

import (
	"testing"

	"amcdistsys/homework2/internal/event"
)

func bebDeliver(from, origin string, value int) event.BEBDeliver {
	return event.BEBDeliver{From: from, Inner: event.RBData{Origin: origin, Value: value}}
}

func TestBroadcastEmitsBEBWithDataAndSelfOrigin(t *testing.T) {
	r := New("n1", []string{"n1", "n2", "n3"})
	out := r.Broadcast(7)
	data, ok := out.Inner.(event.RBData)
	if !ok {
		t.Fatalf("inner type %T", out.Inner)
	}
	if data.Origin != "n1" || data.Value != 7 {
		t.Errorf("got %+v", data)
	}
}

func TestDuplicateDeliveryIsSuppressed(t *testing.T) {
	r := New("n1", []string{"n1", "n2", "n3"})

	first := r.OnBEBDeliver(bebDeliver("n2", "n2", 7))
	if len(first) != 1 {
		t.Fatalf("expected 1 event (RBDeliver), got %d: %+v", len(first), first)
	}
	if _, ok := first[0].(event.RBDeliver); !ok {
		t.Fatalf("first event %T want RBDeliver", first[0])
	}

	second := r.OnBEBDeliver(bebDeliver("n3", "n2", 7))
	if len(second) != 0 {
		t.Fatalf("expected no events on duplicate, got %+v", second)
	}
}

func TestNoRelayWhenOriginAlive(t *testing.T) {
	r := New("n1", []string{"n1", "n2", "n3"})
	out := r.OnBEBDeliver(bebDeliver("n2", "n2", 7))
	for _, e := range out {
		if _, ok := e.(event.BEBBroadcast); ok {
			t.Fatalf("unexpected relay while origin alive: %+v", e)
		}
	}
}

func TestLazyRelayOnLateDeliveryAfterCrash(t *testing.T) {
	r := New("n1", []string{"n1", "n2", "n3"})

	if out := r.OnCrash(event.Crash{Who: "n2"}); len(out) != 0 {
		t.Fatalf("crash with no prior msgs should yield no relays, got %+v", out)
	}

	out := r.OnBEBDeliver(bebDeliver("n3", "n2", 7))
	var sawDeliver, sawRelay bool
	for _, e := range out {
		switch v := e.(type) {
		case event.RBDeliver:
			if v.From != "n2" || v.Value != 7 {
				t.Errorf("RBDeliver = %+v", v)
			}
			sawDeliver = true
		case event.BEBBroadcast:
			data := v.Inner.(event.RBData)
			if data.Origin != "n2" || data.Value != 7 {
				t.Errorf("relay preserves origin: got %+v", data)
			}
			sawRelay = true
		}
	}
	if !sawDeliver || !sawRelay {
		t.Fatalf("expected both RBDeliver and relay, got deliver=%v relay=%v", sawDeliver, sawRelay)
	}
}

func TestCrashTriggersRelayOfPriorMessages(t *testing.T) {
	r := New("n1", []string{"n1", "n2", "n3"})

	r.OnBEBDeliver(bebDeliver("n2", "n2", 7))
	r.OnBEBDeliver(bebDeliver("n2", "n2", 9))

	out := r.OnCrash(event.Crash{Who: "n2"})
	relayed := map[int]bool{}
	for _, e := range out {
		bb, ok := e.(event.BEBBroadcast)
		if !ok {
			t.Fatalf("expected BEBBroadcast, got %T", e)
		}
		data := bb.Inner.(event.RBData)
		if data.Origin != "n2" {
			t.Errorf("origin = %s want n2", data.Origin)
		}
		relayed[data.Value] = true
	}
	if !relayed[7] || !relayed[9] {
		t.Errorf("expected both 7 and 9 relayed, got %+v", relayed)
	}
}

func TestCrashTwiceIsIdempotent(t *testing.T) {
	r := New("n1", []string{"n1", "n2"})
	r.OnBEBDeliver(bebDeliver("n2", "n2", 1))
	r.OnCrash(event.Crash{Who: "n2"})
	out := r.OnCrash(event.Crash{Who: "n2"})
	if len(out) != 0 {
		t.Fatalf("second crash should be no-op, got %+v", out)
	}
}
