package models

import (
	"time"
)

// Direction represents which part of a hierarchical relationship to retrieve
type Direction int

const (
	Outgoing Direction = iota // Get children (specs that this spec points to)
	Incoming                  // Get parents (specs that point to this spec)
)

// SpecNode represents a specification node in the system
type SpecNode struct {
	ID        string    `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// SpecCommitLink represents a link between a spec and a git commit
type SpecCommitLink struct {
	ID        string    `json:"id" db:"id"`
	SpecID    string    `json:"spec_id" db:"spec_id"`
	CommitID  string    `json:"commit_id" db:"commit_id"`
	RepoPath  string    `json:"repo_path" db:"repo_path"`
	LinkType  string    `json:"link_type" db:"link_type"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SpecSpecLink represents a hierarchical link between two specifications (forms a DAG)
type SpecSpecLink struct {
	ID         string    `json:"id" db:"id"`
	FromSpecID string    `json:"from_spec_id" db:"from_spec_id"`
	ToSpecID   string    `json:"to_spec_id" db:"to_spec_id"`
	LinkType   string    `json:"link_type" db:"link_type"` // "child", "fixes", "implements", etc.
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ProjectMetadata represents project-level metadata and configuration
type ProjectMetadata struct {
	ID         int       `json:"id" db:"id"`
	RootSpecID *string   `json:"root_spec_id" db:"root_spec_id"` // Nullable foreign key to specs
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
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
