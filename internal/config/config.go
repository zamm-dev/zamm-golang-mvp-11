package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// Config holds all configuration for the application
type Config struct {
	Storage StorageConfig `mapstructure:"storage"`
	Git     GitConfig     `mapstructure:"git"`
	Logging LoggingConfig `mapstructure:"logging"`
	CLI     CLIConfig     `mapstructure:"cli"`
}

// StorageConfig holds storage-related configuration
type StorageConfig struct {
	Path string `mapstructure:"path"`
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

// LocalMetadata represents the structure of local-metadata.json
type LocalMetadata struct {
	DataRedirect string `json:"data-redirect,omitempty"`
}

// resolveZammDir determines the correct zamm directory to use,
// checking for data-redirect in local-metadata.json
func resolveZammDir(workingDir string) (string, error) {
	localZammDir := filepath.Join(workingDir, ".zamm")
	metadataPath := filepath.Join(localZammDir, "local-metadata.json")

	// Check if local-metadata.json exists
	if _, err := os.Stat(metadataPath); err == nil {
		// Read the metadata file
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			return "", models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("failed to read local-metadata.json: %s", metadataPath), err)
		}

		var metadata LocalMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			return "", models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("failed to parse local-metadata.json: %s", metadataPath), err)
		}

		// If data-redirect is specified, use it
		if metadata.DataRedirect != "" {
			redirectPath := metadata.DataRedirect

			// Convert relative paths to absolute
			if !filepath.IsAbs(redirectPath) {
				redirectPath = filepath.Join(workingDir, redirectPath)
			}

			// Verify the redirected directory exists
			if _, err := os.Stat(redirectPath); os.IsNotExist(err) {
				return "", models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("data-redirect directory does not exist: %s", redirectPath), nil)
			}

			return redirectPath, nil
		}
	}

	// Use default local .zamm directory
	return localZammDir, nil
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Set up viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config paths - use resolved .zamm directory
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to get working directory", err)
	}

	zammDir, err := resolveZammDir(workingDir)
	if err != nil {
		return nil, err
	}

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
	if storagePath := os.Getenv("ZAMM_STORAGE_PATH"); storagePath != "" {
		config.Storage.Path = storagePath
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
	// Storage defaults
	viper.SetDefault("storage.path", zammDir)

	// Git defaults
	viper.SetDefault("git.default_repo", ".")

	// Logging defaults
	homeDir, err := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".zamm", "logs", "zamm.log")
	if err != nil {
		logPath = ".zamm/logs/zamm.log" // fallback
	}
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", logPath)

	// CLI defaults
	viper.SetDefault("cli.output_format", "table")
	viper.SetDefault("cli.color", "auto")
}

// expandPaths expands ~ and relative paths in configuration
func expandPaths(config *Config) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to get working directory", err)
	}

	// Expand storage path
	if config.Storage.Path != "" {
		config.Storage.Path = expandPath(config.Storage.Path, workingDir)
	}

	// Expand log file path
	if config.Logging.File != "" {
		config.Logging.File = expandPath(config.Logging.File, workingDir)
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
	// Ensure storage directory exists
	if config.Storage.Path != "" {
		if err := os.MkdirAll(config.Storage.Path, 0755); err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("failed to create storage directory: %s", config.Storage.Path), err)
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
	workingDir, err := os.Getwd()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to get working directory", err)
	}

	zammDir, err := resolveZammDir(workingDir)
	if err != nil {
		return err
	}

	configPath := filepath.Join(zammDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // Config already exists
	}

	// Create zamm directory
	if err := os.MkdirAll(zammDir, 0755); err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, fmt.Sprintf("failed to create zamm directory: %s", zammDir), err)
	}

	homeDir, err := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".zamm", "logs", "zamm.log")
	if err != nil {
		logPath = ".zamm/logs/zamm.log" // fallback
	}

	// Default config content
	configContent := `storage:
  path: .zamm

git:
  default_repo: .

logging:
  level: info
  file: "` + logPath + `"

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

	workingDir, err := os.Getwd()
	if err != nil {
		return "", models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to get working directory", err)
	}

	zammDir, err := resolveZammDir(workingDir)
	if err != nil {
		return "", err
	}

	return filepath.Join(zammDir, "config.yaml"), nil
}
