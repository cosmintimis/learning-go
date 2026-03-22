package node

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cosmintimis/learning-go/master/amcdistsys/homework1/internal/config"
	"github.com/cosmintimis/learning-go/master/amcdistsys/homework1/internal/logger"
	"github.com/cosmintimis/learning-go/master/amcdistsys/homework1/internal/message"
)

const (
	startupWait = 15 * time.Second
	ioTimeout   = 5 * time.Second
)

// Node represents a single broadcast node.
type Node struct {
	index     int
	config    *config.Config
	conn      *net.UDPConn
	logger    *logger.MsgLogger
	recvCount atomic.Int64
}

// NewNode creates a Node and binds its UDP socket on the node's own address.
func NewNode(index int, cfg *config.Config, lg *logger.MsgLogger) (*Node, error) {
	addr := cfg.Nodes[index]
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", addr.IP, addr.Port))
	if err != nil {
		return nil, fmt.Errorf("NewNode: resolve addr: %w", err)
	}
	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("NewNode: listen UDP on %s:%d: %w", addr.IP, addr.Port, err)
	}
	return &Node{index: index, config: cfg, conn: conn, logger: lg}, nil
}

// Run starts the node lifecycle:
//  1. Receiver goroutine starts immediately (captures early messages from other nodes)
//  2. Sleeps 15 seconds (startup wait for all nodes to spin up)
//  3. Sender goroutine starts broadcasting
//  4. Blocks until both goroutines complete
func (n *Node) Run() {
	defer n.conn.Close()

	M := len(n.config.Nodes)
	N := n.config.N
	total := int64(N * M)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	// 1. Start receiver immediately so we don't miss messages from early-waking nodes
	go n.receiveLoop(ctx, &wg, total)

	// 2. Wait for all nodes to spin up
	fmt.Printf("Node %d: waiting %v before broadcasting...\n", n.index, startupWait)
	time.Sleep(startupWait)

	// 3. Start sender
	fmt.Printf("Node %d: starting broadcasts (N=%d, M=%d, total_expected=%d)\n", n.index, N, M, total)
	go n.sendLoop(ctx, &wg, N, cancel)

	wg.Wait()
	fmt.Printf("Node %d: done\n", n.index)
}

// sendLoop sends N broadcasts to all M nodes (including self), then signals completion via cancel.
func (n *Node) sendLoop(ctx context.Context, wg *sync.WaitGroup, N int, cancel context.CancelFunc) {
	defer wg.Done()
	defer cancel() // signal receiver that all sends are done

	for i := 0; i < N; i++ {
		msg := message.BuildMessage(uint8(n.index))
		for _, dest := range n.config.Nodes {
			destAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", dest.IP, dest.Port))
			if err != nil {
				n.logger.LogError("sendLoop: resolve %s:%d: %v", dest.IP, dest.Port, err)
				continue
			}
			if err := insistWrite(n.conn, msg.Bytes(), destAddr); err != nil {
				n.logger.LogError("sendLoop: send to %s:%d: %v", dest.IP, dest.Port, err)
			}
		}
	}
}

// receiveLoop reads messages until total messages have been received or ctx is done.
func (n *Node) receiveLoop(ctx context.Context, wg *sync.WaitGroup, total int64) {
	defer wg.Done()

	buf := make([]byte, message.MessageSize)
	for {
		if n.recvCount.Load() >= total {
			return // clean exit: received all expected messages
		}

		recvd, err := insistRead(n.conn, buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout: only exit if the sender is done (ctx cancelled)
				select {
				case <-ctx.Done():
					return
				default:
					continue // sender still running, keep waiting
				}
			}
			// Non-timeout error (transient buffer glitch, etc.): log and retry
			n.logger.LogError("receiveLoop: %v", err)
			continue
		}

		msg, err := message.ParseMessage(buf[:recvd])
		if err != nil {
			n.logger.LogError("receiveLoop: parse: %v", err)
			continue
		}

		sentHex, calcHex, ok := msg.Verify()
		n.logger.LogMessage(ok, msg.SenderIndex(), sentHex, calcHex)
		n.recvCount.Add(1)
	}
}

// insistRead performs a single UDP read with a 5-second deadline.
// Returns the number of bytes read, or an error (caller distinguishes timeout vs other errors).
func insistRead(conn *net.UDPConn, buf []byte) (int, error) {
	if err := conn.SetReadDeadline(time.Now().Add(ioTimeout)); err != nil {
		return 0, fmt.Errorf("insistRead: set deadline: %w", err)
	}
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return 0, err
	}
	if n != message.MessageSize {
		return n, fmt.Errorf("insistRead: partial read: got %d bytes, expected %d", n, message.MessageSize)
	}
	return n, nil
}

// insistWrite attempts to write data to dest with a 5-second deadline.
func insistWrite(conn *net.UDPConn, data []byte, dest *net.UDPAddr) error {
	if err := conn.SetWriteDeadline(time.Now().Add(ioTimeout)); err != nil {
		return fmt.Errorf("insistWrite: set deadline: %w", err)
	}
	n, err := conn.WriteToUDP(data, dest)
	if err != nil {
		return fmt.Errorf("insistWrite: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("insistWrite: short write: sent %d/%d bytes", n, len(data))
	}
	return nil
}
