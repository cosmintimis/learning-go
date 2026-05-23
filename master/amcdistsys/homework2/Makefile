BINARY := bin/node
MAELSTROM_JAR := ../../../../maelstrom/maelstrom/lib/maelstrom.jar
NODE_COUNT ?= 5
TIME_LIMIT ?= 20
RATE ?= 10

ifeq ($(OS),Windows_NT)
	BINARY := bin/node.exe
endif

.PHONY: all build test test-clean test-partition clean

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

clean:
	rm -rf bin store
