package interactive

// Spec is a shared data structure representing a specification
// for display in different parts of the interactive UI.
type Spec struct {
	ID      string
	Title   string
	Content string
}

func (s Spec) FilterValue() string {
	return s.Title
}
