package event

import "encoding/json"

type Event interface{ eventTag() }

type PLDeliver struct {
	From string
	Body json.RawMessage
}

type BEBBroadcast struct {
	Inner Event
}

type BEBDeliver struct {
	From  string
	Inner Event
}

type RBBroadcast struct {
	Value int
}

type RBDeliver struct {
	From  string
	Value int
}

type Crash struct {
	Who string
}

type Timeout struct{}

type IncomingMsg struct {
	Src   string
	MsgID int
}

type AppBroadcast struct {
	Msg   IncomingMsg
	Value int
}

type AppRead struct {
	Msg IncomingMsg
}

type AppTopology struct {
	Msg IncomingMsg
}

type AppInit struct {
	Msg     IncomingMsg
	NodeID  string
	NodeIDs []string
}

type HeartbeatReq struct {
	From  string
	MsgID int
}

type HeartbeatOk struct {
	From string
}

type RBData struct {
	Origin string
	Value  int
}

func (PLDeliver) eventTag()    {}
func (BEBBroadcast) eventTag() {}
func (BEBDeliver) eventTag()   {}
func (RBBroadcast) eventTag()  {}
func (RBDeliver) eventTag()    {}
func (Crash) eventTag()        {}
func (Timeout) eventTag()      {}
func (AppBroadcast) eventTag() {}
func (AppRead) eventTag()      {}
func (AppTopology) eventTag()  {}
func (AppInit) eventTag()      {}
func (HeartbeatReq) eventTag() {}
func (HeartbeatOk) eventTag()  {}
func (RBData) eventTag()       {}
