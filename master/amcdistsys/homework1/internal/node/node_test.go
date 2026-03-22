package node

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cosmintimis/learning-go/master/amcdistsys/homework1/internal/config"
	"github.com/cosmintimis/learning-go/master/amcdistsys/homework1/internal/logger"
	"github.com/cosmintimis/learning-go/master/amcdistsys/homework1/internal/message"
)

// getFreePort returns an available UDP port on localhost.
func getFreePort(t *testing.T) int {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port
	conn.Close()
	return port
}

// --- Requirement: NewNode binds UDP socket on the node's address ---

func TestNewNode_BindsUDP(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	port := getFreePort(t)
	cfg := &config.Config{
		N:     1,
		Nodes: []config.NodeAddr{{IP: "127.0.0.1", Port: port}},
	}

	lg, err := logger.NewMsgLogger(0)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	defer lg.Close()

	n, err := NewNode(0, cfg, lg)
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	// Node should have a bound connection — close it via the conn field
	n.conn.Close()
}

// --- Requirement: NewNode should fail if port is already in use ---

func TestNewNode_PortInUse(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	port := getFreePort(t)

	// Occupy the port
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", port))
	occupied, err := net.ListenUDP("udp4", addr)
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	defer occupied.Close()

	cfg := &config.Config{
		N:     1,
		Nodes: []config.NodeAddr{{IP: "127.0.0.1", Port: port}},
	}

	lg, err := logger.NewMsgLogger(0)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	defer lg.Close()

	_, err = NewNode(0, cfg, lg)
	if err == nil {
		t.Fatal("expected error when port is already in use")
	}
}

// --- Requirement: insistWrite sends exactly MessageSize bytes ---

func TestInsistWrite_SendsFullMessage(t *testing.T) {
	// Set up a receiver
	recvAddr, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	recvConn, err := net.ListenUDP("udp4", recvAddr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer recvConn.Close()

	// Set up a sender
	sendAddr, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	sendConn, err := net.ListenUDP("udp4", sendAddr)
	if err != nil {
		t.Fatalf("listen sender: %v", err)
	}
	defer sendConn.Close()

	msg := message.BuildMessage(0)
	destAddr := recvConn.LocalAddr().(*net.UDPAddr)

	if err := insistWrite(sendConn, msg.Bytes(), destAddr); err != nil {
		t.Fatalf("insistWrite: %v", err)
	}

	buf := make([]byte, message.MessageSize+100)
	recvConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := recvConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if n != message.MessageSize {
		t.Errorf("received %d bytes, expected %d", n, message.MessageSize)
	}
}

// --- Requirement: insistRead reads exactly MessageSize bytes ---

func TestInsistRead_ReadsFullMessage(t *testing.T) {
	// Receiver
	recvAddr, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	recvConn, err := net.ListenUDP("udp4", recvAddr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer recvConn.Close()

	// Sender
	sendAddr, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	sendConn, err := net.ListenUDP("udp4", sendAddr)
	if err != nil {
		t.Fatalf("listen sender: %v", err)
	}
	defer sendConn.Close()

	msg := message.BuildMessage(5)
	destAddr := recvConn.LocalAddr().(*net.UDPAddr)
	sendConn.WriteToUDP(msg.Bytes(), destAddr)

	buf := make([]byte, message.MessageSize)
	n, err := insistRead(recvConn, buf)
	if err != nil {
		t.Fatalf("insistRead: %v", err)
	}
	if n != message.MessageSize {
		t.Errorf("insistRead returned %d, expected %d", n, message.MessageSize)
	}
}

// --- Requirement: insistRead times out after 5s (verify deadline is set) ---

func TestInsistRead_Timeout(t *testing.T) {
	addr, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, message.MessageSize)
	start := time.Now()
	_, err = insistRead(conn, buf)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error when nothing is sent")
	}
	netErr, ok := err.(net.Error)
	if !ok || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got: %v", err)
	}
	// Should timeout in ~5s, allow some margin
	if elapsed < 4*time.Second || elapsed > 7*time.Second {
		t.Errorf("timeout should be ~5s, took %v", elapsed)
	}
}

