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

	store := storage.NewFileStorage(cfg.Storage.Path)

	return &App{
		config:      cfg,
		storage:     store,
		specService: services.NewSpecService(store),
		linkService: services.NewLinkService(store),
	}, nil
}

// Close closes the application and cleans up resources
func (a *App) Close() error {
	// File-based storage doesn't need to be closed
	return nil
}
