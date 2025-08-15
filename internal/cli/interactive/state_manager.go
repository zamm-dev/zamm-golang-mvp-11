package interactive

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/common"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/nodes"
)

type StateManager struct {
	state          MenuState
	specs          []Spec
	links          []common.LinkItem
	selectedSpecID string
	message        string
	showMessage    bool
	specListView   nodes.NodeExplorer

	terminalWidth  int
	terminalHeight int

	// Input fields for forms
	inputTitle   string
	inputContent string
	inputCommit  string
	inputRepo    string
	inputType    string
	promptText   string

	// State tracking
	editingSpecID string
	contentLines  []string
	confirmAction string
	parentSpecID  string

	linkEditor           *common.LinkEditor
	nodeEditor           *common.NodeEditor
	nodeTypeSelector     *common.NodeTypeSelector
	pendingNodeType      common.NodeType
	implForm             *common.ImplementationForm
	implRepoURL          *string
	implBranch           *string
	implFolderPath       *string
	slugEditor           *common.SlugEditor
	linkSelector         *common.LinkSelector
	confirmationDialog   *common.DeleteConfirmationDialog

	cursor int
}


func NewStateManager(specListView nodes.NodeExplorer) *StateManager {
	return &StateManager{
		state:          SpecListView,
		specListView:   specListView,
		terminalWidth:  80,
		terminalHeight: 24,
	}
}

func (s *StateManager) GetState() MenuState {
	return s.state
}

func (s *StateManager) SetState(state MenuState) {
	s.state = state
	s.cursor = 0
}

func (s *StateManager) GetSpecs() []Spec {
	return s.specs
}

func (s *StateManager) SetSpecs(specs []Spec) {
	s.specs = specs
}

func (s *StateManager) GetLinks() []common.LinkItem {
	return s.links
}

func (s *StateManager) SetLinks(links []LinkItem) {
	var commonLinks []common.LinkItem
	for _, link := range links {
		commonLinks = append(commonLinks, common.LinkItem{
			ID:        link.ID,
			CommitID:  link.CommitID,
			RepoPath:  link.RepoPath,
			LinkLabel: link.LinkLabel,
		})
	}
	s.links = commonLinks
}

func (s *StateManager) GetSelectedSpecID() string {
	return s.selectedSpecID
}

func (s *StateManager) SetSelectedSpecID(id string) {
	s.selectedSpecID = id
}

func (s *StateManager) ShowMessage(message string) {
	s.message = message
	s.showMessage = true
}

func (s *StateManager) HideMessage() {
	s.showMessage = false
	s.message = ""
}

func (s *StateManager) IsShowingMessage() bool {
	return s.showMessage
}

func (s *StateManager) GetMessage() string {
	return s.message
}

func (s *StateManager) SetSize(width, height int) {
	s.terminalWidth = width
	s.terminalHeight = height
	s.specListView.SetSize(width, height)
}

func (s *StateManager) RefreshSpecListView() tea.Cmd {
	return s.specListView.Refresh()
}

func (s *StateManager) GetInputTitle() string {
	return s.inputTitle
}

func (s *StateManager) SetInputTitle(title string) {
	s.inputTitle = title
}

func (s *StateManager) GetInputContent() string {
	return s.inputContent
}

func (s *StateManager) SetInputContent(content string) {
	s.inputContent = content
}

func (s *StateManager) GetEditingSpecID() string {
	return s.editingSpecID
}

func (s *StateManager) SetEditingSpecID(id string) {
	s.editingSpecID = id
}

func (s *StateManager) GetParentSpecID() string {
	return s.parentSpecID
}

func (s *StateManager) SetParentSpecID(id string) {
	s.parentSpecID = id
}

func (s *StateManager) ResetInputs() {
	s.inputTitle = ""
	s.inputContent = ""
	s.inputCommit = ""
	s.inputRepo = ""
	s.inputType = ""
	s.promptText = ""
	s.editingSpecID = ""
	s.contentLines = []string{}
	s.confirmAction = ""
	s.parentSpecID = ""
	s.implRepoURL = nil
	s.implBranch = nil
	s.implFolderPath = nil
	s.nodeTypeSelector = nil
}

func (s *StateManager) SetNodeEditor(editor common.NodeEditor) {
	s.nodeEditor = &editor
	s.nodeEditor.SetSize(s.terminalWidth, s.terminalHeight)
}

func (s *StateManager) SetLinkEditor(editor common.LinkEditor) {
	s.linkEditor = &editor
	s.linkEditor.SetSize(s.terminalWidth, s.terminalHeight)
}

func (s *StateManager) SetNodeTypeSelector(selector *common.NodeTypeSelector) {
	s.nodeTypeSelector = selector
	if s.nodeTypeSelector != nil {
		s.nodeTypeSelector.SetSize(s.terminalWidth, s.terminalHeight)
	}
}

func (s *StateManager) SetPendingNodeType(nodeType common.NodeType) {
	s.pendingNodeType = nodeType
}

