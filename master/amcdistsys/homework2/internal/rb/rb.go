package rb

import "amcdistsys/homework2/internal/event"

type RB struct {
	self    string
	correct map[string]bool
	from    map[string]map[int]bool
}

func New(self string, nodes []string) *RB {
	r := &RB{
		self:    self,
		correct: make(map[string]bool),
		from:    make(map[string]map[int]bool),
	}
	r.SetNodes(nodes)
	return r
}

func (r *RB) SetSelf(self string) {
	r.self = self
}

func (r *RB) SetNodes(nodes []string) {
	for k := range r.correct {
		delete(r.correct, k)
	}
	for _, n := range nodes {
		r.correct[n] = true
		if _, ok := r.from[n]; !ok {
			r.from[n] = make(map[int]bool)
		}
	}
}

func (r *RB) Broadcast(value int) event.BEBBroadcast {
	return event.BEBBroadcast{Inner: event.RBData{Origin: r.self, Value: value}}
}

func (r *RB) OnBEBDeliver(ev event.BEBDeliver) []event.Event {
	data, ok := ev.Inner.(event.RBData)
	if !ok {
		return nil
	}
	origin := data.Origin
	m := data.Value

	if _, ok := r.from[origin]; !ok {
		r.from[origin] = make(map[int]bool)
	}
	if r.from[origin][m] {
		return nil
	}

	r.from[origin][m] = true
	out := []event.Event{event.RBDeliver{From: origin, Value: m}}

	if !r.correct[origin] {
		out = append(out, event.BEBBroadcast{Inner: event.RBData{Origin: origin, Value: m}})
	}
	return out
}

func (r *RB) OnCrash(ev event.Crash) []event.Event {
	if !r.correct[ev.Who] {
		return nil
	}
	delete(r.correct, ev.Who)
	var out []event.Event
	for m := range r.from[ev.Who] {
		out = append(out, event.BEBBroadcast{Inner: event.RBData{Origin: ev.Who, Value: m}})
	}
	return out
}
