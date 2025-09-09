package nodes

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/teatest"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

// testExplorerCombinedService implements LinkService for testing the explorer
type testExplorerCombinedService struct {
	linkService services.LinkService
	specService services.SpecService
}

func (cs *testExplorerCombinedService) GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error) {
	return cs.linkService.GetCommitsForSpec(specID)
}

func (cs *testExplorerCombinedService) GetChildNodes(specID string) ([]models.Node, error) {
	return cs.specService.GetChildren(specID)
}

func (cs *testExplorerCombinedService) GetNodeByID(specID string) (models.Node, error) {
	return cs.specService.ReadNode(specID)
}

func (cs *testExplorerCombinedService) GetParentNode(specID string) (models.Node, error) {
	parents, err := cs.specService.GetParents(specID)
	if err != nil {
		return nil, err
	}
	if len(parents) == 0 {
		return nil, nil
	}
	return parents[0], nil
}

func (cs *testExplorerCombinedService) GetRootNode() (models.Node, error) {
	return cs.specService.GetRootNode()
}

func waitForExplorerGoldenOutput(t *testing.T, tm *teatest.TestModel, waitFor []byte, goldenName string) {
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

func TestNodeExplorerInitialRender(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("..", "common", "testdata", ".zamm")
	storage, err := storage.New(testDataPath)
	if err != nil {
		t.Fatalf("failed to create file storage: %v", err)
	}
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	combinedSvc := &testExplorerCombinedService{
		linkService: linkService,
		specService: specService,
	}

	// Create node explorer
	explorer := NewSpecExplorer(combinedSvc, specService)
	explorer.SetSize(80, 24)

	tm := teatest.NewTestModel(t, explorer, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render and capture golden output
	waitForExplorerGoldenOutput(t, tm, []byte("Select a child specification"), "TestNodeExplorerInitialRender.golden")
}
