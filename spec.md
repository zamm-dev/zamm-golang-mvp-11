# ZAMM MVP: Spec-to-Commit Linking
## Developer-Ready Implementation Specification

### 1. Executive Summary

This MVP implements the core functionality of linking specification nodes to Git commits in the ZAMM system. The focus is on creating, storing, and querying relationships between human-authored requirements (specs) and their corresponding code implementations (commits).

**Primary Goal**: Enable developers to track which commits implement specific requirements and vice versa.

**Scope Limitations**: This MVP excludes the full hierarchical system, LLM integration, and advanced validation features described in the full ZAMM specification.

---

### 2. Core Requirements

#### 2.1 Functional Requirements

**FR-001**: Create and store specification nodes with unique identifiers
**FR-002**: Link specification nodes to Git commit hashes  
**FR-003**: Query specifications by commit hash
**FR-004**: Query commits by specification ID
**FR-005**: Support multiple commits per specification
**FR-006**: Support multiple specifications per commit
**FR-007**: Persist all data to disk for durability
**FR-008**: Provide CLI interface for all operations

#### 2.2 Non-Functional Requirements

**NFR-001**: Response time < 100ms for single record queries
**NFR-002**: Support up to 10,000 specifications and 50,000 commits
**NFR-003**: Data must survive application restarts
**NFR-004**: CLI commands must provide clear success/error feedback

---

### 3. Data Models

#### 3.1 Specification Node
```go
type SpecNode struct {
    ID          string `json:"id" db:"id"`
    StableID    string `json:"stable_id" db:"stable_id"`
    Version     int    `json:"version" db:"version"`
    Title       string `json:"title" db:"title"`
    Content     string `json:"content" db:"content"`
    NodeType    string `json:"node_type" db:"node_type"` // Always "spec" for MVP
}
```

**Field Specifications**:
- `ID`: UUID v4, unique per version (Primary Key)
- `StableID`: UUID v4, constant across all versions of the same logical spec
- `Version`: Integer starting at 1, auto-incremented per StableID
- `Title`: Human-readable title (max 200 characters)
- `Content`: Markdown content (max 50KB)
- `NodeType`: Fixed value "spec" for MVP

#### 3.2 Spec-Commit Link  
```go
type SpecCommitLink struct {
    ID       string `json:"id" db:"id"`
    SpecID   string `json:"spec_id" db:"spec_id"`
    CommitID string `json:"commit_id" db:"commit_id"`
    RepoPath string `json:"repo_path" db:"repo_path"`
    LinkLabel string `json:"link_label" db:"link_label"`
}
```

**Field Specifications**:
- `ID`: UUID v4 (Primary Key)
- `SpecID`: Foreign key to SpecNode.ID
- `CommitID`: Git commit hash (40-character hex string)
- `RepoPath`: Absolute path to Git repository
- `LinkLabel`: Either "implements" or "fixes" for MVP
- Foreign key constraints ensure referential integrity

---

### 4. Architecture Design

#### 4.1 Project Structure
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
├── testdata/
│   └── fixtures/             # Test data
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

#### 4.2 Technology Stack
- **Language**: Go 1.21+
- **Database**: SQLite 3 (embedded, file-based)
- **CLI Framework**: cobra/cli
- **Database Driver**: mattn/go-sqlite3
- **UUID Generation**: google/uuid
- **Git Integration**: go-git/go-git

#### 4.3 Data Storage Strategy
- Single SQLite database file: `~/.zamm/zamm.db`
- Configuration file: `~/.zamm/config.yaml`
- Atomic transactions for all multi-table operations
- Write-ahead logging (WAL) mode for better concurrency

---

### 5. Database Schema

```sql
-- migrations/001_initial.sql
CREATE TABLE spec_nodes (
    id TEXT PRIMARY KEY,
    stable_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    node_type TEXT NOT NULL DEFAULT 'spec',
    UNIQUE(stable_id, version)
);

CREATE INDEX idx_spec_nodes_stable_id ON spec_nodes(stable_id);

CREATE TABLE spec_commit_links (
    id TEXT PRIMARY KEY,
    spec_id TEXT NOT NULL,
    commit_id TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    link_label TEXT NOT NULL DEFAULT 'implements',
    FOREIGN KEY (spec_id) REFERENCES spec_nodes(id) ON DELETE CASCADE,
    UNIQUE(spec_id, commit_id, repo_path)
);

CREATE INDEX idx_links_spec_id ON spec_commit_links(spec_id);
CREATE INDEX idx_links_commit_id ON spec_commit_links(commit_id);
CREATE INDEX idx_links_repo_path ON spec_commit_links(repo_path);
```

