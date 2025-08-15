package storage

import (
	"os"
	"strings"
	"testing"

	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
)

func TestWriteMarkdownFileWithChildren(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zamm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	fs := NewFileStorage(tempDir)

	err = fs.InitializeStorage()
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	parentNode := &models.Spec{
		NodeBase: models.NodeBase{
			ID:      "parent-id",
			Title:   "Parent Node",
			Content: "This is the parent content.",
			Type:    "specification",
		},
	}

	child1 := &models.Spec{
		NodeBase: models.NodeBase{
			ID:      "child-1-id",
			Title:   "Child 1",
			Content: "Child 1 content",
			Type:    "specification",
		},
	}

	child2 := &models.Spec{
		NodeBase: models.NodeBase{
			ID:      "child-2-id",
			Title:   "Child 2",
			Content: "Child 2 content",
			Type:    "specification",
		},
	}

	children := []models.Node{child1, child2}

	err = fs.WriteNodeWithChildren(parentNode, children)
	if err != nil {
		t.Fatalf("Failed to write node with children: %v", err)
	}

	filePath := fs.GetNodeFilePath(parentNode.GetID())
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "\n---\n") {
		t.Error("File should contain horizontal divider")
	}

	if !strings.Contains(contentStr, "## Child Specifications") {
		t.Error("File should contain child specifications section")
	}

	if !strings.Contains(contentStr, "[Child 1](.zamm/nodes/child-1-id.md)") {
		t.Error("File should contain link to child 1")
	}

	if !strings.Contains(contentStr, "[Child 2](.zamm/nodes/child-2-id.md)") {
		t.Error("File should contain link to child 2")
	}

	var readNode models.Spec
	err = fs.readMarkdownFile(filePath, &readNode)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}

	if strings.Contains(readNode.Content, "## Child Specifications") {
		t.Error("Read content should not include child specifications section")
	}

	if strings.Contains(readNode.Content, "[Child 1](.zamm/nodes/child-1-id.md)") {
		t.Error("Read content should not include child links")
	}

	if readNode.Title != "Parent Node" {
		t.Errorf("Expected title 'Parent Node', got '%s'", readNode.Title)
	}

	if readNode.Content != "This is the parent content." {
		t.Errorf("Expected content 'This is the parent content.', got '%s'", readNode.Content)
	}
}

func TestWriteMarkdownFileWithoutChildren(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zamm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	fs := NewFileStorage(tempDir)

	err = fs.InitializeStorage()
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	node := &models.Spec{
		NodeBase: models.NodeBase{
			ID:      "test-id",
			Title:   "Test Node",
			Content: "This is test content.",
			Type:    "specification",
		},
	}

	err = fs.WriteNodeWithChildren(node, []models.Node{})
	if err != nil {
		t.Fatalf("Failed to write node without children: %v", err)
	}

	filePath := fs.GetNodeFilePath(node.GetID())
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	contentStr := string(content)

	if strings.Contains(contentStr, "## Child Specifications") {
		t.Error("File should not contain child specifications section when no children")
	}

	var readNode models.Spec
	err = fs.readMarkdownFile(filePath, &readNode)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}

	if readNode.Title != "Test Node" {
		t.Errorf("Expected title 'Test Node', got '%s'", readNode.Title)
	}

	if readNode.Content != "This is test content." {
		t.Errorf("Expected content 'This is test content.', got '%s'", readNode.Content)
	}
}
