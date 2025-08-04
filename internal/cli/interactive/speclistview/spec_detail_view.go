package speclistview

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// DebugMsg is a generic debug message for logging
// It can be reused for any debugging purpose
// Only contains a string message
type DebugMsg struct {
	Message string
}

// SpecDetailView manages the viewport for a SpecDetail
// All state and update logic is delegated to SpecDetail
// Only the viewport and passthrough logic remain here
type SpecDetailView struct {
	detail   *SpecDetail
	viewport viewport.Model
	width    int
	height   int
}

func NewSpecDetailView() SpecDetailView {
	return SpecDetailView{
		detail:   NewSpecDetail(),
		viewport: viewport.New(0, 0),
	}
}

func (v *SpecDetailView) Init() tea.Cmd {
	return nil
}

func (v *SpecDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height
	v.detail.SetSize(width, height)
	v.viewport.SetContent(v.detail.View())
}

func (v *SpecDetailView) SetSpec(spec models.Spec, links []*models.SpecCommitLink, childSpecs []*models.Spec) {
	v.detail.SetSpec(spec, links, childSpecs)
	v.viewport.SetContent(v.detail.View())
	v.viewport.SetYOffset(0)
}

func (v *SpecDetailView) GetSelectedChild() *models.Spec {
	return v.detail.GetSelectedChild()
}

func (v *SpecDetailView) SelectNextChild() {
	v.detail.SelectNextChild()
	v.viewport.SetContent(v.detail.View())
}

func (v *SpecDetailView) SelectPrevChild() {
	v.detail.SelectPrevChild()
	v.viewport.SetContent(v.detail.View())
}

func (v *SpecDetailView) ResetCursor() {
	v.detail.ResetCursor()
	v.viewport.SetContent(v.detail.View())
}

func (v *SpecDetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	beforeMsg := DebugMsg{
		Message: fmt.Sprintf(
			"[SpecDetailView] BEFORE | Msg: %T | YOffset: %d | Height: %d | Total lines: %d | Visible lines: %d",
			msg, v.viewport.YOffset, v.viewport.Height, v.viewport.TotalLineCount(), v.viewport.VisibleLineCount(),
		),
	}
	cmds = append(cmds, func() tea.Msg { return beforeMsg })
	v.viewport, _ = v.viewport.Update(msg)
	afterMsg := DebugMsg{
		Message: fmt.Sprintf(
			"[SpecDetailView] AFTER  | Msg: %T | YOffset: %d | Height: %d | Total lines: %d | Visible lines: %d",
			msg, v.viewport.YOffset, v.viewport.Height, v.viewport.TotalLineCount(), v.viewport.VisibleLineCount(),
		),
	}
	cmds = append(cmds, func() tea.Msg { return afterMsg })
	return v, tea.Batch(cmds...)
}

func (v *SpecDetailView) View() string {
	v.viewport.SetContent(v.detail.View())
	return v.viewport.View()
}
