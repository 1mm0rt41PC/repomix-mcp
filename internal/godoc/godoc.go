// ************************************************************************************************
// Package godoc provides Go module documentation retrieval functionality for the repomix-mcp application.
// It handles fetching documentation for Go modules using the `go doc` command when modules
// are not found in the local cache, creating synthetic repositories for AI tool consumption.
package godoc

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"repomix-mcp/pkg/types"
)

// ************************************************************************************************
// GoDocRetriever manages Go module documentation extraction using the go doc command.
// It provides functionality to fetch, parse, and cache Go module documentation
// when modules are not available in the local repository cache.
type GoDocRetriever struct {
	config      *types.GoModuleConfig
	tempDirBase string
	cache       CacheInterface
	verbose     bool
}

// ************************************************************************************************
// CacheInterface defines the interface for cache operations used by GoDocRetriever.
type CacheInterface interface {
	GetRepository(id string) (*types.RepositoryIndex, error)
	StoreRepository(repo *types.RepositoryIndex) error
	ListRepositories() ([]string, error)
}

// ************************************************************************************************
// GoModuleInfo represents comprehensive information about a Go module's documentation.
// It contains all extracted documentation, metadata, and package information.
type GoModuleInfo struct {
	ModulePath      string            `json:"modulePath"`      // Full module path (e.g., golang.org/x/sys/windows/registry)
	Documentation   string            `json:"documentation"`   // Output from `go doc`
	AllDocs         string            `json:"allDocs"`         // Output from `go doc -all`
	PackageList     []string          `json:"packageList"`     // List of discovered packages
	Examples        map[string]string `json:"examples"`        // Code examples if available
	CachedAt        time.Time         `json:"cachedAt"`        // When this info was cached
	Version         string            `json:"version"`         // Module version
	GoVersion       string            `json:"goVersion"`       // Go version used for doc generation
	ErrorInfo       string            `json:"errorInfo"`       // Any errors encountered during retrieval
}

// ************************************************************************************************
// NewGoDocRetriever creates a new Go module documentation retriever.
// It initializes the retriever with configuration and cache interface.
//
// Returns:
//   - *GoDocRetriever: The configured retriever instance.
//   - error: An error if initialization fails.
//
// Example usage:
//
//	retriever, err := NewGoDocRetriever(config, cache)
//	if err != nil {
//		return fmt.Errorf("failed to create Go doc retriever: %w", err)
//	}
func NewGoDocRetriever(config *types.GoModuleConfig, cache CacheInterface) (*GoDocRetriever, error) {
	if config == nil {
		return nil, fmt.Errorf("Go module config cannot be nil")
	}

	if cache == nil {
		return nil, fmt.Errorf("cache interface cannot be nil")
	}

	// Ensure temp directory base exists
	tempDirBase := config.TempDirBase
	if tempDirBase == "" {
		tempDirBase = filepath.Join(mock_osTempDir(), "repomix-mcp-godoc")
	}

	// Expand home directory if needed
	if strings.HasPrefix(tempDirBase, "~") {
		homeDir, err := mock_osUserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		tempDirBase = filepath.Join(homeDir, tempDirBase[1:])
	}

	// Create base temp directory
	if err := mock_osMkdirAll(tempDirBase, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory base %s: %w", tempDirBase, err)
	}

	return &GoDocRetriever{
		config:      config,
		tempDirBase: tempDirBase,
		cache:       cache,
		verbose:     false,
	}, nil
}

// ************************************************************************************************
// SetVerbose enables or disables verbose logging for the retriever.
func (g *GoDocRetriever) SetVerbose(verbose bool) {
	g.verbose = verbose
}

