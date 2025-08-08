package cli

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourorg/zamm-mvp/internal/config"
	"github.com/yourorg/zamm-mvp/internal/services"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

func TestSpewOutputFormat(t *testing.T) {
	// Create a buffer to capture debug output
	var debugBuffer bytes.Buffer

	// Create a minimal app for testing
	cfg := &config.Config{}
	fileStorage := storage.NewFileStorage("interactive/common/testdata/.zamm")
	specService := services.NewSpecService(fileStorage)
	linkService := services.NewLinkService(fileStorage)

	app := &App{
		config:      cfg,
		specService: specService,
		linkService: linkService,
	}

	// Create model with debug writer
	model := NewModel(app, &debugBuffer)

	// Test with a WindowSizeMsg to verify spew formatting
	testMessage := tea.WindowSizeMsg{Width: 80, Height: 24}

	// Call Update with the test message
	_, _ = model.Update(testMessage)

	// Get the output
	output := debugBuffer.String()

	// Verify spew-specific formatting characteristics
	expectedPatterns := []string{
		"(tea.WindowSizeMsg)", // Type information in parentheses
		"Width:",              // Field names
		"Height:",             // Field names
		"(int) 80",            // Type and value formatting
		"(int) 24",            // Type and value formatting
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Expected spew output to contain %q, but got:\n%s", pattern, output)
		}
	}

	// Verify it's properly formatted (should have braces and indentation)
	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Errorf("Expected spew output to contain braces for struct formatting, but got:\n%s", output)
	}
}

func TestComplexMessageTypeFormatting(t *testing.T) {
	// Create a buffer to capture debug output
	var debugBuffer bytes.Buffer

	// Create a minimal app for testing
	cfg := &config.Config{}
	fileStorage := storage.NewFileStorage("interactive/common/testdata/.zamm")
	specService := services.NewSpecService(fileStorage)
	linkService := services.NewLinkService(fileStorage)

	app := &App{
		config:      cfg,
		specService: specService,
		linkService: linkService,
	}

	// Create model with debug writer
	model := NewModel(app, &debugBuffer)

	// Test with a custom message type
	testMessage := operationCompleteMsg{message: "Test operation completed successfully"}

	// Call Update with the test message
	_, _ = model.Update(testMessage)

	// Get the output
	output := debugBuffer.String()

	// Verify the custom message type is properly formatted
	expectedPatterns := []string{
		"(cli.operationCompleteMsg)",            // Custom type with package prefix
		"message:",                              // Field name
		"Test operation completed successfully", // Field value
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Expected spew output to contain %q, but got:\n%s", pattern, output)
		}
	}
}
