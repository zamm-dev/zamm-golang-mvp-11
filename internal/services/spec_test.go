package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/yourorg/zamm-mvp/internal/models"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

// setupTestService creates a new test service with temporary database
func setupTestService(t *testing.T) (SpecService, func()) {
	t.Helper()

	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create storage
	store, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	// Run migrations
	err = store.RunMigrationsIfNeeded()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	service := NewSpecService(store)

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return service, cleanup
}

// createTestSpec creates a test specification
func createTestSpec(title, content string) *models.SpecNode {
	return &models.SpecNode{
		ID:       uuid.New().String(),
		StableID: uuid.New().String(),
		Version:  1,
		Title:    title,
		Content:  content,
		NodeType: "spec",
	}
}

// TestRemoveChildFromParent tests the specific bug that was fixed
func TestRemoveChildFromParent(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Create parent and child specs
	parentSpec := createTestSpec("Parent Specification", "This is the parent spec")
	childSpec := createTestSpec("Child Specification", "This is the child spec")

	// Create the specs in storage
	_, err := service.CreateSpec(parentSpec.Title, parentSpec.Content)
	if err != nil {
		t.Fatalf("Failed to create parent spec: %v", err)
	}

	_, err = service.CreateSpec(childSpec.Title, childSpec.Content)
	if err != nil {
		t.Fatalf("Failed to create child spec: %v", err)
	}

	// Get the actual specs with their generated IDs
	specs, err := service.ListSpecs()
	if err != nil {
		t.Fatalf("Failed to list specs: %v", err)
	}

	if len(specs) != 2 {
		t.Fatalf("Expected 2 specs, got %d", len(specs))
	}

	// Find parent and child by title
	var parent, child *models.SpecNode
	for _, spec := range specs {
		if spec.Title == "Parent Specification" {
			parent = spec
		} else if spec.Title == "Child Specification" {
			child = spec
		}
	}

	if parent == nil || child == nil {
		t.Fatal("Failed to find created specs")
	}

	t.Run("AddChildToParent", func(t *testing.T) {
		// Add child to parent
		link, err := service.AddChildToParent(child.ID, parent.ID)
		if err != nil {
			t.Fatalf("Failed to add child to parent: %v", err)
		}

		if link == nil {
			t.Fatal("Expected link to be created")
		}

		// Verify the relationship exists
		children, err := service.GetChildren(parent.ID)
		if err != nil {
			t.Fatalf("Failed to get children: %v", err)
		}

		if len(children) != 1 {
			t.Fatalf("Expected 1 child, got %d", len(children))
		}

		if children[0].ID != child.ID {
			t.Errorf("Expected child ID %s, got %s", child.ID, children[0].ID)
		}

		// Verify reverse relationship
		parents, err := service.GetParents(child.ID)
		if err != nil {
			t.Fatalf("Failed to get parents: %v", err)
		}

		if len(parents) != 1 {
			t.Fatalf("Expected 1 parent, got %d", len(parents))
		}

		if parents[0].ID != parent.ID {
			t.Errorf("Expected parent ID %s, got %s", parent.ID, parents[0].ID)
		}
	})

	t.Run("RemoveChildFromParent_BugFix", func(t *testing.T) {
		// This is the critical test that verifies the bug fix
		// Before the fix, this would fail with "spec link not found"
		// because the service was calling DeleteSpecLinkBySpecs with wrong parameter order

		err := service.RemoveChildFromParent(child.ID, parent.ID)
		if err != nil {
			t.Fatalf("Failed to remove child from parent (BUG REPRODUCED): %v", err)
		}

		// Verify the relationship no longer exists
		children, err := service.GetChildren(parent.ID)
		if err != nil {
			t.Fatalf("Failed to get children after removal: %v", err)
		}

		if len(children) != 0 {
			t.Errorf("Expected 0 children after removal, got %d", len(children))
		}

		// Verify reverse relationship is also gone
		parents, err := service.GetParents(child.ID)
		if err != nil {
			t.Fatalf("Failed to get parents after removal: %v", err)
		}

		if len(parents) != 0 {
			t.Errorf("Expected 0 parents after removal, got %d", len(parents))
		}
	})

	t.Run("RemoveChildFromParent_NonExistentRelationship", func(t *testing.T) {
		// Test removing a relationship that doesn't exist
		err := service.RemoveChildFromParent(child.ID, parent.ID)
		if err == nil {
			t.Error("Expected error when removing non-existent relationship")
		}

		// Check that it's a "not found" error
		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeNotFound {
			t.Errorf("Expected not found error, got %v", zammErr.Type)
		}
	})
}

