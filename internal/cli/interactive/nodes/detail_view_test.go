package nodes

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

// testCombinedService implements LinkService for testing
type testViewCombinedService struct {
	linkService services.LinkService
	specService services.SpecService
}

func (cs *testViewCombinedService) GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error) {
	return cs.linkService.GetCommitsForSpec(specID)
}

func (cs *testViewCombinedService) GetChildNodes(specID string) ([]models.Node, error) {
	return cs.specService.GetChildren(specID)
}

func (cs *testViewCombinedService) GetNodeByID(specID string) (models.Node, error) {
	return cs.specService.GetNode(specID)
}

func (cs *testViewCombinedService) GetParentNode(specID string) (models.Node, error) {
	parents, err := cs.specService.GetParents(specID)
	if err != nil {
		return nil, err
	}
	if len(parents) == 0 {
		return nil, nil
	}
	return parents[0], nil
}

func (cs *testViewCombinedService) GetRootNode() (models.Node, error) {
	return cs.specService.GetRootNode()
}

func waitForGoldenOutput(t *testing.T, tm *teatest.TestModel, waitFor []byte, goldenName string) {
	var capturedOutput []byte
	teatest.WaitFor(
		t, tm.Output(),
		func(bts []byte) bool {
			if bytes.Contains(bts, waitFor) {
				capturedOutput = make([]byte, len(bts))
				copy(capturedOutput, bts)
				return true
			}
			return false
		},
		teatest.WithCheckInterval(time.Millisecond*100),
		teatest.WithDuration(time.Second*3),
	)
	teatest.RequireEqualOutput(t, capturedOutput)
}

func TestNodeDetailViewInitialRender(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("..", "common", "testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	combinedSvc := &testViewCombinedService{
		linkService: linkService,
		specService: specService,
	}

	// Get "Hello World" spec and its data
	spec, err := specService.GetNode("f38191af-1b23-4129-854b-5ba754a30c3c")
	if err != nil {
		t.Fatalf("Failed to get test spec: %v", err)
	}

	// Create spec detail view with the combined service
	view := NewNodeDetailView(combinedSvc)
	view.SetSize(80, 24)
	view.SetSpec(spec)

	tm := teatest.NewTestModel(t, &view, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render and capture golden output
	waitForGoldenOutput(t, tm, []byte("Lorem ipsum"), "TestNodeDetailViewInitialRender.golden")
}

func TestNodeDetailViewScrolling(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("..", "common", "testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	combinedSvc := &testViewCombinedService{
		linkService: linkService,
		specService: specService,
	}

	// Get a spec from testdata
	spec, err := specService.GetNode("f38191af-1b23-4129-854b-5ba754a30c3c") // "Hello World Function"
	if err != nil {
		t.Fatalf("Failed to get test spec: %v", err)
	}

	// Create spec detail view with smaller height to force scrolling
	view := NewNodeDetailView(combinedSvc)
	view.SetSize(80, 24)
	view.SetSpec(spec)

	tm := teatest.NewTestModel(t, &view, teatest.WithInitialTermSize(80, 24))

	// Simulate page down to scroll
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	// Wait for scrolling and capture golden output
	waitForGoldenOutput(t, tm, []byte("Nullam quis"), "TestNodeDetailViewScrolling.golden")
}
