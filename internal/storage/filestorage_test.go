package storage

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
)

func TestGenerateMarkdownStringWithoutChildren(t *testing.T) {
	testDataPath := filepath.Join("..", "cli", "interactive", "common", "testdata")
	fs, err := New(filepath.Join(testDataPath, ".zamm"))
	if err != nil {
		t.Fatalf("failed to create file storage: %v", err)
	}

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
	fs, err := New(filepath.Join(testDataPath, ".zamm"))
	if err != nil {
		t.Fatalf("failed to create file storage: %v", err)
	}

	parentNode := models.NewSpecWithID("parent-spec-id", "Parent Specification", "This is the parent specification content.")

	child1 := models.NewSpecWithID("child-1-id", "Child 1", "Child 1 content")

	child2 := models.NewSpecWithID("child-2-id", "Child 2", "Child 2 content")

	children := models.ChildGroup{
		Children: []models.Node{child1, child2},
	}

	output, err := fs.generateChildrenString(parentNode, children)
	if err != nil {
		t.Fatalf("Failed to generate markdown string with children: %v", err)
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
