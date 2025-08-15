package interactive

import (
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/config"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
)

type AppAdapter struct {
	specService services.SpecService
	linkService services.LinkService
	storage     storage.Storage
	config      *config.Config
}

func NewAppAdapter(specService services.SpecService, linkService services.LinkService, storage storage.Storage, config *config.Config) *AppAdapter {
	return &AppAdapter{
		specService: specService,
		linkService: linkService,
		storage:     storage,
		config:      config,
	}
}

func (a *AppAdapter) SpecService() services.SpecService {
	return a.specService
}

func (a *AppAdapter) LinkService() services.LinkService {
	return a.linkService
}

func (a *AppAdapter) Storage() StorageInterface {
	return &StorageAdapter{storage: a.storage}
}

func (a *AppAdapter) Config() ConfigInterface {
	return &ConfigAdapter{config: a.config}
}

type StorageAdapter struct {
	storage storage.Storage
}

func (s *StorageAdapter) UpdateNode(node models.Node) error {
	return s.storage.UpdateNode(node)
}

type ConfigAdapter struct {
	config *config.Config
}

func (c *ConfigAdapter) GetGitConfig() GitConfigInterface {
	return &GitConfigAdapter{config: c.config}
}

type GitConfigAdapter struct {
	config *config.Config
}

func (g *GitConfigAdapter) GetDefaultRepo() string {
	return g.config.Git.DefaultRepo
}
