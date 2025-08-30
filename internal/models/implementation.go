package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type ImplementationJSON struct {
	NodeBaseJSON
	RepoURL    *string `json:"repo_url,omitempty"`
	Branch     *string `json:"branch,omitempty"`
	FolderPath *string `json:"folder_path,omitempty"`
}

// Implementation represents an implementation node in the system
type Implementation struct {
	NodeBase
	RepoURL    *string
	Branch     *string
	FolderPath *string
}

// NewImplementation creates a new Implementation with the type field set
func NewImplementation(title, content string) *Implementation {
	return &Implementation{
		NodeBase: NodeBase{
			id:       uuid.New().String(),
			title:    title,
			content:  content,
			nodeType: "implementation",
		},
	}
}

func (impl *Implementation) MarshalJSON() ([]byte, error) {
	return json.Marshal(&ImplementationJSON{
		NodeBaseJSON: impl.asBaseJsonStruct(),
		RepoURL:      impl.RepoURL,
		Branch:       impl.Branch,
		FolderPath:   impl.FolderPath,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Implementation
func (impl *Implementation) UnmarshalJSON(data []byte) error {
	// First unmarshal into a temporary struct that has all fields
	var temp ImplementationJSON

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Set the NodeBase fields
	impl.fromBaseJsonStruct(temp.NodeBaseJSON)

	// Set the Implementation specific fields
	impl.RepoURL = temp.RepoURL
	impl.Branch = temp.Branch
	impl.FolderPath = temp.FolderPath

	return nil
}