// --- Full integration: 2 nodes exchange N broadcasts and log correctly ---

func TestFullBroadcast_TwoNodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	port0 := getFreePort(t)
	port1 := getFreePort(t)

	N := 3
	cfg := &config.Config{
		N: N,
		Nodes: []config.NodeAddr{
			{IP: "127.0.0.1", Port: port0},
			{IP: "127.0.0.1", Port: port1},
		},
	}
	M := len(cfg.Nodes)

	// Override the startup wait so tests don't take 15 seconds
	origStartupWait := startupWait
	// We can't override const, so we'll directly test with a small N
	_ = origStartupWait

	// Create loggers
	lg0, err := logger.NewMsgLogger(0)
	if err != nil {
		t.Fatalf("logger 0: %v", err)
	}
	lg1, err := logger.NewMsgLogger(1)
	if err != nil {
		t.Fatalf("logger 1: %v", err)
	}

	// Create nodes
	n0, err := NewNode(0, cfg, lg0)
	if err != nil {
		t.Fatalf("node 0: %v", err)
	}
	n1, err := NewNode(1, cfg, lg1)
	if err != nil {
		t.Fatalf("node 1: %v", err)
	}

	// Run nodes concurrently (this will block for 15s startup + broadcast time)
	done := make(chan int, 2)
	go func() { n0.Run(); lg0.Close(); done <- 0 }()
	go func() { n1.Run(); lg1.Close(); done <- 1 }()

	// Wait for both nodes with a generous timeout
	timeout := time.After(60 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case idx := <-done:
			t.Logf("node %d finished", idx)
		case <-timeout:
			t.Fatal("test timed out waiting for nodes to complete")
		}
	}

	// --- Verify: each node should have received N*M messages ---
	expectedMessages := N * M

	for nodeIdx := 0; nodeIdx < 2; nodeIdx++ {
		msgPath := filepath.Join("logs", fmt.Sprintf("node_%d_messages.log", nodeIdx))
		data, err := os.ReadFile(msgPath)
		if err != nil {
			t.Fatalf("node %d: read message log: %v", nodeIdx, err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != expectedMessages {
			t.Errorf("node %d: expected %d message log lines, got %d", nodeIdx, expectedMessages, len(lines))
		}

		// All messages should be OK (no corruption in local test)
		for i, line := range lines {
			if !strings.HasPrefix(line, "OK ") {
				t.Errorf("node %d, line %d: expected OK, got: %s", nodeIdx, i, line)
			}
		}
	}

	// --- Verify: error logs should be empty ---
	for nodeIdx := 0; nodeIdx < 2; nodeIdx++ {
		errPath := filepath.Join("logs", fmt.Sprintf("node_%d_errors.log", nodeIdx))
		data, err := os.ReadFile(errPath)
		if err != nil {
			t.Fatalf("node %d: read error log: %v", nodeIdx, err)
		}
		if len(strings.TrimSpace(string(data))) > 0 {
			t.Errorf("node %d: error log should be empty, got: %s", nodeIdx, string(data))
		}
	}
}

// --- Requirement: broadcast sent to source as well (self-send) ---

func TestBroadcast_IncludesSelf(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	port := getFreePort(t)
	N := 2
	cfg := &config.Config{
		N:     N,
		Nodes: []config.NodeAddr{{IP: "127.0.0.1", Port: port}},
	}

	lg, err := logger.NewMsgLogger(0)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}

	n, err := NewNode(0, cfg, lg)
	if err != nil {
		t.Fatalf("node: %v", err)
	}

	done := make(chan struct{})
	go func() { n.Run(); lg.Close(); close(done) }()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out — node should receive its own broadcasts")
	}

	// With M=1 (single node), N=2: node should receive N*M = 2 messages from itself
	msgPath := filepath.Join("logs", "node_0_messages.log")
	data, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != N {
		t.Errorf("single node should receive %d self-messages, got %d", N, len(lines))
	}
	// All should show sender index 0 (self)
	for _, line := range lines {
		if !strings.HasPrefix(line, "OK 0 ") {
			t.Errorf("self-broadcast log should show source 0, got: %s", line)
		}
	}
}
