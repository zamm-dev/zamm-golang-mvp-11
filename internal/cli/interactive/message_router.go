package interactive

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/common"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/nodes"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

type MessageRouter struct {
	stateManager *StateManager
	coordinator  *Coordinator
}

func NewMessageRouter(stateManager *StateManager, coordinator *Coordinator) *MessageRouter {
	return &MessageRouter{
		stateManager: stateManager,
		coordinator:  coordinator,
	}
}

func (r *MessageRouter) RouteMessage(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case NodesLoadedMsg:
		return r.handleNodesLoaded(msg)
	case LinksLoadedMsg:
		return r.handleLinksLoaded(msg)
	case OperationCompleteMsg:
		return r.handleOperationComplete(msg)
	case ReturnToSpecListMsg:
		return r.handleReturnToSpecList()
	case NavigateToNodeMsg:
		return r.handleNavigateToNode(msg)
	case SetCurrentNodeMsg:
		return r.handleSetCurrentNode(msg)

	case nodes.CreateNewSpecMsg:
		return r.handleCreateNewSpec(msg)
	case nodes.EditSpecMsg:
		return r.handleEditSpec(msg)
	case nodes.OpenMarkdownMsg:
		return r.handleOpenMarkdown(msg)
	case nodes.LinkCommitSpecMsg:
		return r.handleLinkCommitSpec(msg)
	case nodes.DeleteSpecMsg:
		return r.handleDeleteSpec(msg)
	case nodes.RemoveLinkSpecMsg:
		return r.handleRemoveLinkSpec(msg)
	case nodes.MoveSpecMsg:
		return r.handleMoveSpec(msg)
	case nodes.EditSlugMsg:
		return r.handleEditSlug(msg)
	case nodes.OrganizeSpecMsg:
		return r.handleOrganizeSpec(msg)
	case nodes.ExitMsg:
		return tea.Quit

	case common.NodeEditorCompleteMsg:
		return r.handleNodeEditorComplete(msg)
	case common.NodeEditorImplementationFormMsg:
		return r.handleNodeEditorImplementationForm(msg)
	case common.ImplementationFormSubmitMsg:
		return r.handleImplementationFormSubmit(msg)
	case common.ImplementationFormCancelMsg:
		return r.handleImplementationFormCancel()
	case common.NodeTypeSelectedMsg:
		return r.handleNodeTypeSelected(msg)
	case common.NodeTypeCancelledMsg:
		return r.handleNodeTypeCancelled()
	case common.NodeEditorCancelMsg:
		return r.handleNodeEditorCancel()
	case common.LinkEditorCompleteMsg:
		return r.handleLinkEditorComplete()
	case common.LinkEditorCancelMsg:
		return r.handleLinkEditorCancel()
	case common.LinkEditorErrorMsg:
		return r.handleLinkEditorError(msg)
	case common.SlugEditorCompleteMsg:
		return r.handleSlugEditorComplete(msg)
	case common.SlugEditorCancelMsg:
		return r.handleSlugEditorCancel()

	case common.LinkSelectorCompleteMsg:
		return r.handleLinkSelectorComplete(msg)
	case common.LinkSelectorCancelMsg:
		return r.handleLinkSelectorCancel()
	case common.ConfirmationAcceptedMsg:
		return r.handleConfirmationAccepted(msg)
	case common.ConfirmationCancelledMsg:
		return r.handleConfirmationCancelled()
	}

	return nil
}

func (r *MessageRouter) handleNodesLoaded(msg NodesLoadedMsg) tea.Cmd {
	if msg.err != nil {
		r.stateManager.ShowMessage(fmt.Sprintf("Error loading specs: %v", msg.err))
		return nil
	}
	r.stateManager.SetSpecs(msg.nodes)
	return r.stateManager.RefreshSpecListView()
}

func (r *MessageRouter) handleLinksLoaded(msg LinksLoadedMsg) tea.Cmd {
	if msg.err != nil {
		r.stateManager.ShowMessage(fmt.Sprintf("Error loading links: %v", msg.err))
		return nil
	}
	r.stateManager.SetLinks(msg.links)

	if len(msg.links) == 0 {
		r.stateManager.ShowMessage("No links found for this specification.")
		return nil
	}

	r.stateManager.SetState(LinkSelection)
	return nil
}

func (r *MessageRouter) handleOperationComplete(msg OperationCompleteMsg) tea.Cmd {
	r.stateManager.ShowMessage(msg.message)
	return nil
}

func (r *MessageRouter) handleReturnToSpecList() tea.Cmd {
	r.stateManager.SetState(SpecListView)
	r.stateManager.ResetInputs()
	return tea.Batch(r.coordinator.LoadSpecsCmd(), r.stateManager.RefreshSpecListView())
}

