package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// setupTestService creates a new test storage with temporary database
func setupTestService(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	err = storage.RunMigrationsIfNeeded()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		storage.Close()
		// os.RemoveAll(tmpDir)
	}

	return storage, cleanup
}

// createTestSpec creates a test specification with default values
func createTestSpec() *models.SpecNode {
	return &models.SpecNode{
		Title:   "Test Spec",
		Content: "This is a test specification",
	}
}

// createTestLink creates a test link with default values
func createTestLink(specID string) *models.SpecCommitLink {
	return &models.SpecCommitLink{
		SpecID:   specID,
		CommitID: "abcdef1234567890abcdef1234567890abcdef12",
		RepoPath: "/test/repo",
		LinkType: "implements",
	}
}

// Test NewSQLiteStorage
func TestNewSQLiteStorage(t *testing.T) {
	t.Run("success with valid path", func(t *testing.T) {
		storage, err := NewSQLiteStorage(":memory:")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		defer storage.Close()

		if storage == nil {
			t.Fatal("Expected storage to be non-nil")
		}
	})

	t.Run("error with invalid path", func(t *testing.T) {
		// Try to create database in non-existent directory
		_, err := NewSQLiteStorage("/nonexistent/path/test.db")
		if err == nil {
			t.Fatal("Expected error for invalid path")
		}

		// Accept either "failed to open database" or "failed to connect to database"
		if !strings.Contains(err.Error(), "failed to open database") && !strings.Contains(err.Error(), "failed to connect to database") {
			t.Errorf("Expected database connection error, got %v", err)
		}
	})
}

// Test CreateSpec
func TestCreateSpec(t *testing.T) {
	t.Run("success with valid spec", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify spec was created with auto-generated fields
		if spec.ID == "" {
			t.Error("Expected ID to be generated")
		}
		if spec.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if spec.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}
	})

	t.Run("success with pre-filled fields", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		spec := &models.SpecNode{
			ID:      uuid.New().String(),
			Title:   "Pre-filled Spec",
			Content: "Content",
		}

		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify original values were preserved
		if spec.Title != "Pre-filled Spec" {
			t.Errorf("Expected title 'Pre-filled Spec', got %s", spec.Title)
		}
	})
}

// Test GetSpec
func TestGetSpec(t *testing.T) {
	t.Run("success with existing spec", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create test spec
		original := createTestSpec()
		err := storage.CreateSpec(original)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		// Retrieve spec
		retrieved, err := storage.GetSpec(original.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify all fields match
		if retrieved.ID != original.ID {
			t.Errorf("Expected ID %s, got %s", original.ID, retrieved.ID)
		}
		if retrieved.Title != original.Title {
			t.Errorf("Expected title %s, got %s", original.Title, retrieved.Title)
		}
		if retrieved.Content != original.Content {
			t.Errorf("Expected content %s, got %s", original.Content, retrieved.Content)
		}
	})

	t.Run("error with non-existent spec", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		nonExistentID := uuid.New().String()
		_, err := storage.GetSpec(nonExistentID)
		if err == nil {
			t.Fatal("Expected error for non-existent spec")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeNotFound {
			t.Errorf("Expected not found error, got %v", zammErr.Type)
		}
	})
}

// Test ListSpecs
func TestListSpecs(t *testing.T) {
	t.Run("success with empty database", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		specs, err := storage.ListSpecs()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(specs) != 0 {
			t.Errorf("Expected 0 specs, got %d", len(specs))
		}
	})

	t.Run("success with multiple specs", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create multiple specs
		expectedCount := 3
		for i := 0; i < expectedCount; i++ {
			spec := &models.SpecNode{
				Title:   fmt.Sprintf("Spec %d", i),
				Content: fmt.Sprintf("Content %d", i),
			}
			err := storage.CreateSpec(spec)
			if err != nil {
				t.Fatalf("Failed to create spec %d: %v", i, err)
			}
			time.Sleep(time.Millisecond) // Ensure different created_at times
		}

		specs, err := storage.ListSpecs()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(specs) != expectedCount {
			t.Errorf("Expected %d specs, got %d", expectedCount, len(specs))
		}

		// Verify ordering (should be DESC by created_at)
		for i := 0; i < len(specs)-1; i++ {
			if specs[i].CreatedAt.Before(specs[i+1].CreatedAt) {
				t.Error("Expected specs to be ordered by created_at DESC")
			}
		}
	})
}

