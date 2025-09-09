package interactive

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
)

type AppInterface interface {
	SpecService() services.SpecService
	LinkService() services.LinkService
	LLMService() services.LLMService
	Storage() StorageInterface
	Config() ConfigInterface
}

type StorageInterface interface {
	WriteNode(node models.Node) error
}

type ConfigInterface interface {
	GetGitConfig() GitConfigInterface
}

type GitConfigInterface interface {
	GetDefaultRepo() string
}

type Coordinator struct {
	app AppInterface
}

func NewCoordinator(app AppInterface) *Coordinator {
	return &Coordinator{
		app: app,
	}
}

func (c *Coordinator) LoadSpecsCmd() tea.Cmd {
	return func() tea.Msg {
		nodes, err := c.app.SpecService().ListNodes()
		if err != nil {
			return NodesLoadedMsg{err: err}
		}

		var nodeItems []Spec
		for _, node := range nodes {
			nodeItems = append(nodeItems, Spec{
				ID:      node.ID(),
				Title:   node.Title(),
				Content: node.Content(),
				Type:    node.Type(),
			})
		}

		return NodesLoadedMsg{nodes: nodeItems}
	}
}

func (c *Coordinator) CreateNodeCmd(title, content, nodeType, parentSpecID string) tea.Cmd {
	switch nodeType {
	case "specification":
		return c.createSpecCmd(title, content, parentSpecID)
	case "implementation":
		return c.createImplementationCmd(title, content, parentSpecID, nil, nil, nil)
	case "project":
		return c.createProjectCmd(title, content, parentSpecID)
	default:
		return func() tea.Msg {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: Unknown node type '%s'. Press Enter to continue...", nodeType)}
		}
	}
}

func (c *Coordinator) CreateImplementationCmd(title, content, parentSpecID string, repoURL, branch, folderPath *string) tea.Cmd {
	return c.createImplementationCmd(title, content, parentSpecID, repoURL, branch, folderPath)
}

func (c *Coordinator) UpdateNodeCmd(nodeID, title, content, nodeType string, implRepoURL, implBranch, implFolderPath *string) tea.Cmd {
	switch nodeType {
	case "specification":
		return c.updateSpecCmd(nodeID, title, content)
	case "implementation":
		return c.updateImplementationCmd(nodeID, title, content, implRepoURL, implBranch, implFolderPath)
	case "project":
		return c.updateProjectCmd(nodeID, title, content)
	default:
		return func() tea.Msg {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: Unknown node type '%s'. Press Enter to continue...", nodeType)}
		}
	}
}

func (c *Coordinator) DeleteSpecCmd(specID string) tea.Cmd {
	return func() tea.Msg {
		if err := c.app.SpecService().DeleteSpec(specID); err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}
		return ReturnToSpecListMsg{}
	}
}

func (c *Coordinator) DeleteLinkCmd(specID, commitID, repoPath string) tea.Cmd {
	return func() tea.Msg {
		if err := c.app.LinkService().UnlinkSpecFromCommit(specID, commitID, repoPath); err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}
		return ReturnToSpecListMsg{}
	}
}

func (c *Coordinator) OrganizeNodeCmd(nodeID string) tea.Cmd {
	return func() tea.Msg {
		if err := c.app.SpecService().OrganizeNodes(nodeID); err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error organizing node: %v. Press Enter to continue...", err)}
		}
		return ReturnToSpecListMsg{}
	}
}

func (c *Coordinator) SetSlugAndOrganizeCmd(nodeID, slug string) tea.Cmd {
	return func() tea.Msg {
		node, err := c.app.SpecService().ReadNode(nodeID)
		if err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error getting node: %v. Press Enter to continue...", err)}
		}

		node.SetSlug(slug)
		if err := c.app.Storage().WriteNode(node); err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error updating slug: %v. Press Enter to continue...", err)}
		}

		if err := c.app.SpecService().OrganizeNodes(nodeID); err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error organizing node: %v. Press Enter to continue...", err)}
		}

		return ReturnToSpecListMsg{}
	}
}

