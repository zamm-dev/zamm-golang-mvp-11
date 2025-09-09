package file_storage_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

// testStorage defines the minimal storage interface for testing
type testStorage interface {
	WriteNode(node models.Node) error
	ReadNode(id string) (models.Node, error)
	DeleteNode(id string) error
}

// nodeTestData represents test data for a node
type nodeTestData struct {
	title       string
	description string
	slug        string
}

// defaultNodeData returns default test data for nodes
func defaultNodeData() nodeTestData {
	return nodeTestData{
		title:       "Basic RWD Test Node",
		description: "This is a test node for basic read, write, and delete operations.",
		slug:        "basic-rwd-test",
	}
}

// testCreateNode tests basic node creation functionality
func testCreateNode(t *testing.T, store testStorage, node models.Node) {
	t.Helper()

	data := defaultNodeData()
	setupNode(node, data)

	// Verify that the node can't be read initially
	_, err := store.ReadNode(node.ID())
	assert.Error(t, err, "node should not exist before creation")

	// Write the node
	err = store.WriteNode(node)
	require.NoError(t, err, "failed to write node")

	// Read and verify the node
	readNode, err := store.ReadNode(node.ID())
	require.NoError(t, err, "failed to read node after creation")
	assertNodeData(t, readNode, data)
}

// testUpdateNode tests node update functionality
func testUpdateNode(t *testing.T, store testStorage, node models.Node) {
	t.Helper()

	originalData := defaultNodeData()
	updatedData := nodeTestData{
		title:       "Overwritten Test Node",
		description: "Overwritten update.",
		slug:        "new-rwd-test",
	}

	// Set up initial node
	setupNode(node, originalData)
	err := store.WriteNode(node)
	require.NoError(t, err, "failed to write initial node")

	// Verify initial data
	readNode, err := store.ReadNode(node.ID())
	require.NoError(t, err, "failed to read initial node")
	assertNodeData(t, readNode, originalData)

	// Update the node
	setupNode(node, updatedData)
	err = store.WriteNode(node)
	require.NoError(t, err, "failed to update node")

	// Verify updated data
	readNode, err = store.ReadNode(node.ID())
	require.NoError(t, err, "failed to read updated node")
	assertNodeData(t, readNode, updatedData)
}

// testDeleteNode tests node deletion functionality
func testDeleteNode(t *testing.T, store testStorage, node models.Node) {
	t.Helper()

	data := defaultNodeData()
	setupNode(node, data)

	// Set up the node
	err := store.WriteNode(node)
	require.NoError(t, err, "failed to write node")

	// Verify the node exists
	readNode, err := store.ReadNode(node.ID())
	require.NoError(t, err, "failed to read node before deletion")
	assertNodeData(t, readNode, data)

	// Delete the node
	err = store.DeleteNode(node.ID())
	require.NoError(t, err, "failed to delete node")

	// Verify the node has been deleted
	_, err = store.ReadNode(node.ID())
	assert.Error(t, err, "node should not exist after deletion")
}

// setupNode configures a node with the given test data
func setupNode(node models.Node, data nodeTestData) {
	node.SetTitle(data.title)
	node.SetContent(data.description)
	node.SetSlug(data.slug)
}

// assertNodeData verifies that a node contains the expected data
func assertNodeData(t *testing.T, node models.Node, expected nodeTestData) {
	t.Helper()
	assert.Equal(t, expected.title, node.Title(), "node title mismatch")
	assert.Equal(t, expected.description, node.Content(), "node content mismatch")
	assert.Equal(t, expected.slug, node.Slug(), "node slug mismatch")
}

// assertFileExists verifies that a node's file exists on disk
func assertFileExists(t *testing.T, store *storage.FileStorage, node models.Node, message string) {
	t.Helper()
	_, err := os.Stat(store.GetNodeFilePath(node.ID()))
	assert.NoError(t, err, message)
}

func assertNoFile(t *testing.T, store *storage.FileStorage, node models.Node, message string) {
	t.Helper()
	_, err := os.Stat(store.GetNodeFilePath(node.ID()))
	assert.Error(t, err, message)
}

// testProjectNode tests project-specific functionality
func testProjectNode(t *testing.T, store testStorage, node *models.Project) {
	t.Helper()

	// Test generic node operations on project node
	testCreateNode(t, store, node)

	// Test project-specific fields
	readNode, err := store.ReadNode(node.ID())
	require.NoError(t, err, "failed to read project node")
	assert.Equal(t, "project", readNode.Type(), "node type should be 'project'")

	_, ok := readNode.(*models.Project)
	assert.True(t, ok, "read node should be a Project type")
}

// testImplementationNode tests implementation-specific functionality
func testImplementationNode(t *testing.T, store testStorage, node *models.Implementation) {
	t.Helper()

	// Set up implementation-specific fields
	repoURL := "http://github.com/example/repo"
	branch := "main"
	folderPath := "/path/to/project"
	node.RepoURL = &repoURL
	node.Branch = &branch
	node.FolderPath = &folderPath

	// Test generic node operations on implementation node
	testCreateNode(t, store, node)

	// Test implementation-specific fields
	readNode, err := store.ReadNode(node.ID())
	require.NoError(t, err, "failed to read implementation node")
	assert.Equal(t, "implementation", readNode.Type(), "node type should be 'implementation'")

	readImpl, ok := readNode.(*models.Implementation)
	require.True(t, ok, "read node should be an Implementation type")
	assert.Equal(t, repoURL, *readImpl.RepoURL, "repo URL mismatch")
	assert.Equal(t, branch, *readImpl.Branch, "branch mismatch")
	assert.Equal(t, folderPath, *readImpl.FolderPath, "folder path mismatch")
}

func TestFileStorage_NodeOperations(t *testing.T) {
	tests := []struct {
		name     string
		nodeType string
		createFn func() models.Node
	}{
		{
			name:     "spec",
			nodeType: "spec",
			createFn: func() models.Node {
				return models.NewSpec("Test Spec", "This should be overwritten.")
			},
		},
		{
			name:     "project",
			nodeType: "project",
			createFn: func() models.Node {
				return models.NewProject("Test Project", "This should be overwritten.")
			},
		},
		{
			name:     "implementation",
			nodeType: "implementation",
			createFn: func() models.Node {
				return models.NewImplementation("Test Implementation", "This should be overwritten.")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			store := storage.NewFileStorage(tempDir)

			t.Run("create", func(t *testing.T) {
				node := tt.createFn()
				testCreateNode(t, store, node)
				assertFileExists(t, store, node, "node file should exist on disk")
			})

			t.Run("update", func(t *testing.T) {
				node := tt.createFn()
				testUpdateNode(t, store, node)
				assertFileExists(t, store, node, "node file should exist on disk after update")
			})

			t.Run("delete", func(t *testing.T) {
				node := tt.createFn()
				testDeleteNode(t, store, node)
				assertNoFile(t, store, node, "node file should not exist on disk after deletion")
			})
		})
	}
}

func TestFileStorage_ProjectNode(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)

	node := models.NewProject("Test Project", "This should be overwritten.")
	testProjectNode(t, store, node)

	assertFileExists(t, store, node, "project node file should exist on disk")
}

func TestFileStorage_ImplementationNode(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)

	node := models.NewImplementation("Test Implementation", "This should be overwritten.")
	testImplementationNode(t, store, node)

	assertFileExists(t, store, node, "implementation node file should exist on disk")
}
