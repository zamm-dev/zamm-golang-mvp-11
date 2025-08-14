package models

import (
	"github.com/google/uuid"
)

// Direction represents which part of a hierarchical relationship to retrieve
type Direction int

const (
	Outgoing Direction = iota // Get children (specs that this spec points to)
	Incoming                  // Get parents (specs that point to this spec)
)

// NodeBase represents the base structure for all nodes in the system
type NodeBase struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Type    string  `json:"type"`
	Slug    *string `json:"slug,omitempty"`
}

// Node interface that all node types must implement
type Node interface {
	GetID() string
	GetTitle() string
	GetContent() string
	GetType() string
	GetSlug() *string
	SetTitle(string)
	SetContent(string)
	SetSlug(*string)
}

// Implement Node interface for NodeBase
func (n *NodeBase) GetID() string      { return n.ID }
func (n *NodeBase) GetTitle() string   { return n.Title }
func (n *NodeBase) GetContent() string { return n.Content }
func (n *NodeBase) GetType() string    { return n.Type }
func (n *NodeBase) GetSlug() *string   { return n.Slug }
func (n *NodeBase) SetTitle(title string) {
	n.Title = title
}
func (n *NodeBase) SetContent(content string) {
	n.Content = content
}
func (n *NodeBase) SetSlug(slug *string) {
	n.Slug = slug
}

// Spec represents a specification node in the system
type Spec struct {
	NodeBase
	// Add any additional fields from SpecNode here if needed
}

// NewSpec creates a new Spec with the type field set
func NewSpec(title, content string) *Spec {
	return &Spec{
		NodeBase: NodeBase{
			ID:      uuid.New().String(),
			Title:   title,
			Content: content,
			Type:    "specification",
		},
	}
}

// Project represents a project node in the system
type Project struct {
	NodeBase
}

// NewProject creates a new Project with the type field set
func NewProject(title, content string) *Project {
	return &Project{
		NodeBase: NodeBase{
			ID:      uuid.New().String(),
			Title:   title,
			Content: content,
			Type:    "project",
		},
	}
}

// Implementation represents an implementation node in the system
type Implementation struct {
	NodeBase
	RepoURL    *string `json:"repo_url,omitempty"`    // Optional repository URL
	Branch     *string `json:"branch,omitempty"`      // Optional branch name
	FolderPath *string `json:"folder_path,omitempty"` // Optional folder path within the repo
}

// NewImplementation creates a new Implementation with the type field set
func NewImplementation(title, content string) *Implementation {
	return &Implementation{
		NodeBase: NodeBase{
			ID:      uuid.New().String(),
			Title:   title,
			Content: content,
			Type:    "implementation",
		},
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
