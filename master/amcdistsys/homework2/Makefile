BINARY := bin/node
MAELSTROM_JAR := ../../../../maelstrom/maelstrom/lib/maelstrom.jar
NODE_COUNT ?= 5
TIME_LIMIT ?= 20
RATE ?= 10
SERVE_PORT ?= 8080

ifeq ($(OS),Windows_NT)
	BINARY := bin/node.exe
endif

.PHONY: all build test test-clean test-clean-verbose test-partition test-partition-verbose test-kill test-kill-verbose results serve clean

all: build

build:
	go build -o $(BINARY) ./cmd/node

test:
	go test ./...

test-clean: build
	java -jar $(MAELSTROM_JAR) test \
		--workload broadcast \
		--bin ./$(BINARY) \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE)

test-partition: build
	java -jar $(MAELSTROM_JAR) test \
		--workload broadcast \
		--bin ./$(BINARY) \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE) \
		--nemesis partition \
		--nemesis-interval 5

# Same as test-clean / test-partition but with per-node event tracing.
# Each node logs every algorithm step to its own stderr (visible at
#   store/broadcast/latest/node-logs/nX.log
# in the Maelstrom UI). Useful for the homework demo.
test-clean-verbose: build
	MAELSTROM_LOG_EVENTS=1 \
	java -jar $(MAELSTROM_JAR) test \
		--workload broadcast \
		--bin ./$(BINARY) \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE)

test-partition-verbose: build
	MAELSTROM_LOG_EVENTS=1 \
	java -jar $(MAELSTROM_JAR) test \
		--workload broadcast \
		--bin ./$(BINARY) \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE) \
		--nemesis partition \
		--nemesis-interval 5

# Crash-stop nemesis: kills one node permanently. This is the fault model
# Algorithm 3.2 is actually specified for (fail-stop, no recovery).
test-kill: build
	java -jar $(MAELSTROM_JAR) test \
		--workload broadcast \
		--bin ./$(BINARY) \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE) \
		--nemesis kill \
		--nemesis-interval 8

test-kill-verbose: build
	MAELSTROM_LOG_EVENTS=1 \
	java -jar $(MAELSTROM_JAR) test \
		--workload broadcast \
		--bin ./$(BINARY) \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE) \
		--nemesis kill \
		--nemesis-interval 8

# Print just the :results block of the most recent Maelstrom run.
results:
	@if [ -L store/broadcast/latest ]; then \
		echo "--- store/broadcast/latest/results.edn ---"; \
		cat store/broadcast/latest/results.edn; \
	else \
		latest=$$(ls -1dt store/broadcast/*/ 2>/dev/null | head -1); \
		if [ -n "$$latest" ]; then \
			echo "--- $${latest}results.edn ---"; \
			cat $${latest}results.edn; \
		else \
			echo "no Maelstrom run found under store/"; exit 1; \
		fi; \
	fi

# Serve the Jepsen/Maelstrom results web UI on http://localhost:$(SERVE_PORT)/
serve:
	java -jar $(MAELSTROM_JAR) serve --port $(SERVE_PORT)

clean:
	rm -rf bin store
