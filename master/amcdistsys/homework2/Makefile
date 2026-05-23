BINARY := bin/node
MAELSTROM_JAR := ../../../../maelstrom/maelstrom/lib/maelstrom.jar
NODE_COUNT ?= 5
TIME_LIMIT ?= 20
RATE ?= 10

ifeq ($(OS),Windows_NT)
	BINARY := bin/node.exe
endif

RESULTS_DIR := results
CLEAN_LOG := $(RESULTS_DIR)/clean.txt
PARTITION_LOG := $(RESULTS_DIR)/partition.txt

.PHONY: all build test test-clean test-partition results clean

all: build

build:
	go build -o $(BINARY) ./cmd/node

test:
	go test ./...

$(RESULTS_DIR):
	mkdir -p $(RESULTS_DIR)

test-clean: build $(RESULTS_DIR)
	java -jar $(MAELSTROM_JAR) test \
		--workload broadcast \
		--bin ./$(BINARY) \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE) 2>&1 | tee $(CLEAN_LOG)

test-partition: build $(RESULTS_DIR)
	java -jar $(MAELSTROM_JAR) test \
		--workload broadcast \
		--bin ./$(BINARY) \
		--node-count $(NODE_COUNT) \
		--time-limit $(TIME_LIMIT) \
		--rate $(RATE) \
		--nemesis partition \
		--nemesis-interval 5 2>&1 | tee $(PARTITION_LOG)

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

clean:
	rm -rf bin store $(RESULTS_DIR)
