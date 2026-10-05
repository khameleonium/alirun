SHELL := /bin/bash
PATH := /usr/local/go/bin:$(PATH)
export PATH

BINARY_NAME := alirun
BUILD_DIR := bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X 'alirun/pkg/updater.Version=$(VERSION)' \
	-X 'alirun/pkg/updater.Commit=$(COMMIT)' \
	-X 'alirun/pkg/updater.BuildDate=$(BUILD_DATE)'

.PHONY: all build clean test install install-system update-deps fmt lint

all: build

build:
	@mkdir -p $(BUILD_DIR)
	@echo "🔨 Building $(BINARY_NAME) ($(VERSION))..."
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/alirun
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

test:
	@echo "🧪 Running tests..."
	go test -v ./...

fmt:
	@echo "🖌️ Formatting code..."
	go fmt ./...

update-deps:
	@echo "🔄 Updating Go dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✅ Dependencies updated!"

install: build
	@echo "📦 Installing to ~/.local/bin/$(BINARY_NAME)..."
	@mkdir -p $(HOME)/.local/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)
	@echo "✅ Installed $(BINARY_NAME) to ~/.local/bin/$(BINARY_NAME)"

install-system: build
	@echo "📦 Installing to /usr/local/bin/$(BINARY_NAME) (requires sudo)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed $(BINARY_NAME) system-wide!"

clean:
	@echo "🧹 Cleaning up..."
	@rm -rf $(BUILD_DIR)
