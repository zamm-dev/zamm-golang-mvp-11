package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// SQLiteStorage implements the Storage interface using SQLite
type SQLiteStorage struct {
	db               *sql.DB
	migrationService *MigrationService
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to open database", err)
	}

	// Find the migrations directory relative to the current working directory
	migrationDir := "migrations"

	// Create migration service
	migrationService := NewMigrationService(db, migrationDir)

	storage := &SQLiteStorage{
		db:               db,
		migrationService: migrationService,
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to connect to database", err)
	}

	return storage, nil
}

// CreateSpec creates a new specification node
func (s *SQLiteStorage) CreateSpec(spec *models.SpecNode) error {
	if spec.ID == "" {
		spec.ID = uuid.New().String()
	}
	if spec.StableID == "" {
		spec.StableID = uuid.New().String()
	}
	if spec.Version == 0 {
		spec.Version = 1
	}
	if spec.NodeType == "" {
		spec.NodeType = "spec"
	}

	now := time.Now()
	spec.CreatedAt = now
	spec.UpdatedAt = now

	query := `
		INSERT INTO spec_nodes (id, stable_id, version, title, content, node_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, spec.ID, spec.StableID, spec.Version, spec.Title, spec.Content, spec.NodeType, spec.CreatedAt, spec.UpdatedAt)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create spec", err)
	}

	return nil
}

// GetSpec retrieves a specification by ID
func (s *SQLiteStorage) GetSpec(id string) (*models.SpecNode, error) {
	query := `
		SELECT id, stable_id, version, title, content, node_type, created_at, updated_at
		FROM spec_nodes WHERE id = ?
	`

	var spec models.SpecNode
	err := s.db.QueryRow(query, id).Scan(
		&spec.ID, &spec.StableID, &spec.Version, &spec.Title,
		&spec.Content, &spec.NodeType, &spec.CreatedAt, &spec.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("spec with ID %s not found", id))
	}
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get spec", err)
	}

	return &spec, nil
}

// GetSpecByStableID retrieves a specification by stable ID and version
func (s *SQLiteStorage) GetSpecByStableID(stableID string, version int) (*models.SpecNode, error) {
	query := `
		SELECT id, stable_id, version, title, content, node_type, created_at, updated_at
		FROM spec_nodes WHERE stable_id = ? AND version = ?
	`

	var spec models.SpecNode
	err := s.db.QueryRow(query, stableID, version).Scan(
		&spec.ID, &spec.StableID, &spec.Version, &spec.Title,
		&spec.Content, &spec.NodeType, &spec.CreatedAt, &spec.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("spec with stable ID %s version %d not found", stableID, version))
	}
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get spec by stable ID", err)
	}

	return &spec, nil
}

// GetLatestSpecByStableID retrieves the latest version of a specification by stable ID
func (s *SQLiteStorage) GetLatestSpecByStableID(stableID string) (*models.SpecNode, error) {
	query := `
		SELECT id, stable_id, version, title, content, node_type, created_at, updated_at
		FROM spec_nodes WHERE stable_id = ? ORDER BY version DESC LIMIT 1
	`

	var spec models.SpecNode
	err := s.db.QueryRow(query, stableID).Scan(
		&spec.ID, &spec.StableID, &spec.Version, &spec.Title,
		&spec.Content, &spec.NodeType, &spec.CreatedAt, &spec.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("spec with stable ID %s not found", stableID))
	}
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get latest spec by stable ID", err)
	}

	return &spec, nil
}

// ListSpecs retrieves all specifications
func (s *SQLiteStorage) ListSpecs() ([]*models.SpecNode, error) {
	query := `
		SELECT id, stable_id, version, title, content, node_type, created_at, updated_at
		FROM spec_nodes ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to list specs", err)
	}
	defer rows.Close()

	var specs []*models.SpecNode
	for rows.Next() {
		var spec models.SpecNode
		err := rows.Scan(
			&spec.ID, &spec.StableID, &spec.Version, &spec.Title,
			&spec.Content, &spec.NodeType, &spec.CreatedAt, &spec.UpdatedAt,
		)
		if err != nil {
			return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to scan spec row", err)
		}
		specs = append(specs, &spec)
	}

	if err = rows.Err(); err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "error iterating spec rows", err)
	}

	return specs, nil
}

// UpdateSpec updates an existing specification
func (s *SQLiteStorage) UpdateSpec(spec *models.SpecNode) error {
	spec.UpdatedAt = time.Now()

	query := `
		UPDATE spec_nodes 
		SET title = ?, content = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := s.db.Exec(query, spec.Title, spec.Content, spec.UpdatedAt, spec.ID)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to update spec", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("spec with ID %s not found", spec.ID))
	}

	return nil
}

// DeleteSpec deletes a specification
func (s *SQLiteStorage) DeleteSpec(id string) error {
	query := `DELETE FROM spec_nodes WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to delete spec", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("spec with ID %s not found", id))
	}

	return nil
}

