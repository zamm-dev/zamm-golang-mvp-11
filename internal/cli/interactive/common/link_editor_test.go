package common

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

func requireGoldenAfterWaitFor(t *testing.T, tm *teatest.TestModel, waitFor []byte) {
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
	// Use testdata storage
	testDataPath := filepath.Join("testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	config := LinkEditorConfig{
		Title:            "Test Link Editor",
		DefaultRepo:      "/test/repo",
		CurrentSpecID:    "201c7092-9367-4a97-837b-98fbbcd7168a", // "Hello World" spec from testdata
		CurrentSpecTitle: "Hello World",
		IsUnlinkMode:     false,
	}
	model := NewLinkEditor(config, linkService, specService)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	requireGoldenAfterWaitFor(t, tm, []byte("Select link type"))
}

func TestLinkEditorPressG(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	config := LinkEditorConfig{
		Title:            "Test Link Editor",
		DefaultRepo:      "/test/repo",
		CurrentSpecID:    "201c7092-9367-4a97-837b-98fbbcd7168a", // "Hello World" spec from testdata
		CurrentSpecTitle: "Hello World",
		IsUnlinkMode:     false,
	}
	model := NewLinkEditor(config, linkService, specService)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Simulate pressing 'g'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})

	requireGoldenAfterWaitFor(t, tm, []byte("Commit Hash"))
}

func TestLinkEditorSpecSelectionMode(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	config := LinkEditorConfig{
		Title:            "Test Link Editor",
		DefaultRepo:      "/test/repo",
		CurrentSpecID:    "201c7092-9367-4a97-837b-98fbbcd7168a", // "Hello World" spec from testdata
		CurrentSpecTitle: "Hello World",
		IsUnlinkMode:     false,
	}
	model := NewLinkEditor(config, linkService, specService)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Simulate pressing 'c' to select child spec link type
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	// Process async update
	tm.Send(nil)

	// Wait for the spec selection screen to render
	requireGoldenAfterWaitFor(t, tm, []byte("Rust Implementation"))
}

func TestLinkEditorUnlinkGitMode(t *testing.T) {
	// Use testdata storage
	testDataPath := filepath.Join("testdata", ".zamm")
	storage := storage.NewFileStorage(testDataPath)
	linkService := services.NewLinkService(storage)
	specService := services.NewSpecService(storage)

	config := LinkEditorConfig{
		Title:            "Test Link Editor",
		DefaultRepo:      "/test/repo",
		CurrentSpecID:    "3e6eec1d-c622-42a5-8fe5-88151ba97090", // "Hello World" spec from testdata
		CurrentSpecTitle: "Hello World Function",
		IsUnlinkMode:     true,
	}
	model := NewLinkEditor(config, linkService, specService)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Simulate pressing 's' to select spec link type
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	// Process async update
	tm.Send(nil)

	// Wait for the spec selection screen to render
	requireGoldenAfterWaitFor(t, tm, []byte("Hello World Function"))
}
