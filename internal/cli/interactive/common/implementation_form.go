package common

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ImplementationForm collects implementation-specific fields
// Fields: RepoURL, Branch, FolderPath

type ImplementationForm struct {
	title       string
	repoInput   textinput.Model
	branchInput textinput.Model
	pathInput   textinput.Model
	focusIndex  int
	width       int
	height      int
}

type ImplementationFormSubmitMsg struct {
	RepoURL    *string
	Branch     *string
	FolderPath *string
}

type ImplementationFormCancelMsg struct{}

func NewImplementationForm(title string) ImplementationForm {
	repo := textinput.New()
	repo.Placeholder = "Repository URL (optional)"
	repo.Focus()

	branch := textinput.New()
	branch.Placeholder = "Branch (optional)"

	path := textinput.New()
	path.Placeholder = "Path within repo (optional)"

	return ImplementationForm{
		title:       title,
		repoInput:   repo,
		branchInput: branch,
		pathInput:   path,
		focusIndex:  0,
	}
}

func (f *ImplementationForm) SetSize(width, height int) {
	f.width = width
	f.height = height
	f.repoInput.Width = width
	f.branchInput.Width = width
	f.pathInput.Width = width
}

func (f ImplementationForm) Init() tea.Cmd { return nil }

func (f ImplementationForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			f.focusIndex = (f.focusIndex + 1) % 3
			f.applyFocus()
			return f, nil
		case "shift+tab":
			f.focusIndex = (f.focusIndex + 2) % 3
			f.applyFocus()
			return f, nil
		case "enter":
			repo := strings.TrimSpace(f.repoInput.Value())
			branch := strings.TrimSpace(f.branchInput.Value())
			path := strings.TrimSpace(f.pathInput.Value())
			var repoPtr, branchPtr, pathPtr *string
			if repo != "" {
				repoPtr = &repo
			}
			if branch != "" {
				branchPtr = &branch
			}
			if path != "" {
				pathPtr = &path
			}
			return f, func() tea.Msg {
				return ImplementationFormSubmitMsg{RepoURL: repoPtr, Branch: branchPtr, FolderPath: pathPtr}
			}
		case "esc":
			return f, func() tea.Msg { return ImplementationFormCancelMsg{} }
		}
	}

	switch f.focusIndex {
	case 0:
		f.repoInput, _ = f.repoInput.Update(msg)
	case 1:
		f.branchInput, _ = f.branchInput.Update(msg)
	case 2:
		f.pathInput, _ = f.pathInput.Update(msg)
	}
	return f, nil
}

func (f *ImplementationForm) applyFocus() {
	f.repoInput.Blur()
	f.branchInput.Blur()
	f.pathInput.Blur()
	switch f.focusIndex {
	case 0:
		f.repoInput.Focus()
	case 1:
		f.branchInput.Focus()
	case 2:
		f.pathInput.Focus()
	}
}

func (f ImplementationForm) View() string {
	b := &strings.Builder{}
	b.WriteString(f.title + "\n")
	b.WriteString(strings.Repeat("=", len(f.title)) + "\n\n")
	b.WriteString(f.repoInput.View() + "\n\n")
	b.WriteString(f.branchInput.View() + "\n\n")
	b.WriteString(f.pathInput.View() + "\n\n")
	b.WriteString("Tab to navigate, Enter to submit, Esc to cancel")
	return b.String()
}
