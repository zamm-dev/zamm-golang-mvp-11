# AGENT.md - Development Guide

## Build/Test Commands
- `make build` - Build the binary to bin/zamm
- `make test` - Run all tests with race detection
- `make test-coverage` - Generate HTML coverage report (after `go test -coverprofile=coverage.out ./...`)
- `make update-golden` - Update golden test files with -update flag
- `make lint` - Run golangci-lint
- `make fmt` - Format code with go fmt
- `make clean` - Clean build artifacts and test cache
- `make dev-setup` - Install dev dependencies (golangci-lint)
- `make migrations-up` - Apply database migrations
- `go test -v ./internal/storage` - Run tests for specific package

## Architecture
- **CLI Tool**: Go-based CLI using Cobra for commands and Bubble Tea for interactive UI
- **Storage**: .zamm/ folder with nodes/<id>.json (specs), spec-links.csv, commit-links.csv, project_metadata.json
- **Data Models**: Spec Nodes with UUIDs, embedded NodeBase pattern, ZammError with categorized types
- **Structure**: cmd/zamm (entry), internal/{cli,storage,services,models,config}
- **Config**: Viper-based config with mapstructure tags, ~/.zamm/config.yaml
- **UI**: Bubble Tea with message-driven architecture, delegates for list rendering

## Code Style & Safety Rules
- **Imports**: stdlib, third-party, local with blank line separation
- **Errors**: ZammError with types (Validation, NotFound, Conflict, Storage, Git, System)
- **Naming**: Interfaces end with service name, constructors use New prefix, messages end with Msg
- **Tests**: t.Helper() in helpers, setupTestService pattern, t.TempDir() for isolation
- **Tea Models**: NEVER use pointer receivers for Update() methods (interface requirement)
- **SAFETY**: NEVER run `./bin/zamm` commands on root directory - only in test directories
- **Build Rule**: Always run `make` after code changes to ensure compilation
