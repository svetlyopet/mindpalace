GO   ?= go
BIN  ?= bin/mp
PKG  ?= ./cmd/mp
SERVE_FLAGS ?=

.PHONY: help out build test test-race test-cover test-e2e serve clean vendor-htmx

help:
	@echo "Mindpalace — common targets:"
	@echo "  make build          Build $(BIN)"
	@echo "  make test           Run go test ./..."
	@echo "  make test-race      Run go test -race ./..."
	@echo "  make test-cover     Run tests and print coverage summary"
	@echo "  make test-e2e       Build mp and run CLI subprocess E2E (separate from test)"
	@echo "  make serve          Build and run mp serve (needs initialized vault)"
	@echo "  make clean          Remove $(BIN)"
	@echo "  make vendor-htmx    Download htmx into web/static/"
	@echo ""
	@echo "Overrides:"
	@echo "  BIN=path            Output binary (default: bin/mp)"
	@echo "  SERVE_FLAGS=...     Passed to mp serve (e.g. --open, --addr HOST:PORT)"
	@echo ""
	@echo "Chrome extension      Load unpacked: extension/ (see extension/README.md)"

out:
	@mkdir -p $(dir $(BIN))

build: out
	$(GO) build -o $(BIN) $(PKG)

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

test-e2e: build
	$(GO) test -tags=e2e ./e2e/...

serve: build
	$(BIN) serve $(SERVE_FLAGS)

clean:
	rm -f $(BIN)

vendor-htmx:
	mkdir -p web/static
	curl -fsSL -o web/static/htmx.min.js https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js

.DEFAULT_GOAL := help
