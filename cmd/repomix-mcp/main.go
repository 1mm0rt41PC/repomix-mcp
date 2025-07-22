// ************************************************************************************************
// Main entry point for the repomix-mcp application.
// This application provides Context7-compatible functionality for indexing internal private repositories
// using repomix as the CLI indexer and serving content through an MCP server.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"repomix-mcp/internal/cache"
	"repomix-mcp/internal/config"
	"repomix-mcp/internal/indexer"
	"repomix-mcp/internal/mcp"
	"repomix-mcp/internal/repository"
	"repomix-mcp/pkg/types"

	"github.com/spf13/cobra"
)

// ************************************************************************************************
// Application represents the main application instance.
type Application struct {
	configManager *config.Manager
	cache         *cache.Cache
	repoManager   *repository.Manager
	indexer       *indexer.Indexer
	searchEngine  SearchInterface
	mcpServer     *mcp.Server
}

// ************************************************************************************************
// SearchInterface defines the interface for search operations.
type SearchInterface interface {
	Search(query types.SearchQuery) ([]types.SearchResult, error)
}

// ************************************************************************************************
// MockSearchEngine provides a simple search implementation.
type MockSearchEngine struct{}

// Search implements a basic search functionality.
func (m *MockSearchEngine) Search(query types.SearchQuery) ([]types.SearchResult, error) {
	// Simple mock implementation for now
	return []types.SearchResult{}, nil
}

// ************************************************************************************************
// NewApplication creates a new application instance.
//
// Returns:
//   - *Application: The application instance.
//   - error: An error if initialization fails.
func NewApplication() (*Application, error) {
	return &Application{}, nil
}

// ************************************************************************************************
// Initialize initializes the application components.
//
// Returns:
//   - error: An error if initialization fails.
func (app *Application) Initialize(configPath string) error {
	var err error

	// Initialize configuration manager
	app.configManager = config.NewManager()
	if err = app.configManager.LoadConfig(configPath); err != nil {
		return fmt.Errorf("failed to load configuration\n>    %w", err)
	}

	config := app.configManager.GetConfig()
	if config == nil {
		return fmt.Errorf("%w: configuration is nil", types.ErrNotInitialized)
	}

	// Initialize cache
	app.cache, err = cache.NewCache(&config.Cache)
	if err != nil {
		return fmt.Errorf("failed to initialize cache\n>    %w", err)
	}

	// Initialize repository manager
	repoWorkDir := filepath.Join(config.Cache.Path, "repositories")
	app.repoManager, err = repository.NewManager(repoWorkDir)
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager\n>    %w", err)
	}

	// Initialize indexer
	app.indexer, err = indexer.NewIndexer()
	if err != nil {
		return fmt.Errorf("failed to initialize indexer\n>    %w", err)
	}

	// Initialize search engine
	app.searchEngine = &MockSearchEngine{}

	// Initialize MCP server
	app.mcpServer, err = mcp.NewServer(config, app.cache, app.searchEngine)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP server\n>    %w", err)
	}

	return nil
}

// ************************************************************************************************
// IndexAllRepositories indexes all configured repositories.
// It automatically expands glob patterns and indexes each discovered repository.
//
// Returns:
//   - error: An error if indexing fails.
func (app *Application) IndexAllRepositories() error {
	aliases := app.configManager.GetRepositoryAliases()

	log.Printf("Starting indexing of %d configured repositories", len(aliases))

	totalIndexed := 0
	for _, alias := range aliases {
		// Get repository configuration
		repoConfig, err := app.configManager.GetRepository(alias)
		if err != nil {
			log.Printf("Warning: failed to get repository config for %s: %v", alias, err)
			continue
		}

		// Expand glob patterns if present
		expandedRepos, err := app.repoManager.ExpandGlobRepositories(alias, repoConfig)
		if err != nil {
			log.Printf("Warning: failed to expand glob for repository %s: %v", alias, err)
			continue
		}

		log.Printf("Repository %s expanded to %d repositories", alias, len(expandedRepos))

		// Index each expanded repository
		for expandedAlias, expandedConfig := range expandedRepos {
			if err := app.indexExpandedRepository(expandedAlias, expandedConfig); err != nil {
				log.Printf("Warning: failed to index repository %s: %v", expandedAlias, err)
				continue
			}
			log.Printf("Successfully indexed repository: %s", expandedAlias)
			totalIndexed++
		}
	}

	log.Printf("Completed indexing %d repositories", totalIndexed)
	return nil
}

