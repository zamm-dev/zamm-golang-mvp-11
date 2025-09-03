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

var testTitle = "Basic RWD Test Node"
var testDescription = "This is a test node for basic read, write, and delete operations."
var testSlug = "basic-rwd-test"

func GenericStorage_TestCreateNode(t *testing.T, storage Storage, node models.Node) {
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

func GenericStorage_TestUpdateNode(t *testing.T, storage Storage, node models.Node) {
	newTestTitle := "Overwritten Test Node"
	newTestDescription := "Overwritten update."
	newTestSlug := "new-rwd-test"

	node.SetTitle(testTitle)
	node.SetContent(testDescription)
	node.SetSlug(testSlug)

	// set up new node
	err := storage.WriteNode(node)
	assert.NoError(t, err)

	// first, verify that we have old data
	readNode, err := storage.ReadNode(node.ID())
	assert.NoError(t, err)
	assert.Equal(t, readNode.Title(), testTitle)
	assert.Equal(t, readNode.Content(), testDescription)
	assert.Equal(t, readNode.Slug(), testSlug)

	// now, update the node
	node.SetTitle(newTestTitle)
	node.SetContent(newTestDescription)
	node.SetSlug(newTestSlug)

	err = storage.WriteNode(node)
	assert.NoError(t, err)

	// finally, verify that we have new data
	readNode, err = storage.ReadNode(node.ID())
	assert.NoError(t, err)
	assert.Equal(t, readNode.Title(), newTestTitle)
	assert.Equal(t, readNode.Content(), newTestDescription)
	assert.Equal(t, readNode.Slug(), newTestSlug)
}

func GenericStorage_TestDeleteNode(t *testing.T, storage Storage, node models.Node) {
	node.SetTitle(testTitle)
	node.SetContent(testDescription)
	node.SetSlug(testSlug)

	// set up new node
	err := storage.WriteNode(node)
	assert.NoError(t, err)

	// first, verify that the node exists and we can read it
	readNode, err := storage.ReadNode(node.ID())
	assert.NoError(t, err)
	assert.Equal(t, readNode.Title(), testTitle)
	assert.Equal(t, readNode.Content(), testDescription)
	assert.Equal(t, readNode.Slug(), testSlug)

	// now, delete the node
	err = storage.DeleteNode(node.ID())
	assert.NoError(t, err)

	// finally, verify that the node has been deleted
	_, err = storage.ReadNode(node.ID())
	assert.Error(t, err)
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

func TestFileStorage_UpdateNode(t *testing.T) {
	tempDir := t.TempDir()
	storage := storage.NewFileStorage(tempDir)

	node := models.NewSpec("Test Spec", "This should be overwritten.")
	GenericStorage_TestUpdateNode(t, storage, node)

	// check that actual file exists
	_, err := os.Stat(storage.GetNodeFilePath(node.ID()))
	assert.NoError(t, err)
}

func TestFileStorage_DeleteNode(t *testing.T) {
	tempDir := t.TempDir()
	storage := storage.NewFileStorage(tempDir)

	node := models.NewSpec("Test Spec", "This should be deleted.")
	GenericStorage_TestDeleteNode(t, storage, node)

	// check that actual file no longer exists
	_, err := os.Stat(storage.GetNodeFilePath(node.ID()))
	assert.Error(t, err)
}
