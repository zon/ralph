# Read version from VERSION file
VERSION := $(shell cat VERSION)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build settings
BINARY_NAME := ralph
MAIN_PATH := ./cmd/ralph
INSTALL_PATH := $(GOPATH)/bin

# ldflags to inject version information
LDFLAGS := -X main.Version=$(VERSION) -X main.Date=$(BUILD_DATE) -X github.com/zon/ralph/internal/workflow.DefaultContainerVersion=$(VERSION)

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the ralph binary with version information
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: ./$(BINARY_NAME)"

.PHONY: install
install: ## Install ralph to GOPATH/bin with version information
	@echo "Installing $(BINARY_NAME) v$(VERSION) to $(INSTALL_PATH)..."
	go install -ldflags "$(LDFLAGS)" $(MAIN_PATH)
	@echo "Installation complete: $(INSTALL_PATH)/$(BINARY_NAME)"

.PHONY: version
version: ## Display version information
	@echo "Version: $(VERSION)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Git Commit: $(GIT_COMMIT)"

.PHONY: clean
clean: ## Remove built binaries
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

.PHONY: container-build
container-build: ## Build container image
	@REPOSITORY="ghcr.io/zon/ralph"; \
	IMAGE="$$REPOSITORY:$(VERSION)"; \
	echo "Building container $$IMAGE..."; \
	podman build -t "$$IMAGE" -f Containerfile .

.PHONY: push
push: ## Push container image to registry
	@./scripts/push-image.sh
