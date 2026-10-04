GO ?= go
NPM ?= npm
INSTALL ?= install
SYSTEMCTL ?= systemctl

UI_DIR := assets/dashboard-ui
EMBED_DIR := internal/http/webassets/static
BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/dashboard-server
CLI_BIN := $(BIN_DIR)/dashboard

# Installation is deliberately scoped to the current user. The service uses
# the same HOME, SQLite database and OpenCode state as the interactive CLI.
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
SYSTEMD_USER_DIR ?= $(HOME)/.config/systemd/user
SERVICE_NAME ?= dashboard.locals.service
INSTALL_SCRIPT := scripts/install.sh
SERVICE_TEMPLATE := contrib/systemd/dashboard.locals.service.in

.PHONY: fmt vet test frontend-install frontend-build build-server build-cli build install quality

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

# Install both executables and register the user-level systemd service. The
# script is intentionally not run with sudo: the daemon must run as the user
# who owns the application data and the OpenCode service state.
install: build
	$(SHELL) $(INSTALL_SCRIPT) \
		--server-bin "$(SERVER_BIN)" \
		--cli-bin "$(CLI_BIN)" \
		--bindir "$(BINDIR)" \
		--systemd-user-dir "$(SYSTEMD_USER_DIR)" \
		--service-name "$(SERVICE_NAME)" \
		--service-template "$(SERVICE_TEMPLATE)" \
		--install "$(INSTALL)" \
		--systemctl "$(SYSTEMCTL)"

quality: fmt vet test