// Test UpdateSpec
func TestUpdateSpec(t *testing.T) {
	t.Run("success with existing spec", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create original spec
		original := createTestSpec()
		err := storage.CreateSpec(original)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		// Update spec
		original.Title = "Updated Title"
		original.Content = "Updated Content"
		originalUpdatedAt := original.UpdatedAt

		err = storage.UpdateSpec(original)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify UpdatedAt was modified
		if !original.UpdatedAt.After(originalUpdatedAt) {
			t.Error("Expected UpdatedAt to be updated")
		}

		// Retrieve and verify changes
		retrieved, err := storage.GetSpec(original.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve updated spec: %v", err)
		}

		if retrieved.Title != "Updated Title" {
			t.Errorf("Expected title 'Updated Title', got %s", retrieved.Title)
		}
		if retrieved.Content != "Updated Content" {
			t.Errorf("Expected content 'Updated Content', got %s", retrieved.Content)
		}
	})

	t.Run("error with non-existent spec", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		nonExistentSpec := &models.SpecNode{
			ID:      uuid.New().String(),
			Title:   "Non-existent",
			Content: "Content",
		}

		err := storage.UpdateSpec(nonExistentSpec)
		if err == nil {
			t.Fatal("Expected error for non-existent spec")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeNotFound {
			t.Errorf("Expected not found error, got %v", zammErr.Type)
		}
	})
}

