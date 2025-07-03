# AGENT.md - Development Guide

## Build/Test Commands
- `make build` - Build the binary to bin/zamm
- `make test` - Run all tests with race detection
- `make test-coverage` - Generate HTML coverage report
- `make lint` - Run golangci-lint
- `make fmt` - Format code with go fmt
- `make clean` - Clean build artifacts
- `make dev-setup` - Install dev dependencies
- `make migrations-up` - Apply database migrations
- `go test -v ./internal/storage` - Run tests for specific package

## Architecture
- **CLI Tool**: Go-based CLI for linking specs to Git commits
- **Storage**: SQLite database (~/.zamm/zamm.db) with migration support
- **Structure**: cmd/zamm (CLI entry), internal/{storage,services,models,config,cli}
- **Database**: Two main tables: spec_nodes and spec_commit_links
- **Config**: YAML config at ~/.zamm/config.yaml
- **Models**: SpecNode and SpecCommitLink with UUID primary keys

## Code Style
- **Imports**: stdlib, third-party, local imports with blank line separation
- **Errors**: Custom ZammError type with categorized error handling
- **Naming**: PascalCase exports, camelCase private, descriptive names
- **Structs**: Multiple tags same line `json:"id" db:"id"`
- **SQL**: Multi-line formatting with proper indentation
- **Tests**: Use t.Helper() for test helpers, table-driven tests preferred
