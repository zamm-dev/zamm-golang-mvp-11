package nodes

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
)

// DebugMsg is a generic debug message for logging
// It can be reused for any debugging purpose
// Only contains a string message
type DebugMsg struct {
	Message string
}

// NodeDetailView manages the viewport for a SpecDetail
// All state and update logic is delegated to SpecDetail
// Only the viewport and passthrough logic remain here
type NodeDetailView struct {
	detail   *NodeDetail
	viewport viewport.Model
	width    int
	height   int
}

func NewNodeDetailView() NodeDetailView {
	return NodeDetailView{
		detail:   NewNodeDetail(),
		viewport: viewport.New(0, 0),
	}
}

func (v *NodeDetailView) Init() tea.Cmd {
	return nil
}

func (v *NodeDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height
	v.detail.SetSize(width, height)
	v.viewport.SetContent(v.detail.View())
}

func (v *NodeDetailView) SetSpec(node models.Node, links []*models.SpecCommitLink, childNodes []models.Node) {
	v.detail.SetSpec(node, links, childNodes)
	v.viewport.SetContent(v.detail.View())
	v.viewport.SetYOffset(0)
}

func (v *NodeDetailView) GetSelectedChild() models.Node {
	return v.detail.GetSelectedChild()
}

func (v *NodeDetailView) SelectNextChild() {
	v.detail.SelectNextChild()
	v.viewport.SetContent(v.detail.View())
}

func (v *NodeDetailView) SelectPrevChild() {
	v.detail.SelectPrevChild()
	v.viewport.SetContent(v.detail.View())
}

func (v *NodeDetailView) ResetCursor() {
	v.detail.ResetCursor()
	v.viewport.SetContent(v.detail.View())
}

func (v *NodeDetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	beforeMsg := DebugMsg{
		Message: fmt.Sprintf(
			"[NodeDetailView] BEFORE | Msg: %T | YOffset: %d | Height: %d | Total lines: %d | Visible lines: %d",
			msg, v.viewport.YOffset, v.viewport.Height, v.viewport.TotalLineCount(), v.viewport.VisibleLineCount(),
		),
	}
	cmds = append(cmds, func() tea.Msg { return beforeMsg })
	v.viewport, _ = v.viewport.Update(msg)
	afterMsg := DebugMsg{
		Message: fmt.Sprintf(
			"[NodeDetailView] AFTER  | Msg: %T | YOffset: %d | Height: %d | Total lines: %d | Visible lines: %d",
			msg, v.viewport.YOffset, v.viewport.Height, v.viewport.TotalLineCount(), v.viewport.VisibleLineCount(),
		),
	}
	cmds = append(cmds, func() tea.Msg { return afterMsg })
	return v, tea.Batch(cmds...)
}

func (v *NodeDetailView) View() string {
	v.viewport.SetContent(v.detail.View())
	return v.viewport.View()
}
