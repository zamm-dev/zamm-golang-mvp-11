package storage

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
)

func TestReadProjectNode(t *testing.T) {
	// Use the existing testdata from the CLI package
	testDataPath := filepath.Join("..", "cli", "interactive", "common", "testdata")
	fs := NewFileStorage(filepath.Join(testDataPath, ".zamm"))

	// Read the project node
	var project models.Project
	nodePath := fs.GetNodeFilePath("4c09428a-ce7e-43d0-85da-6f671453c06f")

	err := fs.readMarkdownFile(nodePath, &project)
	if err != nil {
		t.Fatalf("Failed to read project node: %v", err)
	}

	// Verify all metadata is correctly parsed
	if project.GetID() != "4c09428a-ce7e-43d0-85da-6f671453c06f" {
		t.Errorf("Expected ID '4c09428a-ce7e-43d0-85da-6f671453c06f', got '%s'", project.GetID())
	}

	if project.GetType() != "project" {
		t.Errorf("Expected type 'project', got '%s'", project.GetType())
	}

	if project.GetTitle() != "Test Project" {
		t.Errorf("Expected title 'Test Project', got '%s'", project.GetTitle())
	}

	expectedContent := "This project is meant to help tests pass"
	if project.GetContent() != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, project.GetContent())
	}

	// Verify that divider lines and child sections are ignored
	if strings.Contains(project.GetContent(), "---") {
		t.Error("Content should not contain YAML front matter dividers")
	}

	if strings.Contains(project.GetContent(), "## Child") {
		t.Error("Content should not contain child specification sections")
	}
}

func TestReadImplementationNode(t *testing.T) {
	// Use the existing testdata from the CLI package
	testDataPath := filepath.Join("..", "cli", "interactive", "common", "testdata")
	fs := NewFileStorage(filepath.Join(testDataPath, ".zamm"))

	// Read the implementation node
	var impl models.Implementation
	nodePath := fs.GetNodeFilePath("eb76cdc6-f24c-432a-bfa3-c2ac3257146c")

	err := fs.readMarkdownFile(nodePath, &impl)
	if err != nil {
		t.Fatalf("Failed to read implementation node: %v", err)
	}

	// Verify all metadata is correctly parsed
	if impl.GetID() != "eb76cdc6-f24c-432a-bfa3-c2ac3257146c" {
		t.Errorf("Expected ID 'eb76cdc6-f24c-432a-bfa3-c2ac3257146c', got '%s'", impl.GetID())
	}

	if impl.GetType() != "implementation" {
		t.Errorf("Expected type 'implementation', got '%s'", impl.GetType())
	}

	if impl.GetTitle() != "Rust Implementation" {
		t.Errorf("Expected title 'Rust Implementation', got '%s'", impl.GetTitle())
	}

	expectedContent := "This is an implementation of the project in Rust"
	if impl.GetContent() != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, impl.GetContent())
	}

	// Check that branch metadata is correctly parsed
	if impl.Branch == nil || *impl.Branch != "rust" {
		t.Errorf("Expected branch 'rust', got %v", impl.Branch)
	}

	// Verify that divider lines are ignored
	if strings.Contains(impl.GetContent(), "---") {
		t.Error("Content should not contain YAML front matter dividers")
	}
}

func TestGenerateMarkdownStringWithoutChildren(t *testing.T) {
	testDataPath := filepath.Join("..", "cli", "interactive", "common", "testdata")
	fs := NewFileStorage(filepath.Join(testDataPath, ".zamm"))

	node := models.NewSpecWithID("test-spec-id", "Test Specification", "This is a test specification content.")

	output, err := fs.generateMarkdownString(node)
	if err != nil {
		t.Fatalf("Failed to generate markdown string: %v", err)
	}

	// Verify YAML frontmatter
	if !strings.Contains(output, "---\n") {
		t.Error("Output should contain YAML frontmatter delimiters")
	}

	if !strings.Contains(output, "id: test-spec-id") {
		t.Error("Output should contain node ID in frontmatter")
	}

	if !strings.Contains(output, "type: specification") {
		t.Error("Output should contain node type in frontmatter")
	}

	// Verify title as markdown header
	if !strings.Contains(output, "# Test Specification") {
		t.Error("Output should contain title as level 1 heading")
	}

	// Verify content
	if !strings.Contains(output, "This is a test specification content.") {
		t.Error("Output should contain node content")
	}

	// Verify no child section
	if strings.Contains(output, "## Child Specifications") {
		t.Error("Output should not contain child specifications section when no children")
	}

	if strings.Contains(output, "\n---\n\n## Child") {
		t.Error("Output should not contain divider before children when no children")
	}
}

func TestGenerateMarkdownStringWithChildren(t *testing.T) {
	testDataPath := filepath.Join("..", "cli", "interactive", "common", "testdata")
	fs := NewFileStorage(filepath.Join(testDataPath, ".zamm"))

	parentNode := models.NewSpecWithID("parent-spec-id", "Parent Specification", "This is the parent specification content.")

	child1 := models.NewSpecWithID("child-1-id", "Child 1", "Child 1 content")

	child2 := models.NewSpecWithID("child-2-id", "Child 2", "Child 2 content")

	children := models.ChildGroup{
		Children: []models.Node{child1, child2},
	}

	output, err := fs.generateMarkdownStringWithChildren(parentNode, children)
	if err != nil {
		t.Fatalf("Failed to generate markdown string with children: %v", err)
	}

	// Verify YAML frontmatter
	if !strings.Contains(output, "---\n") {
		t.Error("Output should contain YAML frontmatter delimiters")
	}

	if !strings.Contains(output, "id: parent-spec-id") {
		t.Error("Output should contain parent node ID in frontmatter")
	}

	if !strings.Contains(output, "type: specification") {
		t.Error("Output should contain node type in frontmatter")
	}

	// Verify title as markdown header
	if !strings.Contains(output, "# Parent Specification") {
		t.Error("Output should contain title as level 1 heading")
	}

	// Verify content
	if !strings.Contains(output, "This is the parent specification content.") {
		t.Error("Output should contain parent node content")
	}

	// Verify additional divider before children
	if !strings.Contains(output, "\n---\n\n## Child Specifications") {
		t.Error("Output should contain horizontal divider before child specifications section")
	}

	// Verify child specifications section
	if !strings.Contains(output, "## Child Specifications") {
		t.Error("Output should contain child specifications section")
	}

	// Verify child links
	if !strings.Contains(output, "[Child 1](child-1-id.md)") {
		t.Error("Output should contain link to child 1", output)
	}

	if !strings.Contains(output, "[Child 2](child-2-id.md)") {
		t.Error("Output should contain link to child 2", output)
	}
}
