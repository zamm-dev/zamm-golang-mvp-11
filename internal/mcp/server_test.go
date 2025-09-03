package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

func setupTestStorage(t *testing.T) (storage.Storage, string) {
	tempDir := t.TempDir()
	testDataDir := filepath.Join(tempDir, ".zamm")

	srcDir := "../cli/interactive/common/testdata/.zamm"
	err := os.CopyFS(testDataDir, os.DirFS(srcDir))
	require.NoError(t, err, "Failed to copy testdata")

	store := storage.NewFileStorage(testDataDir)

	return store, testDataDir
}

func TestCreateChildSpec_Success(t *testing.T) {
	store, _ := setupTestStorage(t)
	specService := services.NewSpecService(store)
	server := NewServer(specService)

	parentSpec, err := specService.CreateSpec("Parent Spec", "Parent content")
	require.NoError(t, err)

	args := CreateChildSpecArgs{
		ParentID: parentSpec.ID(),
		Title:    "Child Spec",
		Content:  "Child content",
	}

	params := &mcp.CallToolParamsFor[CreateChildSpecArgs]{
		Name:      "create_child_spec",
		Arguments: args,
	}

	ctx := context.Background()
	result, err := server.CreateChildSpec(ctx, nil, params)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "Expected TextContent")

	var resultData CreateChildSpecResult
	err = json.Unmarshal([]byte(textContent.Text), &resultData)
	require.NoError(t, err, "Failed to unmarshal result JSON")

	assert.Equal(t, parentSpec.ID(), resultData.ParentID)
	assert.Equal(t, "Child Spec", resultData.Title)
	assert.Equal(t, "Child content", resultData.Content)
	assert.NotEmpty(t, resultData.ChildID)
	assert.Contains(t, resultData.Message, "Successfully created child spec")
	assert.Contains(t, resultData.Message, "Child Spec")
	assert.Contains(t, resultData.Message, "Parent Spec")

	childSpec, err := specService.GetNode(resultData.ChildID)
	require.NoError(t, err, "Child spec should exist in storage")
	assert.Equal(t, "Child Spec", childSpec.Title())
	assert.Equal(t, "Child content", childSpec.Content())

	children, err := specService.GetChildren(parentSpec.ID())
	require.NoError(t, err)
	require.Len(t, children, 1)
	assert.Equal(t, resultData.ChildID, children[0].ID())

	childFound := false
	for _, child := range children {
		if child.ID() == resultData.ChildID {
			childFound = true
			break
		}
	}
	assert.True(t, childFound, "Child spec should be linked to parent")
}

func TestCreateChildSpec_InvalidParentID(t *testing.T) {
	store, _ := setupTestStorage(t)
	specService := services.NewSpecService(store)
	server := NewServer(specService)

	args := CreateChildSpecArgs{
		ParentID: "nonexistent-id",
		Title:    "Child Spec",
		Content:  "Child content",
	}

	params := &mcp.CallToolParamsFor[CreateChildSpecArgs]{
		Name:      "create_child_spec",
		Arguments: args,
	}

	ctx := context.Background()
	result, err := server.CreateChildSpec(ctx, nil, params)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "Expected TextContent")

	assert.Contains(t, textContent.Text, "Error: Parent spec with ID 'nonexistent-id' not found")
}

func TestCreateChildSpec_EmptyTitle(t *testing.T) {
	store, _ := setupTestStorage(t)
	specService := services.NewSpecService(store)
	server := NewServer(specService)

	parentSpec, err := specService.CreateSpec("Parent Spec", "Parent content")
	require.NoError(t, err)

	args := CreateChildSpecArgs{
		ParentID: parentSpec.ID(),
		Title:    "",
		Content:  "Child content",
	}

	params := &mcp.CallToolParamsFor[CreateChildSpecArgs]{
		Name:      "create_child_spec",
		Arguments: args,
	}

	ctx := context.Background()
	result, err := server.CreateChildSpec(ctx, nil, params)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "Expected TextContent")

	assert.Contains(t, textContent.Text, "Error creating child spec")
}

func TestNewServer(t *testing.T) {
	store, _ := setupTestStorage(t)
	specService := services.NewSpecService(store)

	server := NewServer(specService)

	assert.NotNil(t, server)
	assert.Equal(t, specService, server.specService)
}
