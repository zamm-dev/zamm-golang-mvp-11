# Implementation Node Design Specification

## Overview
Add support for tracking implementations of program specifications through new node types: `Implementation` and `SpecImplementationLink`.

## Design Requirements

### Base Node Structure
Create a base `Node` struct that contains common fields, then embed it in existing and new types:

```go
// Node represents a base node in the system with common fields
type Node struct {
    ID      string `json:"id"`
    Title   string `json:"title"`
    Content string `json:"content"`
    Type    string `json:"type"`
}
```

### Type Refactoring
Refactor existing `SpecNode` to embed `Node`, and rename it to `Spec` for clarity:

```go
// Spec represents a specification node in the system
type Spec struct {
    Node
}

// NewSpec creates a new Spec with the type field set
func NewSpec(id, title, content string) *Spec {
    return &Spec{
        Node: Node{
            ID:      id,
            Title:   title,
            Content: content,
            Type:    "Spec",
        },
    }
}
```

### New Implementation Type
Create `Implementation` type to track implementation environments:

```go
// Implementation represents an implementation node with additional context
type Implementation struct {
    Node
    RepoPath   string `json:"repo_path"`   // Repository identifier
    FolderPath string `json:"folder_path"` // Relative path within repo
}

// NewImplementation creates a new Implementation with the type field set
func NewImplementation(id, title, content, repoPath, folderPath string) *Implementation {
    return &Implementation{
        Node: Node{
            ID:      id,
            Title:   title,
            Content: content,
            Type:    "Implementation",
        },
        RepoPath:   repoPath,
        FolderPath: folderPath,
    }
}
```

### Implementation Link Type
Create `SpecImplementationLink` as a first-class entity (not just a relationship row):

```go
// SpecImplementationLink represents a link between a spec and its implementation
type SpecImplementationLink struct {
    Node
    SpecID               string            `json:"spec_id"`
    ImplementationID     string            `json:"implementation_id"`
    FilePaths            []string          `json:"file_paths,omitempty"`        // Files that implement this spec
    FilePathSummaries    map[string]string `json:"file_path_summaries,omitempty"` // File path -> summary mapping
    CommitIDs            []string          `json:"commit_ids,omitempty"`        // Commits related to this implementation
}

// NewSpecImplementationLink creates a new SpecImplementationLink with the type field set
func NewSpecImplementationLink(id, title, content, specID, implementationID string) *SpecImplementationLink {
    return &SpecImplementationLink{
        Node: Node{
            ID:      id,
            Title:   title,
            Content: content,
            Type:    "SpecImplementationLink",
        },
        SpecID:           specID,
        ImplementationID: implementationID,
    }
}
```

## Key Design Principles

1. **Embedded Structs**: Use anonymous embedding (`Node`) to promote fields to top-level JSON
2. **First-Class Links**: `SpecImplementationLink` is a full entity with its own ID, not just a relationship
3. **No Redundant Fields**: 
   - No `LinkLabel` (link type is implicit)
   - No `Notes` field (use `Content` from embedded `Node`)
4. **File Mapping**: Support both file lists and AI-generated summaries per file
5. **Implementation Status**: Tracked by presence/absence of links, not explicit status fields
6. **Consistency**: Follow existing patterns from `SpecCommitLink` and `SpecSpecLink`

## Expected JSON Structure

### Spec
```json
{
  "id": "spec-123",
  "title": "User Authentication",
  "content": "Specification for user authentication system",
  "type": "Spec"
}
```

### Implementation
```json
{
  "id": "impl-456",
  "title": "Python Backend",
  "content": "Backend implementation description",
  "type": "Implementation",
  "repo_path": "github.com/user/repo",
  "folder_path": "backend/"
}
```

### SpecImplementationLink
```json
{
  "id": "link-789",
  "title": "User Authentication Implementation",
  "content": "Links auth spec to Python backend implementation",
  "type": "SpecImplementationLink",
  "spec_id": "spec-123",
  "implementation_id": "impl-456",
  "file_paths": ["auth/models.py", "auth/views.py"],
  "file_path_summaries": {
    "auth/models.py": "User model with authentication fields",
    "auth/views.py": "Login/logout API endpoints"
  },
  "commit_ids": ["abc123", "def456"]
}
```

## Implementation Tasks

- [x] Add `Node` base struct to models.go
- [x] Refactor existing `SpecNode` to embed `Node` and be renamed to `Spec`
- [x] Add new `Implementation` struct
- [x] Add new `SpecImplementationLink` struct
- [x] Update any existing code that directly references `SpecNode` fields
- [ ] Do datastore file migration:
   - store new types in the "type" field of the existing JSON
   - rename `.zamm/specs` to `./zamm/nodes` because we now have a more base `Node` type than `SpecNode`
   - change all existing links between nodes and commits to be `SpecImplementationLink` nodes. The CSV of `commit-links` should now include a column for the `SpecImplementationLink` ID, and the `SpecImplementationLink` JSON file should live inside the same folder as the rest of the nodes.