package storagetest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

type AdvancedStorage interface {
	WriteNodeWithExtraData(node models.Node, extraData string) error
	ReadNode(id string) (models.Node, error)
}

func TestFileStorage_CreateNodeWithExtraData(t *testing.T) {
	tempDir := t.TempDir()
	storage := storage.NewFileStorage(tempDir)

	node := models.NewSpec("Test Spec", "This should stay the same.")
	err := storage.WriteNodeWithExtraData(node, "\n---\n\nExtra data")
	assert.NoError(t, err)

	// check that regular data is read back in
	readNode, err := storage.ReadNode(node.ID())
	assert.NoError(t, err)
	assert.Equal(t, readNode.Title(), node.Title())
	assert.Equal(t, readNode.Content(), node.Content())

	// check that the file itself contains the extra data
	fileContent, err := os.ReadFile(storage.GetNodeFilePath(node.ID()))
	assert.NoError(t, err)
	assert.Contains(t, string(fileContent), "Extra data")
}