// Test DeleteSpec
func TestDeleteSpec(t *testing.T) {
	t.Run("success with existing spec", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create test spec
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		// Delete spec
		err = storage.DeleteSpec(spec.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify spec is gone
		_, err = storage.GetSpec(spec.ID)
		if err == nil {
			t.Fatal("Expected error when getting deleted spec")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeNotFound {
			t.Errorf("Expected not found error, got %v", zammErr.Type)
		}
	})

	t.Run("success cascades to links", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create spec and link
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		link := createTestLink(spec.ID)
		err = storage.CreateLink(link)
		if err != nil {
			t.Fatalf("Failed to create test link: %v", err)
		}

		// Delete spec
		err = storage.DeleteSpec(spec.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify link is also gone (cascade delete)
		_, err = storage.GetLink(link.ID)
		if err == nil {
			t.Fatal("Expected error when getting link for deleted spec")
		}
	})

	t.Run("error with non-existent spec", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		nonExistentID := uuid.New().String()
		err := storage.DeleteSpec(nonExistentID)
		if err == nil {
			t.Fatal("Expected error for non-existent spec")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeNotFound {
			t.Errorf("Expected not found error, got %v", zammErr.Type)
		}
	})
}

// Test CreateLink
func TestCreateLink(t *testing.T) {
	t.Run("success with valid link", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create spec first
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		// Create link
		link := createTestLink(spec.ID)
		err = storage.CreateLink(link)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify auto-generated fields
		if link.ID == "" {
			t.Error("Expected ID to be generated")
		}
		if link.LinkType != "implements" {
			t.Errorf("Expected link_type 'implements', got %s", link.LinkType)
		}
		if link.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("success with pre-filled fields", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create spec first
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		// Create link with pre-filled fields
		link := &models.SpecCommitLink{
			ID:       uuid.New().String(),
			SpecID:   spec.ID,
			CommitID: "abcdef1234567890abcdef1234567890abcdef12",
			RepoPath: "/test/repo",
			LinkType: "fixes",
		}

		err = storage.CreateLink(link)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify original values were preserved
		if link.LinkType != "fixes" {
			t.Errorf("Expected link_type 'fixes', got %s", link.LinkType)
		}
	})

	t.Run("error with non-existent spec", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		nonExistentSpecID := uuid.New().String()
		link := createTestLink(nonExistentSpecID)

		err := storage.CreateLink(link)
		if err == nil {
			t.Fatal("Expected error for non-existent spec")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeStorage {
			t.Errorf("Expected storage error, got %v", zammErr.Type)
		}
	})

	t.Run("error with duplicate link", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create spec first
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		// Create first link
		link1 := createTestLink(spec.ID)
		err = storage.CreateLink(link1)
		if err != nil {
			t.Fatalf("Expected no error for first link, got %v", err)
		}

		// Try to create duplicate link (same spec_id, commit_id, repo_path)
		link2 := createTestLink(spec.ID)
		err = storage.CreateLink(link2)
		if err == nil {
			t.Fatal("Expected error for duplicate link")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeStorage {
			t.Errorf("Expected storage error, got %v", zammErr.Type)
		}
	})
}

// Test GetLink
func TestGetLink(t *testing.T) {
	t.Run("success with existing link", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create spec and link
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		original := createTestLink(spec.ID)
		err = storage.CreateLink(original)
		if err != nil {
			t.Fatalf("Failed to create test link: %v", err)
		}

		// Retrieve link
		retrieved, err := storage.GetLink(original.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify all fields match
		if retrieved.ID != original.ID {
			t.Errorf("Expected ID %s, got %s", original.ID, retrieved.ID)
		}
		if retrieved.SpecID != original.SpecID {
			t.Errorf("Expected SpecID %s, got %s", original.SpecID, retrieved.SpecID)
		}
		if retrieved.CommitID != original.CommitID {
			t.Errorf("Expected CommitID %s, got %s", original.CommitID, retrieved.CommitID)
		}
		if retrieved.RepoPath != original.RepoPath {
			t.Errorf("Expected RepoPath %s, got %s", original.RepoPath, retrieved.RepoPath)
		}
		if retrieved.LinkType != original.LinkType {
			t.Errorf("Expected LinkType %s, got %s", original.LinkType, retrieved.LinkType)
		}
	})

	t.Run("error with non-existent link", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		nonExistentID := uuid.New().String()
		_, err := storage.GetLink(nonExistentID)
		if err == nil {
			t.Fatal("Expected error for non-existent link")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeNotFound {
			t.Errorf("Expected not found error, got %v", zammErr.Type)
		}
	})
}

// Test GetLinksBySpec
func TestGetLinksBySpec(t *testing.T) {
	t.Run("success with existing links", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create spec
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		// Create multiple links for the spec
		expectedCount := 3
		for i := 0; i < expectedCount; i++ {
			link := &models.SpecCommitLink{
				SpecID:   spec.ID,
				CommitID: fmt.Sprintf("commit%d%s", i, strings.Repeat("0", 35)),
				RepoPath: fmt.Sprintf("/repo/%d", i),
				LinkType: "implements",
			}
			err = storage.CreateLink(link)
			if err != nil {
				t.Fatalf("Failed to create link %d: %v", i, err)
			}
			time.Sleep(time.Millisecond) // Ensure different created_at times
		}

		// Retrieve links
		links, err := storage.GetLinksBySpec(spec.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(links) != expectedCount {
			t.Errorf("Expected %d links, got %d", expectedCount, len(links))
		}

		// Verify all links belong to the spec
		for _, link := range links {
			if link.SpecID != spec.ID {
				t.Errorf("Expected SpecID %s, got %s", spec.ID, link.SpecID)
			}
		}

		// Verify ordering (should be DESC by created_at)
		for i := 0; i < len(links)-1; i++ {
			if links[i].CreatedAt.Before(links[i+1].CreatedAt) {
				t.Error("Expected links to be ordered by created_at DESC")
			}
		}
	})

	t.Run("success with no links", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create spec without links
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		links, err := storage.GetLinksBySpec(spec.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(links) != 0 {
			t.Errorf("Expected 0 links, got %d", len(links))
		}
	})
}

// Test GetLinksByCommit
func TestGetLinksByCommit(t *testing.T) {
	t.Run("success with existing links", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		commitID := "abcdef1234567890abcdef1234567890abcdef12"
		repoPath := "/test/repo"

		// Create multiple specs and link them to the same commit
		expectedCount := 3
		for i := 0; i < expectedCount; i++ {
			spec := &models.SpecNode{
				Title:   fmt.Sprintf("Spec %d", i),
				Content: fmt.Sprintf("Content %d", i),
			}
			err := storage.CreateSpec(spec)
			if err != nil {
				t.Fatalf("Failed to create spec %d: %v", i, err)
			}

			link := &models.SpecCommitLink{
				SpecID:   spec.ID,
				CommitID: commitID,
				RepoPath: repoPath,
				LinkType: "implements",
			}
			err = storage.CreateLink(link)
			if err != nil {
				t.Fatalf("Failed to create link %d: %v", i, err)
			}
			time.Sleep(time.Millisecond) // Ensure different created_at times
		}

		// Retrieve links
		links, err := storage.GetLinksByCommit(commitID, repoPath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(links) != expectedCount {
			t.Errorf("Expected %d links, got %d", expectedCount, len(links))
		}

		// Verify all links belong to the commit
		for _, link := range links {
			if link.CommitID != commitID {
				t.Errorf("Expected CommitID %s, got %s", commitID, link.CommitID)
			}
			if link.RepoPath != repoPath {
				t.Errorf("Expected RepoPath %s, got %s", repoPath, link.RepoPath)
			}
		}

		// Verify ordering (should be DESC by created_at)
		for i := 0; i < len(links)-1; i++ {
			if links[i].CreatedAt.Before(links[i+1].CreatedAt) {
				t.Error("Expected links to be ordered by created_at DESC")
			}
		}
	})

	t.Run("success with no links", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		nonExistentCommit := "nonexistent1234567890abcdef1234567890abcdef"
		links, err := storage.GetLinksByCommit(nonExistentCommit, "/test/repo")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(links) != 0 {
			t.Errorf("Expected 0 links, got %d", len(links))
		}
	})

	t.Run("different repo paths return different results", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		commitID := "abcdef1234567890abcdef1234567890abcdef12"
		repoPath1 := "/repo/path1"
		repoPath2 := "/repo/path2"

		// Create spec and links to same commit but different repo paths
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		link1 := &models.SpecCommitLink{
			SpecID:   spec.ID,
			CommitID: commitID,
			RepoPath: repoPath1,
			LinkType: "implements",
		}
		err = storage.CreateLink(link1)
		if err != nil {
			t.Fatalf("Failed to create link1: %v", err)
		}

		// Query for repo path 1 should return 1 link
		links1, err := storage.GetLinksByCommit(commitID, repoPath1)
		if err != nil {
			t.Fatalf("Expected no error for repo1, got %v", err)
		}
		if len(links1) != 1 {
			t.Errorf("Expected 1 link for repo1, got %d", len(links1))
		}

		// Query for repo path 2 should return 0 links
		links2, err := storage.GetLinksByCommit(commitID, repoPath2)
		if err != nil {
			t.Fatalf("Expected no error for repo2, got %v", err)
		}
		if len(links2) != 0 {
			t.Errorf("Expected 0 links for repo2, got %d", len(links2))
		}
	})
}