func (r *MessageRouter) handleNavigateToNode(msg NavigateToNodeMsg) tea.Cmd {
	if r.stateManager.GetState() != SpecListView {
		r.stateManager.SetState(SpecListView)
		r.stateManager.ResetInputs()
	}
	return func() tea.Msg {
		node, err := r.coordinator.app.SpecService().GetNode(msg.nodeID)
		if err != nil {
			return r.coordinator.LoadSpecsCmd()()
		}
		return SetCurrentNodeMsg{node: node}
	}
}

func (r *MessageRouter) handleSetCurrentNode(msg SetCurrentNodeMsg) tea.Cmd {
	if r.stateManager.GetState() == SpecListView {
		return tea.Batch(r.coordinator.LoadSpecsCmd(), r.stateManager.RefreshSpecListView())
	}
	return nil
}

func (r *MessageRouter) handleCreateNewSpec(msg nodes.CreateNewSpecMsg) tea.Cmd {
	r.stateManager.ResetInputs()
	r.stateManager.SetParentSpecID(msg.ParentSpecID)

	nts := common.NewNodeTypeSelector("Choose node type to create:")
	r.stateManager.SetNodeTypeSelector(&nts)
	r.stateManager.SetState(NodeTypeSelection)
	return nil
}

func (r *MessageRouter) handleEditSpec(msg nodes.EditSpecMsg) tea.Cmd {
	r.stateManager.ResetInputs()
	r.stateManager.SetEditingSpecID(msg.SpecID)

	node, err := r.coordinator.app.SpecService().GetNode(msg.SpecID)
	if err != nil {
		return func() tea.Msg {
			return OperationCompleteMsg{message: fmt.Sprintf("Error loading node: %v. Press Enter to continue...", err)}
		}
	}

	config := common.NodeEditorConfig{
		Title:          "✏️  Edit Node",
		InitialTitle:   node.GetTitle(),
		InitialContent: node.GetContent(),
		NodeType:       node.GetType(),
	}
	r.stateManager.SetNodeEditor(common.NewNodeEditor(config))
	r.stateManager.SetState(NodeEditor)
	return nil
}

func (r *MessageRouter) handleOpenMarkdown(msg nodes.OpenMarkdownMsg) tea.Cmd {
	// Cast storage to FileStorage to access GetNodeFilePath
	fileStorage, ok := r.coordinator.app.Storage().(*StorageAdapter).storage.(*storage.FileStorage)
	if !ok {
		return func() tea.Msg {
			return OperationCompleteMsg{message: "Error: Cannot access file storage. Press Enter to continue..."}
		}
	}

	markdownPath := fileStorage.GetNodeFilePath(msg.SpecID)

	return func() tea.Msg {
		cmd := exec.Command("code", markdownPath)
		err := cmd.Start()
		if err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error opening in VSCode: %v. Press Enter to continue...", err)}
		}
		return ReturnToSpecListMsg{}
	}
}

func (r *MessageRouter) handleLinkCommitSpec(msg nodes.LinkCommitSpecMsg) tea.Cmd {
	r.stateManager.ResetInputs()
	r.stateManager.SetSelectedSpecID(msg.SpecID)

	var specTitle string
	for _, spec := range r.stateManager.GetSpecs() {
		if spec.ID == msg.SpecID {
			specTitle = spec.Title
			break
		}
	}

	config := common.LinkEditorConfig{
		Title:            "Link Specification",
		DefaultRepo:      r.coordinator.app.Config().GetGitConfig().GetDefaultRepo(),
		CurrentSpecID:    msg.SpecID,
		CurrentSpecTitle: specTitle,
		IsUnlinkMode:     false,
		IsMoveMode:       false,
	}
	r.stateManager.SetLinkEditor(common.NewLinkEditor(config, r.coordinator.app.LinkService(), r.coordinator.app.SpecService()))
	r.stateManager.SetState(LinkEditor)
	return nil
}

func (r *MessageRouter) handleDeleteSpec(msg nodes.DeleteSpecMsg) tea.Cmd {
	r.stateManager.ResetInputs()
	r.stateManager.SetSelectedSpecID(msg.SpecID)

	var specTitle string
	for _, spec := range r.stateManager.GetSpecs() {
		if spec.ID == msg.SpecID {
			specTitle = spec.Title
			break
		}
	}

	config := common.ConfirmationDialogConfig{
		Title:         "Confirm Deletion",
		Message:       fmt.Sprintf("Are you sure you want to delete the specification '%s'?", specTitle),
		ConfirmAction: "delete_spec",
		TargetID:      msg.SpecID,
		TargetTitle:   specTitle,
	}
	r.stateManager.SetConfirmationDialog(common.NewDeleteConfirmationDialog(config))
	r.stateManager.SetState(ConfirmDelete)
	return nil
}

