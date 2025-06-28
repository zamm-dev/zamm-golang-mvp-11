package cli

import (
	"github.com/yourorg/zamm-mvp/internal/config"
	"github.com/yourorg/zamm-mvp/internal/services"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

// App represents the CLI application
type App struct {
	config      *config.Config
	storage     storage.Storage
	specService services.SpecService
	linkService services.LinkService
}

// NewApp creates a new CLI application
func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if err := config.EnsureDirectories(cfg); err != nil {
		return nil, err
	}

	store, err := storage.NewSQLiteStorage(cfg.Database.Path)
	if err != nil {
		return nil, err
	}

	return &App{
		config:      cfg,
		storage:     store,
		specService: services.NewSpecService(store),
		linkService: services.NewLinkService(store),
	}, nil
}

// Close closes the application and cleans up resources
func (a *App) Close() error {
	if a.storage != nil {
		return a.storage.Close()
	}
	return nil
}
