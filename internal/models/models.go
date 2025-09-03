package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Direction represents which part of a hierarchical relationship to retrieve
type Direction int

const (
	Outgoing Direction = iota // Get children (specs that this spec points to)
	Incoming                  // Get parents (specs that point to this spec)
)

// to keep NodeBase fields private https://stackoverflow.com/a/11129633
type NodeBaseJSON struct {
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Content       string      `json:"content"`
	Type          string      `json:"type"`
	Slug          *string     `json:"slug,omitempty"`
	ChildGrouping *ChildGroup `json:"child_grouping,omitempty"`
}

// NodeBase represents the base structure for all nodes in the system
type NodeBase struct {
	id            string
	title         string
	content       string
	nodeType      string
	slug          *string
	childGrouping *ChildGroup
}

func (n *NodeBase) asBaseJsonStruct() NodeBaseJSON {
	return NodeBaseJSON{
		ID:            n.id,
		Title:         n.title,
		Content:       n.content,
		Type:          n.nodeType,
		Slug:          n.slug,
		ChildGrouping: n.childGrouping,
	}
}

func (n *NodeBase) fromBaseJsonStruct(jsonStruct NodeBaseJSON) {
	n.id = jsonStruct.ID
	n.title = jsonStruct.Title
	n.content = jsonStruct.Content
	n.nodeType = jsonStruct.Type
	n.slug = jsonStruct.Slug
	n.childGrouping = jsonStruct.ChildGrouping
}

func (n *NodeBase) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.asBaseJsonStruct())
}

func (n *NodeBase) UnmarshalJSON(data []byte) error {
	var nodeJSON NodeBaseJSON
	if err := json.Unmarshal(data, &nodeJSON); err != nil {
		return err
	}

	n.fromBaseJsonStruct(nodeJSON)

	return nil
}

// Node interface that all node types must implement
type Node interface {
	ID() string

	Title() string
	SetTitle(string)

	Content() string
	SetContent(string)

	Type() string
	SetType(string)

	GetSlug() *string
	SetSlug(*string)

	GetChildGrouping() ChildGroup
	SetChildGrouping(ChildGroup)
}

// Implement Node interface for NodeBase
func (n *NodeBase) ID() string      { return n.id }
func (n *NodeBase) Title() string   { return n.title }
func (n *NodeBase) Content() string { return n.content }
func (n *NodeBase) Type() string    { return n.nodeType }
func (n *NodeBase) SetType(nodeType string) {
	n.nodeType = nodeType
}
func (n *NodeBase) GetSlug() *string { return n.slug }
func (n *NodeBase) SetTitle(title string) {
	n.title = title
}
func (n *NodeBase) SetContent(content string) {
	n.content = content
}
func (n *NodeBase) SetSlug(slug *string) {
	n.slug = slug
}

func (n *NodeBase) GetChildGrouping() ChildGroup {
	if n.childGrouping == nil {
		return ChildGroup{}
	}
	return *n.childGrouping
}

func (n *NodeBase) SetChildGrouping(grouping ChildGroup) {
	n.childGrouping = &grouping
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
			id:       uuid.New().String(),
			title:    title,
			content:  content,
			nodeType: "specification",
		},
	}
}

// NewSpecWithID creates a new Spec with a specific ID for testing
func NewSpecWithID(id, title, content string) *Spec {
	return &Spec{
		NodeBase: NodeBase{
			id:       id,
			title:    title,
			content:  content,
			nodeType: "specification",
		},
	}
}

// Project represents a project node in the system
type Project struct {
	NodeBase
}

func (p *Project) MarshalJSON() ([]byte, error) {
	return json.Marshal(&p.NodeBase)
}

func (p *Project) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &p.NodeBase)
}

// NewProject creates a new Project with the type field set
func NewProject(title, content string) *Project {
	return &Project{
		NodeBase: NodeBase{
			id:       uuid.New().String(),
			title:    title,
			content:  content,
			nodeType: "project",
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
