# ZAMM MVP: Spec-to-Commit Linking

A Go-based CLI tool for linking specification nodes to Git commits, enabling traceability between requirements and implementation.

## Features

- Create and manage specification nodes with unique identifiers
- Link specifications to Git commit hashes
- Query specifications by commit hash and vice versa
- Support for multiple commits per specification and multiple specifications per commit
- Persistent SQLite storage
- CLI interface with JSON output support

## Installation

### Prerequisites

- Go 1.21 or higher
- SQLite 3 (for manual database operations)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourorg/zamm-mvp
cd zamm-mvp

# Set up development environment
make dev-setup

# Build the binary
make build

# Install globally
make install
```

## Quick Start

### Initialize ZAMM

```bash
# Initialize zamm in your project
./bin/zamm init

# Check status
./bin/zamm status
```

### Create a Specification

```bash
# Create a new spec
./bin/zamm spec create --title "User Authentication" --content "Users must be able to log in with username and password"

# List all specs
./bin/zamm spec list

# Show spec details
./bin/zamm spec show <spec-id>
```

### Link Specs to Commits

```bash
# Link a spec to the current commit
./bin/zamm link create --spec <spec-id> --commit $(git rev-parse HEAD)

# Link with explicit repository path
./bin/zamm link create --spec <spec-id> --commit <commit-hash> --repo /path/to/repo

# List commits for a spec
./bin/zamm link list-by-spec <spec-id>

# List specs for a commit
./bin/zamm link list-by-commit <commit-hash>
```

## Usage

### Specification Management

```bash
# Create a specification
zamm spec create --title "Feature Title" --content "Detailed description"

# List all specifications
zamm spec list

# Show a specific specification
zamm spec show <spec-id>

# Update a specification
zamm spec update <spec-id> --title "New Title" --content "New content"

# Delete a specification
zamm spec delete <spec-id>
```

### Link Management

```bash
# Create a link between spec and commit
zamm link create --spec <spec-id> --commit <commit-hash> [--repo <repo-path>] [--type implements|references]

# List commits linked to a spec
zamm link list-by-spec <spec-id>

# List specs linked to a commit
zamm link list-by-commit <commit-hash> [--repo <repo-path>]

# Delete a link
zamm link delete --spec <spec-id> --commit <commit-hash> [--repo <repo-path>]
```

### Utility Commands

```bash
# Initialize zamm
zamm init

# Show system status
zamm status

# Show version
zamm version
```

### Output Formats

All commands support JSON output for programmatic use:

```bash
# JSON output
zamm spec list --json

# Quiet mode (minimal output)
zamm spec create --title "Test" --content "Test content" --quiet
```

## Configuration

ZAMM uses a configuration file located at `~/.zamm/config.yaml`:

```yaml
database:
  path: ~/.zamm/zamm.db
  timeout: 30s

git:
  default_repo: .

logging:
  level: info
  file: ~/.zamm/logs/zamm.log

cli:
  output_format: table
  color: auto
```

### Environment Variables

- `ZAMM_CONFIG_PATH`: Override config file location
- `ZAMM_DB_PATH`: Override database path
- `ZAMM_LOG_LEVEL`: Override log level
- `ZAMM_NO_COLOR`: Disable colored output

## Database Schema

ZAMM uses SQLite with the following schema:

### Specifications Table

```sql
CREATE TABLE spec_nodes (
    id TEXT PRIMARY KEY,
    stable_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    node_type TEXT NOT NULL DEFAULT 'spec',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(stable_id, version)
);
```

### Links Table

```sql
CREATE TABLE spec_commit_links (
    id TEXT PRIMARY KEY,
    spec_id TEXT NOT NULL,
    commit_id TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    link_type TEXT NOT NULL DEFAULT 'implements',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (spec_id) REFERENCES spec_nodes(id) ON DELETE CASCADE,
    UNIQUE(spec_id, commit_id, repo_path)
);
```

## Development

### Building

```bash
# Build the project
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Lint code
make lint

# Format code
make fmt
```

### Project Structure

```
zamm-mvp/
├── cmd/
│   └── zamm/
│       └── main.go           # CLI entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── storage/
│   │   ├── sqlite.go         # SQLite implementation
│   │   └── interfaces.go     # Storage interfaces
│   ├── models/
│   │   └── models.go         # Data structures
│   ├── services/
│   │   ├── spec.go           # Spec management
│   │   └── link.go           # Link management
│   └── cli/
│       └── commands.go       # CLI command handlers
├── migrations/
│   └── 001_initial.sql       # Database schema
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### Testing

Run the test suite:

```bash
make test
```

Generate coverage report:

```bash
make test-coverage
open coverage.html
```

### Database Migrations

To manually apply database migrations:

```bash
make migrations-up
```

Or directly with sqlite3:

```bash
sqlite3 ~/.zamm/zamm.db < migrations/001_initial.sql
```

## Error Handling

ZAMM provides structured error handling with the following error types:

- **Validation**: Input validation failures
- **Not Found**: Resource doesn't exist
- **Conflict**: Duplicate keys, constraint violations
- **Storage**: Database/filesystem issues
- **Git**: Git repository or commit issues
- **System**: Unexpected system failures

Exit codes:
- `0`: Success
- `1`: User error (validation, not found, conflict, git)
- `2`: System error (storage, system)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

[Add your license information here]

## Roadmap

This MVP provides the foundation for the full ZAMM system. Future extensions include:

- Hierarchical specifications with versioning
- Implementation scopes and architectural branches
- LLM integration for automated code generation
- Advanced querying and relationship management
- Web interface
- Advanced validation and change management