// TestRemoveChildFromParent_ParameterValidation tests input validation
func TestRemoveChildFromParent_ParameterValidation(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	t.Run("EmptyChildSpecID", func(t *testing.T) {
		err := service.RemoveChildFromParent("", "some-parent-id")
		if err == nil {
			t.Error("Expected error for empty child spec ID")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeValidation {
			t.Errorf("Expected validation error, got %v", zammErr.Type)
		}
	})

	t.Run("EmptyParentSpecID", func(t *testing.T) {
		err := service.RemoveChildFromParent("some-child-id", "")
		if err == nil {
			t.Error("Expected error for empty parent spec ID")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeValidation {
			t.Errorf("Expected validation error, got %v", zammErr.Type)
		}
	})
}

// TestSpecHierarchyIntegration tests the complete spec hierarchy workflow
func TestSpecHierarchyIntegration(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Create a hierarchy: Root -> Level1 -> Level2
	rootSpec, err := service.CreateSpec("Root Specification", "Root level")
	if err != nil {
		t.Fatalf("Failed to create root spec: %v", err)
	}

	level1Spec, err := service.CreateSpec("Level 1 Specification", "Level 1")
	if err != nil {
		t.Fatalf("Failed to create level 1 spec: %v", err)
	}

	level2Spec, err := service.CreateSpec("Level 2 Specification", "Level 2")
	if err != nil {
		t.Fatalf("Failed to create level 2 spec: %v", err)
	}

	t.Run("BuildHierarchy", func(t *testing.T) {
		// Add Level1 as child of Root
		_, err := service.AddChildToParent(level1Spec.ID, rootSpec.ID)
		if err != nil {
			t.Fatalf("Failed to add level1 to root: %v", err)
		}

		// Add Level2 as child of Level1
		_, err = service.AddChildToParent(level2Spec.ID, level1Spec.ID)
		if err != nil {
			t.Fatalf("Failed to add level2 to level1: %v", err)
		}

		// Verify Root has Level1 as child
		children, err := service.GetChildren(rootSpec.ID)
		if err != nil {
			t.Fatalf("Failed to get root children: %v", err)
		}
		if len(children) != 1 || children[0].ID != level1Spec.ID {
			t.Error("Root should have Level1 as child")
		}

		// Verify Level1 has Level2 as child
		children, err = service.GetChildren(level1Spec.ID)
		if err != nil {
			t.Fatalf("Failed to get level1 children: %v", err)
		}
		if len(children) != 1 || children[0].ID != level2Spec.ID {
			t.Error("Level1 should have Level2 as child")
		}

		// Verify Level2 has no children
		children, err = service.GetChildren(level2Spec.ID)
		if err != nil {
			t.Fatalf("Failed to get level2 children: %v", err)
		}
		if len(children) != 0 {
			t.Error("Level2 should have no children")
		}
	})

	t.Run("RemoveMiddleLevel", func(t *testing.T) {
		// Remove Level1 from Root (this tests the bug fix)
		err := service.RemoveChildFromParent(level1Spec.ID, rootSpec.ID)
		if err != nil {
			t.Fatalf("Failed to remove level1 from root: %v", err)
		}

		// Verify Root no longer has Level1 as child
		children, err := service.GetChildren(rootSpec.ID)
		if err != nil {
			t.Fatalf("Failed to get root children after removal: %v", err)
		}
		if len(children) != 0 {
			t.Error("Root should have no children after removal")
		}

		// Verify Level1 still has Level2 as child (only removed one relationship)
		children, err = service.GetChildren(level1Spec.ID)
		if err != nil {
			t.Fatalf("Failed to get level1 children after removal: %v", err)
		}
		if len(children) != 1 || children[0].ID != level2Spec.ID {
			t.Error("Level1 should still have Level2 as child")
		}
	})

	t.Run("RemoveRemainingRelationship", func(t *testing.T) {
		// Remove Level2 from Level1
		err := service.RemoveChildFromParent(level2Spec.ID, level1Spec.ID)
		if err != nil {
			t.Fatalf("Failed to remove level2 from level1: %v", err)
		}

		// Verify Level1 no longer has Level2 as child
		children, err := service.GetChildren(level1Spec.ID)
		if err != nil {
			t.Fatalf("Failed to get level1 children after removal: %v", err)
		}
		if len(children) != 0 {
			t.Error("Level1 should have no children after removal")
		}

		// Verify Level2 has no parents
		parents, err := service.GetParents(level2Spec.ID)
		if err != nil {
			t.Fatalf("Failed to get level2 parents after removal: %v", err)
		}
		if len(parents) != 0 {
			t.Error("Level2 should have no parents after removal")
		}
	})
}

// TestAddChildToParent tests the add functionality for completeness
func TestAddChildToParent(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	parentSpec, err := service.CreateSpec("Parent", "Parent content")
	if err != nil {
		t.Fatalf("Failed to create parent spec: %v", err)
	}

	childSpec, err := service.CreateSpec("Child", "Child content")
	if err != nil {
		t.Fatalf("Failed to create child spec: %v", err)
	}

	t.Run("ValidAddition", func(t *testing.T) {
		link, err := service.AddChildToParent(childSpec.ID, parentSpec.ID)
		if err != nil {
			t.Fatalf("Failed to add child to parent: %v", err)
		}

		if link.FromSpecID != childSpec.ID {
			t.Errorf("Expected FromSpecID to be child %s, got %s", childSpec.ID, link.FromSpecID)
		}
		if link.ToSpecID != parentSpec.ID {
			t.Errorf("Expected ToSpecID to be parent %s, got %s", parentSpec.ID, link.ToSpecID)
		}
		if link.LinkType != "child" {
			t.Errorf("Expected LinkType 'child', got %s", link.LinkType)
		}
	})

	t.Run("PreventCycle", func(t *testing.T) {
		// Try to add parent as child of child (would create cycle)
		_, err := service.AddChildToParent(parentSpec.ID, childSpec.ID)
		if err == nil {
			t.Error("Expected error when creating cycle")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeValidation {
			t.Errorf("Expected validation error, got %v", zammErr.Type)
		}
	})

	t.Run("PreventSelfLink", func(t *testing.T) {
		// Try to add spec as child of itself
		_, err := service.AddChildToParent(parentSpec.ID, parentSpec.ID)
		if err == nil {
			t.Error("Expected error when linking spec to itself")
		}

		zammErr, ok := err.(*models.ZammError)
		if !ok {
			t.Fatalf("Expected ZammError, got %T", err)
		}
		if zammErr.Type != models.ErrTypeValidation {
			t.Errorf("Expected validation error, got %v", zammErr.Type)
		}
	})
}

func TestInitializeRootSpec(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	t.Run("CreateRootSpecWhenNoneExists", func(t *testing.T) {
		// Test creating a root spec when none exists
		err := service.InitializeRootSpec()
		if err != nil {
			t.Fatalf("Failed to initialize root spec: %v", err)
		}

		// Verify root spec was created
		rootSpec, err := service.GetRootSpec()
		if err != nil {
			t.Fatalf("Failed to get root spec: %v", err)
		}

		if rootSpec == nil {
			t.Fatal("Root spec should not be nil")
		}

		if rootSpec.Title != "New Project" {
			t.Errorf("Expected title 'New Project', got '%s'", rootSpec.Title)
		}
	})

	t.Run("DoesNotRecreateExistingRootSpec", func(t *testing.T) {
		// Get the current root spec
		originalRoot, err := service.GetRootSpec()
		if err != nil {
			t.Fatalf("Failed to get original root spec: %v", err)
		}

		// Call InitializeRootSpec again
		err = service.InitializeRootSpec()
		if err != nil {
			t.Fatalf("Failed to initialize root spec: %v", err)
		}

		// Verify the same root spec is returned
		currentRoot, err := service.GetRootSpec()
		if err != nil {
			t.Fatalf("Failed to get current root spec: %v", err)
		}

		if currentRoot.ID != originalRoot.ID {
			t.Errorf("Root spec ID changed from %s to %s", originalRoot.ID, currentRoot.ID)
		}
	})

	t.Run("LinksOrphanedSpecs", func(t *testing.T) {
		// Create some orphaned specs
		orphan1, err := service.CreateSpec("Orphan 1", "Content 1")
		if err != nil {
			t.Fatalf("Failed to create orphan spec 1: %v", err)
		}

		orphan2, err := service.CreateSpec("Orphan 2", "Content 2")
		if err != nil {
			t.Fatalf("Failed to create orphan spec 2: %v", err)
		}

		// Get root spec before initialization
		rootSpec, err := service.GetRootSpec()
		if err != nil {
			t.Fatalf("Failed to get root spec: %v", err)
		}

		// Initialize root spec (should link orphans)
		err = service.InitializeRootSpec()
		if err != nil {
			t.Fatalf("Failed to initialize root spec: %v", err)
		}

		// Verify orphans are linked to root
		children, err := service.GetChildren(rootSpec.ID)
		if err != nil {
			t.Fatalf("Failed to get root children: %v", err)
		}

		// Should contain at least our two orphaned specs
		foundOrphan1, foundOrphan2 := false, false
		for _, child := range children {
			if child.ID == orphan1.ID {
				foundOrphan1 = true
			}
			if child.ID == orphan2.ID {
				foundOrphan2 = true
			}
		}

		if !foundOrphan1 {
			t.Error("Orphan 1 was not linked to root spec")
		}
		if !foundOrphan2 {
			t.Error("Orphan 2 was not linked to root spec")
		}
	})
	t.Run("WorksWithFreshDatabase", func(t *testing.T) {
		// Create a fresh service to test that the fix works
		freshService, freshCleanup := setupTestService(t)
		defer freshCleanup()

		// This should now work successfully after fixing the foreign key constraint issue
		err := freshService.InitializeRootSpec()
		
		if err != nil {
			t.Errorf("Expected no error with fresh database, but got: %v", err)
		} else {
			t.Log("Successfully initialized root spec with fresh database")
			
			// Verify root spec was created
			rootSpec, err := freshService.GetRootSpec()
			if err != nil {
				t.Errorf("Failed to get root spec after initialization: %v", err)
			} else if rootSpec == nil {
				t.Error("Root spec should not be nil after initialization")
			} else {
				t.Logf("Root spec created successfully with ID: %s", rootSpec.ID)
			}
		}
	})
}