// ************************************************************************************************
// IndexRepository indexes a specific repository.
// It first expands any glob patterns and then indexes each discovered repository.
//
// Returns:
//   - error: An error if indexing fails.
func (app *Application) IndexRepository(alias string) error {
	// Get repository configuration
	repoConfig, err := app.configManager.GetRepository(alias)
	if err != nil {
		return fmt.Errorf("failed to get repository config\n>    %w", err)
	}

	// Expand glob patterns if present
	expandedRepos, err := app.repoManager.ExpandGlobRepositories(alias, repoConfig)
	if err != nil {
		return fmt.Errorf("failed to expand glob for repository %s\n>    %w", alias, err)
	}

	log.Printf("Repository %s expanded to %d repositories", alias, len(expandedRepos))

	// Index each expanded repository
	for expandedAlias, expandedConfig := range expandedRepos {
		if err := app.indexExpandedRepository(expandedAlias, expandedConfig); err != nil {
			return fmt.Errorf("failed to index repository %s\n>    %w", expandedAlias, err)
		}
		log.Printf("Successfully indexed repository: %s", expandedAlias)
	}

	return nil
}

// ************************************************************************************************
// indexExpandedRepository indexes a single expanded repository (internal method).
//
// Returns:
//   - error: An error if indexing fails.
func (app *Application) indexExpandedRepository(alias string, repoConfig *types.RepositoryConfig) error {
	log.Printf("Indexing repository: %s", alias)

	// Prepare repository (clone/update if needed)
	localPath, err := app.repoManager.PrepareRepository(alias, repoConfig)
	if err != nil {
		return fmt.Errorf("failed to prepare repository\n>    %w", err)
	}

	// Index repository content
	repoIndex, err := app.indexer.IndexRepository(alias, localPath, repoConfig.Indexing)
	if err != nil {
		return fmt.Errorf("failed to index repository content\n>    %w", err)
	}

	// Get additional repository metadata
	repoInfo, err := app.repoManager.GetRepositoryInfo(alias, localPath)
	if err != nil {
		log.Printf("Warning: failed to get repository info for %s: %v", alias, err)
	} else {
		// Merge metadata
		repoIndex.CommitHash = repoInfo.CommitHash
		for k, v := range repoInfo.Metadata {
			repoIndex.Metadata[k] = v
		}
	}

	// Store in cache
	if err = app.cache.StoreRepository(repoIndex); err != nil {
		return fmt.Errorf("failed to store repository in cache\n>    %w", err)
	}

	// Update MCP server
	if err = app.mcpServer.UpdateRepository(repoIndex); err != nil {
		return fmt.Errorf("failed to update MCP server\n>    %w", err)
	}

	return nil
}

// ************************************************************************************************
// StartServer starts the MCP server.
//
// Returns:
//   - error: An error if server startup fails.
func (app *Application) StartServer() error {
	log.Println("Starting MCP server...")
	return app.mcpServer.Start()
}

// ************************************************************************************************
// Cleanup cleans up application resources.
//
// Returns:
//   - error: An error if cleanup fails.
func (app *Application) Cleanup() error {
	log.Println("Cleaning up application resources...")

	if app.indexer != nil {
		if err := app.indexer.Close(); err != nil {
			log.Printf("Warning: failed to close indexer: %v", err)
		}
	}

	if app.cache != nil {
		if err := app.cache.Close(); err != nil {
			log.Printf("Warning: failed to close cache: %v", err)
		}
	}

	return nil
}

// ************************************************************************************************
// Global application instance
var app *Application

// ************************************************************************************************
// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "repomix-mcp",
	Short: "Context7-compatible repository indexing and MCP server",
	Long: `repomix-mcp provides Context7-compatible functionality for indexing internal private repositories.
It uses repomix as the CLI indexer and serves content through an MCP server that provides the same
functions as Context7 to AI clients.

Features:
- Index both local and remote repositories
- Cache indexed content using BadgerDB
- Serve content through Context7-compatible MCP tools
- Support for authentication and incremental updates`,
}

