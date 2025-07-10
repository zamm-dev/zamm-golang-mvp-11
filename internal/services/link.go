package services

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/yourorg/zamm-mvp/internal/models"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

// LinkService interface defines operations for managing spec-commit links
type LinkService interface {
	LinkSpecToCommit(specID, commitID, repoPath, label string) (*models.SpecCommitLink, error)
	GetSpecsForCommit(commitID, repoPath string) ([]*models.Spec, error)
	GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error)
	UnlinkSpecFromCommit(specID, commitID, repoPath string) error
}

// linkService implements the LinkService interface
type linkService struct {
	storage storage.Storage
}

// NewLinkService creates a new LinkService instance
func NewLinkService(storage storage.Storage) LinkService {
	return &linkService{
		storage: storage,
	}
}

// LinkSpecToCommit creates a link between a spec and a commit
func (s *linkService) LinkSpecToCommit(specID, commitID, repoPath, label string) (*models.SpecCommitLink, error) {
	// Validate inputs
	if err := s.validateLinkInput(specID, commitID, repoPath, label); err != nil {
		return nil, err
	}

	// Verify spec exists
	_, err := s.storage.GetSpecNode(specID)
	if err != nil {
		return nil, err
	}

	// Verify repository path exists
	if err := s.validateRepoPath(repoPath); err != nil {
		return nil, err
	}

	link := &models.SpecCommitLink{
		SpecID:    specID,
		CommitID:  strings.TrimSpace(commitID),
		RepoPath:  strings.TrimSpace(repoPath),
		LinkLabel: strings.TrimSpace(label),
	}

	if err := s.storage.CreateSpecCommitLink(link); err != nil {
		return nil, err
	}

	return link, nil
}

// GetSpecsForCommit retrieves all specs linked to a commit
func (s *linkService) GetSpecsForCommit(commitID, repoPath string) ([]*models.Spec, error) {
	if err := s.validateCommitInput(commitID, repoPath); err != nil {
		return nil, err
	}

	// Get links for this commit
	links, err := s.storage.GetLinksByCommit(commitID, repoPath)
	if err != nil {
		return nil, err
	}

	// Get specs for each link
	var specs []*models.Spec
	for _, link := range links {
		spec, err := s.storage.GetSpecNode(link.SpecID)
		if err != nil {
			// Skip if spec not found (orphaned link)
			if zammErr, ok := err.(*models.ZammError); ok && zammErr.Type == models.ErrTypeNotFound {
				continue
			}
			return nil, err
		}
		specs = append(specs, spec)
	}

	return specs, nil
}

// GetCommitsForSpec retrieves all commits linked to a spec
func (s *linkService) GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error) {
	if specID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	// Verify spec exists
	_, err := s.storage.GetSpecNode(specID)
	if err != nil {
		return nil, err
	}

	return s.storage.GetLinksBySpec(specID)
}

// UnlinkSpecFromCommit removes a link between a spec and commit
func (s *linkService) UnlinkSpecFromCommit(specID, commitID, repoPath string) error {
	if err := s.validateLinkInput(specID, commitID, repoPath, ""); err != nil {
		return err
	}

	return s.storage.DeleteSpecCommitLinkByFields(specID, commitID, repoPath)
}

// validateLinkInput validates input for link operations
func (s *linkService) validateLinkInput(specID, commitID, repoPath, label string) error {
	if specID == "" {
		return models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	if err := s.validateCommitID(commitID); err != nil {
		return err
	}

	if repoPath == "" {
		return models.NewZammError(models.ErrTypeValidation, "repository path cannot be empty")
	}

	return nil
}

// validateCommitInput validates commit-related input
func (s *linkService) validateCommitInput(commitID, repoPath string) error {
	if err := s.validateCommitID(commitID); err != nil {
		return err
	}

	if repoPath == "" {
		return models.NewZammError(models.ErrTypeValidation, "repository path cannot be empty")
	}

	return nil
}

// validateCommitID validates a Git commit hash
func (s *linkService) validateCommitID(commitID string) error {
	commitID = strings.TrimSpace(commitID)

	if commitID == "" {
		return models.NewZammError(models.ErrTypeValidation, "commit ID cannot be empty")
	}

	// Git commit hashes are 40-character hex strings (SHA-1) or 64-character hex strings (SHA-256)
	if len(commitID) != 40 && len(commitID) != 64 {
		return models.NewZammError(models.ErrTypeValidation, "commit ID must be a 40 or 64 character hex string")
	}

	// Validate hex characters
	matched, err := regexp.MatchString("^[a-fA-F0-9]+$", commitID)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to validate commit ID format", err)
	}

	if !matched {
		return models.NewZammError(models.ErrTypeValidation, "commit ID must contain only hexadecimal characters")
	}

	return nil
}

// validateRepoPath validates that a repository path exists and is accessible
func (s *linkService) validateRepoPath(repoPath string) error {
	repoPath = strings.TrimSpace(repoPath)

	if repoPath == "" {
		return models.NewZammError(models.ErrTypeValidation, "repository path cannot be empty")
	}

	// Check if path exists
	info, err := os.Stat(repoPath)
	if os.IsNotExist(err) {
		return models.NewZammError(models.ErrTypeValidation, fmt.Sprintf("repository path does not exist: %s", repoPath))
	}
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("failed to access repository path: %s", repoPath), err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return models.NewZammError(models.ErrTypeValidation, fmt.Sprintf("repository path is not a directory: %s", repoPath))
	}

	// Check if it's a Git repository (contains .git directory)
	gitPath := fmt.Sprintf("%s/.git", repoPath)
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		return models.NewZammError(models.ErrTypeGit, fmt.Sprintf("path is not a Git repository: %s", repoPath))
	}

	return nil
}
