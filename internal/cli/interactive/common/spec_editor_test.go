package common

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
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

func TestSpecEditorEscWithDirtyFormShowsOverlay(t *testing.T) {
	config := SpecEditorConfig{
		Title:          "Test Spec Editor",
		InitialTitle:   "Initial Title",
		InitialContent: "Initial Content",
	}
	editor := NewSpecEditor(config)
	tm := teatest.NewTestModel(t, &editor, teatest.WithInitialTermSize(80, 24))

	// Simulate editing the title (dirty form)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})

	// Simulate pressing esc
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Wait for the overlay to appear and capture golden output
	waitForGoldenOutput(t, tm, []byte("Unsaved Changes"), "TestSpecEditorEscWithDirtyFormShowsOverlay.golden")
}

func TestSpecEditorPressNToDismissOverlay(t *testing.T) {
	config := SpecEditorConfig{
		Title:          "Test Spec Editor",
		InitialTitle:   "Initial Title",
		InitialContent: "Initial Content",
	}
	editor := NewSpecEditor(config)
	tm := teatest.NewTestModel(t, &editor, teatest.WithInitialTermSize(80, 24))

	// Simulate editing the title (dirty form)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})

	// Simulate pressing esc to show overlay
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Simulate pressing 'n' to dismiss overlay
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	// Simulate pressing 'Z' to confirm text on screen changes
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Z")})

	// Wait for return to editor and capture golden output
	waitForGoldenOutput(t, tm, []byte("Initial TitleXZ"), "TestSpecEditorPressNToDismissOverlay.golden")
}
