package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"time"

	"amcdistsys/homework2/internal/app"
	"amcdistsys/homework2/internal/beb"
	"amcdistsys/homework2/internal/event"
	"amcdistsys/homework2/internal/pl"
	"amcdistsys/homework2/internal/pfd"
	"amcdistsys/homework2/internal/rb"
)

type stderrLogger struct {
	*log.Logger
	verbose bool
}

func (l stderrLogger) Printf(format string, args ...any) {
	l.Logger.Printf(format, args...)
}

func (l stderrLogger) Eventf(format string, args ...any) {
	if !l.verbose {
		return
	}
	l.Logger.Printf("[event] "+format, args...)
}

func main() {
	timeoutMs := flag.Int("pfd-timeout-ms", 1500, "PFD heartbeat timeout in ms")
	logEvents := flag.Bool("log-events", false, "log every algorithm event to stderr (per-node log)")
	flag.Parse()

	verbose := *logEvents || os.Getenv("MAELSTROM_LOG_EVENTS") == "1"
	logger := stderrLogger{
		Logger:  log.New(os.Stderr, "[node] ", log.LstdFlags|log.Lmicroseconds),
		verbose: verbose,
	}

	q := event.NewQueue(1024)
	plLayer := pl.New("", os.Stdout, q)
	bebLayer := beb.New(nil, plLayer, logger)
	pfdLayer := pfd.New("", nil, plLayer, logger)
	rbLayer := rb.New("", nil)
	appLayer := app.New(plLayer, logger)

	initDone := make(chan struct{})
	initOnce := false

	go stdinReader(q, logger)
	go pfdTicker(q, initDone, time.Duration(*timeoutMs)*time.Millisecond)

	for {
		e := q.Dequeue()
		dispatch(e, q, plLayer, bebLayer, pfdLayer, rbLayer, appLayer, logger, func() {
			if !initOnce {
				initOnce = true
				close(initDone)
			}
		})
	}
}

func stdinReader(q *event.Queue, log stderrLogger) {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := append([]byte(nil), sc.Bytes()...)
		ev, _, err := pl.ParseLine(line)
		if err != nil {
			log.Printf("stdin parse error: %v", err)
			continue
		}
		q.Enqueue(ev)
	}
	if err := sc.Err(); err != nil {
		log.Printf("stdin scanner error: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	os.Exit(0)
}

func pfdTicker(q *event.Queue, initDone <-chan struct{}, period time.Duration) {
	<-initDone
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for range ticker.C {
		q.Enqueue(event.Timeout{})
	}
}

func dispatch(
	e event.Event,
	q *event.Queue,
	plLayer *pl.PL,
	bebLayer *beb.BEB,
	pfdLayer *pfd.PFD,
	rbLayer *rb.RB,
	appLayer *app.App,
	logger stderrLogger,
	signalInitDone func(),
) {
	switch ev := e.(type) {
	case event.PLDeliver:
		head, err := pl.PeekType(ev.Body)
		if err != nil {
			logger.Printf("peek type: %v", err)
			return
		}
		switch head.Type {
		case "init":
			parsed, err := app.ParseInit(ev.From, ev.Body)
			if err != nil {
				logger.Printf("parse init: %v", err)
				return
			}
			plLayer.SetSelf(parsed.NodeID)
			bebLayer.SetNodes(parsed.NodeIDs)
			pfdLayer.SetSelf(parsed.NodeID)
			pfdLayer.SetNodes(parsed.NodeIDs)
			rbLayer.SetSelf(parsed.NodeID)
			rbLayer.SetNodes(parsed.NodeIDs)
			appLayer.SetSelf(parsed.NodeID)
			appLayer.OnInit(parsed)
			signalInitDone()
			logger.Eventf("init: self=%s peers=%v", parsed.NodeID, parsed.NodeIDs)
		case "topology":
			parsed, err := app.ParseTopology(ev.From, ev.Body)
			if err != nil {
				logger.Printf("parse topology: %v", err)
				return
			}
			appLayer.OnTopology(parsed)
		case "broadcast":
			parsed, err := app.ParseBroadcast(ev.From, ev.Body)
			if err != nil {
				logger.Printf("parse broadcast: %v", err)
				return
			}
			logger.Eventf("app: client %s requests broadcast %d", parsed.Msg.Src, parsed.Value)
			q.Enqueue(appLayer.OnBroadcast(parsed))
		case "read":
			parsed, err := app.ParseRead(ev.From, ev.Body)
			if err != nil {
				logger.Printf("parse read: %v", err)
				return
			}
			appLayer.OnRead(parsed)
		case "rb_data":
			next, ok := bebLayer.OnPLDeliver(ev)
			if ok {
				q.Enqueue(next)
			}
		case "heartbeat":
			q.Enqueue(event.HeartbeatReq{From: ev.From, MsgID: head.MsgID})
		case "heartbeat_ok":
			q.Enqueue(event.HeartbeatOk{From: ev.From})
		default:
			logger.Printf("ignored unknown body type: %s", head.Type)
		}

	case event.RBBroadcast:
		logger.Eventf("rb: broadcast value=%d", ev.Value)
		q.Enqueue(rbLayer.Broadcast(ev.Value))

	case event.BEBBroadcast:
		data, ok := ev.Inner.(event.RBData)
		if !ok {
			logger.Printf("BEBBroadcast inner not RBData: %T", ev.Inner)
			return
		}
		logger.Eventf("beb: broadcast rb_data origin=%s value=%d", data.Origin, data.Value)
		bebLayer.Broadcast(data)

	case event.BEBDeliver:
		out := rbLayer.OnBEBDeliver(ev)
		if data, ok := ev.Inner.(event.RBData); ok {
			relayed := false
			for _, n := range out {
				if _, isBeb := n.(event.BEBBroadcast); isBeb {
					relayed = true
					break
				}
			}
			switch {
			case len(out) == 0:
				logger.Eventf("rb: dedup drop rb_data from=%s origin=%s value=%d", ev.From, data.Origin, data.Value)
			case relayed:
				logger.Eventf("rb: deliver+lazy-relay origin=%s value=%d (sender suspected)", data.Origin, data.Value)
			default:
				logger.Eventf("rb: deliver origin=%s value=%d", data.Origin, data.Value)
			}
		}
		for _, next := range out {
			q.Enqueue(next)
		}

	case event.RBDeliver:
		logger.Eventf("app: rb-deliver from=%s value=%d", ev.From, ev.Value)
		appLayer.OnRBDeliver(ev)

	case event.Crash:
		out := rbLayer.OnCrash(ev)
		logger.Eventf("rb: crash(%s) -> %d relay(s) from from[%s]", ev.Who, len(out), ev.Who)
		for _, next := range out {
			q.Enqueue(next)
		}

	case event.Timeout:
		out := pfdLayer.OnTimeout()
		for _, next := range out {
			if c, ok := next.(event.Crash); ok {
				logger.Eventf("pfd: suspect %s -> Crash", c.Who)
			}
			q.Enqueue(next)
		}

	case event.HeartbeatReq:
		pfdLayer.OnHeartbeatReq(ev)

	case event.HeartbeatOk:
		pfdLayer.OnHeartbeatOk(ev)

	default:
		logger.Printf("unknown event type %T", ev)
	}
}
