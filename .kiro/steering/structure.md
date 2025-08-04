# Project Structure

## Directory Organization

```
zamm-mvp/
├── cmd/zamm/           # Application entry point
├── internal/           # Private application code
│   ├── cli/            # Command-line interface layer
│   │   └── interactive/ # Bubble Tea TUI components
│   ├── config/         # Configuration management
│   ├── models/         # Data structures and domain models
│   ├── services/       # Business logic layer
│   └── storage/        # Data persistence layer
├── .zamm/              # Project metadata and data storage
├── bin/                # Built binaries (generated)
└── Makefile           # Build automation
```

## Architecture Layers

### CLI Layer (`internal/cli/`)
- **Cobra Commands**: Root command and subcommands for spec/link management
- **Interactive Mode**: Bubble Tea-based TUI in `interactive/` subdirectory
- **Common Components**: Reusable TUI components in `interactive/common/`
- **Specialized Views**: Feature-specific views like `speclistview/`

### Services Layer (`internal/services/`)
- **SpecService**: Specification CRUD and hierarchical operations
- **LinkService**: Git commit linking operations
- Clean interfaces with error handling using custom `ZammError` types

### Storage Layer (`internal/storage/`)
- **FileStorage**: JSON-based persistence with CSV for relationships
- **Interfaces**: Abstract storage contracts for testability
- Data stored in `.zamm/` directory structure

### Models Layer (`internal/models/`)
- **Domain Models**: `Spec`, `SpecCommitLink`, `SpecSpecLink`
- **Error Types**: Structured error handling with `ZammError`
- **Interfaces**: `Node` interface for extensible node types

## File Naming Conventions

- **Go Files**: Snake_case for multi-word files (`spec_detail_view.go`)
- **Test Files**: `*_test.go` suffix with golden files in `testdata/`
- **Interfaces**: Defined in separate `interfaces.go` files
- **Messages**: Bubble Tea messages in dedicated `messages.go` files

## Data Storage Structure

```
.zamm/
├── nodes/              # Individual spec JSON files (UUID named)
├── commit-links.csv    # Spec-to-commit relationships
├── spec-links.csv      # Spec-to-spec hierarchical links
└── project_metadata.json # Project-level configuration
```

## Testing Patterns

- **Unit Tests**: Standard Go testing with table-driven tests
- **TUI Tests**: `teatest` for Bubble Tea component testing
- **Golden Files**: Expected output stored in `testdata/` directories
- **Test Data**: Isolated test fixtures in component-specific `testdata/`