// Test DeleteLink
func TestDeleteLink(t *testing.T) {
	t.Run("success with existing link", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		// Create spec and link
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		link := createTestLink(spec.ID)
		err = storage.CreateLink(link)
		if err != nil {
			t.Fatalf("Failed to create test link: %v", err)
		}

		// Delete link
		err = storage.DeleteLink(link.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify link is gone
		_, err = storage.GetLink(link.ID)
		if err == nil {
			t.Fatal("Expected error when getting deleted link")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeNotFound {
			t.Errorf("Expected not found error, got %v", zammErr.Type)
		}
	})

	t.Run("error with non-existent link", func(t *testing.T) {
		storage, cleanup := setupTestService(t)
		defer cleanup()

		nonExistentID := uuid.New().String()
		err := storage.DeleteLink(nonExistentID)
		if err == nil {
			t.Fatal("Expected error for non-existent link")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeNotFound {
			t.Errorf("Expected not found error, got %v", zammErr.Type)
		}
	})
}

// Test Close
func TestClose(t *testing.T) {
	t.Run("success with open connection", func(t *testing.T) {
		storage, err := NewSQLiteStorage(":memory:")
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		err = storage.Close()
		if err != nil {
			t.Errorf("Expected no error closing storage, got %v", err)
		}
	})

	t.Run("success with already closed connection", func(t *testing.T) {
		storage, err := NewSQLiteStorage(":memory:")
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Close once
		err = storage.Close()
		if err != nil {
			t.Errorf("Expected no error on first close, got %v", err)
		}

		// Close again should not error
		err = storage.Close()
		if err != nil {
			t.Errorf("Expected no error on second close, got %v", err)
		}
	})

	t.Run("success with nil database", func(t *testing.T) {
		storage := &SQLiteStorage{db: nil}
		err := storage.Close()
		if err != nil {
			t.Errorf("Expected no error with nil db, got %v", err)
		}
	})
}

// Benchmark tests for performance requirements
func BenchmarkCreateSpec(b *testing.B) {
	storage, cleanup := setupTestService(new(testing.T))
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spec := &models.SpecNode{
			Title:   fmt.Sprintf("Benchmark Spec %d", i),
			Content: "Benchmark content",
		}
		err := storage.CreateSpec(spec)
		if err != nil {
			b.Fatalf("Failed to create spec: %v", err)
		}
	}
}

func BenchmarkGetSpec(b *testing.B) {
	storage, cleanup := setupTestService(new(testing.T))
	defer cleanup()

	// Create test spec
	spec := createTestSpec()
	err := storage.CreateSpec(spec)
	if err != nil {
		b.Fatalf("Failed to create test spec: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.GetSpec(spec.ID)
		if err != nil {
			b.Fatalf("Failed to get spec: %v", err)
		}
	}
}

func BenchmarkListSpecs(b *testing.B) {
	storage, cleanup := setupTestService(new(testing.T))
	defer cleanup()

	// Create multiple specs
	for i := 0; i < 100; i++ {
		spec := &models.SpecNode{
			Title:   fmt.Sprintf("Spec %d", i),
			Content: "Content",
		}
		err := storage.CreateSpec(spec)
		if err != nil {
			b.Fatalf("Failed to create spec %d: %v", i, err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.ListSpecs()
		if err != nil {
			b.Fatalf("Failed to list specs: %v", err)
		}
	}
}

// Test performance requirements (NFR-001: Response time < 100ms for single record queries)
func TestPerformanceRequirements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	storage, cleanup := setupTestService(t)
	defer cleanup()

	// Create test data
	spec := createTestSpec()
	err := storage.CreateSpec(spec)
	if err != nil {
		t.Fatalf("Failed to create test spec: %v", err)
	}

	link := createTestLink(spec.ID)
	err = storage.CreateLink(link)
	if err != nil {
		t.Fatalf("Failed to create test link: %v", err)
	}

	// Test GetSpec performance
	t.Run("GetSpec under 100ms", func(t *testing.T) {
		start := time.Now()
		_, err := storage.GetSpec(spec.ID)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("GetSpec failed: %v", err)
		}
		if duration > 100*time.Millisecond {
			t.Errorf("GetSpec took %v, expected < 100ms", duration)
		}
	})

	// Test GetLink performance
	t.Run("GetLink under 100ms", func(t *testing.T) {
		start := time.Now()
		_, err := storage.GetLink(link.ID)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("GetLink failed: %v", err)
		}
		if duration > 100*time.Millisecond {
			t.Errorf("GetLink took %v, expected < 100ms", duration)
		}
	})
}

// Test concurrent access scenarios
func TestConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent access tests in short mode")
	}

	storage, cleanup := setupTestService(t)
	defer cleanup()

	t.Run("concurrent spec creation", func(t *testing.T) {
		const numGoroutines = 5
		const specsPerGoroutine = 2

		type result struct {
			routineID int
			err       error
			created   int
		}

		results := make(chan result, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				var created int
				var lastErr error

				for j := 0; j < specsPerGoroutine; j++ {
					spec := &models.SpecNode{
						Title:   fmt.Sprintf("Concurrent Spec %d-%d", routineID, j),
						Content: fmt.Sprintf("Content from routine %d, spec %d", routineID, j),
					}
					err := storage.CreateSpec(spec)
					if err != nil {
						lastErr = err
						break
					}
					created++
				}
				results <- result{routineID: routineID, err: lastErr, created: created}
			}(i)
		}

		// Wait for all goroutines to complete and collect results
		totalCreated := 0
		var errors []error

		for i := 0; i < numGoroutines; i++ {
			res := <-results
			if res.err != nil {
				errors = append(errors, fmt.Errorf("routine %d: %v", res.routineID, res.err))
			}
			totalCreated += res.created
		}

		// Some concurrent failures are acceptable in SQLite with high concurrency
		// But we should have created at least some specs
		if totalCreated == 0 {
			t.Fatalf("No specs were created successfully. Errors: %v", errors)
		}

		// Give a small delay to ensure all transactions are complete
		time.Sleep(10 * time.Millisecond)

		// Verify the count matches what we actually created
		// Try a few times in case of temporary database locking
		var specs []*models.SpecNode
		var listErr error
		for attempt := 0; attempt < 3; attempt++ {
			specs, listErr = storage.ListSpecs()
			if listErr == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}

		if listErr != nil {
			t.Fatalf("Failed to list specs after %d attempts: %v", 3, listErr)
		}

		if len(specs) != totalCreated {
			t.Errorf("Expected %d specs (actually created), got %d from database", totalCreated, len(specs))
		}

		// If we have more than a few errors, that might indicate a problem
		if len(errors) > numGoroutines/2 {
			t.Logf("Warning: High number of concurrent errors (%d/%d): %v", len(errors), numGoroutines, errors)
		}
	})
}