// ************************************************************************************************
// IsGoModulePath checks if a given string looks like a valid Go module path.
// It uses pattern matching to identify potential Go module imports.
//
// Returns:
//   - bool: True if the string appears to be a Go module path.
//
// Example usage:
//
//	if IsGoModulePath("golang.org/x/sys/windows") {
//		// Handle as Go module
//	}
func IsGoModulePath(libraryName string) bool {
	if libraryName == "" {
		return false
	}

	// Clean and normalize the input
	libraryName = strings.TrimSpace(libraryName)
	libraryName = strings.ToLower(libraryName)

	// Standard library packages (simple names)
	stdLibPattern := `^[a-z][a-z0-9]*(/[a-z][a-z0-9]*)*$`
	if matched, _ := regexp.MatchString(stdLibPattern, libraryName); matched {
		// Additional check for common standard library patterns
		commonStdLib := []string{"fmt", "os", "io", "net", "http", "time", "context", "sync", "crypto", "encoding"}
		for _, pkg := range commonStdLib {
			if strings.HasPrefix(libraryName, pkg) {
				return true
			}
		}
	}

	// External module patterns (domain.com/path or github.com/user/repo style)
	externalModulePattern := `^([a-z0-9.-]+\.[a-z]{2,}|github\.com|gitlab\.com|bitbucket\.org|golang\.org)/[a-z0-9.\-_/]+$`
	if matched, _ := regexp.MatchString(externalModulePattern, libraryName); matched {
		return true
	}

	// Go experimental packages (golang.org/x/...)
	if strings.HasPrefix(libraryName, "golang.org/x/") {
		return true
	}

	return false
}

