package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
	interactive "github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/nodes"
)

// Model represents the state of our TUI application
type Model struct {
	app           *App
	stateManager  *interactive.StateManager
	messageRouter *interactive.MessageRouter
	coordinator   *interactive.Coordinator
	textInput     textinput.Model
	debugWriter   io.Writer
}

// createInteractiveCommand creates the interactive mode command
func (a *App) createInteractiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interactive",
		Short: "Interactive mode for managing specs and links",
		Long:  "Start an interactive session to manage specifications and links using arrow keys for navigation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get the debug flag value from this command's local flags
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				return fmt.Errorf("failed to get debug flag: %w", err)
			}
			return a.runInteractiveMode(debug)
		},
	}

	// Add debug flag specific to this command
	cmd.Flags().Bool("debug", false, "Enable debug logging for bubbletea messages")

	return cmd
}

// NewModel creates a new Model with the given debug writer
func NewModel(app *App, debugWriter io.Writer) *Model {
	ti := textinput.New()
	ti.Focus()

	combinedSvc := interactive.NewCombinedService(app.linkService, app.specService)
	specListView := nodes.NewSpecExplorer(combinedSvc)

	stateManager := interactive.NewStateManager(specListView)
	appAdapter := interactive.NewAppAdapter(app.specService, app.linkService, app.llmService, app.storage, app.config)
	coordinator := interactive.NewCoordinator(appAdapter)
	messageRouter := interactive.NewMessageRouter(stateManager, coordinator)

	return &Model{
		app:           app,
		stateManager:  stateManager,
		messageRouter: messageRouter,
		coordinator:   coordinator,
		textInput:     ti,
		debugWriter:   debugWriter,
	}
}

// runInteractiveMode starts the interactive mode with TUI
func (a *App) runInteractiveMode(debug bool) error {
	// Perform complete initialization
	if err := a.InitializeZamm(); err != nil {
		return fmt.Errorf("failed to initialize zamm: %w", err)
	}

	var debugWriter io.Writer
	var debugFile *os.File
	if debug {
		var err error
		debugFile, err = createDebugLogFile()
		if err != nil {
			return fmt.Errorf("failed to create debug log file: %w", err)
		}
		debugWriter = debugFile
	}

	model := NewModel(a, debugWriter)

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()

	// Ensure proper cleanup of debug file on program exit
	if debugFile != nil {
		if closeErr := debugFile.Close(); closeErr != nil {
			// Log to stderr but don't override the main error
			fmt.Fprintf(os.Stderr, "Warning: failed to close debug log file: %v\n", closeErr)
		}
	}

	return err
}

// Init is the first function that will be called
func (m *Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.coordinator.LoadSpecsCmd())
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Debug logging: dump all messages when debug writer is available
	if m.debugWriter != nil {
		spew.Fdump(m.debugWriter, msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.stateManager.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		// Handle message dismissal
		if m.stateManager.HandleMessageDismissal(msg) {
			return m, tea.Batch(m.coordinator.LoadSpecsCmd(), m.stateManager.RefreshSpecListView())
		}
	}

	// Always try to forward messages to StateManager components first
	if cmd := m.stateManager.UpdateComponent(msg); cmd != nil {
		return m, cmd
	}

	if cmd := m.messageRouter.RouteMessage(msg); cmd != nil {
		return m, cmd
	}

	return m, nil
}

// View renders the UI
func (m *Model) View() string {
	return m.stateManager.View()
}
