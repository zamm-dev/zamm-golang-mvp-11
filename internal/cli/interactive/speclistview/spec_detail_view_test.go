package speclistview

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/yourorg/zamm-mvp/internal/services"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

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

func TestSpecDetailViewInitialRender(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("..", "common", "testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	// Get "Hello World" spec and its data
	spec, err := specService.GetSpec("f38191af-1b23-4129-854b-5ba754a30c3c")
	if err != nil {
		t.Fatalf("Failed to get test spec: %v", err)
	}

	// Get links for the spec
	links, err := linkService.GetCommitsForSpec(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get links for spec: %v", err)
	}

	// Get child specs
	childSpecs, err := specService.GetChildren(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get child specs: %v", err)
	}

	// Create spec detail view
	view := NewSpecDetailView()
	view.SetSize(80, 24)
	view.SetSpec(*spec, links, childSpecs)

	tm := teatest.NewTestModel(t, &view, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render and capture golden output
	waitForGoldenOutput(t, tm, []byte("Lorem ipsum"), "TestSpecDetailViewInitialRender.golden")
}

func TestSpecDetailViewScrolling(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("..", "common", "testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	// Get a spec from testdata
	spec, err := specService.GetSpec("f38191af-1b23-4129-854b-5ba754a30c3c") // "Hello World Function"
	if err != nil {
		t.Fatalf("Failed to get test spec: %v", err)
	}

	// Get links for the spec
	links, err := linkService.GetCommitsForSpec(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get links for spec: %v", err)
	}

	// Get child specs
	childSpecs, err := specService.GetChildren(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get child specs: %v", err)
	}

	// Create spec detail view with smaller height to force scrolling
	view := NewSpecDetailView()
	view.SetSize(80, 24)
	view.SetSpec(*spec, links, childSpecs)

	tm := teatest.NewTestModel(t, &view, teatest.WithInitialTermSize(80, 24))

	// Simulate page down to scroll
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	// Wait for scrolling and capture golden output
	waitForGoldenOutput(t, tm, []byte("Nullam quis"), "TestSpecDetailViewScrolling.golden")
}
