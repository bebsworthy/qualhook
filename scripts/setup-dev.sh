#!/bin/bash
# setup-dev.sh - Development environment setup script for qualhook

set -e

echo "Setting up development environment for qualhook..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "Error: Go version $REQUIRED_VERSION or later is required. Current version: $GO_VERSION"
    exit 1
fi

echo "✓ Go version $GO_VERSION detected"

# Install development tools
echo "Installing development tools..."
make tools

# Install pre-commit if available
if command -v pre-commit &> /dev/null; then
    echo "Installing pre-commit hooks..."
    pre-commit install
    echo "✓ Pre-commit hooks installed"
else
    echo "⚠ pre-commit not found. Install it with: pip install pre-commit"
    echo "  Then run: pre-commit install"
fi

# Download dependencies
echo "Downloading Go dependencies..."
make download

# Run initial build
echo "Running initial build..."
make build

# Run tests
echo "Running tests..."
make test

echo ""
echo "✅ Development environment setup complete!"
echo ""
echo "Available make targets:"
make help