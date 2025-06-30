package storage

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/yourorg/zamm-mvp/internal/models"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// MigrationService handles database migrations using golang-migrate
type MigrationService struct {
	db *sql.DB
}

// NewMigrationService creates a new migration service
func NewMigrationService(db *sql.DB, migrationDir string) *MigrationService {
	return &MigrationService{
		db: db,
	}
}

// RunMigrationsIfNeeded checks for pending migrations and runs them
func (m *MigrationService) RunMigrationsIfNeeded() error {
	// Get database path from the connection string
	rows, err := m.db.Query("PRAGMA database_list")
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get database info", err)
	}
	defer rows.Close()
	
	var seq int
	var name, file string
	if rows.Next() {
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to scan database info", err)
		}
	}
	rows.Close()
	
	// Create source driver from embedded files
	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create source driver", err)
	}
	
	// Create migrate instance
	databaseURL := fmt.Sprintf("sqlite3://%s", file)
	migrateInstance, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create migrate instance", err)
	}
	defer migrateInstance.Close()

	// Get current version
	currentVersion, dirty, err := migrateInstance.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get migration version", err)
	}

	if dirty {
		return models.NewZammError(models.ErrTypeStorage, "database is in dirty state, manual intervention required")
	}

	// Run migrations to latest version
	err = migrateInstance.Up()
	if err != nil && err != migrate.ErrNoChange {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to run migrations", err)
	}

	// Get new version to report what happened
	newVersion, _, err := migrateInstance.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get new migration version", err)
	}

	if err == migrate.ErrNilVersion {
		fmt.Println("No migrations found or database is empty")
	} else if currentVersion != newVersion {
		fmt.Printf("Database migrated from version %d to %d\n", currentVersion, newVersion)
	} else {
		fmt.Println("Database is already up to date")
	}

	return nil
}

// GetCurrentVersion returns the current migration version
func (m *MigrationService) GetCurrentVersion() (uint, bool, error) {
	// Get database path from the connection string
	rows, err := m.db.Query("PRAGMA database_list")
	if err != nil {
		return 0, false, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get database info", err)
	}
	defer rows.Close()
	
	var seq int
	var name, file string
	if rows.Next() {
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return 0, false, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to scan database info", err)
		}
	}
	rows.Close()

	// Create source driver from embedded files
	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return 0, false, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create source driver", err)
	}
	
	// Create migrate instance
	databaseURL := fmt.Sprintf("sqlite3://%s", file)
	migrateInstance, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return 0, false, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create migrate instance", err)
	}
	defer migrateInstance.Close()

	// Get current version
	version, dirty, err := migrateInstance.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get migration version", err)
	}

	if err == migrate.ErrNilVersion {
		return 0, dirty, nil
	}

	return version, dirty, nil
}

// ForceMigrationVersion forces the migration version (for recovery)
func (m *MigrationService) ForceMigrationVersion(version uint) error {
	// Get database path from the connection string
	rows, err := m.db.Query("PRAGMA database_list")
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get database info", err)
	}
	defer rows.Close()
	
	var seq int
	var name, file string
	if rows.Next() {
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to scan database info", err)
		}
	}
	rows.Close()

	// Create source driver from embedded files
	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create source driver", err)
	}
	
	// Create migrate instance
	databaseURL := fmt.Sprintf("sqlite3://%s", file)
	migrateInstance, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create migrate instance", err)
	}
	defer migrateInstance.Close()

	// Force version
	err = migrateInstance.Force(int(version))
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to force migration version", err)
	}

	fmt.Printf("Forced migration version to %d\n", version)
	return nil
}
