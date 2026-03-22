# --- OS detection ---
ifeq ($(OS),Windows_NT)
    BINARY = bcastnode.exe
    RM = del /f /q
else
    BINARY = bcastnode
    RM = rm -f
endif

.PHONY: test test-short test-verbose build clean run

## Run all tests
test:
	go test ./...

## Run fast tests only (skip integration)
test-short:
	go test -short ./...

## Run all tests with verbose output
test-verbose:
	go test -v ./...

## Build the binary
build:
	go build -o $(BINARY) ./cmd/bcastnode

## Remove build artifacts
clean:
	$(RM) $(BINARY)

## Run nodes (usage: make run CONFIG=config.txt FIRST=0 LAST=2)
run: build
ifeq ($(OS),Windows_NT)
	powershell -File scripts/startnodes.ps1 $(CONFIG) $(FIRST) $(LAST)
else
	bash scripts/startnodes.sh $(CONFIG) $(FIRST) $(LAST)
endif
