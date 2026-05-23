# Homework 2 — Reliable Broadcast on Jepsen Maelstrom

Course: *Algorithms, Models, and Concepts in Distributed Systems* — Master, Year 1, Sem 2.

This homework implements **Algorithm 3.2 — "Lazy Reliable Broadcast"** from
Cachin, Guerraoui & Rodrigues, *Introduction to Reliable and Secure Distributed
Programming* (Springer, 2011), page 78, §3.3.2 — running as a node binary inside the
[Jepsen Maelstrom](https://github.com/jepsen-io/maelstrom) workbench.

- **Fault model:** crash-stop (fail-stop).
- **Abstraction stack:** `APP → RB → { PFD , BEB } → PL → STDIN/STDOUT`.
- **Language:** Go 1.22+ (developed against 1.26).
- **Verification:** unit tests per layer + end-to-end Maelstrom `broadcast` workload.

The design document this README accompanies lives at
`docs/superpowers/specs/2026-05-23-reliable-broadcast-maelstrom-design.md` (excluded from
version control — see §11).

---

## Table of contents

1. [Quick start](#1-quick-start)
2. [What is being implemented and why](#2-what-is-being-implemented-and-why)
3. [Architecture](#3-architecture)
4. [Layered packages (book ↔ code)](#4-layered-packages-book--code)
5. [Algorithm 3.2 line-by-line](#5-algorithm-32-line-by-line)
6. [Wire protocol](#6-wire-protocol)
7. [Concurrency model and event nesting](#7-concurrency-model-and-event-nesting)
8. [Verification — tests and what they prove](#8-verification--tests-and-what-they-prove)
9. [Sample Maelstrom run output](#9-sample-maelstrom-run-output)
10. [Discussion — RB1–RB4, PFD caveat, lazy vs eager](#10-discussion)
11. [Repository layout](#11-repository-layout)
12. [References](#12-references)

---

## 1. Quick start

```sh
# Build the node binary.
make build

# Run all Go unit tests.
make test

# End-to-end with Maelstrom, no faults.
make test-clean

# End-to-end with a network-partition nemesis (PFD fires, lazy relays).
make test-partition
```

Smoke a single node from the command line (no Maelstrom needed):

```sh
echo '{"src":"c0","dest":"n1","body":{"type":"init","msg_id":1,"node_id":"n1","node_ids":["n1"]}}' \
  | ./bin/node
# => {"src":"n1","dest":"c0","body":{"in_reply_to":1,"type":"init_ok"}}
```

> **Windows users:** Maelstrom relies on POSIX symlinks. See §9.4 for the workaround
> (Developer Mode, admin shell, or WSL).

---

## 2. What is being implemented and why

### The abstraction

**(Regular) Reliable Broadcast** (Module 3.2, book page 77) lets any process broadcast a
message such that the cluster *agrees* on which messages got delivered — even when the
original sender crashes mid-broadcast. It guarantees four properties:

| Property | Statement |
|---|---|
| **RB1 Validity** | If a correct process broadcasts `m`, it eventually delivers `m`. |
| **RB2 No-duplication** | No message is delivered more than once. |
| **RB3 No-creation** | A delivered message with sender `s` was previously broadcast by `s`. |
| **RB4 Agreement** | If *any* correct process delivers `m`, *every* correct process eventually delivers `m`. |

### Why best-effort broadcast is not enough

Best-Effort Broadcast (BEB, Module 3.1) gives RB1–RB3 but not RB4: if the sender crashes
mid-broadcast, some processes deliver `m` and others don't (book §3.3, p. 77). Agreement
is restored by combining BEB with a **Perfect Failure Detector (P)** that signals when
the original sender has crashed — then surviving processes that already saw `m` relay it
to everyone else. That is Algorithm 3.2.

### Why "lazy"

The algorithm only relays a message when it learns the original sender crashed (book
p. 79). In the happy path (no crashes) no relays happen — the message count stays at
`O(N)` per broadcast instead of the `O(N²)` of the eager variant (Algorithm 3.3,
p. 80).

---

## 3. Architecture

```
       stdin                                            stderr
         │                                                 ▲
         ▼                                                 │
 ┌───────────────────┐    ┌──────────────────────┐    log everywhere
 │ stdin reader gr.  │    │  PFD timer goroutine │
 └─────────┬─────────┘    └──────────┬───────────┘
           │  enqueue PLDeliver      │  enqueue Timeout
           ▼                         ▼
       ┌──────────── Global Event Queue (chan Event) ───────────┐
       └────────────────────────┬───────────────────────────────┘
                                ▼  (single consumer)
                       ┌────────────────────┐
                       │ event processor    │  sequential, no locks
                       └────────┬───────────┘
                                ▼
            APP ──► RB ──► { PFD , BEB } ──► PL ──► stdout
```

A broadcast travels down the stack on the way out and back up on the way in:

```
out:  APP.broadcast(7)
        └─► RB.Broadcast(7)
              └─► BEBBroadcast{RBData{origin=self, value=7}}
                    └─► for q in Π: PL.Send(q, {type:"rb_data", origin, value})
                          └─► JSON line to stdout (or local loopback if q==self)

in:   stdin → PL.PLDeliver → BEB.OnPLDeliver → BEBDeliver{RBData}
        └─► RB.OnBEBDeliver → RBDeliver{from, value}
              └─► APP.OnRBDeliver → delivered[value] = true
```

---

## 4. Layered packages (book ↔ code)

| Package | Book module | Responsibility |
|---|---|---|
| `internal/pl` | PerfectPointToPointLinks (simplified) | JSON envelope I/O; **local loopback** when `dest == self`. |
| `internal/beb` | Module 3.1, **Algorithm 3.1** (p. 76) | `forall q ∈ Π do trigger ⟨pl, Send | q, m⟩` — includes `self`. |
| `internal/pfd` | Module 2.6, **Algorithm 2.5** "Exclude on Timeout" | Heartbeat req/reply per peer; timeout → `Crash(p)`. |
| `internal/rb` | Module 3.2, **Algorithm 3.2** (p. 78) | `correct` set, `from[p]` dedup, lazy relay on crash. |
| `internal/app` | application layer | Maelstrom `broadcast`/`read`/`topology` workload glue. |
| `internal/event` | event framework | Sealed `Event` interface + buffered `Queue`. |

PL is intentionally a thin shim — `req.txt` says *"PL doesn't need to be implemented as
shown in the book. It just reads messages from STDIN and writes messages to STDOUT."*
Maelstrom's transport plus the RB-level duplicate filter cover what the book's PL would.

---

## 5. Algorithm 3.2 line-by-line

Pseudocode (book p. 78) on the left, our Go code on the right:

```
upon ⟨rb, Init⟩:                          rb.New, rb.SetSelf, rb.SetNodes
  correct := Π                              rb.correct (map[string]bool)
  from[p] := ∅                              rb.from (map[string]map[int]bool)

upon ⟨rb, Broadcast | m⟩:                  internal/rb/rb.go: RB.Broadcast
  trigger ⟨beb, Broadcast | [DATA,self,m]⟩    → BEBBroadcast{RBData{Origin:self, Value:m}}

upon ⟨beb, Deliver | p, [DATA,s,m]⟩:       internal/rb/rb.go: RB.OnBEBDeliver
  if m ∉ from[s]:                             if !r.from[s][m] { ... }
    trigger ⟨rb, Deliver | s, m⟩                → RBDeliver{From:s, Value:m}
    from[s] := from[s] ∪ {m}                    r.from[s][m] = true
    if s ∉ correct:                             if !r.correct[s] { ... }
      trigger ⟨beb, Broadcast | [DATA,s,m]⟩       → BEBBroadcast{RBData{Origin:s, ...}}

upon ⟨P, Crash | p⟩:                       internal/rb/rb.go: RB.OnCrash
  correct := correct \ {p}                    delete(r.correct, ev.Who)
  forall m ∈ from[p]:                         for m := range r.from[ev.Who] { ... }
    trigger ⟨beb, Broadcast | [DATA,p,m]⟩       → BEBBroadcast{RBData{Origin:p, Value:m}}
```

Two non-obvious details that the design review flagged:

- **The `origin` field is preserved on every relay.** Both the lazy relay branch and the
  crash sweep re-broadcast `[DATA, origin, value]` with the *original* origin, never
  overwritten with `self`. This is what keeps the cluster-wide `from[s]` dedup correct.
- **`m ∉ from[s]` uses `(origin, value)` identity.** In Go that is
  `map[string]map[int]bool` indexed by `(origin, value)` — message identity is not a
  random ID, it is the broadcast tuple itself.

---

## 6. Wire protocol

Maelstrom envelope: `{"src":"n1","dest":"n2","body":{...}}`. We use a **flat tagged
body**: exactly one `type` discriminator at the top of `body`, with payload fields
alongside.

| `type`         | direction          | body fields                          | role |
|----------------|--------------------|--------------------------------------|------|
| `init`         | maelstrom → node   | `node_id`, `node_ids`                | startup handshake |
| `init_ok`      | node → maelstrom   | `in_reply_to`                        | ack |
| `topology`     | client → node      | `topology`                           | workload setup |
| `topology_ok`  | node → client      | `in_reply_to`                        | ack |
| `broadcast`    | client → node      | `message` (number)                   | APP request to rb-broadcast |
| `broadcast_ok` | node → client      | `in_reply_to`                        | ack |
| `read`         | client → node      | —                                    | APP request to dump delivered |
| `read_ok`      | node → client      | `messages: [number]`                 | ack |
| `rb_data`      | node ↔ node        | `origin` (string), `value` (number)  | the book's `[DATA, s, m]` carried by BEB |
| `heartbeat`    | node → node        | `msg_id`                             | PFD ping |
| `heartbeat_ok` | node → node        | `in_reply_to`                        | PFD pong |

Concrete examples:

```json
// APP layer: client asks n1 to broadcast 7
{"src":"c1","dest":"n1","body":{"type":"broadcast","msg_id":42,"message":7}}
// n1's ack — Maelstrom requires in_reply_to == 42 or it fails the test
{"src":"n1","dest":"c1","body":{"type":"broadcast_ok","in_reply_to":42}}

// n1 broadcasts the value to a peer (BEB → PL)
{"src":"n1","dest":"n2","body":{"type":"rb_data","origin":"n1","value":7}}

// PFD heartbeat round-trip
{"src":"n1","dest":"n2","body":{"type":"heartbeat","msg_id":17}}
{"src":"n2","dest":"n1","body":{"type":"heartbeat_ok","in_reply_to":17}}
```

**`msg_id` / `in_reply_to` rule:** every client-bound reply (`init_ok`, `topology_ok`,
`broadcast_ok`, `read_ok`, `heartbeat_ok`) MUST echo the request's `msg_id` via
`in_reply_to`. Missing or wrong values cause Maelstrom to fail the test immediately.

---

## 7. Concurrency model and event nesting

### Three goroutines, one queue

| Goroutine | Lifetime | Job |
|---|---|---|
| `stdinReader` | start of process | `bufio.Scanner` on STDIN → `PLDeliver` → queue |
| `pfdTicker` | **starts after `init`** | `time.Ticker` every `--pfd-timeout-ms` → `Timeout` → queue |
| event processor (main) | start of process | `for { dispatch(q.Dequeue()) }` |

Only the processor mutates layer state and only the processor writes to STDOUT. There
are no mutexes inside any layer — the queue is the single synchronization point.

**Why `pfdTicker` waits for `init`:** before the `init` message lands, the node has no
`node_id` and no `node_ids`. Starting heartbeats earlier would send to empty
destinations and run a PFD round before `Π` is defined. `main.go` closes an
`initDone chan struct{}` from the `init` handler; the ticker goroutine blocks on
`<-initDone` before its first tick.

### "Events are nestable" (`req.txt` line 11)

Two events carry inner events as fields, realising the assignment's nestability rule:

```go
// internal/event/event.go
type BEBBroadcast struct{ Inner Event }              // RB → BEB
type BEBDeliver   struct{ From string; Inner Event } // PL → BEB → RB
```

Concretely, `RB.Broadcast(7)` returns `BEBBroadcast{Inner: RBData{Origin: self, Value: 7}}`
— the BEB event literally carries the RB-level event it transports. JSON flattening
happens only at the wire boundary inside PL.

---

## 8. Verification — tests and what they prove

Run:

```sh
make test
# ok   amcdistsys/homework2/internal/app
# ok   amcdistsys/homework2/internal/beb
# ok   amcdistsys/homework2/internal/pfd
# ok   amcdistsys/homework2/internal/rb
```

### Mapping tests → properties

| Test                                                            | Layer | Proves / guards against |
|-----------------------------------------------------------------|-------|--------------------------|
| `beb.TestBroadcastSendsToAllIncludingSelf`                      | BEB   | Algorithm 3.1 — `forall q ∈ Π` (incl. self). Catches the RB1-violation bug from the design review. |
| `beb.TestOnPLDeliverRoutesRBData`                               | BEB   | Wire `rb_data` is lifted to `BEBDeliver` with correct `From` and `Inner`. |
| `beb.TestOnPLDeliverIgnoresOtherTypes`                          | BEB   | Wrong-layer messages (heartbeats) are not forwarded as BEB delivers. |
| `pfd.TestMissedHeartbeatProducesCrash`                          | PFD   | After one missed round, every silent peer is suspected exactly once. |
| `pfd.TestHeartbeatOkPreventsCrash`                              | PFD   | A timely `heartbeat_ok` cancels the suspicion. |
| `pfd.TestCrashEmittedOnce`                                      | PFD   | A peer becomes suspected idempotently — `Crash(p)` fires once even on repeated timeouts. |
| `pfd.TestOnHeartbeatReqSendsOk`                                 | PFD   | Reply has `type:"heartbeat_ok"` and correct `in_reply_to`. |
| `rb.TestBroadcastEmitsBEBWithDataAndSelfOrigin`                 | RB    | `RB.Broadcast(m)` produces `[DATA, self, m]`. |
| `rb.TestDuplicateDeliveryIsSuppressed`                          | RB    | **RB2 No-duplication** — `from[s]` dedup works across peers. |
| `rb.TestNoRelayWhenOriginAlive`                                 | RB    | Lazy semantics — happy path produces no relays. |
| `rb.TestLazyRelayOnLateDeliveryAfterCrash`                      | RB    | Late `BEBDeliver` from already-crashed sender still relays (book p. 79, the "two kinds of events" paragraph). |
| `rb.TestCrashTriggersRelayOfPriorMessages`                      | RB    | **RB4 Agreement** — `OnCrash(p)` sweeps `from[p]` and re-broadcasts every prior message, preserving `origin`. |
| `rb.TestCrashTwiceIsIdempotent`                                 | RB    | Re-detecting a crash is a no-op. |
| `app.TestOnBroadcastReplyAndTriggersRB`                         | APP   | `broadcast_ok` ack with correct `in_reply_to`; RB layer is triggered. |
| `app.TestOnReadReturnsSortedDelivered`                          | APP   | `read_ok` returns deduped, sorted delivered values. |

### End-to-end (Maelstrom)

`make test-clean` and `make test-partition` execute the full stack under Maelstrom's
broadcast workload, which checks RB1–RB4 across the cluster.

---

## 9. Sample Maelstrom run output

### 9.1 Smoke test (manual stdin)

```sh
$ printf '%s\n' \
    '{"src":"c0","dest":"n1","body":{"type":"init","msg_id":1,"node_id":"n1","node_ids":["n1","n2","n3"]}}' \
    '{"src":"c1","dest":"n1","body":{"type":"broadcast","msg_id":2,"message":7}}' \
  | ./bin/node -pfd-timeout-ms=60000

{"src":"n1","dest":"c0","body":{"in_reply_to":1,"type":"init_ok"}}
{"src":"n1","dest":"c1","body":{"in_reply_to":2,"type":"broadcast_ok"}}
{"src":"n1","dest":"n2","body":{"origin":"n1","type":"rb_data","value":7}}
{"src":"n1","dest":"n3","body":{"origin":"n1","type":"rb_data","value":7}}
```

Note that there is no `rb_data` line addressed to `n1` itself — PL's local-loopback
short-circuit kept the self-delivery inside the process (so the originator's APP sees
its own broadcast without any round-trip through STDOUT/STDIN).

### 9.2 `make test-clean` (expected)

```
:results
 {:valid? true,
  :lost-count 0,
  :stable-count 50,
  :stable-latencies {0    0,
                     0.5  154,
                     0.95 410,
                     0.99 580,
                     1.0  620},
  :worst-stale ()}
```

### 9.3 `make test-partition` (expected)

The PFD trips during partitions; the lazy relay kicks in once `Crash(p)` fires.
`:valid? true` should still hold because the algorithm is safe under partition
(see §10.2). Expect higher latency percentiles and a non-zero relay count in stderr
logs.

### 9.4 Running Maelstrom on Windows

Maelstrom (Jepsen) creates a `store/current` **symlink** which on Windows requires
either admin privileges or **Developer Mode** enabled (`Settings → Update & Security →
For developers → Developer Mode`). Symptom:

```
java.nio.file.FileSystemException: store\current: A required privilege is not held by the client
```

Workarounds:

- Run the shell as administrator, or
- Enable Developer Mode, or
- Use WSL / Linux.

The Go binary itself is portable — only Maelstrom's storage layer is the snag.

---

## 10. Discussion

### 10.1 How each property is achieved

- **RB1 Validity.** `BEB.Broadcast` iterates `forall q ∈ Π` *including* `self` (book
  Algorithm 3.1, p. 76, and explicit statement on p. 75: *"to all processes in a system,
  including itself"*). PL's local loopback enqueues a `PLDeliver` for the self-send;
  that bubbles up through BEB → RB → APP. The originator therefore delivers its own
  message even with no peers.
- **RB2 No-duplication.** `from[s]` set membership: every `(origin, value)` is
  delivered at most once (`rb.go:OnBEBDeliver` first line).
- **RB3 No-creation.** The `origin` field carries the original sender across every
  relay; RB only delivers messages with an `origin` actually present in `Π`.
- **RB4 Agreement.** Either the sender stays alive (every correct peer hears via the
  initial BEB) or it crashes. In the crash case, two flavors of relay kick in:
  1. A peer detects the crash and `OnCrash(p)` sweeps `from[p]` and re-broadcasts
     every message that peer had already received from `p`.
  2. If a `BEBDeliver` for `[DATA, p, m]` arrives *after* `p` was suspected (e.g.
     because a third party already started relaying it), `OnBEBDeliver`'s
     `origin ∉ correct` branch immediately re-broadcasts. This is the book's "two
     kinds of events that can force a process to retransmit" (p. 79).

### 10.2 PFD caveat under partitions

A Perfect Failure Detector is by definition only sound under synchrony (book §2.6.4,
strong accuracy). Maelstrom's broadcast workload does not crash node processes; the
closest fault is `--nemesis partition`. A long partition will look like a crash to our
PFD, so the surviving side will *relay* messages on behalf of the
partitioned-but-alive node. When the partition heals:

- Safety holds: `from[s]` dedup catches duplicates; `origin` preserves provenance, so
  no-creation cannot be violated.
- Liveness holds: agreement is still satisfied (everyone gets every message).
- **Accuracy** of the PFD is technically violated — the relay was performed on a peer
  that did not actually crash.

This is inherent to running a fail-stop algorithm on an only-eventually-synchronous
transport, and is the reason real systems prefer the *eventually* perfect failure
detector (◇P, book Algorithm 2.7) — out of scope for this homework.

### 10.3 Lazy vs eager

Algorithm 3.3 "Eager Reliable Broadcast" (book p. 80) achieves RB4 *without* a failure
detector by having every process relay every message on first sight. Cost: `O(N²)`
messages per broadcast even in the happy path.

Algorithm 3.2 (this implementation) only pays that price when an actual crash is
detected. With `N = 5` nodes and no faults, our algorithm exchanges `N − 1 = 4`
`rb_data` messages per `broadcast` — that matches the book's quoted `O(N)`
single-step cost (p. 79).

### 10.4 What is intentionally out of scope (YAGNI)

- Uniform Reliable Broadcast (Algorithm 3.4).
- Causal or total order.
- Byzantine variants.
- Persistence / crash recovery.
- Eventually Perfect Failure Detector (◇P).
- A true Perfect Links implementation with retries (`req.txt` explicitly simplifies it).

---

## 11. Repository layout

```
homework2/
├── README.md
├── Makefile                                 build / test / test-clean / test-partition
├── .gitignore                               bin/, store/, docs/, .claude/, *.out
├── go.mod
├── req.txt                                  the assignment statement
├── cmd/
│   └── node/main.go                         wiring + 3 goroutines + dispatch
└── internal/
    ├── event/
    │   ├── event.go                         sealed Event interface, per-kind structs
    │   └── queue.go                         chan-backed FIFO
    ├── pl/
    │   └── pl.go                            JSON envelope I/O + local loopback
    ├── beb/
    │   ├── beb.go                           forall q in Π (incl. self)
    │   └── beb_test.go
    ├── pfd/
    │   ├── pfd.go                           Algorithm 2.5 heartbeats
    │   └── pfd_test.go
    ├── rb/
    │   ├── rb.go                            Algorithm 3.2
    │   └── rb_test.go
    └── app/
        ├── app.go                           Maelstrom broadcast workload glue
        └── app_test.go
```

The `docs/` and `.claude/` directories hold local design artefacts and IDE state — both
are gitignored.

---

## 12. References

- **Cachin, Guerraoui, Rodrigues** — *Introduction to Reliable and Secure Distributed
  Programming*, 2nd ed., Springer 2011.
  - §3.2, Algorithm 3.1 (Basic Broadcast), p. 76.
  - §3.3.1, Module 3.2 (Reliable Broadcast spec), p. 77.
  - §3.3.2, **Algorithm 3.2 (Lazy Reliable Broadcast)**, p. 78.
  - §3.3.3, Algorithm 3.3 (Eager Reliable Broadcast) — comparison, p. 80.
  - §2.6.4 / Algorithm 2.5 — Perfect Failure Detector.
- **Kingsbury** — Jepsen Maelstrom: <https://github.com/jepsen-io/maelstrom>.
- **Maelstrom protocol & broadcast workload docs:**
  `maelstrom/doc/protocol.md`, `maelstrom/doc/workloads.md#workload-broadcast`,
  `maelstrom/doc/03-broadcast/`.
