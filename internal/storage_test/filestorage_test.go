package storagetest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

type Storage interface {
	WriteNode(node models.Node) error
	ReadNode(id string) (models.Node, error)
	DeleteNode(id string) error
}

func GenericStorage_TestCreateNode(t *testing.T, storage Storage, node models.Node) {
	testTitle := "Basic RWD Test Node"
	testDescription := "This is a test node for basic read, write, and delete operations."
	testSlug := "basic-rwd-test"

	node.SetTitle(testTitle)
	node.SetContent(testDescription)
	node.SetSlug(testSlug)

	// verify that the node can't be read in yet
	_, err := storage.ReadNode(node.ID())
	assert.Error(t, err)

	err = storage.WriteNode(node)
	assert.NoError(t, err)

	readNode, err := storage.ReadNode(node.ID())
	assert.NoError(t, err)
	assert.Equal(t, readNode.Title(), testTitle)
	assert.Equal(t, readNode.Content(), testDescription)
	assert.Equal(t, readNode.Slug(), testSlug)
}

func TestFileStorage_CreateNode(t *testing.T) {
	tempDir := t.TempDir()
	storage := storage.NewFileStorage(tempDir)

	node := models.NewSpec("Test Spec", "This should be overwritten.")
	GenericStorage_TestCreateNode(t, storage, node)

	// check that actual file exists
	_, err := os.Stat(storage.GetNodeFilePath(node.ID()))
	assert.NoError(t, err)
}