---

### 6. API Design

#### 6.1 Storage Interface
```go
type Storage interface {
    // Spec operations
    CreateSpec(spec *SpecNode) error
    GetSpec(id string) (*SpecNode, error)
    GetSpecByStableID(stableID string, version int) (*SpecNode, error)
    GetLatestSpecByStableID(stableID string) (*SpecNode, error)
    ListSpecs() ([]*SpecNode, error)
    UpdateSpec(spec *SpecNode) error
    DeleteSpec(id string) error
    
    // Link operations  
    CreateLink(link *SpecCommitLink) error
    GetLink(id string) (*SpecCommitLink, error)
    GetLinksBySpec(specID string) ([]*SpecCommitLink, error)
    GetLinksByCommit(commitID, repoPath string) ([]*SpecCommitLink, error)
    DeleteLink(id string) error
    
    // Utility
    Close() error
}
```

#### 6.2 Service Layer Interfaces
```go
type SpecService interface {
    CreateSpec(title, content string) (*SpecNode, error)
    GetSpec(id string) (*SpecNode, error)
    UpdateSpec(id, title, content string) (*SpecNode, error)
    ListSpecs() ([]*SpecNode, error)
    DeleteSpec(id string) error
}

type LinkService interface {
    LinkSpecToCommit(specID, commitID, repoPath, label string) (*SpecCommitLink, error)
    GetSpecsForCommit(commitID, repoPath string) ([]*SpecNode, error)
    GetCommitsForSpec(specID string) ([]*SpecCommitLink, error)
    UnlinkSpecFromCommit(specID, commitID, repoPath string) error
}
```

---

### 7. CLI Interface

#### 7.1 Command Structure
```bash
# Spec management
zamm spec create --title "User Authentication" --content "Users must be able to log in"
zamm spec list
zamm spec show <spec-id>
zamm spec update <spec-id> --title "New Title" --content "New content"
zamm spec delete <spec-id>

# Link management  
zamm link create --spec <spec-id> --commit <commit-hash> [--repo <repo-path>] [--type implements|fixes]
zamm link list-by-spec <spec-id>
zamm link list-by-commit <commit-hash> [--repo <repo-path>]
zamm link delete --spec <spec-id> --commit <commit-hash> [--repo <repo-path>]

# Utility commands
zamm init                    # Initialize zamm in current directory
zamm status                 # Show system status and statistics
zamm version               # Show version information
```

#### 7.2 Output Formats
- Default: Human-readable table format
- JSON flag: `--json` for machine-readable output
- Quiet flag: `--quiet` for minimal output

---

### 8. Error Handling Strategy

#### 8.1 Error Categories
```go
type ErrorType string

const (
    ErrTypeValidation    ErrorType = "validation"
    ErrTypeNotFound     ErrorType = "not_found" 
    ErrTypeConflict     ErrorType = "conflict"
    ErrTypeStorage      ErrorType = "storage"
    ErrTypeGit          ErrorType = "git"
    ErrTypeSystem       ErrorType = "system"
)

type ZammError struct {
    Type    ErrorType `json:"type"`
    Message string    `json:"message"`
    Details string    `json:"details,omitempty"`
    Cause   error     `json:"-"`
}
```

#### 8.2 Error Handling Rules
1. **Validation Errors**: Input validation failures (400-level)
2. **Not Found Errors**: Resource doesn't exist (404-level)  
3. **Conflict Errors**: Duplicate keys, constraint violations (409-level)
4. **Storage Errors**: Database/filesystem issues (500-level)
5. **Git Errors**: Git repository or commit issues (422-level)
6. **System Errors**: Unexpected system failures (500-level)

#### 8.3 CLI Error Display
- Exit codes: 0 (success), 1 (user error), 2 (system error)
- Error messages include suggested actions when possible
- Stack traces only in debug mode (`--debug` flag)

---

### 9. Configuration Management

#### 9.1 Configuration File (~/.zamm/config.yaml)
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

#### 9.2 Environment Variables
- `ZAMM_CONFIG_PATH`: Override config file location
- `ZAMM_DB_PATH`: Override database path
- `ZAMM_LOG_LEVEL`: Override log level
- `ZAMM_NO_COLOR`: Disable colored output

---

### 10. Testing Strategy

