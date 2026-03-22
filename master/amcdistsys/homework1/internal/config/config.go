package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type NodeAddr struct {
	IP   string
	Port int
}

type Config struct {
	N     int        // number of broadcasts each node sends
	Nodes []NodeAddr // index = node index
}

// ParseConfig reads the config file and returns a Config.
// First line: N (number of broadcasts). Remaining lines: IP PORT.
// Lines starting with '#' or empty lines are ignored after the first line.
func ParseConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ParseConfig: open %q: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	if !scanner.Scan() {
		return nil, fmt.Errorf("ParseConfig: empty config file")
	}
	n, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		return nil, fmt.Errorf("ParseConfig: invalid N %q: %w", scanner.Text(), err)
	}

	cfg := &Config{N: n}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// strip inline comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return nil, fmt.Errorf("ParseConfig: malformed line %q", line)
		}
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("ParseConfig: invalid port %q: %w", parts[1], err)
		}
		cfg.Nodes = append(cfg.Nodes, NodeAddr{IP: parts[0], Port: port})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ParseConfig: scan: %w", err)
	}
	if len(cfg.Nodes) == 0 {
		return nil, fmt.Errorf("ParseConfig: no nodes defined")
	}
	return cfg, nil
}
