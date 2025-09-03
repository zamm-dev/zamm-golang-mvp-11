package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type implementationJSON struct {
	nodeBaseJSON
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
	return json.Marshal(&implementationJSON{
		nodeBaseJSON: impl.asBaseJsonStruct(),
		RepoURL:      impl.RepoURL,
		Branch:       impl.Branch,
		FolderPath:   impl.FolderPath,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Implementation
func (impl *Implementation) UnmarshalJSON(data []byte) error {
	// First unmarshal into a temporary struct that has all fields
	var temp implementationJSON

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Set the NodeBase fields
	impl.fromBaseJsonStruct(temp.nodeBaseJSON)

	// Set the Implementation specific fields
	impl.RepoURL = temp.RepoURL
	impl.Branch = temp.Branch
	impl.FolderPath = temp.FolderPath

	return nil
}
