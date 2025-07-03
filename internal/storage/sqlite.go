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
	// No longer needed since migrations are embedded

	// Create migration service
	migrationService := NewMigrationService(db, "")

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

	now := time.Now()
	spec.CreatedAt = now
	spec.UpdatedAt = now

	query := `
		INSERT INTO spec_nodes (id, title, content, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, spec.ID, spec.Title, spec.Content, spec.CreatedAt, spec.UpdatedAt)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create spec", err)
	}

	return nil
}

// GetSpec retrieves a specification by ID
func (s *SQLiteStorage) GetSpec(id string) (*models.SpecNode, error) {
	query := `
		SELECT id, title, content, created_at, updated_at
		FROM spec_nodes WHERE id = ?
	`

	var spec models.SpecNode
	err := s.db.QueryRow(query, id).Scan(
		&spec.ID, &spec.Title, &spec.Content, &spec.CreatedAt, &spec.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.NewZammError(models.ErrTypeNotFound, fmt.Sprintf("spec with ID %s not found", id))
	}
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get spec", err)
	}

	return &spec, nil
}

// ListSpecs retrieves all specifications
func (s *SQLiteStorage) ListSpecs() ([]*models.SpecNode, error) {
	query := `
		SELECT id, title, content, created_at, updated_at
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
			&spec.ID, &spec.Title, &spec.Content, &spec.CreatedAt, &spec.UpdatedAt,
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
		fmt.Println("[DEBUG] Error creating spec link:", err)
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

// GetLinkedSpecs retrieves all related specs in a given direction relative to the specified spec
// ID
func (s *SQLiteStorage) GetLinkedSpecs(specID string, direction models.Direction) ([]*models.SpecNode, error) {
	var desiredMatch, retrievedNode string

	switch direction {
	case models.Incoming: // we're matching on links "to" this spec and retrieving the "from" specs
		desiredMatch = "to_spec_id"
		retrievedNode = "from_spec_id"
	case models.Outgoing: // we're matching on links "from" this spec and retrieving the "to" specs
		desiredMatch = "from_spec_id"
		retrievedNode = "to_spec_id"
	default:
		return nil, models.NewZammError(models.ErrTypeValidation, "invalid direction")
	}

	query := fmt.Sprintf(`
		SELECT sn.id, sn.title, sn.content, sn.created_at, sn.updated_at
		FROM spec_nodes sn
		INNER JOIN spec_spec_links ssl ON sn.id = ssl.%s
		WHERE ssl.%s = ? AND ssl.link_type = 'child'
		ORDER BY sn.created_at DESC`, retrievedNode, desiredMatch)

	rows, err := s.db.Query(query, specID)
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to query related specs", err)
	}
	defer rows.Close()

	var specs []*models.SpecNode
	for rows.Next() {
		var spec models.SpecNode
		err := rows.Scan(
			&spec.ID, &spec.Title, &spec.Content, &spec.CreatedAt, &spec.UpdatedAt,
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

// DeleteSpecLinkBySpecs deletes a spec-spec link by fromSpecID and toSpecID
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

// WouldCreateCycle checks if adding a link from fromSpecID to toSpecID would create a cycle
func (s *SQLiteStorage) WouldCreateCycle(fromSpecID, toSpecID string) (bool, error) {
	// If from and to are the same, it's a direct cycle
	if fromSpecID == toSpecID {
		return true, nil
	}

	// Use a recursive CTE to check if there's already a path from toSpecID to fromSpecID
	// If such a path exists, adding fromSpecID -> toSpecID would create a cycle
	query := `
		WITH RECURSIVE spec_path(spec_id) AS (
			SELECT to_spec_id FROM spec_spec_links WHERE from_spec_id = ?
			UNION
			SELECT ssl.to_spec_id
			FROM spec_spec_links ssl
			INNER JOIN spec_path sp ON ssl.from_spec_id = sp.spec_id
		)
		SELECT 1 FROM spec_path WHERE spec_id = ? LIMIT 1`

	row := s.db.QueryRow(query, toSpecID, fromSpecID)
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

// GetProjectMetadata retrieves the project metadata
func (s *SQLiteStorage) GetProjectMetadata() (*models.ProjectMetadata, error) {
	query := `
		SELECT id, root_spec_id, created_at, updated_at 
		FROM project_metadata 
		WHERE id = 1
	`

	var metadata models.ProjectMetadata
	row := s.db.QueryRow(query)
	err := row.Scan(&metadata.ID, &metadata.RootSpecID, &metadata.CreatedAt, &metadata.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.NewZammError(models.ErrTypeNotFound, "project metadata not found")
		}
		return nil, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get project metadata", err)
	}

	return &metadata, nil
}

// SetRootSpecID sets the root specification ID in project metadata
func (s *SQLiteStorage) SetRootSpecID(specID string) error {
	// First try to update existing metadata
	updateQuery := `
		UPDATE project_metadata 
		SET root_spec_id = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = 1
	`

	result, err := s.db.Exec(updateQuery, specID)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to update root spec ID", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to check update result", err)
	}

	// If no rows were affected, insert new metadata
	if rowsAffected == 0 {
		insertQuery := `
			INSERT INTO project_metadata (id, root_spec_id) 
			VALUES (1, ?)
		`

		_, err = s.db.Exec(insertQuery, specID)
		if err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to insert root spec ID", err)
		}
	}

	return nil
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

// GetOrphanSpecs returns all specs that don't have any outgoing "child" links
// (i.e., specs that are not children of any parent)
func (s *SQLiteStorage) GetOrphanSpecs() ([]*models.SpecNode, error) {
	query := `
		SELECT s.id, s.title, s.content, s.created_at, s.updated_at
		FROM spec_nodes s
		LEFT JOIN spec_spec_links ssl ON s.id = ssl.from_spec_id AND ssl.link_type = 'child'
		WHERE ssl.from_spec_id IS NULL
		ORDER BY s.created_at ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, models.NewZammError(models.ErrTypeStorage, "failed to query orphan specs: "+err.Error())
	}
	defer rows.Close()

	var specs []*models.SpecNode
	for rows.Next() {
		spec := &models.SpecNode{}
		err := rows.Scan(
			&spec.ID,
			&spec.Title,
			&spec.Content,
			&spec.CreatedAt,
			&spec.UpdatedAt,
		)
		if err != nil {
			return nil, models.NewZammError(models.ErrTypeStorage, "failed to scan orphan spec: "+err.Error())
		}
		specs = append(specs, spec)
	}

	if err = rows.Err(); err != nil {
		return nil, models.NewZammError(models.ErrTypeStorage, "error iterating orphan specs: "+err.Error())
	}

	return specs, nil
}

// MigrateDown migrates down to a specific version
func (s *SQLiteStorage) MigrateDown(targetVersion uint) error {
	return s.migrationService.MigrateDown(targetVersion)
}