func (r *MessageRouter) handleRemoveLinkSpec(msg nodes.RemoveLinkSpecMsg) tea.Cmd {
	r.stateManager.ResetInputs()
	r.stateManager.SetSelectedSpecID(msg.SpecID)

	var specTitle string
	for _, spec := range r.stateManager.GetSpecs() {
		if spec.ID == msg.SpecID {
			specTitle = spec.Title
			break
		}
	}

	config := common.LinkEditorConfig{
		Title:            "Remove Links",
		DefaultRepo:      r.coordinator.app.Config().GetGitConfig().GetDefaultRepo(),
		CurrentSpecID:    msg.SpecID,
		CurrentSpecTitle: specTitle,
		IsUnlinkMode:     true,
		IsMoveMode:       false,
	}
	r.stateManager.SetLinkEditor(common.NewLinkEditor(config, r.coordinator.app.LinkService(), r.coordinator.app.SpecService()))
	r.stateManager.SetState(LinkEditor)
	return nil
}

func (r *MessageRouter) handleMoveSpec(msg nodes.MoveSpecMsg) tea.Cmd {
	r.stateManager.ResetInputs()
	r.stateManager.SetSelectedSpecID(msg.SpecID)

	var specTitle string
	for _, spec := range r.stateManager.GetSpecs() {
		if spec.ID == msg.SpecID {
			specTitle = spec.Title
			break
		}
	}

	config := common.LinkEditorConfig{
		Title:            "Move Spec",
		DefaultRepo:      r.coordinator.app.Config().GetGitConfig().GetDefaultRepo(),
		CurrentSpecID:    msg.SpecID,
		CurrentSpecTitle: specTitle,
		IsUnlinkMode:     false,
		IsMoveMode:       true,
	}
	linkEditor := common.NewLinkEditor(config, r.coordinator.app.LinkService(), r.coordinator.app.SpecService())
	r.stateManager.SetLinkEditor(linkEditor)
	r.stateManager.SetState(LinkEditor)
	return linkEditor.Init()
}

func (r *MessageRouter) handleEditSlug(msg nodes.EditSlugMsg) tea.Cmd {
	r.stateManager.ResetInputs()
	r.stateManager.SetSelectedSpecID(msg.SpecID)

	llmService := r.coordinator.app.LLMService()
	slugEditor := common.NewSlugEditor(msg.SpecID, msg.OriginalTitle, msg.InitialSlug, llmService)
	r.stateManager.SetSlugEditor(slugEditor)
	r.stateManager.SetState(SlugEditor)
	return slugEditor.Init()
}

func (r *MessageRouter) handleOrganizeSpec(msg nodes.OrganizeSpecMsg) tea.Cmd {
	return r.coordinator.OrganizeNodeCmd(msg.SpecID)
}

func (r *MessageRouter) handleNodeEditorComplete(msg common.NodeEditorCompleteMsg) tea.Cmd {
	editingSpecID := r.stateManager.GetEditingSpecID()
	parentSpecID := r.stateManager.GetParentSpecID()

	if editingSpecID != "" {
		return r.coordinator.UpdateNodeCmd(editingSpecID, msg.Title, msg.Content, msg.NodeType, nil, nil, nil)
	} else {
		return r.coordinator.CreateNodeCmd(msg.Title, msg.Content, msg.NodeType, parentSpecID)
	}
}

func (r *MessageRouter) handleNodeEditorImplementationForm(msg common.NodeEditorImplementationFormMsg) tea.Cmd {
	r.stateManager.SetInputTitle(msg.Title)
	r.stateManager.SetInputContent(msg.Content)

	editingSpecID := r.stateManager.GetEditingSpecID()
	if editingSpecID != "" {
		node, err := r.coordinator.app.SpecService().GetNode(editingSpecID)
		if err != nil {
			r.stateManager.ShowMessage(fmt.Sprintf("Error fetching implementation details: %v", err))
			return nil
		}

		if impl, ok := node.(*models.Implementation); ok {
			r.stateManager.SetImplementationForm(common.NewImplementationFormWithValues("🔧 Implementation Details", impl.RepoURL, impl.Branch, impl.FolderPath))
		} else {
			r.stateManager.SetImplementationForm(common.NewImplementationForm("🔧 Implementation Details"))
		}
	} else {
		r.stateManager.SetImplementationForm(common.NewImplementationForm("🔧 Implementation Details"))
	}

	r.stateManager.SetState(ImplementationForm)
	return nil
}

