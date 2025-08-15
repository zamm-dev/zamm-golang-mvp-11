package nodes

// Message types for communication
type CreateNewSpecMsg struct {
	ParentSpecID string // ID of parent spec
}

type LinkCommitSpecMsg struct {
	SpecID string
}

type EditSpecMsg struct {
	SpecID string
}

type DeleteSpecMsg struct {
	SpecID string
}

type RemoveLinkSpecMsg struct {
	SpecID string
}

type MoveSpecMsg struct {
	SpecID string
}

type OrganizeSpecMsg struct {
	SpecID string
}

type EditSlugMsg struct {
	SpecID        string
	OriginalTitle string
	InitialSlug   string
}

type OpenMarkdownMsg struct {
	SpecID string
}

type ExitMsg struct{}