// CreateLink creates a new spec-commit link
func (s *SQLiteStorage) CreateLink(link *models.SpecCommitLink) error {
	if link.ID == "" {
		link.ID = uuid.New().String()
	}
	if link.LinkType == "" {
		link.LinkType = "implements"
	}

	link.CreatedAt = time.Now()

	query := `
		INSERT INTO spec_commit_links (id, spec_id, commit_id, repo_path, link_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, link.ID, link.SpecID, link.CommitID, link.RepoPath, link.LinkType, link.CreatedAt)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create link", err)
	}

	return nil
}

// GetLink retrieves a link by ID
func (s *SQLiteStorage) GetLink(id string) (*models.SpecCommitLink, error) {
	query := `
		SELECT id, spec_id, commit_id, repo_path, link_type, created_at
		FROM spec_commit_links WHERE id = ?
	`

	var link models.SpecCommitLink
	err := s.db.QueryRow(query, id).Scan(
		&link.ID, &link.SpecID, &link.CommitID, &link.RepoPath, &link.LinkType, &link.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("link with ID %s not found", id))
	}
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get link", err)
	}

	return &link, nil
}

// GetLinksBySpec retrieves all links for a specification
func (s *SQLiteStorage) GetLinksBySpec(specID string) ([]*models.SpecCommitLink, error) {
	query := `
		SELECT id, spec_id, commit_id, repo_path, link_type, created_at
		FROM spec_commit_links WHERE spec_id = ? ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, specID)
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get links by spec", err)
	}
	defer rows.Close()

	var links []*models.SpecCommitLink
	for rows.Next() {
		var link models.SpecCommitLink
		err := rows.Scan(
			&link.ID, &link.SpecID, &link.CommitID, &link.RepoPath, &link.LinkType, &link.CreatedAt,
		)
		if err != nil {
			return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to scan link row", err)
		}
		links = append(links, &link)
	}

	if err = rows.Err(); err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "error iterating link rows", err)
	}

	return links, nil
}

// GetLinksByCommit retrieves all links for a commit
func (s *SQLiteStorage) GetLinksByCommit(commitID, repoPath string) ([]*models.SpecCommitLink, error) {
	query := `
		SELECT id, spec_id, commit_id, repo_path, link_type, created_at
		FROM spec_commit_links WHERE commit_id = ? AND repo_path = ? ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, commitID, repoPath)
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get links by commit", err)
	}
	defer rows.Close()

	var links []*models.SpecCommitLink
	for rows.Next() {
		var link models.SpecCommitLink
		err := rows.Scan(
			&link.ID, &link.SpecID, &link.CommitID, &link.RepoPath, &link.LinkType, &link.CreatedAt,
		)
		if err != nil {
			return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to scan link row", err)
		}
		links = append(links, &link)
	}

	if err = rows.Err(); err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "error iterating link rows", err)
	}

	return links, nil
}

// DeleteLink deletes a link
func (s *SQLiteStorage) DeleteLink(id string) error {
	query := `DELETE FROM spec_commit_links WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to delete link", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("link with ID %s not found", id))
	}

	return nil
}

// CreateSpecLink creates a new hierarchical link between two specifications
func (s *SQLiteStorage) CreateSpecLink(link *models.SpecSpecLink) error {
	if link.ID == "" {
		link.ID = uuid.New().String()
	}

	now := time.Now()
	link.CreatedAt = now

	// First check if this would create a cycle
	wouldCycle, err := s.WouldCreateCycle(link.FromSpecID, link.ToSpecID)
	if err != nil {
		return err
	}
	if wouldCycle {
		return models.NewZammError(models.ErrTypeValidation, "creating this link would create a cycle in the spec hierarchy")
	}

	query := `INSERT INTO spec_spec_links (id, from_spec_id, to_spec_id, link_type, created_at) 
			  VALUES (?, ?, ?, ?, ?)`

	_, err = s.db.Exec(query, link.ID, link.FromSpecID, link.ToSpecID, link.LinkType, link.CreatedAt)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create spec link", err)
	}

	return nil
}

// GetSpecLink retrieves a spec-spec link by ID
func (s *SQLiteStorage) GetSpecLink(id string) (*models.SpecSpecLink, error) {
	query := `SELECT id, from_spec_id, to_spec_id, link_type, created_at 
			  FROM spec_spec_links WHERE id = ?`

	row := s.db.QueryRow(query, id)

	var link models.SpecSpecLink
	err := row.Scan(&link.ID, &link.FromSpecID, &link.ToSpecID, &link.LinkType, &link.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("spec link with ID %s not found", id))
		}
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get spec link", err)
	}

	return &link, nil
}

