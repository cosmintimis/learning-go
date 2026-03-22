package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cosmintimis/learning-go/master/amcdistsys/homework1/internal/config"
	"github.com/cosmintimis/learning-go/master/amcdistsys/homework1/internal/logger"
	"github.com/cosmintimis/learning-go/master/amcdistsys/homework1/internal/node"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: bcastnode <config_file> <node_index>\n")
		os.Exit(1)
	}

	configPath := os.Args[1]
	nodeIndex, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid node index %q: %v\n", os.Args[2], err)
		os.Exit(1)
	}

	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	if nodeIndex < 0 || nodeIndex >= len(cfg.Nodes) {
		fmt.Fprintf(os.Stderr, "node index %d out of range [0, %d)\n", nodeIndex, len(cfg.Nodes))
		os.Exit(1)
	}

	lg, err := logger.NewMsgLogger(nodeIndex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}
	defer lg.Close()

	n, err := node.NewNode(nodeIndex, cfg, lg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "node error: %v\n", err)
		os.Exit(1)
	}

	n.Run()
}
