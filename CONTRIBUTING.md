# Contributing to Qualhook

Thank you for your interest in contributing to Qualhook! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Make
- Git
- (Optional) pre-commit for automatic code formatting

### Initial Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/boyd/qualhook.git
   cd qualhook
   ```

2. Run the development setup script:
   ```bash
   ./scripts/setup-dev.sh
   ```

   This will:
   - Verify Go installation
   - Install development tools
   - Set up pre-commit hooks (if available)
   - Download dependencies
   - Run initial build and tests

### Manual Setup

If you prefer manual setup:

```bash
# Install development tools
make tools

# Download dependencies
make download

# Install pre-commit hooks (optional)
pip install pre-commit
pre-commit install

# Build the project
make build

# Run tests
make test
```

## Development Workflow

### Available Make Commands

Run `make help` to see all available commands:

- `make build` - Build the binary
- `make test` - Run tests
- `make lint` - Run golangci-lint
- `make fmt` - Format code
- `make pre-commit` - Run pre-commit checks
- `make clean` - Clean build artifacts

### Code Style

We use the following tools to maintain code quality:

1. **gofmt** - Standard Go formatting
2. **goimports** - Organize imports
3. **golangci-lint** - Comprehensive linting with multiple linters

All code must pass linting before being merged. Run `make lint` to check your code.

### Pre-commit Hooks

If you have pre-commit installed, hooks will automatically:
- Format your code
- Run linters
- Check for common issues

To run pre-commit manually: `pre-commit run --all-files`

### Testing

- Write tests for all new functionality
- Maintain test coverage above 80%
- Run `make test` before submitting PRs
- Use `make test-coverage` to generate coverage reports

### Commit Messages

Follow conventional commit format:
- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `test:` - Test additions or modifications
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

Examples:
```
feat: add support for Python project detection
fix: correct timeout handling in command executor
docs: update configuration examples
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Make your changes following the coding standards
4. Add tests for new functionality
5. Run `make pre-commit` to ensure code quality
6. Commit your changes with descriptive messages
7. Push to your fork: `git push origin feature/your-feature`
8. Create a Pull Request

### PR Requirements

- All tests must pass
- Code must pass linting
- Maintain or improve test coverage
- Update documentation if needed
- Include a clear description of changes

## Continuous Integration

GitHub Actions runs on all PRs and includes:
- Linting with golangci-lint
- Unit tests on multiple OS/architectures
- Security scanning with gosec
- Test coverage reporting

## Release Process

Releases are automated via GitHub Actions when a tag is pushed:

```bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

This will:
- Run all tests
- Build binaries for multiple platforms
- Create a GitHub release with artifacts

## Getting Help

- Open an issue for bugs or feature requests
- Join discussions in GitHub Discussions
- Check existing issues before creating new ones

## Code of Conduct

Please be respectful and inclusive in all interactions. We strive to maintain a welcoming environment for all contributors.