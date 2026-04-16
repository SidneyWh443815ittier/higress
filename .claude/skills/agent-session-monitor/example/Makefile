# Makefile for agent-session-monitor example

BINARY := session_monitor
GO := go
GOFLAGS := -ldflags "-s -w"

.PHONY: all build test clean run lint fmt

## all: build and test
all: build test

## build: compile the session monitor binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY) .

## test: run unit tests with coverage
test:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

## test-integration: run integration tests using demo log files
test-integration: build
	@echo "=== Testing v1 log format ==="
	./$(BINARY) -config config.yaml -log test_access.log
	@echo ""
	@echo "=== Testing v2 log format ==="
	./$(BINARY) -config config.yaml -log test_access_v2.log

## run: run with default config and test log
run: build
	./$(BINARY) -config config.yaml -log test_access.log

## fmt: format Go source files
fmt:
	$(GO) fmt ./...

## lint: run golangci-lint if available
lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

## clean: remove build artifacts
clean:
	rm -f $(BINARY) coverage.out

## demo: run the bash demo script
demo:
	bash demo.sh

## demo-v2: run the v2 bash demo script
demo-v2:
	bash demo_v2.sh

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
