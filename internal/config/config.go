package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// Config holds all configuration for the application
type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Git      GitConfig      `mapstructure:"git"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	CLI      CLIConfig      `mapstructure:"cli"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Path    string        `mapstructure:"path"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// GitConfig holds git-related configuration
type GitConfig struct {
	DefaultRepo string `mapstructure:"default_repo"`
}

// LoggingConfig holds logging-related configuration
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// CLIConfig holds CLI-related configuration
type CLIConfig struct {
	OutputFormat string `mapstructure:"output_format"`
	Color        string `mapstructure:"color"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Set up viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to get user home directory", err)
	}

	zammDir := filepath.Join(homeDir, ".zamm")
	viper.AddConfigPath(zammDir)
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults(zammDir)

	// Environment variable support
	viper.SetEnvPrefix("ZAMM")
	viper.AutomaticEnv()

	// Handle environment variable overrides
	if configPath := os.Getenv("ZAMM_CONFIG_PATH"); configPath != "" {
		viper.SetConfigFile(configPath)
	}

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is OK, we'll use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to read config file", err)
		}
	}

	// Unmarshal into struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to unmarshal config", err)
	}

	// Apply environment variable overrides
	if dbPath := os.Getenv("ZAMM_DB_PATH"); dbPath != "" {
		config.Database.Path = dbPath
	}
	if logLevel := os.Getenv("ZAMM_LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}
	if os.Getenv("ZAMM_NO_COLOR") != "" {
		config.CLI.Color = "never"
	}

	// Expand paths
	if err := expandPaths(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(zammDir string) {
	// Database defaults
	viper.SetDefault("database.path", filepath.Join(zammDir, "zamm.db"))
	viper.SetDefault("database.timeout", "30s")

	// Git defaults
	viper.SetDefault("git.default_repo", ".")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", filepath.Join(zammDir, "logs", "zamm.log"))

	// CLI defaults
	viper.SetDefault("cli.output_format", "table")
	viper.SetDefault("cli.color", "auto")
}

// expandPaths expands ~ and relative paths in configuration
func expandPaths(config *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to get user home directory", err)
	}

	// Expand database path
	if config.Database.Path != "" {
		config.Database.Path = expandPath(config.Database.Path, homeDir)
	}

	// Expand log file path
	if config.Logging.File != "" {
		config.Logging.File = expandPath(config.Logging.File, homeDir)
	}

	return nil
}

// expandPath expands ~ to home directory and resolves relative paths
func expandPath(path, homeDir string) string {
	if path == "" {
		return path
	}

	// Expand ~ to home directory
	if path[0] == '~' {
		if len(path) == 1 {
			return homeDir
		}
		if path[1] == '/' || path[1] == filepath.Separator {
			return filepath.Join(homeDir, path[2:])
		}
	}

	// Convert to absolute path
	if !filepath.IsAbs(path) {
		if absPath, err := filepath.Abs(path); err == nil {
			return absPath
		}
	}

	return path
}

// EnsureDirectories creates necessary directories for the configuration
func EnsureDirectories(config *Config) error {
	// Ensure database directory exists
	if config.Database.Path != "" {
		dbDir := filepath.Dir(config.Database.Path)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("failed to create database directory: %s", dbDir), err)
		}
	}

	// Ensure log directory exists
	if config.Logging.File != "" {
		logDir := filepath.Dir(config.Logging.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("failed to create log directory: %s", logDir), err)
		}
	}

	return nil
}

// WriteDefaultConfig writes a default configuration file
func WriteDefaultConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to get user home directory", err)
	}

	zammDir := filepath.Join(homeDir, ".zamm")
	configPath := filepath.Join(zammDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // Config already exists
	}

	// Create zamm directory
	if err := os.MkdirAll(zammDir, 0755); err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("failed to create zamm directory: %s", zammDir), err)
	}

	// Default config content
	configContent := `database:
  path: ~/.zamm/zamm.db
  timeout: 30s

git:
  default_repo: .

logging:
  level: info
  file: ~/.zamm/logs/zamm.log

cli:
  output_format: table
  color: auto
`

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("failed to write config file: %s", configPath), err)
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath() (string, error) {
	if configPath := os.Getenv("ZAMM_CONFIG_PATH"); configPath != "" {
		return configPath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to get user home directory", err)
	}

	return filepath.Join(homeDir, ".zamm", "config.yaml"), nil
}
