package cli

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	interactive "github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/config"
	"github.com/yourorg/zamm-mvp/internal/services"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

func TestMessageDumpingWithDebugWriter(t *testing.T) {
	// Create a buffer to capture debug output
	var debugBuffer bytes.Buffer

	// Create a minimal app for testing
	cfg := &config.Config{}
	fileStorage := storage.NewFileStorage("testdata")
	specService := services.NewSpecService(fileStorage)
	linkService := services.NewLinkService(fileStorage)

	app := &App{
		config:      cfg,
		specService: specService,
		linkService: linkService,
	}

	// Create model with debug writer
	model := NewModel(app, &debugBuffer)

	// Test different message types
	testCases := []struct {
		name        string
		message     tea.Msg
		expectedStr string
	}{
		{
			name:        "WindowSizeMsg",
			message:     tea.WindowSizeMsg{Width: 80, Height: 24},
			expectedStr: "tea.WindowSizeMsg",
		},
		{
			name:        "KeyMsg",
			message:     tea.KeyMsg{Type: tea.KeyEnter},
			expectedStr: "tea.KeyMsg",
		},
		{
			name:        "Custom specsLoadedMsg",
			message:     specsLoadedMsg{specs: []interactive.Spec{}, err: nil},
			expectedStr: "specsLoadedMsg",
		},
		{
			name:        "Custom operationCompleteMsg",
			message:     operationCompleteMsg{message: "test complete"},
			expectedStr: "operationCompleteMsg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear the buffer before each test
			debugBuffer.Reset()

			// Call Update with the test message
			_, _ = model.Update(tc.message)

			// Check that the message was dumped to the debug buffer
			output := debugBuffer.String()
			if !strings.Contains(output, tc.expectedStr) {
				t.Errorf("Expected debug output to contain %q, but got: %s", tc.expectedStr, output)
			}

			// Verify that spew formatting is used (should contain parentheses and type info)
			if !strings.Contains(output, "(") || !strings.Contains(output, ")") {
				t.Errorf("Expected spew formatting with parentheses, but got: %s", output)
			}
		})
	}
}

func TestMessageDumpingWithoutDebugWriter(t *testing.T) {
	// Create a minimal app for testing
	cfg := &config.Config{}
	fileStorage := storage.NewFileStorage("testdata")
	specService := services.NewSpecService(fileStorage)
	linkService := services.NewLinkService(fileStorage)

	app := &App{
		config:      cfg,
		specService: specService,
		linkService: linkService,
	}

	// Create model without debug writer (nil)
	model := NewModel(app, nil)

	// Test that no dumping occurs when debugWriter is nil
	testMessage := tea.WindowSizeMsg{Width: 80, Height: 24}

	// This should not panic or cause any issues
	_, _ = model.Update(testMessage)

	// Since there's no debug writer, we can't capture output, but we can verify
	// that the method completes successfully without errors
	if model.debugWriter != nil {
		t.Error("Expected debugWriter to be nil")
	}
}

func TestDebugWriterFieldExists(t *testing.T) {
	// Create a minimal app for testing
	cfg := &config.Config{}
	fileStorage := storage.NewFileStorage("testdata")
	specService := services.NewSpecService(fileStorage)
	linkService := services.NewLinkService(fileStorage)

	app := &App{
		config:      cfg,
		specService: specService,
		linkService: linkService,
	}

	var debugBuffer bytes.Buffer
	model := NewModel(app, &debugBuffer)

	// Verify that the debugWriter field is properly set
	if model.debugWriter == nil {
		t.Error("Expected debugWriter to be set, but it was nil")
	}

	// Verify that it's the same buffer we passed in
	if model.debugWriter != &debugBuffer {
		t.Error("Expected debugWriter to be the same buffer we passed in")
	}
}
