# Makefile for qualhook

# Variables
BINARY_NAME=qualhook
BINARY_DIR=./bin
MODULE=$(shell go list -m)
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags="-X 'main.Version=$(VERSION)'"

# Build variables
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOGET=$(GOCMD) get
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Tools
GOLANGCI_LINT_VERSION=v1.61.0
GOLANGCI_LINT=$(shell which golangci-lint 2> /dev/null)

.PHONY: all
all: clean lint test build

.PHONY: build
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/qualhook

.PHONY: build-all
build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BINARY_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/qualhook
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/qualhook
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/qualhook
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/qualhook

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: lint
lint: check-golangci-lint ## Run golangci-lint
	@echo "Running linters..."
	$(GOLANGCI_LINT) run --config .golangci.yml

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	$(GOCMD) fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

.PHONY: tidy
tidy: ## Tidy go modules
	@echo "Tidying modules..."
	$(GOMOD) tidy

.PHONY: download
download: ## Download go modules
	@echo "Downloading modules..."
	$(GOMOD) download

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.out coverage.html

.PHONY: install
install: build ## Install the binary
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BINARY_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

.PHONY: uninstall
uninstall: ## Uninstall the binary
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)

.PHONY: run
run: build ## Build and run the binary
	@echo "Running $(BINARY_NAME)..."
	$(BINARY_DIR)/$(BINARY_NAME)

.PHONY: check-golangci-lint
check-golangci-lint:
ifndef GOLANGCI_LINT
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(eval GOLANGCI_LINT=$(shell which golangci-lint))
endif

.PHONY: tools
tools: check-golangci-lint ## Install development tools
	@echo "Installing development tools..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/segmentio/golines@latest
	@echo "Tools installed successfully"

.PHONY: pre-commit
pre-commit: fmt vet lint ## Run pre-commit checks
	@echo "Pre-commit checks passed!"

.PHONY: ci
ci: clean download lint test build ## Run CI pipeline

.PHONY: help
help: ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help