func (s *StateManager) SetImplementationForm(form common.ImplementationForm) {
	s.implForm = &form
	s.implForm.SetSize(s.terminalWidth, s.terminalHeight)
}

func (s *StateManager) SetImplRepoURL(url *string) {
	s.implRepoURL = url
}

func (s *StateManager) SetImplBranch(branch *string) {
	s.implBranch = branch
}

func (s *StateManager) SetImplFolderPath(path *string) {
	s.implFolderPath = path
}

func (s *StateManager) SetSlugEditor(editor *common.SlugEditor) {
	s.slugEditor = editor
	if s.slugEditor != nil {
		s.slugEditor.SetSize(s.terminalWidth, s.terminalHeight)
	}
}

func (s *StateManager) SetLinkSelector(selector *common.LinkSelector) {
	s.linkSelector = selector
	if s.linkSelector != nil {
		s.linkSelector.SetSize(s.terminalWidth, s.terminalHeight)
	}
}

func (s *StateManager) SetConfirmationDialog(dialog common.DeleteConfirmationDialog) {
	s.confirmationDialog = &dialog
	s.confirmationDialog.SetSize(s.terminalWidth, s.terminalHeight)
}

func (s *StateManager) UpdateComponent(msg tea.Msg) tea.Cmd {
	switch s.state {
	case SpecListView:
		var cmd tea.Cmd
		s.specListView, cmd = s.specListView.Update(msg)
		return cmd
	case LinkSelection:
		if s.linkSelector != nil {
			var cmd tea.Cmd
			selector, cmd := s.linkSelector.Update(msg)
			if ls, ok := selector.(*common.LinkSelector); ok {
				s.linkSelector = ls
			}
			return cmd
		}
	case NodeEditor:
		if s.nodeEditor != nil {
			editor, editorCmd := s.nodeEditor.Update(msg)
			if nodeEditor, ok := editor.(*common.NodeEditor); ok {
				s.nodeEditor = nodeEditor
			}
			return editorCmd
		}
	case ConfirmDelete:
		if s.confirmationDialog != nil {
			var cmd tea.Cmd
			dialog, cmd := s.confirmationDialog.Update(msg)
			if cd, ok := dialog.(*common.DeleteConfirmationDialog); ok {
				s.confirmationDialog = cd
			}
			return cmd
		}
	case SlugEditor:
		if s.slugEditor != nil {
			editor, editorCmd := s.slugEditor.Update(msg)
			if slugEditor, ok := editor.(*common.SlugEditor); ok {
				s.slugEditor = slugEditor
			}
			return editorCmd
		}
	case NodeTypeSelection:
		if s.nodeTypeSelector != nil {
			var cmd tea.Cmd
			selector, cmd := s.nodeTypeSelector.Update(msg)
			if nts, ok := selector.(*common.NodeTypeSelector); ok {
				s.nodeTypeSelector = nts
			}
			return cmd
		}
	case ImplementationForm:
		if s.implForm != nil {
			var cmd tea.Cmd
			formModel, cmd := s.implForm.Update(msg)
			if f, ok := formModel.(common.ImplementationForm); ok {
				*s.implForm = f
			}
			return cmd
		}
	case LinkEditor:
		if s.linkEditor != nil {
			var cmd tea.Cmd
			editor, cmd := s.linkEditor.Update(msg)
			if linkEditor, ok := editor.(common.LinkEditor); ok {
				*s.linkEditor = linkEditor
			}
			return cmd
		}
	}
	return nil
}

func (s *StateManager) View() string {
	if s.showMessage {
		return s.message
	}

	switch s.state {
	case SpecListView:
		return s.specListView.View()
	case LinkSelection:
		if s.linkSelector != nil {
			return s.linkSelector.View()
		}
		return "Loading link selector..."
	case NodeEditor:
		if s.nodeEditor != nil {
			return s.nodeEditor.View()
		}
		return "Loading node editor..."
	case ImplementationForm:
		if s.implForm != nil {
			return s.implForm.View()
		}
		return "Loading implementation form..."
	case NodeTypeSelection:
		if s.nodeTypeSelector != nil {
			return s.nodeTypeSelector.View()
		}
		return "Choose node type..."
	case LinkEditor:
		if s.linkEditor != nil {
			return s.linkEditor.View()
		}
		return "Loading link editor..."
	case SlugEditor:
		if s.slugEditor != nil {
			return s.slugEditor.View()
		}
		return "Loading slug editor..."
	case ConfirmDelete:
		if s.confirmationDialog != nil {
			return s.confirmationDialog.View()
		}
		return "Loading confirmation dialog..."
	default:
		return "Loading..."
	}
}

func (s *StateManager) HandleMessageDismissal(msg tea.KeyMsg) bool {
	if s.showMessage {
		if msg.String() == "enter" || msg.String() == " " || msg.String() == "esc" {
			s.showMessage = false
			s.message = ""
			s.state = SpecListView
			s.cursor = 0
			return true
		}
	}
	return false
}