// Test storage interface compliance
func TestStorageInterfaceCompliance(t *testing.T) {
	storage, cleanup := setupTestService(t)
	defer cleanup()

	// Verify the storage implements the Storage interface
	var _ Storage = storage

	t.Run("all interface methods exist", func(t *testing.T) {
		// This test ensures all interface methods are implemented
		// If any method is missing, this won't compile

		// Spec operations
		_ = storage.CreateSpec
		_ = storage.GetSpec
		_ = storage.ListSpecs
		_ = storage.UpdateSpec
		_ = storage.DeleteSpec

		// Link operations
		_ = storage.CreateLink
		_ = storage.GetLink
		_ = storage.GetLinksBySpec
		_ = storage.GetLinksByCommit
		_ = storage.DeleteLink

		// Utility
		_ = storage.Close
	})
}

// Test edge cases and validation
func TestEdgeCases(t *testing.T) {
	storage, cleanup := setupTestService(t)
	defer cleanup()

	t.Run("spec with maximum content length", func(t *testing.T) {
		// Create spec with 50KB content (spec limit)
		maxContent := strings.Repeat("x", 50*1024)
		spec := &models.SpecNode{
			Title:   "Max Content Spec",
			Content: maxContent,
		}

		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create spec with max content: %v", err)
		}

		// Verify content was stored correctly
		retrieved, err := storage.GetSpec(spec.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve spec: %v", err)
		}

		if len(retrieved.Content) != len(maxContent) {
			t.Errorf("Expected content length %d, got %d", len(maxContent), len(retrieved.Content))
		}
	})

	t.Run("spec with maximum title length", func(t *testing.T) {
		// Create spec with 200 character title (spec limit)
		maxTitle := strings.Repeat("T", 200)
		spec := &models.SpecNode{
			Title:   maxTitle,
			Content: "Content",
		}

		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create spec with max title: %v", err)
		}

		// Verify title was stored correctly
		retrieved, err := storage.GetSpec(spec.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve spec: %v", err)
		}

		if retrieved.Title != maxTitle {
			t.Errorf("Expected title %s, got %s", maxTitle, retrieved.Title)
		}
	})

	t.Run("link with valid commit hash format", func(t *testing.T) {
		spec := createTestSpec()
		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create test spec: %v", err)
		}

		// Test with 40-character hex commit hash
		validCommitHash := "abcdef1234567890abcdef1234567890abcdef12"
		link := &models.SpecCommitLink{
			SpecID:   spec.ID,
			CommitID: validCommitHash,
			RepoPath: "/test/repo",
			LinkType: "implements",
		}

		err = storage.CreateLink(link)
		if err != nil {
			t.Fatalf("Failed to create link with valid commit hash: %v", err)
		}

		retrieved, err := storage.GetLink(link.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve link: %v", err)
		}

		if retrieved.CommitID != validCommitHash {
			t.Errorf("Expected commit hash %s, got %s", validCommitHash, retrieved.CommitID)
		}
	})

	t.Run("empty string handling", func(t *testing.T) {
		// Test with empty title (should be allowed)
		spec := &models.SpecNode{
			Title:   "",
			Content: "Content with empty title",
		}

		err := storage.CreateSpec(spec)
		if err != nil {
			t.Fatalf("Failed to create spec with empty title: %v", err)
		}

		retrieved, err := storage.GetSpec(spec.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve spec: %v", err)
		}

		if retrieved.Title != "" {
			t.Errorf("Expected empty title, got %s", retrieved.Title)
		}
	})
}

