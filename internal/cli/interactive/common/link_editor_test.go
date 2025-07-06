package common

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// Mock services for testing
type mockLinkService struct{}
type mockSpecService struct{}

func (m *mockLinkService) LinkSpecToCommit(specID, commitHash, repoPath, linkType string) (*models.SpecCommitLink, error) {
	return &models.SpecCommitLink{}, nil
}

func (m *mockLinkService) UnlinkSpecFromCommit(specID, commitID, repoPath string) error {
	return nil
}

func (m *mockLinkService) GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error) {
	return []*models.SpecCommitLink{}, nil
}

func (m *mockLinkService) GetSpecsForCommit(commitID, repoPath string) ([]*models.SpecNode, error) {
	return []*models.SpecNode{}, nil
}

func (m *mockSpecService) ListSpecs() ([]*models.SpecNode, error) {
	return []*models.SpecNode{
		{
			ID:      "test-spec-1",
			Title:   "Test Spec",
			Content: "Test content",
		},
		{
			ID:      "other-spec-2",
			Title:   "Other Spec",
			Content: "Other content",
		},
	}, nil
}

func (m *mockSpecService) GetChildren(specID string) ([]*models.SpecNode, error) {
	return []*models.SpecNode{}, nil
}

func (m *mockSpecService) AddChildToParent(childID, parentID, linkType string) (*models.SpecSpecLink, error) {
	return &models.SpecSpecLink{}, nil
}

func (m *mockSpecService) RemoveChildFromParent(childID, parentID string) error {
	return nil
}

func (m *mockSpecService) CreateSpec(title, content string) (*models.SpecNode, error) {
	return &models.SpecNode{}, nil
}

func (m *mockSpecService) GetSpec(id string) (*models.SpecNode, error) {
	return &models.SpecNode{}, nil
}

func (m *mockSpecService) UpdateSpec(id, title, content string) (*models.SpecNode, error) {
	return &models.SpecNode{}, nil
}

func (m *mockSpecService) DeleteSpec(id string) error {
	return nil
}

func (m *mockSpecService) GetParents(specID string) ([]*models.SpecNode, error) {
	return []*models.SpecNode{}, nil
}

func (m *mockSpecService) InitializeRootSpec() error {
	return nil
}

func (m *mockSpecService) GetRootSpec() (*models.SpecNode, error) {
	return &models.SpecNode{}, nil
}

func (m *mockSpecService) GetOrphanSpecs() ([]*models.SpecNode, error) {
	return []*models.SpecNode{}, nil
}

func requireGoldenAfterWaitFor(t *testing.T, tm *teatest.TestModel, waitFor []byte, goldenName string) {
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

func TestLinkEditorInitialRender(t *testing.T) {
	config := LinkEditorConfig{
		Title:             "Test Link Editor",
		DefaultRepo:       "/test/repo",
		SelectedSpecID:    "test-spec-1",
		SelectedSpecTitle: "Test Spec",
		IsUnlinkMode:      false,
	}
	linkService := &mockLinkService{}
	specService := &mockSpecService{}
	model := NewLinkEditor(config, linkService, specService)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	requireGoldenAfterWaitFor(t, tm, []byte("Link Type Selection"), "TestLinkEditorInitialRender.golden")
}

func TestLinkEditorPressG(t *testing.T) {
	config := LinkEditorConfig{
		Title:             "Test Link Editor",
		DefaultRepo:       "/test/repo",
		SelectedSpecID:    "test-spec-1",
		SelectedSpecTitle: "Test Spec",
		IsUnlinkMode:      false,
	}
	linkService := &mockLinkService{}
	specService := &mockSpecService{}
	model := NewLinkEditor(config, linkService, specService)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Simulate pressing 'g'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})

	requireGoldenAfterWaitFor(t, tm, []byte("Git Commit"), "TestLinkEditorPressG.golden")
}

func TestLinkEditorSpecSelectionMode(t *testing.T) {
	config := LinkEditorConfig{
		Title:             "Test Link Editor",
		DefaultRepo:       "/test/repo",
		SelectedSpecID:    "test-spec-1",
		SelectedSpecTitle: "Test Spec",
		IsUnlinkMode:      false,
	}
	linkService := &mockLinkService{}
	specService := &mockSpecService{}
	model := NewLinkEditor(config, linkService, specService)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Simulate pressing 's' to select spec link type
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	// Process async update
	tm.Send(nil)

	// Wait for the spec selection screen to render
	requireGoldenAfterWaitFor(t, tm, []byte("Other Spec"), "TestLinkEditorSpecSelectionMode.golden")
}