func (r *MessageRouter) handleImplementationFormSubmit(msg common.ImplementationFormSubmitMsg) tea.Cmd {
	r.stateManager.SetImplRepoURL(msg.RepoURL)
	r.stateManager.SetImplBranch(msg.Branch)
	r.stateManager.SetImplFolderPath(msg.FolderPath)

	editingSpecID := r.stateManager.GetEditingSpecID()
	inputTitle := r.stateManager.GetInputTitle()
	inputContent := r.stateManager.GetInputContent()
	parentSpecID := r.stateManager.GetParentSpecID()

	if editingSpecID != "" {
		return r.coordinator.UpdateNodeCmd(editingSpecID, inputTitle, inputContent, "implementation", msg.RepoURL, msg.Branch, msg.FolderPath)
	} else {
		return r.coordinator.CreateImplementationCmd(inputTitle, inputContent, parentSpecID, msg.RepoURL, msg.Branch, msg.FolderPath)
	}
}

func (r *MessageRouter) handleImplementationFormCancel() tea.Cmd {
	return func() tea.Msg { return ReturnToSpecListMsg{} }
}

func (r *MessageRouter) handleNodeTypeSelected(msg common.NodeTypeSelectedMsg) tea.Cmd {
	r.stateManager.SetPendingNodeType(msg.Type)

	var title, nodeType string
	if msg.Type == common.NodeTypeImplementation {
		title = "🧩 Create New Implementation"
		nodeType = "implementation"
	} else {
		title = "📝 Create New Specification"
		nodeType = "specification"
	}

	config := common.NodeEditorConfig{
		Title:          title,
		InitialTitle:   "",
		InitialContent: "",
		NodeType:       nodeType,
	}
	r.stateManager.SetNodeEditor(common.NewNodeEditor(config))
	r.stateManager.SetState(NodeEditor)
	return nil
}

func (r *MessageRouter) handleNodeTypeCancelled() tea.Cmd {
	return func() tea.Msg { return ReturnToSpecListMsg{} }
}

func (r *MessageRouter) handleNodeEditorCancel() tea.Cmd {
	return func() tea.Msg { return ReturnToSpecListMsg{} }
}

func (r *MessageRouter) handleLinkEditorComplete() tea.Cmd {
	return func() tea.Msg { return ReturnToSpecListMsg{} }
}

func (r *MessageRouter) handleLinkEditorCancel() tea.Cmd {
	return func() tea.Msg { return ReturnToSpecListMsg{} }
}

func (r *MessageRouter) handleLinkEditorError(msg common.LinkEditorErrorMsg) tea.Cmd {
	return func() tea.Msg {
		return OperationCompleteMsg{message: fmt.Sprintf("Error: %s. Press Enter to continue...", msg.Error)}
	}
}

func (r *MessageRouter) handleSlugEditorComplete(msg common.SlugEditorCompleteMsg) tea.Cmd {
	return r.coordinator.SetSlugAndOrganizeCmd(msg.SpecID, msg.Slug)
}

func (r *MessageRouter) handleSlugEditorCancel() tea.Cmd {
	return func() tea.Msg { return ReturnToSpecListMsg{} }
}

func (r *MessageRouter) handleLinkSelectorComplete(msg common.LinkSelectorCompleteMsg) tea.Cmd {
	if msg.Action == "delete_link" {
		config := common.ConfirmationDialogConfig{
			Title:         "Confirm Link Deletion",
			Message:       fmt.Sprintf("Are you sure you want to delete the link to commit %s?", msg.SelectedLink.CommitID[:12]+"..."),
			ConfirmAction: "delete_link",
			TargetID:      r.stateManager.GetSelectedSpecID(),
			ExtraData:     msg.SelectedLink,
		}
		r.stateManager.SetConfirmationDialog(common.NewDeleteConfirmationDialog(config))
		r.stateManager.SetState(ConfirmDelete)
	}
	return nil
}

func (r *MessageRouter) handleLinkSelectorCancel() tea.Cmd {
	r.stateManager.SetState(SpecListView)
	return tea.Batch(r.coordinator.LoadSpecsCmd(), r.stateManager.RefreshSpecListView())
}

func (r *MessageRouter) handleConfirmationAccepted(msg common.ConfirmationAcceptedMsg) tea.Cmd {
	switch msg.Action {
	case "delete_spec":
		return r.coordinator.DeleteSpecCmd(msg.TargetID)
	case "delete_link":
		if linkData, ok := msg.ExtraData.(common.LinkItem); ok {
			return r.coordinator.DeleteLinkCmd(msg.TargetID, linkData.CommitID, linkData.RepoPath)
		}
	}
	r.stateManager.SetState(SpecListView)
	return tea.Batch(r.coordinator.LoadSpecsCmd(), r.stateManager.RefreshSpecListView())
}

func (r *MessageRouter) handleConfirmationCancelled() tea.Cmd {
	r.stateManager.SetState(SpecListView)
	return tea.Batch(r.coordinator.LoadSpecsCmd(), r.stateManager.RefreshSpecListView())
}
