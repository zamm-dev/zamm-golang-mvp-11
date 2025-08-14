package interactive

// MenuState represents the current state of the interactive menu
type MenuState int

const (
	SpecListView MenuState = iota
	NodeTypeSelection
	LinkSelection
	NodeEditor
	ImplementationForm
	ConfirmDelete
	LinkEditor
	UnlinkEditor
	SlugEditor
)

// Spec is a shared data structure representing a specification
// for display in different parts of the interactive UI.
type Spec struct {
	ID      string
	Title   string
	Content string
	Type    string
}

func (s Spec) FilterValue() string {
	return s.Title
}
