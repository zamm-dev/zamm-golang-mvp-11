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

// verifyFileMove checks that a file has been moved from originalPath to newPath
func verifyFileMove(t *testing.T, tempDir, originalPath, newPath string) {
	t.Helper()

	// Verify the file was moved to the new location
	expectedNewPath := filepath.Join(filepath.Dir(tempDir), newPath)
	_, err := os.Stat(expectedNewPath)
	assert.NoError(t, err, "file should exist at new location")

	// Verify the old file no longer exists
	_, err = os.Stat(originalPath)
	assert.True(t, os.IsNotExist(err), "file should not exist at original location")
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
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			fs, err := storage.New(tempDir)
			require.NoError(t, err)

			spec := models.NewSpec("Test Spec", "Test content for move operation")
			testCreateNode(t, fs, spec)
			originalPath := fs.GetNodeFilePath(spec.ID())

			err = fs.MoveNodeFile(spec, tt.newPath)
			require.NoError(t, err)
			verifyFileMove(t, tempDir, originalPath, tt.newPath)

			testUpdateNode(t, fs, spec)
			// Verify the file is still at the correct location after update
			verifyFileMove(t, tempDir, originalPath, tt.newPath)
		})
	}
}