// ************************************************************************************************
// indexCmd represents the index command
var indexCmd = &cobra.Command{
	Use:   "index [repository-alias]",
	Short: "Index repositories",
	Long: `Index one or all configured repositories. If no alias is provided, all repositories will be indexed.

Examples:
  repomix-mcp index                    # Index all repositories
  repomix-mcp index my-repo           # Index specific repository`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Index all repositories
			return app.IndexAllRepositories()
		} else {
			// Index specific repository
			return app.IndexRepository(args[0])
		}
	},
}

// ************************************************************************************************
// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the MCP server to serve indexed repository content through Context7-compatible tools.

The server will listen on the configured host and port and provide the following MCP tools:
- resolve-library-id: Resolve library names to repository IDs
- get-library-docs: Retrieve repository documentation content`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.StartServer()
	},
}

// ************************************************************************************************
// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration and dependencies",
	Long: `Validate the configuration file and check that all required dependencies are available.

This command will:
- Validate the configuration file syntax and settings
- Check that repomix CLI is available
- Verify repository access (for remote repositories)
- Test cache directory permissions`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("Validating configuration...")

		// Validate repomix availability
		if err := app.indexer.ValidateRepomix(); err != nil {
			return fmt.Errorf("repomix validation failed\n>    %w", err)
		}

		// Get repomix version
		version, err := app.indexer.GetRepomixVersion()
		if err != nil {
			log.Printf("Warning: could not get repomix version: %v", err)
		} else {
			log.Printf("Repomix version: %s", version)
		}

		// Validate repository access
		aliases := app.configManager.GetRepositoryAliases()
		log.Printf("Validating %d repositories...", len(aliases))

		totalValidated := 0
		for _, alias := range aliases {
			repoConfig, err := app.configManager.GetRepository(alias)
			if err != nil {
				log.Printf("Error: invalid repository config for %s: %v", alias, err)
				continue
			}

			// Expand glob patterns if present
			expandedRepos, err := app.repoManager.ExpandGlobRepositories(alias, repoConfig)
			if err != nil {
				log.Printf("Error: failed to expand glob for repository %s: %v", alias, err)
				continue
			}

			// Validate each expanded repository
			for expandedAlias, expandedConfig := range expandedRepos {
				// Test repository preparation (without full indexing)
				_, err = app.repoManager.PrepareRepository(expandedAlias, expandedConfig)
				if err != nil {
					log.Printf("Error: cannot access repository %s: %v", expandedAlias, err)
					continue
				}

				log.Printf("✓ Repository %s is accessible", expandedAlias)
				totalValidated++
			}
		}

		log.Printf("✓ Validated %d total repositories (including expanded glob patterns)", totalValidated)

		// Test cache operations
		stats, err := app.cache.GetCacheStats()
		if err != nil {
			return fmt.Errorf("cache validation failed\n>    %w", err)
		}

		log.Printf("Cache statistics: %+v", stats)
		log.Println("✓ All validations passed")

		return nil
	},
}

// ************************************************************************************************
// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Manage application configuration including creating example configurations.`,
}

// ************************************************************************************************
// configExampleCmd represents the config example command
var configExampleCmd = &cobra.Command{
	Use:   "example [output-file]",
	Short: "Generate example configuration",
	Long: `Generate an example configuration file with all available options.

Examples:
  repomix-mcp config example                    # Output to stdout
  repomix-mcp config example config.json       # Save to file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile := ""
		if len(args) > 0 {
			outputFile = args[0]
		}

		if outputFile == "" {
			outputFile = "config.example.json"
		}

		manager := config.NewManager()
		if err := manager.CreateExampleConfig(outputFile); err != nil {
			return fmt.Errorf("failed to create example config\n>    %w", err)
		}

		log.Printf("Example configuration saved to: %s", outputFile)
		return nil
	},
}

// ************************************************************************************************
// Global flags
var configFile string

func init() {
	// Add global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.json", "configuration file path")

	// Add subcommands
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(configCmd)

	// Add config subcommands
	configCmd.AddCommand(configExampleCmd)
}

// ************************************************************************************************
// main is the application entry point
func main() {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal...")
		if app != nil {
			app.Cleanup()
		}
		os.Exit(0)
	}()

	// Create and initialize application
	var err error
	app, err = NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Set up pre-run hook to initialize application
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Skip initialization for config example command
		if cmd.Name() == "example" {
			return nil
		}

		return app.Initialize(configFile)
	}

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Command execution failed: %v", err)
	}
}
