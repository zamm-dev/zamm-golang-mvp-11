package interactive

import (
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/config"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

type AppAdapter struct {
	specService services.SpecService
	linkService services.LinkService
	llmService  services.LLMService
	storage     storage.Storage
	config      *config.Config
}

func NewAppAdapter(specService services.SpecService, linkService services.LinkService, llmService services.LLMService, storage storage.Storage, config *config.Config) *AppAdapter {
	return &AppAdapter{
		specService: specService,
		linkService: linkService,
		llmService:  llmService,
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

func (a *AppAdapter) LLMService() services.LLMService {
	return a.llmService
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

func (s *StorageAdapter) WriteNode(node models.Node) error {
	return s.storage.WriteNode(node)
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
