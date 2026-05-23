package app

import (
	"encoding/json"
	"sort"

	"amcdistsys/homework2/internal/event"
)

type Sender interface {
	Send(dest string, body any) error
}

type Logger interface {
	Printf(format string, args ...any)
}

type App struct {
	self      string
	delivered map[int]bool
	pl        Sender
	log       Logger
}

func New(pl Sender, log Logger) *App {
	return &App{
		delivered: make(map[int]bool),
		pl:        pl,
		log:       log,
	}
}

func (a *App) SetSelf(self string) { a.self = self }

type IncomingBroadcast struct {
	Type    string `json:"type"`
	MsgID   int    `json:"msg_id"`
	Message int    `json:"message"`
}

type IncomingRead struct {
	Type  string `json:"type"`
	MsgID int    `json:"msg_id"`
}

type IncomingTopology struct {
	Type     string                 `json:"type"`
	MsgID    int                    `json:"msg_id"`
	Topology map[string]interface{} `json:"topology"`
}

type IncomingInit struct {
	Type    string   `json:"type"`
	MsgID   int      `json:"msg_id"`
	NodeID  string   `json:"node_id"`
	NodeIDs []string `json:"node_ids"`
}

func ParseBroadcast(from string, body json.RawMessage) (event.AppBroadcast, error) {
	var b IncomingBroadcast
	if err := json.Unmarshal(body, &b); err != nil {
		return event.AppBroadcast{}, err
	}
	return event.AppBroadcast{
		Msg:   event.IncomingMsg{Src: from, MsgID: b.MsgID},
		Value: b.Message,
	}, nil
}

func ParseRead(from string, body json.RawMessage) (event.AppRead, error) {
	var r IncomingRead
	if err := json.Unmarshal(body, &r); err != nil {
		return event.AppRead{}, err
	}
	return event.AppRead{Msg: event.IncomingMsg{Src: from, MsgID: r.MsgID}}, nil
}

func ParseTopology(from string, body json.RawMessage) (event.AppTopology, error) {
	var t IncomingTopology
	if err := json.Unmarshal(body, &t); err != nil {
		return event.AppTopology{}, err
	}
	return event.AppTopology{Msg: event.IncomingMsg{Src: from, MsgID: t.MsgID}}, nil
}

func ParseInit(from string, body json.RawMessage) (event.AppInit, error) {
	var i IncomingInit
	if err := json.Unmarshal(body, &i); err != nil {
		return event.AppInit{}, err
	}
	return event.AppInit{
		Msg:     event.IncomingMsg{Src: from, MsgID: i.MsgID},
		NodeID:  i.NodeID,
		NodeIDs: i.NodeIDs,
	}, nil
}

func (a *App) OnBroadcast(ev event.AppBroadcast) event.RBBroadcast {
	a.replyOk(ev.Msg, "broadcast_ok", nil)
	return event.RBBroadcast{Value: ev.Value}
}

func (a *App) OnRead(ev event.AppRead) {
	vals := make([]int, 0, len(a.delivered))
	for v := range a.delivered {
		vals = append(vals, v)
	}
	sort.Ints(vals)
	a.replyOk(ev.Msg, "read_ok", map[string]any{"messages": vals})
}

func (a *App) OnTopology(ev event.AppTopology) {
	a.replyOk(ev.Msg, "topology_ok", nil)
}

func (a *App) OnInit(ev event.AppInit) {
	a.replyOk(ev.Msg, "init_ok", nil)
}

func (a *App) OnRBDeliver(ev event.RBDeliver) {
	a.delivered[ev.Value] = true
}

func (a *App) replyOk(in event.IncomingMsg, replyType string, extra map[string]any) {
	body := map[string]any{
		"type":        replyType,
		"in_reply_to": in.MsgID,
	}
	for k, v := range extra {
		body[k] = v
	}
	if err := a.pl.Send(in.Src, body); err != nil {
		a.log.Printf("app: reply %s to %s failed: %v", replyType, in.Src, err)
	}
}
