package nodes

import (
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/exp/teatest"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

// testCombinedService implements LinkService for testing
type testCombinedService struct {
	linkService services.LinkService
	specService services.SpecService
}

func (cs *testCombinedService) GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error) {
	return cs.linkService.GetCommitsForSpec(specID)
}

func (cs *testCombinedService) GetChildNodes(specID string) ([]models.Node, error) {
	return cs.specService.GetChildren(specID)
}

func (cs *testCombinedService) GetNodeByID(specID string) (models.Node, error) {
	return cs.specService.GetNode(specID)
}

func (cs *testCombinedService) GetParentNode(specID string) (models.Node, error) {
	parents, err := cs.specService.GetParents(specID)
	if err != nil {
		return nil, err
	}
	if len(parents) == 0 {
		return nil, nil
	}
	return parents[0], nil
}

func (cs *testCombinedService) GetRootNode() (models.Node, error) {
	return cs.specService.GetRootNode()
}

func TestNodeDetailProjectRender(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("..", "common", "testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	specService := services.NewSpecService(storage)
	linkService := services.NewLinkService(storage)

	combinedSvc := &testCombinedService{
		linkService: linkService,
		specService: specService,
	}

	// Get "Test Project" and its data
	project, err := specService.GetNode("4c09428a-ce7e-43d0-85da-6f671453c06f")
	if err != nil {
		t.Fatalf("Failed to get test project: %v", err)
	}

	// Verify it's a project node
	if project.GetType() != "project" {
		t.Fatalf("Expected project node, got: %s", project.GetType())
	}

	// Create project detail with the combined service
	detail := NewNodeDetail(combinedSvc)
	detail.SetSize(80, 24)
	detail.SetSpec(project)

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

	combinedSvc := &testCombinedService{
		linkService: linkService,
		specService: specService,
	}

	// Get "Hello World Function" spec and its data
	spec, err := specService.GetNode("201c7092-9367-4a97-837b-98fbbcd7168a")
	if err != nil {
		t.Fatalf("Failed to get test spec: %v", err)
	}

	// Verify it's a specification node
	if spec.GetType() != "specification" {
		t.Fatalf("Expected specification node, got: %s", spec.GetType())
	}

	// Create spec detail with the combined service
	detail := NewNodeDetail(combinedSvc)
	detail.SetSize(80, 24)
	detail.SetSpec(spec)

	tm := teatest.NewTestModel(t, detail, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render and capture golden output (should NOT contain Implementations section)
	waitForGoldenOutput(t, tm, []byte("No children"), "TestNodeDetailSpecificationRender.golden")
}
