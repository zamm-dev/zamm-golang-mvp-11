package file_storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

// testSetup holds the setup data for a test
type testSetup struct {
	fs           *storage.FileStorage
	spec         *models.Spec
	originalPath string
	tempDir      string
}

// setupTestSpec creates a test spec and returns the test setup
func setupTestSpec(t *testing.T) testSetup {
	t.Helper()

	tempDir := t.TempDir()
	fs, err := storage.New(tempDir)
	require.NoError(t, err)

	spec := models.NewSpec("Test Spec", "Test content for move operation")
	testCreateNode(t, fs, spec)
	originalPath := fs.GetNodeFilePath(spec.ID())

	return testSetup{
		fs:           fs,
		spec:         spec,
		originalPath: originalPath,
		tempDir:      tempDir,
	}
}

// moveNodeFileAndVerify moves a file and verifies the move operation, returns the expected new file location
func moveNodeFileAndVerify(t *testing.T, setup testSetup, newPath string) string {
	t.Helper()

	err := setup.fs.MoveNodeFile(setup.spec, newPath)
	require.NoError(t, err)

	// Calculate the expected new file location based on tempDir and newPath
	expectedNewPath := filepath.Join(filepath.Dir(setup.tempDir), newPath)
	verifyFileMove(t, setup.originalPath, expectedNewPath)

	return expectedNewPath
}

// verifyFileMove checks that a file has been moved from originalPath to newPath
func verifyFileMove(t *testing.T, originalPath, newPath string) {
	t.Helper()

	// Verify the file was moved to the new location
	_, err := os.Stat(newPath)
	assert.NoError(t, err, "file should exist at new location")

	// Verify the old file no longer exists
	_, err = os.Stat(originalPath)
	assert.True(t, os.IsNotExist(err), "file should not exist at original location")
}

// verifyFilesDeleted checks that both the original and moved files are deleted
func verifyFilesDeleted(t *testing.T, originalPath, newPath string) {
	t.Helper()

	// Verify the file at new location is deleted
	_, err := os.Stat(newPath)
	assert.True(t, os.IsNotExist(err), "file should not exist at new location after deletion")

	// Verify the original file is still deleted
	_, err = os.Stat(originalPath)
	assert.True(t, os.IsNotExist(err), "file should not exist at original location after deletion")
}

func TestMoveNodeFile(t *testing.T) {
	tests := []struct {
		name    string
		newPath string
	}{
		{
			name:    "move spec to new location",
			newPath: "docs/moved-spec.md",
		},
		{
			name:    "move spec to subdirectory",
			newPath: "docs/category/new-location.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" update", func(t *testing.T) {
			setup := setupTestSpec(t)
			movedPath := moveNodeFileAndVerify(t, setup, tt.newPath)

			// Verify the update operation works on the new path
			testUpdateNode(t, setup.fs, setup.spec)
			verifyFileMove(t, setup.originalPath, movedPath)
		})

		t.Run(tt.name+" delete", func(t *testing.T) {
			setup := setupTestSpec(t)
			movedPath := moveNodeFileAndVerify(t, setup, tt.newPath)

			// Verify the delete operation works on the new path
			testDeleteNode(t, setup.fs, setup.spec)
			verifyFilesDeleted(t, setup.originalPath, movedPath)
		})
	}
}