// ************************************************************************************************
// RetrieveDocumentation fetches documentation for a Go module using the go doc command.
// It creates a temporary module, fetches the target module, and extracts documentation.
//
// Returns:
//   - *GoModuleInfo: Complete module documentation information.
//   - error: An error if retrieval fails.
//
// Example usage:
//
//	info, err := retriever.RetrieveDocumentation("golang.org/x/sys/windows/registry")
//	if err != nil {
//		return fmt.Errorf("failed to retrieve docs: %w", err)
//	}
func (g *GoDocRetriever) RetrieveDocumentation(modulePath string) (*GoModuleInfo, error) {
	if modulePath == "" {
		return nil, fmt.Errorf("module path cannot be empty")
	}

	// Validate the module path
	if err := g.validateModulePath(modulePath); err != nil {
		return nil, fmt.Errorf("invalid module path: %w", err)
	}

	if g.verbose {
		log.Printf("Starting Go module documentation retrieval for: %s", modulePath)
	}

	// Check if Go command is available
	if err := g.validateGoCommand(); err != nil {
		return nil, fmt.Errorf("Go command validation failed: %w", err)
	}

	// Create and use temporary directory
	var moduleInfo *GoModuleInfo
	err := g.withTempDir(func(tempDir string) error {
		var err error
		moduleInfo, err = g.executeGoCommands(modulePath, tempDir)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to execute Go commands for %s: %w", modulePath, err)
	}

	if g.verbose {
		log.Printf("Successfully retrieved documentation for module: %s", modulePath)
	}

	return moduleInfo, nil
}

// ************************************************************************************************
// GetOrRetrieveDocumentation attempts to get module documentation from cache first,
// then falls back to live retrieval if not found or expired.
//
// Returns:
//   - *GoModuleInfo: Module documentation information.
//   - error: An error if retrieval fails.
//
// Example usage:
//
//	info, err := retriever.GetOrRetrieveDocumentation("github.com/gin-gonic/gin")
//	if err != nil {
//		return fmt.Errorf("failed to get docs: %w", err)
//	}
func (g *GoDocRetriever) GetOrRetrieveDocumentation(modulePath string) (*GoModuleInfo, error) {
	// Generate cache key
	cacheKey := g.getCacheKey(modulePath)

	// Try to get from cache first
	if cached, err := g.cache.GetRepository(cacheKey); err == nil {
		if g.verbose {
			log.Printf("Found cached documentation for module: %s", modulePath)
		}

		// Check if cache is still valid
		if moduleInfo := g.parseRepositoryToModuleInfo(cached); moduleInfo != nil {
			if g.isCacheValid(moduleInfo) {
				return moduleInfo, nil
			}
		}

		if g.verbose {
			log.Printf("Cached documentation for %s is expired, retrieving fresh", modulePath)
		}
	}

	// Cache miss or expired, retrieve fresh documentation
	moduleInfo, err := g.RetrieveDocumentation(modulePath)
	if err != nil {
		return nil, err
	}

	// Cache the results
	if err := g.cacheModuleInfo(modulePath, moduleInfo); err != nil {
		// Log error but don't fail the request
		log.Printf("Warning: failed to cache module documentation for %s: %v", modulePath, err)
	}

	return moduleInfo, nil
}

// ************************************************************************************************
// CreateSyntheticRepository converts Go module information into a RepositoryIndex
// that can be stored in the cache and served through MCP tools.
//
// Returns:
//   - *types.RepositoryIndex: Synthetic repository containing module documentation.
//
// Example usage:
//
//	repo := retriever.CreateSyntheticRepository("example.com/module", moduleInfo)
//	cache.StoreRepository(repo)
func (g *GoDocRetriever) CreateSyntheticRepository(modulePath string, info *GoModuleInfo) *types.RepositoryIndex {
	repoID := g.getCacheKey(modulePath)

	files := make(map[string]types.IndexedFile)

	// Add basic documentation file
	if info.Documentation != "" {
		files["go-doc.md"] = types.IndexedFile{
			Path:         "go-doc.md",
			Content:      g.formatDocumentation("go doc", info.Documentation),
			Hash:         g.calculateContentHash(info.Documentation),
			Size:         int64(len(info.Documentation)),
			ModTime:      info.CachedAt,
			Language:     "markdown",
			RepositoryID: repoID,
			Metadata: map[string]string{
				"source":      "go_doc",
				"type":        "documentation",
				"module_path": modulePath,
			},
		}
	}

	// Add comprehensive documentation file
	if info.AllDocs != "" {
		files["go-doc-all.md"] = types.IndexedFile{
			Path:         "go-doc-all.md",
			Content:      g.formatDocumentation("go doc -all", info.AllDocs),
			Hash:         g.calculateContentHash(info.AllDocs),
			Size:         int64(len(info.AllDocs)),
			ModTime:      info.CachedAt,
			Language:     "markdown",
			RepositoryID: repoID,
			Metadata: map[string]string{
				"source":      "go_doc_all",
				"type":        "comprehensive_documentation",
				"module_path": modulePath,
			},
		}
	}

	// Add package list if available
	if len(info.PackageList) > 0 {
		packageContent := strings.Join(info.PackageList, "\n")
		files["packages.txt"] = types.IndexedFile{
			Path:         "packages.txt",
			Content:      packageContent,
			Hash:         g.calculateContentHash(packageContent),
			Size:         int64(len(packageContent)),
			ModTime:      info.CachedAt,
			Language:     "text",
			RepositoryID: repoID,
			Metadata: map[string]string{
				"source":      "go_list",
				"type":        "package_list",
				"module_path": modulePath,
			},
		}
	}

	// Add examples if available
	for name, example := range info.Examples {
		fileName := fmt.Sprintf("example-%s.go", strings.ReplaceAll(name, "/", "_"))
		files[fileName] = types.IndexedFile{
			Path:         fileName,
			Content:      example,
			Hash:         g.calculateContentHash(example),
			Size:         int64(len(example)),
			ModTime:      info.CachedAt,
			Language:     "go",
			RepositoryID: repoID,
			Metadata: map[string]string{
				"source":      "go_example",
				"type":        "code_example",
				"module_path": modulePath,
				"example_name": name,
			},
		}
	}

	return &types.RepositoryIndex{
		ID:          repoID,
		Name:        fmt.Sprintf("Go Module: %s", modulePath),
		Path:        modulePath,
		LastUpdated: info.CachedAt,
		Files:       files,
		Metadata: map[string]interface{}{
			"source":       "go_module_docs",
			"module_path":  modulePath,
			"doc_type":     "go_documentation",
			"cached_at":    info.CachedAt.Format(time.RFC3339),
			"go_version":   info.GoVersion,
			"version":      info.Version,
			"file_count":   len(files),
			"has_examples": len(info.Examples) > 0,
		},
		CommitHash: "", // Not applicable for Go modules
	}
}

// ************************************************************************************************
// Private helper methods

// getCacheKey generates a consistent cache key for a Go module.
func (g *GoDocRetriever) getCacheKey(modulePath string) string {
	return fmt.Sprintf("gomod:%s", modulePath)
}

// validateModulePath validates that a module path is safe and properly formatted.
func (g *GoDocRetriever) validateModulePath(modulePath string) error {
	// Check for command injection attempts
	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]", "<", ">"}
	for _, char := range dangerousChars {
		if strings.Contains(modulePath, char) {
			return fmt.Errorf("module path contains dangerous characters: %s", char)
		}
	}

	// Check length limits
	if len(modulePath) > 256 {
		return fmt.Errorf("module path too long (max 256 characters)")
	}

	// Additional validation for Go module path format
	if !IsGoModulePath(modulePath) {
		return fmt.Errorf("invalid Go module path format")
	}

	return nil
}

