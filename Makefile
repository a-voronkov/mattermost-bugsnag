# Mattermost Bugsnag Plugin Makefile

PLUGIN_ID := com.mattermost.bugsnag
PLUGIN_VERSION := 0.0.1
BUNDLE_NAME := $(PLUGIN_ID)-$(PLUGIN_VERSION).tar.gz

GO := go
NPM := npm

SERVER_DIR := server
WEBAPP_DIR := webapp
DIST_DIR := dist

# Go build settings
GOOS_LINUX := linux
GOOS_DARWIN := darwin
GOOS_WINDOWS := windows
GOARCH := amd64
GOARCH_ARM := arm64

.PHONY: all clean server webapp bundle test lint check-style

## all: Build everything and create plugin bundle
all: clean server webapp bundle

## clean: Remove build artifacts
clean:
	rm -rf $(DIST_DIR)
	rm -rf $(SERVER_DIR)/dist
	rm -rf $(WEBAPP_DIR)/dist
	rm -f $(BUNDLE_NAME)

## server: Build server binaries for all platforms
server: server-linux server-darwin-amd64 server-darwin-arm64 server-windows

server-linux:
	@echo "Building server for linux-amd64..."
	cd $(SERVER_DIR) && GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) $(GO) build -o dist/plugin-linux-amd64 .

server-darwin-amd64:
	@echo "Building server for darwin-amd64..."
	cd $(SERVER_DIR) && GOOS=$(GOOS_DARWIN) GOARCH=$(GOARCH) $(GO) build -o dist/plugin-darwin-amd64 .

server-darwin-arm64:
	@echo "Building server for darwin-arm64..."
	cd $(SERVER_DIR) && GOOS=$(GOOS_DARWIN) GOARCH=$(GOARCH_ARM) $(GO) build -o dist/plugin-darwin-arm64 .

server-windows:
	@echo "Building server for windows-amd64..."
	cd $(SERVER_DIR) && GOOS=$(GOOS_WINDOWS) GOARCH=$(GOARCH) $(GO) build -o dist/plugin-windows-amd64.exe .

## webapp: Build webapp bundle
webapp:
	@echo "Building webapp..."
	cd $(WEBAPP_DIR) && $(NPM) install && $(NPM) run build

## bundle: Create plugin tar.gz bundle
bundle:
	@echo "Creating plugin bundle..."
	mkdir -p $(DIST_DIR)/$(PLUGIN_ID)
	cp plugin.json $(DIST_DIR)/$(PLUGIN_ID)/
	cp -r $(SERVER_DIR)/dist $(DIST_DIR)/$(PLUGIN_ID)/server/
	mkdir -p $(DIST_DIR)/$(PLUGIN_ID)/webapp/dist
	cp $(WEBAPP_DIR)/dist/main.js $(DIST_DIR)/$(PLUGIN_ID)/webapp/dist/
	cd $(DIST_DIR) && tar -czvf ../$(BUNDLE_NAME) $(PLUGIN_ID)
	@echo "Bundle created: $(BUNDLE_NAME)"

## test: Run all tests
test:
	cd $(SERVER_DIR) && $(GO) test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	cd $(SERVER_DIR) && $(GO) test -v -coverprofile=coverage.out ./...
	cd $(SERVER_DIR) && $(GO) tool cover -html=coverage.out -o coverage.html

## lint: Run linters
lint:
	cd $(SERVER_DIR) && golangci-lint run ./...

## check-style: Check code formatting
check-style:
	cd $(SERVER_DIR) && gofmt -d .

## fmt: Format Go code
fmt:
	cd $(SERVER_DIR) && gofmt -w .

## tidy: Tidy Go modules
tidy:
	cd $(SERVER_DIR) && $(GO) mod tidy

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/## /  /'

