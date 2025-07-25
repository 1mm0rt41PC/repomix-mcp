// ************************************************************************************************
// Package config provides configuration management for the repomix-mcp application.
// It handles loading, validation, and management of application settings including
// repository configurations, cache settings, and server parameters.
package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"repomix-mcp/pkg/types"
)

// ************************************************************************************************
// Manager handles configuration loading, validation, and management.
// It provides centralized access to application configuration with validation
// and environment variable support.
type Manager struct {
	config     *types.Config
	configPath string
}

// ************************************************************************************************
// NewManager creates a new configuration manager instance.
// It initializes the configuration system with default values and prepares
// for configuration loading from files and environment variables.
//
// Returns:
//   - *Manager: The configuration manager instance.
//
// Example usage:
//
//	manager := NewManager()
//	err := manager.LoadConfig("./config.json")
func NewManager() *Manager {
	return &Manager{}
}

// ************************************************************************************************
// LoadConfig loads configuration from the specified file path.
// It supports JSON configuration files and validates the loaded configuration.
//
// Returns:
//   - error: An error if configuration loading or validation fails.
//
// Example usage:
//
//	err := manager.LoadConfig("./config.json")
//	if err != nil {
//		return fmt.Errorf("failed to load config: %w", err)
//	}
func (m *Manager) LoadConfig(configPath string) error {
	if configPath == "" {
		return fmt.Errorf("%w: config path is empty", types.ErrInvalidConfig)
	}
	
	// Expand home directory if needed
	if strings.HasPrefix(configPath, "~") {
		homeDir, err := mock_osUserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory\n>    %w", err)
		}
		configPath = filepath.Join(homeDir, configPath[1:])
	}
	
	// Check if file exists
	if _, err := mock_osStat(configPath); mock_osIsNotExist(err) {
		return fmt.Errorf("%w: %s", types.ErrConfigNotFound, configPath)
	}
	
	m.configPath = configPath
	
	// Read file content
	data, err := mock_osReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file\n>    %w", err)
	}
	
	// Parse JSON
	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file\n>    %w", err)
	}
	
	// Validate configuration
	if err := m.validateConfig(&config); err != nil {
		return fmt.Errorf("config validation failed\n>    %w", err)
	}
	
	m.config = &config
	return nil
}

// ************************************************************************************************
// LoadConfigFromJSON loads configuration directly from JSON bytes.
// This method is useful for testing or when configuration comes from sources
// other than files.
//
// Returns:
//   - error: An error if configuration parsing or validation fails.
//
// Example usage:
//
//	jsonData := []byte(`{"repositories": {...}}`)
//	err := manager.LoadConfigFromJSON(jsonData)
func (m *Manager) LoadConfigFromJSON(jsonData []byte) error {
	var config types.Config
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return fmt.Errorf("failed to parse JSON config\n>    %w", err)
	}
	
	if err := m.validateConfig(&config); err != nil {
		return fmt.Errorf("config validation failed\n>    %w", err)
	}
	
	m.config = &config
	return nil
}

// ************************************************************************************************
// validateConfig validates the loaded configuration for consistency and completeness.
// It checks repository configurations, cache settings, and server parameters.
//
// Returns:
//   - error: An error if validation fails.
func (m *Manager) validateConfig(config *types.Config) error {
	if config == nil {
		return fmt.Errorf("%w: config is nil", types.ErrInvalidConfig)
	}
	
	// Validate repositories
	if len(config.Repositories) == 0 {
		return fmt.Errorf("%w: no repositories configured", types.ErrInvalidConfig)
	}
	
	for alias, repo := range config.Repositories {
		if err := m.validateRepository(alias, &repo); err != nil {
			return fmt.Errorf("invalid repository '%s'\n>    %w", alias, err)
		}
	}
	
	// Validate cache configuration
	if err := m.validateCache(&config.Cache); err != nil {
		return fmt.Errorf("invalid cache config\n>    %w", err)
	}
	
	// Validate server configuration
	if err := m.validateServer(&config.Server); err != nil {
		return fmt.Errorf("invalid server config\n>    %w", err)
	}
	
	return nil
}

// ************************************************************************************************
// validateRepository validates a single repository configuration.
//
// Returns:
//   - error: An error if repository configuration is invalid.
func (m *Manager) validateRepository(alias string, repo *types.RepositoryConfig) error {
	if alias == "" {
		return fmt.Errorf("%w: repository alias cannot be empty", types.ErrInvalidConfig)
	}
	
	if repo.Type != types.RepositoryTypeLocal && repo.Type != types.RepositoryTypeRemote {
		return fmt.Errorf("%w: %s", types.ErrInvalidRepositoryType, repo.Type)
	}
	
	// Validate paths/URLs
	if repo.Type == types.RepositoryTypeLocal {
		if repo.Path == "" {
			return fmt.Errorf("%w: local repository path cannot be empty", types.ErrInvalidConfig)
		}
	} else {
		if repo.URL == "" {
			return fmt.Errorf("%w: remote repository URL cannot be empty", types.ErrInvalidConfig)
		}
	}
	
	// Validate authentication
	if err := m.validateAuth(&repo.Auth); err != nil {
		return fmt.Errorf("invalid auth config\n>    %w", err)
	}
	
	// Set default branch if not specified
	if repo.Branch == "" {
		repo.Branch = "main"
	}
	
	return nil
}

