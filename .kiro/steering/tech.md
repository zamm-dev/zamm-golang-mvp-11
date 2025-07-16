# Technology Stack

## Core Technologies

- **Language**: Go 1.23.4+ (with Go 1.24.4 toolchain)
- **CLI Framework**: Cobra for command-line interface
- **Configuration**: Viper for configuration management
- **TUI Framework**: Bubble Tea ecosystem for interactive terminal UI
  - `bubbletea` - Core TUI framework
  - `bubbles` - Pre-built UI components (tables, text input, viewport)
  - `lipgloss` - Styling and layout
  - `bubbletea-overlay` - Modal/overlay components
- **Storage**: File-based JSON with CSV for relationships
- **Testing**: Go standard testing with `teatest` for TUI testing
- **UUID Generation**: Google UUID library

## Build System

The project uses a Makefile for build automation:

### Common Commands

```bash
# Build the project
make build

# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Clean build artifacts
make clean

# Install binary globally
make install

# Development setup (downloads deps, installs linters)
make dev-setup

# Code formatting
make fmt

# Lint code
make lint
```

### Development Workflow

1. Use `make dev-setup` for initial environment setup
2. Build with `make build` - outputs to `bin/zamm`
3. Test with `make test` for unit tests
4. Use `make fmt` and `make lint` before commits

## Code Quality Tools

- **Linter**: golangci-lint (installed via dev-setup)
- **Formatter**: Standard Go fmt
- **Testing**: Go race detector enabled (`go test -race`)
- **Coverage**: HTML coverage reports via `make test-coverage`