package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T) (originalDir string, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return orig, func() { os.Chdir(orig) }
}

// --- Requirement: log messages to node_<index>_messages.log ---
// --- Requirement: log errors to a separate file ---

func TestNewMsgLogger_CreatesLogsDir(t *testing.T) {
	orig, cleanup := setupTestDir(t)
	_ = orig
	defer cleanup()

	lg, err := NewMsgLogger(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lg.Close()

	// logs/ directory should exist
	info, err := os.Stat(logsDir)
	if err != nil {
		t.Fatalf("logs dir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("logs should be a directory")
	}
}

func TestNewMsgLogger_CreatesLogFiles(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	lg, err := NewMsgLogger(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer lg.Close()

	msgPath := filepath.Join(logsDir, "node_3_messages.log")
	errPath := filepath.Join(logsDir, "node_3_errors.log")

	if _, err := os.Stat(msgPath); err != nil {
		t.Errorf("message log file should exist: %v", err)
	}
	if _, err := os.Stat(errPath); err != nil {
		t.Errorf("error log file should exist: %v", err)
	}
}

// --- Requirement: log format "OK/FAIL <source_index> <sent_sha1_hex> <calc_sha1_hex>" ---

func TestLogMessage_OKFormat(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	lg, err := NewMsgLogger(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sentHex := "0123456789abcdef0123456789abcdef01234567"
	calcHex := "0123456789abcdef0123456789abcdef01234567"
	lg.LogMessage(true, 1, sentHex, calcHex)
	lg.Close()

	msgPath := filepath.Join(logsDir, "node_0_messages.log")
	data, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	line := strings.TrimSpace(string(data))
	expected := "OK 1 " + sentHex + " " + calcHex
	if line != expected {
		t.Errorf("log line mismatch:\ngot:      %q\nexpected: %q", line, expected)
	}
}

func TestLogMessage_FAILFormat(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	lg, err := NewMsgLogger(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sentHex := "0123456789abcdef0123456789abcdef01234567"
	calcHex := "0123456789abcdef0123456789abcdef01234aaa"
	lg.LogMessage(false, 0, sentHex, calcHex)
	lg.Close()

	msgPath := filepath.Join(logsDir, "node_0_messages.log")
	data, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	line := strings.TrimSpace(string(data))
	expected := "FAIL 0 " + sentHex + " " + calcHex
	if line != expected {
		t.Errorf("log line mismatch:\ngot:      %q\nexpected: %q", line, expected)
	}
}

// --- Multiple log entries ---

func TestLogMessage_MultipleEntries(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	lg, err := NewMsgLogger(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	lg.LogMessage(true, 1, sha, sha)
	lg.LogMessage(true, 2, sha, sha)
	lg.LogMessage(false, 0, sha, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	lg.Close()

	msgPath := filepath.Join(logsDir, "node_2_messages.log")
	data, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 log lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "OK 1") {
		t.Errorf("line 0: expected OK 1..., got %q", lines[0])
	}
	if !strings.HasPrefix(lines[2], "FAIL 0") {
		t.Errorf("line 2: expected FAIL 0..., got %q", lines[2])
	}
}

// --- Requirement: errors in separate file ---

func TestLogError_WritesToErrorFile(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	lg, err := NewMsgLogger(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lg.LogError("sendLoop: resolve %s:%d: %v", "127.0.0.1", 5000, "connection refused")
	lg.Close()

	errPath := filepath.Join(logsDir, "node_1_errors.log")
	data, err := os.ReadFile(errPath)
	if err != nil {
		t.Fatalf("read error log: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "sendLoop") {
		t.Errorf("error log should contain error message, got: %q", content)
	}
	if !strings.Contains(content, "connection refused") {
		t.Errorf("error log should contain error details, got: %q", content)
	}
}

// --- Error log should have timestamps (LstdFlags) ---

func TestLogError_HasTimestamp(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	lg, err := NewMsgLogger(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lg.LogError("test error")
	lg.Close()

	errPath := filepath.Join(logsDir, "node_0_errors.log")
	data, err := os.ReadFile(errPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	// LstdFlags format includes date like "2024/01/15"
	if !strings.Contains(content, "/") {
		t.Errorf("error log should have timestamp (LstdFlags), got: %q", content)
	}
}

// --- Message log should have NO prefix (matches required format exactly) ---

func TestLogMessage_NoTimestamp(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	lg, err := NewMsgLogger(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sha := "0123456789abcdef0123456789abcdef01234567"
	lg.LogMessage(true, 0, sha, sha)
	lg.Close()

	msgPath := filepath.Join(logsDir, "node_0_messages.log")
	data, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	line := strings.TrimSpace(string(data))
	// Should start directly with "OK", no timestamp prefix
	if !strings.HasPrefix(line, "OK") {
		t.Errorf("message log should start with OK/FAIL (no prefix), got: %q", line)
	}
}

// --- Multiple node indices should create separate files ---

func TestNewMsgLogger_DifferentIndices(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	lg0, err := NewMsgLogger(0)
	if err != nil {
		t.Fatalf("node 0: %v", err)
	}
	lg1, err := NewMsgLogger(1)
	if err != nil {
		t.Fatalf("node 1: %v", err)
	}

	sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	lg0.LogMessage(true, 0, sha, sha)
	lg1.LogMessage(true, 1, sha, sha)
	lg0.Close()
	lg1.Close()

	data0, _ := os.ReadFile(filepath.Join(logsDir, "node_0_messages.log"))
	data1, _ := os.ReadFile(filepath.Join(logsDir, "node_1_messages.log"))

	if !strings.Contains(string(data0), "OK 0") {
		t.Error("node 0 log should contain 'OK 0'")
	}
	if !strings.Contains(string(data1), "OK 1") {
		t.Error("node 1 log should contain 'OK 1'")
	}
}