// TestSpecHierarchy tests spec-to-spec linking functionality
func TestSpecHierarchy(t *testing.T) {
	storage, cleanup := setupTestService(t)
	defer cleanup()

	// Create test specs
	fromSpec := &models.SpecNode{
		ID:      uuid.New().String(),
		Title:   "From Specification",
		Content: "From content",
	}

	toSpec := &models.SpecNode{
		ID:      uuid.New().String(),
		Title:   "To Specification",
		Content: "To content",
	}

	// Create the specs
	err := storage.CreateSpec(fromSpec)
	if err != nil {
		t.Fatalf("Failed to create from spec: %v", err)
	}

	err = storage.CreateSpec(toSpec)
	if err != nil {
		t.Fatalf("Failed to create to spec: %v", err)
	}

	t.Run("CreateSpecLink", func(t *testing.T) {
		link := &models.SpecSpecLink{
			FromSpecID: fromSpec.ID,
			ToSpecID:   toSpec.ID,
			LinkType:   "child",
		}

		err := storage.CreateSpecLink(link)
		if err != nil {
			t.Fatalf("Failed to create spec link: %v", err)
		}

		if link.ID == "" {
			t.Error("Expected link ID to be generated")
		}

		if link.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("GetLinkedSpecs", func(t *testing.T) {
		// Test getting outgoing links (from fromSpec to toSpec)
		outgoing, err := storage.GetLinkedSpecs(fromSpec.ID, models.Outgoing)
		if err != nil {
			t.Fatalf("Failed to get outgoing links: %v", err)
		}

		if len(outgoing) != 1 {
			t.Fatalf("Expected 1 outgoing link, got %d", len(outgoing))
		}

		if outgoing[0].ID != toSpec.ID {
			t.Errorf("Expected outgoing spec ID %s, got %s", toSpec.ID, outgoing[0].ID)
		}

		// Test getting incoming links (to toSpec from fromSpec)
		incoming, err := storage.GetLinkedSpecs(toSpec.ID, models.Incoming)
		if err != nil {
			t.Fatalf("Failed to get incoming links: %v", err)
		}

		if len(incoming) != 1 {
			t.Fatalf("Expected 1 incoming link, got %d", len(incoming))
		}

		if incoming[0].ID != fromSpec.ID {
			t.Errorf("Expected incoming spec ID %s, got %s", fromSpec.ID, incoming[0].ID)
		}
	})

	t.Run("DeleteSpecLinkBySpecs", func(t *testing.T) {
		// Test successful deletion using fromSpecID and toSpecID
		err := storage.DeleteSpecLinkBySpecs(fromSpec.ID, toSpec.ID)
		if err != nil {
			t.Fatalf("Failed to delete spec link: %v", err)
		}

		// Verify the link is gone
		outgoing, err := storage.GetLinkedSpecs(fromSpec.ID, models.Outgoing)
		if err != nil {
			t.Fatalf("Failed to get outgoing links after deletion: %v", err)
		}

		if len(outgoing) != 0 {
			t.Errorf("Expected 0 outgoing links after deletion, got %d", len(outgoing))
		}

		// Test deletion of non-existent link
		err = storage.DeleteSpecLinkBySpecs(fromSpec.ID, toSpec.ID)
		if err == nil {
			t.Error("Expected error when deleting non-existent link")
		}

		// Check error type
		if !strings.Contains(err.Error(), "spec link not found") {
			t.Errorf("Expected 'spec link not found' error, got: %v", err)
		}
	})

	t.Run("DeleteSpecLinkBySpecs_WrongOrder", func(t *testing.T) {
		// Create a new link for this test
		link := &models.SpecSpecLink{
			FromSpecID: fromSpec.ID,
			ToSpecID:   toSpec.ID,
			LinkType:   "child",
		}

		err := storage.CreateSpecLink(link)
		if err != nil {
			t.Fatalf("Failed to create spec link: %v", err)
		}

		// Test deletion with wrong parameter order (should fail)
		err = storage.DeleteSpecLinkBySpecs(toSpec.ID, fromSpec.ID)
		if err == nil {
			t.Error("Expected error when using wrong parameter order")
		}

		// Verify the link still exists
		outgoing, err := storage.GetLinkedSpecs(fromSpec.ID, models.Outgoing)
		if err != nil {
			t.Fatalf("Failed to get outgoing links: %v", err)
		}

		if len(outgoing) != 1 {
			t.Errorf("Expected 1 outgoing link (link should still exist), got %d", len(outgoing))
		}

		// Clean up - delete with correct order
		err = storage.DeleteSpecLinkBySpecs(fromSpec.ID, toSpec.ID)
		if err != nil {
			t.Fatalf("Failed to delete spec link with correct order: %v", err)
		}
	})
}

// TestSpecHierarchyIntegration tests the integration between services and storage
func TestSpecHierarchyIntegration(t *testing.T) {
	storage, cleanup := setupTestService(t)
	defer cleanup()

	// Create test specs
	fromSpec := &models.SpecNode{
		ID:      uuid.New().String(),
		Title:   "Integration From Spec",
		Content: "From content for integration test",
	}

	toSpec := &models.SpecNode{
		ID:      uuid.New().String(),
		Title:   "Integration To Spec",
		Content: "To content for integration test",
	}

	// Create the specs
	err := storage.CreateSpec(fromSpec)
	if err != nil {
		t.Fatalf("Failed to create from spec: %v", err)
	}

	err = storage.CreateSpec(toSpec)
	if err != nil {
		t.Fatalf("Failed to create to spec: %v", err)
	}

	// Create a link
	link := &models.SpecSpecLink{
		FromSpecID: fromSpec.ID,
		ToSpecID:   toSpec.ID,
		LinkType:   "child",
	}

	err = storage.CreateSpecLink(link)
	if err != nil {
		t.Fatalf("Failed to create spec link: %v", err)
	}

	t.Run("VerifyDatabaseSchema", func(t *testing.T) {
		// Query the database directly to verify the schema
		var fromSpecID, toSpecID string
		query := `SELECT from_spec_id, to_spec_id FROM spec_spec_links WHERE id = ?`
		err := storage.db.QueryRow(query, link.ID).Scan(&fromSpecID, &toSpecID)
		if err != nil {
			t.Fatalf("Failed to query spec link: %v", err)
		}

		if fromSpecID != fromSpec.ID {
			t.Errorf("Expected from_spec_id to be %s, got %s", fromSpec.ID, fromSpecID)
		}

		if toSpecID != toSpec.ID {
			t.Errorf("Expected to_spec_id to be %s, got %s", toSpec.ID, toSpecID)
		}
	})

	t.Run("DeleteUsingCorrectDirection", func(t *testing.T) {
		// This simulates the service layer call:
		// DeleteSpecLinkBySpecs(fromSpecID, toSpecID)
		err := storage.DeleteSpecLinkBySpecs(fromSpec.ID, toSpec.ID)
		if err != nil {
			t.Fatalf("Failed to delete spec link using correct direction: %v", err)
		}

		// Verify deletion was successful
		outgoing, err := storage.GetLinkedSpecs(fromSpec.ID, models.Outgoing)
		if err != nil {
			t.Fatalf("Failed to get outgoing links after deletion: %v", err)
		}

		if len(outgoing) != 0 {
			t.Errorf("Expected 0 outgoing links after deletion, got %d", len(outgoing))
		}
	})
}