// GetParentSpecs retrieves all parent spec links for a given spec ID
func (s *SQLiteStorage) GetParentSpecs(specID string) ([]*models.SpecSpecLink, error) {
	query := `SELECT id, from_spec_id, to_spec_id, link_type, created_at 
			  FROM spec_spec_links WHERE from_spec_id = ? AND link_type = 'child' ORDER BY created_at DESC`

	rows, err := s.db.Query(query, specID)
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to query parent spec links", err)
	}
	defer rows.Close()

	var links []*models.SpecSpecLink
	for rows.Next() {
		var link models.SpecSpecLink
		err := rows.Scan(&link.ID, &link.FromSpecID, &link.ToSpecID, &link.LinkType, &link.CreatedAt)
		if err != nil {
			return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to scan spec link row", err)
		}
		links = append(links, &link)
	}

	if err = rows.Err(); err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "error iterating spec link rows", err)
	}

	return links, nil
}

// GetChildSpecs retrieves all child spec links for a given spec ID
func (s *SQLiteStorage) GetChildSpecs(specID string) ([]*models.SpecSpecLink, error) {
	query := `SELECT id, from_spec_id, to_spec_id, link_type, created_at 
			  FROM spec_spec_links WHERE to_spec_id = ? AND link_type = 'child' ORDER BY created_at DESC`

	rows, err := s.db.Query(query, specID)
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to query child spec links", err)
	}
	defer rows.Close()

	var links []*models.SpecSpecLink
	for rows.Next() {
		var link models.SpecSpecLink
		err := rows.Scan(&link.ID, &link.FromSpecID, &link.ToSpecID, &link.LinkType, &link.CreatedAt)
		if err != nil {
			return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to scan spec link row", err)
		}
		links = append(links, &link)
	}

	if err = rows.Err(); err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "error iterating spec link rows", err)
	}

	return links, nil
}

// DeleteSpecLink deletes a spec-spec link by ID
func (s *SQLiteStorage) DeleteSpecLink(id string) error {
	query := `DELETE FROM spec_spec_links WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to delete spec link", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("spec link with ID %s not found", id))
	}

	return nil
}

// DeleteSpecLinkBySpecs deletes a spec-spec link by parent and child spec IDs
func (s *SQLiteStorage) DeleteSpecLinkBySpecs(fromSpecID, toSpecID string) error {
	query := `DELETE FROM spec_spec_links WHERE from_spec_id = ? AND to_spec_id = ?`

	result, err := s.db.Exec(query, fromSpecID, toSpecID)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to delete spec link", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return models.NewZammError(models.ErrTypeNotFound, "spec link not found")
	}

	return nil
}

// WouldCreateCycle checks if adding a link from parentSpecID to childSpecID would create a cycle
func (s *SQLiteStorage) WouldCreateCycle(parentSpecID, childSpecID string) (bool, error) {
	// If parent and child are the same, it's a direct cycle
	if parentSpecID == childSpecID {
		return true, nil
	}

	// Use a recursive CTE to check if there's already a path from childSpecID to parentSpecID
	// If such a path exists, adding parentSpecID -> childSpecID would create a cycle
	query := `
		WITH RECURSIVE spec_path(spec_id) AS (
			SELECT from_spec_id FROM spec_spec_links WHERE to_spec_id = ?
			UNION
			SELECT ssl.from_spec_id
			FROM spec_spec_links ssl
			INNER JOIN spec_path sp ON ssl.to_spec_id = sp.spec_id
		)
		SELECT 1 FROM spec_path WHERE spec_id = ? LIMIT 1`

	row := s.db.QueryRow(query, childSpecID, parentSpecID)
	var exists int
	err := row.Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // No cycle would be created
		}
		return false, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to check for cycles", err)
	}

	return true, nil // A cycle would be created
}

// RunMigrationsIfNeeded runs any pending migrations
func (s *SQLiteStorage) RunMigrationsIfNeeded() error {
	return s.migrationService.RunMigrationsIfNeeded()
}

// GetMigrationVersion returns the current migration version
func (s *SQLiteStorage) GetMigrationVersion() (uint, bool, error) {
	return s.migrationService.GetCurrentVersion()
}

// ForceMigrationVersion forces the migration version (for recovery)
func (s *SQLiteStorage) ForceMigrationVersion(version uint) error {
	return s.migrationService.ForceMigrationVersion(version)
}

// BackupDatabase creates a backup of the database to the specified path
func (s *SQLiteStorage) BackupDatabase(backupPath string) error {
	// Use SQLite's backup API via SQL commands
	query := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
	_, err := s.db.Exec(query)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to backup database", err)
	}
	return nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