// ************************************************************************************************
// validateAuth validates authentication configuration.
//
// Returns:
//   - error: An error if authentication configuration is invalid.
func (m *Manager) validateAuth(auth *types.RepositoryAuth) error {
	switch auth.Type {
	case types.AuthTypeNone:
		// No validation needed
	case types.AuthTypeSSH:
		if auth.KeyPath == "" {
			return fmt.Errorf("%w: SSH key path required for SSH auth", types.ErrInvalidConfig)
		}
	case types.AuthTypeToken:
		if auth.Token == "" {
			return fmt.Errorf("%w: token required for token auth", types.ErrInvalidConfig)
		}
	default:
		return fmt.Errorf("%w: unknown auth type: %s", types.ErrInvalidConfig, auth.Type)
	}
	
	return nil
}

// ************************************************************************************************
// validateCache validates cache configuration.
//
// Returns:
//   - error: An error if cache configuration is invalid.
func (m *Manager) validateCache(cache *types.CacheConfig) error {
	if cache.Path == "" {
		return fmt.Errorf("%w: cache path cannot be empty", types.ErrInvalidConfig)
	}
	
	// Expand home directory in cache path
	if strings.HasPrefix(cache.Path, "~") {
		homeDir, err := mock_osUserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory\n>    %w", err)
		}
		cache.Path = filepath.Join(homeDir, cache.Path[1:])
	}
	
	return nil
}

// ************************************************************************************************
// validateServer validates server configuration.
//
// Returns:
//   - error: An error if server configuration is invalid.
func (m *Manager) validateServer(server *types.ServerConfig) error {
	if server.Port <= 0 || server.Port > 65535 {
		return fmt.Errorf("%w: invalid port number: %d", types.ErrInvalidConfig, server.Port)
	}
	
	validLogLevels := []string{"trace", "debug", "info", "warning", "error", "critical"}
	isValidLevel := false
	for _, level := range validLogLevels {
		if server.LogLevel == level {
			isValidLevel = true
			break
		}
	}
	
	if !isValidLevel {
		return fmt.Errorf("%w: invalid log level: %s", types.ErrInvalidConfig, server.LogLevel)
	}
	
	// Set HTTPS defaults
	if server.HTTPSPort == 0 {
		server.HTTPSPort = 9443
	}
	
	// Validate HTTPS configuration
	if server.HTTPSEnabled {
		if server.HTTPSPort <= 0 || server.HTTPSPort > 65535 {
			return fmt.Errorf("%w: invalid HTTPS port number: %d", types.ErrInvalidConfig, server.HTTPSPort)
		}
		
		// If not auto-generating certificate, validate paths
		if !server.AutoGenCert {
			if server.CertPath == "" {
				return fmt.Errorf("%w: certificate path required when HTTPS is enabled and autoGenCert is false", types.ErrInvalidConfig)
			}
			if server.KeyPath == "" {
				return fmt.Errorf("%w: key path required when HTTPS is enabled and autoGenCert is false", types.ErrInvalidConfig)
			}
			
			// Expand home directory in certificate paths
			if strings.HasPrefix(server.CertPath, "~") {
				homeDir, err := mock_osUserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory for cert path\n>    %w", err)
				}
				server.CertPath = filepath.Join(homeDir, server.CertPath[1:])
			}
			
			if strings.HasPrefix(server.KeyPath, "~") {
				homeDir, err := mock_osUserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory for key path\n>    %w", err)
				}
				server.KeyPath = filepath.Join(homeDir, server.KeyPath[1:])
			}
		}
		
		// Ensure HTTP and HTTPS ports are different
		if server.Port == server.HTTPSPort {
			return fmt.Errorf("%w: HTTP and HTTPS ports must be different", types.ErrInvalidConfig)
		}
	}
	
	return nil
}

// ************************************************************************************************
// GetConfig returns the current configuration.
// Returns nil if no configuration has been loaded.
//
// Returns:
//   - *types.Config: The current configuration, or nil if not loaded.
//
// Example usage:
//
//	config := manager.GetConfig()
//	if config == nil {
//		return fmt.Errorf("configuration not loaded")
//	}
func (m *Manager) GetConfig() *types.Config {
	return m.config
}