#### 10.1 Test Coverage Requirements
- **Unit Tests**: 90% coverage for business logic
- **Integration Tests**: All CLI commands with real database  
- **Performance Tests**: Query performance under load
- **Error Tests**: All error conditions and edge cases

#### 10.2 Test Structure
```
internal/
├── storage/
│   ├── sqlite_test.go
│   └── testdata/
├── services/
│   ├── spec_test.go
│   ├── link_test.go
│   └── testdata/
└── cli/
    ├── commands_test.go
    └── testdata/
```

#### 10.3 Test Categories

**Unit Tests**:
- Model validation
- Service layer logic
- Storage interface compliance
- Error handling paths

**Integration Tests**:
- CLI command execution
- Database transactions
- File system operations
- Git repository integration

**Performance Tests**:
- Query response times
- Bulk operation performance
- Memory usage patterns
- Concurrent access scenarios

#### 10.4 Test Data Management
- Use separate test database: `:memory:` for unit tests
- Temporary directories for integration tests
- Fixtures in `testdata/` directories
- Cleanup after each test

---

### 11. Build and Deployment

#### 11.1 Makefile
```makefile
.PHONY: build test clean install dev-setup

build:
	go build -o bin/zamm ./cmd/zamm

test:
	go test -v -race -coverprofile=coverage.out ./...

test-coverage:
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/ coverage.out coverage.html

install:
	go install ./cmd/zamm

dev-setup:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	golangci-lint run

fmt:
	go fmt ./...

migrations-up:
	sqlite3 ~/.zamm/zamm.db < migrations/001_initial.sql
```

#### 11.2 Dependencies
```go
// go.mod
module github.com/zamm-dev/zamm-golang-mvp-11

go 1.21

require (
    github.com/google/uuid v1.3.0
    github.com/mattn/go-sqlite3 v1.14.17
    github.com/spf13/cobra v1.7.0
    github.com/spf13/viper v1.16.0
    gopkg.in/yaml.v3 v3.0.1
)
```

---

### 12. Implementation Phases

#### 12.1 Phase 1: Core Data Layer (Week 1)
- [x] Set up project structure
- [x] Implement data models
- [x] Create SQLite storage implementation
- [x] Write storage layer tests
- [x] Database migrations

#### 12.2 Phase 2: Service Layer (Week 2)  
- [x] Implement SpecService
- [x] Implement LinkService
- [ ] Add business logic validation
- [ ] Write service layer tests
- [ ] Error handling implementation

#### 12.3 Phase 3: CLI Interface (Week 3)
- [ ] CLI framework setup
- [ ] Implement all commands
- [ ] Configuration management
- [ ] CLI integration tests
- [ ] Documentation

#### 12.4 Phase 4: Polish & Performance (Week 4)
- [ ] Performance optimization
- [ ] Error message improvement
- [ ] Code review and refactoring
- [ ] Final testing and validation
- [ ] Release preparation

---

### 13. Success Criteria

#### 13.1 Functional Success
- [ ] All CLI commands work as specified
- [ ] Data persists across application restarts
- [ ] Spec-commit relationships are correctly maintained
- [ ] Git commit validation works properly

#### 13.2 Quality Success  
- [ ] 90%+ test coverage achieved
- [ ] All tests pass consistently
- [ ] Performance requirements met
- [ ] Code passes linting standards

#### 13.3 Usability Success
- [ ] Clear error messages with actionable guidance
- [ ] Intuitive command structure
- [ ] Comprehensive help documentation
- [ ] Easy installation and setup

---

### 14. Future Extension Points

This MVP is designed to easily extend into the full ZAMM system:

1. **Hierarchical Specs**: The `StableID` and versioning system supports spec evolution
2. **Implementation Scopes**: The link system can be extended to support scope nodes
3. **LLM Integration**: Services can be extended with LLM workflow methods
4. **Advanced Querying**: Storage interface can support complex relationship queries
5. **Web Interface**: Services are designed to support both CLI and web frontends

---

### 15. Getting Started

#### 15.1 Initial Setup
```bash
# Clone and build
git clone <repo-url>
cd zamm-mvp
make dev-setup
make build

# Initialize zamm
./bin/zamm init

# Create your first spec
./bin/zamm spec create --title "MVP Feature" --content "Link specs to commits"

# Link it to a commit
./bin/zamm link create --spec <spec-id> --commit $(git rev-parse HEAD)
```

This specification provides everything needed to begin immediate implementation of the ZAMM MVP feature for tying specs to commits.