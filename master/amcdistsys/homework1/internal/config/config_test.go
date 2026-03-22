package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.txt")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTempConfig: %v", err)
	}
	return p
}

// --- Requirement: Configuration file first line = N (broadcasts per node) ---

func TestParseConfig_ValidSimple(t *testing.T) {
	p := writeTempConfig(t, "100\n127.0.0.1 5000\n127.0.0.1 5001\n")
	cfg, err := ParseConfig(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.N != 100 {
		t.Errorf("expected N=100, got %d", cfg.N)
	}
	if len(cfg.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(cfg.Nodes))
	}
	if cfg.Nodes[0].IP != "127.0.0.1" || cfg.Nodes[0].Port != 5000 {
		t.Errorf("node 0 mismatch: %+v", cfg.Nodes[0])
	}
	if cfg.Nodes[1].IP != "127.0.0.1" || cfg.Nodes[1].Port != 5001 {
		t.Errorf("node 1 mismatch: %+v", cfg.Nodes[1])
	}
}

// --- Requirement: Example config with 5 nodes ---

func TestParseConfig_FiveNodes(t *testing.T) {
	content := "1000\n127.0.0.1 5000\n127.0.0.1 5001\n127.0.0.1 5002\n127.0.0.1 5003\n127.0.0.1 5004\n"
	p := writeTempConfig(t, content)
	cfg, err := ParseConfig(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.N != 1000 {
		t.Errorf("expected N=1000, got %d", cfg.N)
	}
	if len(cfg.Nodes) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(cfg.Nodes))
	}
	for i, port := range []int{5000, 5001, 5002, 5003, 5004} {
		if cfg.Nodes[i].Port != port {
			t.Errorf("node %d: expected port %d, got %d", i, port, cfg.Nodes[i].Port)
		}
	}
}

// --- Inline comments should be stripped ---

func TestParseConfig_InlineComments(t *testing.T) {
	content := "10\n127.0.0.1 5000  # index 0\n127.0.0.1 5001  # index 1\n"
	p := writeTempConfig(t, content)
	cfg, err := ParseConfig(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(cfg.Nodes))
	}
}

// --- Empty lines should be ignored ---

func TestParseConfig_EmptyLinesIgnored(t *testing.T) {
	content := "5\n\n127.0.0.1 5000\n\n127.0.0.1 5001\n\n"
	p := writeTempConfig(t, content)
	cfg, err := ParseConfig(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(cfg.Nodes))
	}
}

// --- Error cases ---

func TestParseConfig_EmptyFile(t *testing.T) {
	p := writeTempConfig(t, "")
	_, err := ParseConfig(p)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestParseConfig_InvalidN(t *testing.T) {
	p := writeTempConfig(t, "abc\n127.0.0.1 5000\n")
	_, err := ParseConfig(p)
	if err == nil {
		t.Fatal("expected error for non-numeric N")
	}
}

func TestParseConfig_NoNodes(t *testing.T) {
	p := writeTempConfig(t, "10\n")
	_, err := ParseConfig(p)
	if err == nil {
		t.Fatal("expected error when no nodes are defined")
	}
}

func TestParseConfig_InvalidPort(t *testing.T) {
	p := writeTempConfig(t, "10\n127.0.0.1 not_a_port\n")
	_, err := ParseConfig(p)
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestParseConfig_MalformedLine(t *testing.T) {
	p := writeTempConfig(t, "10\njust_an_ip\n")
	_, err := ParseConfig(p)
	if err == nil {
		t.Fatal("expected error for line with only one field")
	}
}

func TestParseConfig_FileNotFound(t *testing.T) {
	_, err := ParseConfig("/nonexistent/path/config.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// --- Node index mapping: index = line order ---

func TestParseConfig_NodeIndexMatchesLineOrder(t *testing.T) {
	content := "1\n10.0.0.1 6000\n10.0.0.2 6001\n10.0.0.3 6002\n"
	p := writeTempConfig(t, content)
	cfg, err := ParseConfig(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []struct {
		ip   string
		port int
	}{
		{"10.0.0.1", 6000},
		{"10.0.0.2", 6001},
		{"10.0.0.3", 6002},
	}
	for i, e := range expected {
		if cfg.Nodes[i].IP != e.ip || cfg.Nodes[i].Port != e.port {
			t.Errorf("node %d: expected %s:%d, got %s:%d", i, e.ip, e.port, cfg.Nodes[i].IP, cfg.Nodes[i].Port)
		}
	}
}
