GO ?= go
NPM ?= npm

UI_DIR := assets/dashboard-ui
EMBED_DIR := internal/http/webassets/static
BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/dashboard-server
CLI_BIN := $(BIN_DIR)/dashboard

.PHONY: fmt vet test frontend-install frontend-build build-server build-cli build quality

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

frontend-install:
	cd $(UI_DIR) && $(NPM) install

# The React application remains outside the server implementation. This target
# only makes its compiled output available to go:embed for a production build.
frontend-build: frontend-install
	cd $(UI_DIR) && $(NPM) run build
	rm -rf $(EMBED_DIR)/*
	cp -R $(UI_DIR)/dist/. $(EMBED_DIR)/

build-server: frontend-build fmt vet test
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(SERVER_BIN) ./cmd/server

# The CLI only depends on the Go sources and talks to the server over HTTP.
build-cli: fmt vet test
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(CLI_BIN) ./cmd/dashboard

# Production build order: frontend, Go checks, then the embedded server.
build: build-server build-cli

quality: fmt vet test
