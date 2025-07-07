package models

// Direction represents which part of a hierarchical relationship to retrieve
type Direction int

const (
	Outgoing Direction = iota // Get children (specs that this spec points to)
	Incoming                  // Get parents (specs that point to this spec)
)

// Node represents a base node in the system with common fields
type Node struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

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

// SpecNode is an alias for Spec to maintain backward compatibility
// TODO: Remove this alias after migrating all references
type SpecNode = Spec

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

// SpecCommitLink represents a link between a spec and a git commit
type SpecCommitLink struct {
	SpecID    string `json:"spec_id"`
	CommitID  string `json:"commit_id"`
	RepoPath  string `json:"repo_path"`
	LinkLabel string `json:"link_label"`
}

// SpecSpecLink represents a hierarchical link between two specifications (forms a DAG)
type SpecSpecLink struct {
	FromSpecID string `json:"from_spec_id"`
	ToSpecID   string `json:"to_spec_id"`
	LinkLabel  string `json:"link_label"` // "child", "fixes", "implements", etc.
}

// ProjectMetadata represents project-level metadata and configuration
type ProjectMetadata struct {
	RootSpecID *string `json:"root_spec_id"` // Nullable foreign key to specs
}

// ErrorType represents different categories of errors in the system
type ErrorType string

const (
	ErrTypeValidation ErrorType = "validation"
	ErrTypeNotFound   ErrorType = "not_found"
	ErrTypeConflict   ErrorType = "conflict"
	ErrTypeStorage    ErrorType = "storage"
	ErrTypeGit        ErrorType = "git"
	ErrTypeSystem     ErrorType = "system"
)

// ZammError represents a structured error with type and context
type ZammError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Cause   error     `json:"-"`
}

// Error implements the error interface
func (e *ZammError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// Unwrap returns the underlying cause of the error
func (e *ZammError) Unwrap() error {
	return e.Cause
}

// NewZammError creates a new ZammError with the given type and message
func NewZammError(errType ErrorType, message string) *ZammError {
	return &ZammError{
		Type:    errType,
		Message: message,
	}
}

// NewZammErrorWithCause creates a new ZammError with an underlying cause
func NewZammErrorWithCause(errType ErrorType, message string, cause error) *ZammError {
	return &ZammError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}
