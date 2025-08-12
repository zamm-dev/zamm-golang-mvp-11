package nodes

import (
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/exp/teatest"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

func TestNodeDetailProjectRender(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("..", "common", "testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	specService := services.NewSpecService(storage)

	// Get "Test Project" and its data
	project, err := specService.GetNode("4c09428a-ce7e-43d0-85da-6f671453c06f")
	if err != nil {
		t.Fatalf("Failed to get test project: %v", err)
	}

	// Verify it's a project node
	if project.GetType() != "project" {
		t.Fatalf("Expected project node, got: %s", project.GetType())
	}

	// Get links for the project (projects don't have commit links, so pass empty slice)
	var links []*models.SpecCommitLink

	// Get child nodes
	childNodes, err := specService.GetChildren(project.GetID())
	if err != nil {
		t.Fatalf("Failed to get child nodes: %v", err)
	}

	// Create project detail
	detail := NewNodeDetail()
	detail.SetSize(80, 24)
	detail.SetSpec(project, links, childNodes)

	tm := teatest.NewTestModel(t, detail, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render and capture golden output
	waitForGoldenOutput(t, tm, []byte("Implementations:"), "TestNodeDetailProjectRender.golden")
}

func TestNodeDetailSpecificationRender(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("..", "common", "testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	// Get "Hello World Function" spec and its data
	spec, err := specService.GetNode("201c7092-9367-4a97-837b-98fbbcd7168a")
	if err != nil {
		t.Fatalf("Failed to get test spec: %v", err)
	}

	// Verify it's a specification node
	if spec.GetType() != "specification" {
		t.Fatalf("Expected specification node, got: %s", spec.GetType())
	}

	// Get links for the spec
	links, err := linkService.GetCommitsForSpec(spec.GetID())
	if err != nil {
		t.Fatalf("Failed to get links for spec: %v", err)
	}

	// Get child nodes
	childNodes, err := specService.GetChildren(spec.GetID())
	if err != nil {
		t.Fatalf("Failed to get child nodes: %v", err)
	}

	// Create spec detail
	detail := NewNodeDetail()
	detail.SetSize(80, 24)
	detail.SetSpec(spec, links, childNodes)

	tm := teatest.NewTestModel(t, detail, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render and capture golden output (should NOT contain Implementations section)
	waitForGoldenOutput(t, tm, []byte("Child Nodes:"), "TestNodeDetailSpecificationRender.golden")
}