// validateGoCommand checks if the go command is available and working.
func (g *GoDocRetriever) validateGoCommand() error {
	cmd := mock_execCommand("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("go command not available: %w", err)
	}

	if g.verbose {
		log.Printf("Go version: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// withTempDir creates a temporary directory, executes a function, and cleans up.
func (g *GoDocRetriever) withTempDir(fn func(string) error) error {
	tempDir, err := mock_osMkdirTemp(g.tempDirBase, "gomod-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	defer func() {
		if removeErr := mock_osRemoveAll(tempDir); removeErr != nil {
			log.Printf("Warning: failed to cleanup temp directory %s: %v", tempDir, removeErr)
		}
	}()

	if g.verbose {
		log.Printf("Created temp directory: %s", tempDir)
	}

	return fn(tempDir)
}

// calculateContentHash generates a simple hash for content change detection.
func (g *GoDocRetriever) calculateContentHash(content string) string {
	if len(content) == 0 {
		return "empty"
	}

	// Simple hash based on content length and first/last characters
	first := content[0]
	last := content[len(content)-1]

	return fmt.Sprintf("godoc_%d_%c_%c", len(content), first, last)
}

// formatDocumentation formats raw go doc output into markdown.
func (g *GoDocRetriever) formatDocumentation(command, content string) string {
	var formatted strings.Builder

	formatted.WriteString(fmt.Sprintf("# Go Documentation\n\n"))
	formatted.WriteString(fmt.Sprintf("**Generated with:** `%s`\n", command))
	formatted.WriteString(fmt.Sprintf("**Retrieved at:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	formatted.WriteString("---\n\n")
	
	// Add the raw documentation content in a code block
	formatted.WriteString("```\n")
	formatted.WriteString(content)
	formatted.WriteString("\n```\n")

	return formatted.String()
}

// cacheModuleInfo stores module information in the cache.
func (g *GoDocRetriever) cacheModuleInfo(modulePath string, info *GoModuleInfo) error {
	repo := g.CreateSyntheticRepository(modulePath, info)
	return g.cache.StoreRepository(repo)
}

// parseRepositoryToModuleInfo converts a cached repository back to module info.
func (g *GoDocRetriever) parseRepositoryToModuleInfo(repo *types.RepositoryIndex) *GoModuleInfo {
	if repo == nil || !strings.HasPrefix(repo.ID, "gomod:") {
		return nil
	}

	info := &GoModuleInfo{
		ModulePath:  strings.TrimPrefix(repo.ID, "gomod:"),
		CachedAt:    repo.LastUpdated,
		PackageList: []string{},
		Examples:    make(map[string]string),
	}

	// Extract information from metadata
	if modulePath, exists := repo.Metadata["module_path"].(string); exists {
		info.ModulePath = modulePath
	}
	if version, exists := repo.Metadata["version"].(string); exists {
		info.Version = version
	}
	if goVersion, exists := repo.Metadata["go_version"].(string); exists {
		info.GoVersion = goVersion
	}

	// Extract documentation from files
	for _, file := range repo.Files {
		switch file.Path {
		case "go-doc.md":
			info.Documentation = file.Content
		case "go-doc-all.md":
			info.AllDocs = file.Content
		case "packages.txt":
			info.PackageList = strings.Split(file.Content, "\n")
		default:
			if strings.HasPrefix(file.Path, "example-") && strings.HasSuffix(file.Path, ".go") {
				exampleName := strings.TrimSuffix(strings.TrimPrefix(file.Path, "example-"), ".go")
				info.Examples[exampleName] = file.Content
			}
		}
	}

	return info
}

// isCacheValid checks if cached module info is still valid based on TTL.
func (g *GoDocRetriever) isCacheValid(info *GoModuleInfo) bool {
	if g.config.CacheTimeout == "" {
		return true // No expiration configured
	}

	timeout, err := mock_timeParseDuration(g.config.CacheTimeout)
	if err != nil {
		log.Printf("Warning: invalid cache timeout format %s, assuming valid", g.config.CacheTimeout)
		return true
	}

	return time.Since(info.CachedAt) < timeout
}