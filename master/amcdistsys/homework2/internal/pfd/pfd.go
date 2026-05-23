package pfd

import "amcdistsys/homework2/internal/event"

type Sender interface {
	Send(dest string, body any) error
}

type Logger interface {
	Printf(format string, args ...any)
}

type PFD struct {
	self      string
	peers     []string
	alive     map[string]bool
	suspected map[string]bool
	pl        Sender
	log       Logger
	nextMsgID int
}

func New(self string, nodes []string, sender Sender, log Logger) *PFD {
	p := &PFD{
		self:      self,
		alive:     make(map[string]bool),
		suspected: make(map[string]bool),
		pl:        sender,
		log:       log,
	}
	p.SetNodes(nodes)
	return p
}

func (p *PFD) SetSelf(self string) {
	p.self = self
}

func (p *PFD) SetNodes(nodes []string) {
	p.peers = p.peers[:0]
	for _, n := range nodes {
		if n != p.self {
			p.peers = append(p.peers, n)
			p.alive[n] = true
		}
	}
}

func (p *PFD) OnTimeout() []event.Event {
	var out []event.Event
	for _, q := range p.peers {
		if !p.alive[q] && !p.suspected[q] {
			p.suspected[q] = true
			out = append(out, event.Crash{Who: q})
		}
		p.nextMsgID++
		body := map[string]any{
			"type":   "heartbeat",
			"msg_id": p.nextMsgID,
		}
		if err := p.pl.Send(q, body); err != nil {
			p.log.Printf("pfd: heartbeat to %s failed: %v", q, err)
		}
	}
	for k := range p.alive {
		delete(p.alive, k)
	}
	return out
}

func (p *PFD) OnHeartbeatReq(ev event.HeartbeatReq) {
	body := map[string]any{
		"type":        "heartbeat_ok",
		"in_reply_to": ev.MsgID,
	}
	if err := p.pl.Send(ev.From, body); err != nil {
		p.log.Printf("pfd: heartbeat_ok to %s failed: %v", ev.From, err)
	}
}

func (p *PFD) OnHeartbeatOk(ev event.HeartbeatOk) {
	p.alive[ev.From] = true
}

func (p *PFD) IsSuspected(node string) bool {
	return p.suspected[node]
}