// ************************************************************************************************
// GetRepository returns the configuration for a specific repository by alias.
//
// Returns:
//   - *types.RepositoryConfig: The repository configuration.
//   - error: An error if the repository is not found.
//
// Example usage:
//
//	repo, err := manager.GetRepository("my-repo")
//	if err != nil {
//		return fmt.Errorf("repository not found: %w", err)
//	}
func (m *Manager) GetRepository(alias string) (*types.RepositoryConfig, error) {
	if m.config == nil {
		return nil, fmt.Errorf("%w: configuration not loaded", types.ErrNotInitialized)
	}
	
	repo, exists := m.config.Repositories[alias]
	if !exists {
		return nil, fmt.Errorf("%w: %s", types.ErrRepositoryNotFound, alias)
	}
	
	return &repo, nil
}

// ************************************************************************************************
// GetRepositoryAliases returns all configured repository aliases.
//
// Returns:
//   - []string: List of repository aliases.
//
// Example usage:
//
//	aliases := manager.GetRepositoryAliases()
//	for _, alias := range aliases {
//		repo, _ := manager.GetRepository(alias)
//		// Process repository...
//	}
func (m *Manager) GetRepositoryAliases() []string {
	if m.config == nil {
		return nil
	}
	
	aliases := make([]string, 0, len(m.config.Repositories))
	for alias := range m.config.Repositories {
		aliases = append(aliases, alias)
	}
	
	return aliases
}

// ************************************************************************************************
// SaveConfig saves the current configuration to the specified file path.
//
// Returns:
//   - error: An error if saving fails.
//
// Example usage:
//
//	err := manager.SaveConfig("./config.json")
//	if err != nil {
//		return fmt.Errorf("failed to save config: %w", err)
//	}
func (m *Manager) SaveConfig(configPath string) error {
	if m.config == nil {
		return fmt.Errorf("%w: no configuration to save", types.ErrNotInitialized)
	}
	
	if configPath == "" {
		configPath = m.configPath
	}
	
	if configPath == "" {
		return fmt.Errorf("%w: no config path specified", types.ErrInvalidPath)
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := mock_osMkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory\n>    %w", err)
	}
	
	// Marshal config to JSON
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config\n>    %w", err)
	}
	
	// Write to file
	if err := mock_osWriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file\n>    %w", err)
	}
	
	m.configPath = configPath
	return nil
}

// ************************************************************************************************
// CreateExampleConfig creates an example configuration file.
//
// Returns:
//   - error: An error if creation fails.
//
// Example usage:
//
//	err := manager.CreateExampleConfig("./config.example.json")
func (m *Manager) CreateExampleConfig(configPath string) error {
	exampleConfig := &types.Config{
		Repositories: map[string]types.RepositoryConfig{
			"my-local-repo": {
				Type: types.RepositoryTypeLocal,
				Path: "/path/to/local/repo",
				Auth: types.RepositoryAuth{Type: types.AuthTypeNone},
				Indexing: types.IndexingConfig{
					Enabled:           true,
					ExcludePatterns:   []string{"*.log", "node_modules", ".git", "vendor"},
					IncludePatterns:   []string{"*.go", "*.md", "*.json", "*.yaml", "*.yml"},
					MaxFileSize:       "1MB",
					IncludeNonExported: false,
				},
				Branch: "main",
			},
			"my-remote-repo": {
				Type: types.RepositoryTypeRemote,
				URL:  "git@github.com:org/repo.git",
				Auth: types.RepositoryAuth{
					Type:    types.AuthTypeSSH,
					KeyPath: "~/.ssh/id_rsa",
				},
				Indexing: types.IndexingConfig{
					Enabled:           true,
					ExcludePatterns:   []string{"*.log", "node_modules", ".git"},
					IncludePatterns:   []string{"*.js", "*.ts", "*.md"},
					MaxFileSize:       "1MB",
					IncludeNonExported: false,
				},
				Branch: "main",
			},
		},
		Cache: types.CacheConfig{
			Path:    "~/.repomix-mcp",
			MaxSize: "1GB",
			TTL:     "24h",
		},
		Server: types.ServerConfig{
			Port:         8080,
			Host:         "localhost",
			LogLevel:     "info",
			HTTPSEnabled: true,
			HTTPSPort:    9443,
			CertPath:     "~/.repomix-mcp/server.crt",
			KeyPath:      "~/.repomix-mcp/server.key",
			AutoGenCert:  true,
		},
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := mock_osMkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory\n>    %w", err)
	}
	
	// Marshal config to JSON
	data, err := json.MarshalIndent(exampleConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal example config\n>    %w", err)
	}
	
	// Write to file
	if err := mock_osWriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write example config file\n>    %w", err)
	}
	
	return nil
}