// createProjectCmd returns a command to create a new project
func (c *Coordinator) createProjectCmd(title, content, parentSpecID string) tea.Cmd {
	return func() tea.Msg {
		project, err := c.app.SpecService().CreateProject(title, content)
		if err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}

		if parentSpecID != "" {
			_, err := c.app.SpecService().AddChildToParent(project.ID(), parentSpecID, "child")
			if err != nil {
				return OperationCompleteMsg{message: fmt.Sprintf("Error creating parent-child relationship: %v. Press Enter to continue...", err)}
			}
		}

		return NavigateToNodeMsg{nodeID: project.ID()}
	}
}

// createSpecCmd returns a command to create a new spec
func (c *Coordinator) createSpecCmd(title, content, parentSpecID string) tea.Cmd {
	return func() tea.Msg {
		spec, err := c.app.SpecService().CreateSpec(title, content)
		if err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}

		if parentSpecID != "" {
			_, err := c.app.SpecService().AddChildToParent(spec.ID(), parentSpecID, "child")
			if err != nil {
				return OperationCompleteMsg{message: fmt.Sprintf("Error creating parent-child relationship: %v. Press Enter to continue...", err)}
			}
		}

		return NavigateToNodeMsg{nodeID: spec.ID()}
	}
}

// createImplementationCmd returns a command to create a new implementation node
func (c *Coordinator) createImplementationCmd(title, content, parentSpecID string, repoURL, branch, folderPath *string) tea.Cmd {
	return func() tea.Msg {
		impl, err := c.app.SpecService().CreateImplementation(title, content, repoURL, branch, folderPath)
		if err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}

		if parentSpecID != "" {
			_, err := c.app.SpecService().AddChildToParent(impl.ID(), parentSpecID, "child")
			if err != nil {
				return OperationCompleteMsg{message: fmt.Sprintf("Error creating parent-child relationship: %v. Press Enter to continue...", err)}
			}
		}

		return NavigateToNodeMsg{nodeID: impl.ID()}
	}
}

// updateSpecCmd returns a command to update an existing spec
func (c *Coordinator) updateSpecCmd(specID, title, content string) tea.Cmd {
	return func() tea.Msg {
		_, err := c.app.SpecService().UpdateSpec(specID, title, content)
		if err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}
		return NavigateToNodeMsg{nodeID: specID}
	}
}

// updateImplementationCmd returns a command to update an existing implementation
func (c *Coordinator) updateImplementationCmd(nodeID, title, content string, repoURL, branch, folderPath *string) tea.Cmd {
	return func() tea.Msg {
		_, err := c.app.SpecService().UpdateImplementation(nodeID, title, content, repoURL, branch, folderPath)
		if err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}
		return NavigateToNodeMsg{nodeID: nodeID}
	}
}

// updateProjectCmd returns a command to update an existing project
func (c *Coordinator) updateProjectCmd(nodeID, title, content string) tea.Cmd {
	return func() tea.Msg {
		_, err := c.app.SpecService().WriteNode(nodeID, title, content)
		if err != nil {
			return OperationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}
		return NavigateToNodeMsg{nodeID: nodeID}
	}
}

// combinedService adapts both LinkService and SpecService to provide
// the interface needed by speclistview
type combinedService struct {
	linkService services.LinkService
	specService services.SpecService
}

func (cs *combinedService) GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error) {
	return cs.linkService.GetCommitsForSpec(specID)
}

func (cs *combinedService) GetChildNodes(specID string) ([]models.Node, error) {
	nodes, err := cs.specService.GetChildren(specID)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (cs *combinedService) GetNodeByID(specID string) (models.Node, error) {
	node, err := cs.specService.ReadNode(specID)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (cs *combinedService) GetParentNode(specID string) (models.Node, error) {
	parents, err := cs.specService.GetParents(specID)
	if err != nil {
		return nil, err
	}

	if len(parents) == 0 {
		return nil, nil
	}

	return parents[0], nil
}

func (cs *combinedService) GetRootNode() (models.Node, error) {
	rootNode, err := cs.specService.GetRootNode()
	if err != nil {
		return nil, err
	}
	if rootNode == nil {
		return nil, nil
	}
	return rootNode, nil
}

func NewCombinedService(linkService services.LinkService, specService services.SpecService) *combinedService {
	return &combinedService{
		linkService: linkService,
		specService: specService,
	}
}
