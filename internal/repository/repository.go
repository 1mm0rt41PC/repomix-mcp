// ************************************************************************************************
// Package repository provides repository management functionality for the repomix-mcp application.
// It handles Git repository operations including cloning, updating, and authentication
// for both local and remote repositories.
package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"repomix-mcp/pkg/types"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// ************************************************************************************************
// Manager handles repository operations including cloning, updating, and authentication.
// It provides unified access to both local and remote repositories with proper
// authentication and change detection capabilities.
type Manager struct {
	workDir string
}

// ************************************************************************************************
// NewManager creates a new repository manager instance.
// It initializes the manager with a working directory for repository operations.
//
// Returns:
//   - *Manager: The repository manager instance.
//   - error: An error if initialization fails.
//
// Example usage:
//
//	manager, err := NewManager("/tmp/repomix-repos")
//	if err != nil {
//		return fmt.Errorf("failed to create repository manager: %w", err)
//	}
func NewManager(workDir string) (*Manager, error) {
	if workDir == "" {
		homeDir, err := mock_osUserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory\n>    %w", err)
		}
		workDir = filepath.Join(homeDir, ".repomix-mcp", "repositories")
	}

	// Ensure work directory exists
	if err := mock_osMkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory\n>    %w", err)
	}

	return &Manager{
		workDir: workDir,
	}, nil
}

// ************************************************************************************************
// PrepareRepository prepares a repository for indexing based on its configuration.
// It handles cloning for remote repositories and validates local repositories.
// For local repositories with glob patterns, it returns the first matching path.
//
// Returns:
//   - string: The local path to the prepared repository.
//   - error: An error if preparation fails.
//
// Example usage:
//
//	path, err := manager.PrepareRepository("my-repo", &repoConfig)
//	if err != nil {
//		return fmt.Errorf("failed to prepare repository: %w", err)
//	}
func (m *Manager) PrepareRepository(alias string, config *types.RepositoryConfig) (string, error) {
	if alias == "" || config == nil {
		return "", fmt.Errorf("%w: invalid parameters", types.ErrInvalidConfig)
	}

	switch config.Type {
	case types.RepositoryTypeLocal:
		return m.prepareLocalRepository(config)
	case types.RepositoryTypeRemote:
		return m.prepareRemoteRepository(alias, config)
	default:
		return "", fmt.Errorf("%w: %s", types.ErrInvalidRepositoryType, config.Type)
	}
}

// ************************************************************************************************
// ExpandGlobRepositories expands a repository configuration with glob patterns into multiple repositories.
// This allows a single config entry like "c:\xxx\*" to discover and create multiple repository configurations.
//
// Returns:
//   - map[string]*types.RepositoryConfig: Map of discovered repositories with generated aliases.
//   - error: An error if glob expansion fails.
//
// Example usage:
//
//	expanded, err := manager.ExpandGlobRepositories("base-alias", &repoConfig)
//	if err != nil {
//		return fmt.Errorf("failed to expand glob: %w", err)
//	}
func (m *Manager) ExpandGlobRepositories(baseAlias string, config *types.RepositoryConfig) (map[string]*types.RepositoryConfig, error) {
	if config.Type != types.RepositoryTypeLocal {
		// Only expand local repositories
		return map[string]*types.RepositoryConfig{baseAlias: config}, nil
	}

	// Check if path contains glob patterns
	if !strings.ContainsAny(config.Path, "*?[]{}") {
		// No glob patterns, return as-is
		return map[string]*types.RepositoryConfig{baseAlias: config}, nil
	}

	// Expand home directory if needed
	path := config.Path
	if strings.HasPrefix(path, "~") {
		homeDir, err := mock_osUserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory\n>    %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Find all matching paths
	matches, err := doublestar.FilepathGlob(path)
	if err != nil {
		return nil, fmt.Errorf("failed to expand glob pattern %s\n>    %w", path, err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no directories found matching pattern: %s", path)
	}

	// Create repository configurations for each match
	expanded := make(map[string]*types.RepositoryConfig)
	for i, matchPath := range matches {
		// Check if it's a directory
		if info, err := mock_osStat(matchPath); err != nil || !info.IsDir() {
			continue // Skip files, only process directories
		}

		// Generate alias for this match
		dirName := filepath.Base(matchPath)
		alias := fmt.Sprintf("%s-%s", baseAlias, dirName)
		
		// If there's only one match, use the original alias
		if len(matches) == 1 {
			alias = baseAlias
		} else if i == 0 && dirName == baseAlias {
			// If the directory name matches the base alias, use it directly
			alias = baseAlias
		}

		// Create new config for this path
		newConfig := &types.RepositoryConfig{
			Type:     types.RepositoryTypeLocal,
			Path:     matchPath,
			Auth:     config.Auth,
			Indexing: config.Indexing,
			Branch:   config.Branch,
		}

		expanded[alias] = newConfig
	}

	if len(expanded) == 0 {
		return nil, fmt.Errorf("no valid directories found matching pattern: %s", path)
	}

	return expanded, nil
}

// ************************************************************************************************
// prepareLocalRepository validates and prepares a local repository.
//
// Returns:
//   - string: The local path to the repository.
//   - error: An error if validation fails.
func (m *Manager) prepareLocalRepository(config *types.RepositoryConfig) (string, error) {
	// Expand home directory if needed
	path := config.Path
	if strings.HasPrefix(path, "~") {
		homeDir, err := mock_osUserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory\n>    %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Check if path exists
	if _, err := mock_osStat(path); mock_osIsNotExist(err) {
		return "", fmt.Errorf("%w: %s", types.ErrInvalidPath, path)
	}

	// Check if it's a directory
	if info, err := mock_osStat(path); err != nil {
		return "", fmt.Errorf("failed to stat path %s\n>    %w", path, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("%w: path is not a directory: %s", types.ErrInvalidPath, path)
	}

	// For local repositories, we don't require them to be git repositories
	// This allows indexing of directories that contain multiple codebases
	// The git validation is only done for remote repositories that need cloning
	
	return path, nil
}

// ************************************************************************************************
// prepareRemoteRepository clones or updates a remote repository.
//
// Returns:
//   - string: The local path to the cloned repository.
//   - error: An error if cloning/updating fails.
func (m *Manager) prepareRemoteRepository(alias string, config *types.RepositoryConfig) (string, error) {
	localPath := filepath.Join(m.workDir, alias)

	// Check if repository already exists
	if _, err := mock_osStat(localPath); err == nil {
		// Repository exists, try to update it
		return m.updateRepository(localPath, config)
	}

	// Repository doesn't exist, clone it
	return m.cloneRepository(localPath, config)
}

// ************************************************************************************************
// cloneRepository clones a remote repository to the specified local path.
//
// Returns:
//   - string: The local path to the cloned repository.
//   - error: An error if cloning fails.
func (m *Manager) cloneRepository(localPath string, config *types.RepositoryConfig) (string, error) {
	// Create authentication
	auth, err := m.createAuth(config.Auth)
	if err != nil {
		return "", fmt.Errorf("failed to create authentication\n>    %w", err)
	}

	// Clone options
	cloneOptions := &git.CloneOptions{
		URL:           config.URL,
		Auth:          auth,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", config.Branch)),
		Progress:      nil, // We can add progress reporting later
	}

	// Clone repository
	_, err = mock_gitPlainClone(localPath, false, cloneOptions)
	if err != nil {
		return "", fmt.Errorf("%w: failed to clone repository\n>    %w", types.ErrGitCloneFailed, err)
	}

	return localPath, nil
}

// ************************************************************************************************
// updateRepository updates an existing local repository.
//
// Returns:
//   - string: The local path to the updated repository.
//   - error: An error if updating fails.
func (m *Manager) updateRepository(localPath string, config *types.RepositoryConfig) (string, error) {
	// Open repository
	repo, err := mock_gitPlainOpen(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository\n>    %w", err)
	}

	// Get worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree\n>    %w", err)
	}

	// Create authentication
	auth, err := m.createAuth(config.Auth)
	if err != nil {
		return "", fmt.Errorf("failed to create authentication\n>    %w", err)
	}

	// Pull latest changes
	pullOptions := &git.PullOptions{
		Auth:     auth,
		Progress: nil,
	}

	err = worktree.Pull(pullOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return "", fmt.Errorf("%w: failed to pull repository\n>    %w", types.ErrGitPullFailed, err)
	}

	return localPath, nil
}

// ************************************************************************************************
// createAuth creates authentication configuration for Git operations.
//
// Returns:
//   - transport.AuthMethod: The authentication method.
//   - error: An error if authentication creation fails.
func (m *Manager) createAuth(authConfig types.RepositoryAuth) (transport.AuthMethod, error) {
	switch authConfig.Type {
	case types.AuthTypeNone:
		return nil, nil

	case types.AuthTypeSSH:
		if authConfig.KeyPath == "" {
			return nil, fmt.Errorf("%w: SSH key path is required", types.ErrAuthenticationFailed)
		}

		// Expand home directory if needed
		keyPath := authConfig.KeyPath
		if strings.HasPrefix(keyPath, "~") {
			homeDir, err := mock_osUserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory\n>    %w", err)
			}
			keyPath = filepath.Join(homeDir, keyPath[1:])
		}

		// Create SSH authentication
		auth, err := ssh.NewPublicKeysFromFile("git", keyPath, "")
		if err != nil {
			return nil, fmt.Errorf("%w: failed to create SSH auth\n>    %w", types.ErrAuthenticationFailed, err)
		}

		return auth, nil

	case types.AuthTypeToken:
		if authConfig.Token == "" {
			return nil, fmt.Errorf("%w: token is required", types.ErrAuthenticationFailed)
		}

		username := authConfig.Username
		if username == "" {
			username = "token" // Default username for token auth
		}

		auth := &http.BasicAuth{
			Username: username,
			Password: authConfig.Token,
		}

		return auth, nil

	default:
		return nil, fmt.Errorf("%w: unknown auth type: %s", types.ErrAuthenticationFailed, authConfig.Type)
	}
}

// ************************************************************************************************
// GetRepositoryInfo retrieves information about a prepared repository.
// It returns metadata including commit hash and last update time.
// For non-git directories, it returns basic filesystem metadata.
//
// Returns:
//   - *types.RepositoryIndex: Repository metadata.
//   - error: An error if information retrieval fails.
//
// Example usage:
//
//	info, err := manager.GetRepositoryInfo("my-repo", "/path/to/repo")
//	if err != nil {
//		return fmt.Errorf("failed to get repository info: %w", err)
//	}
func (m *Manager) GetRepositoryInfo(repositoryID, localPath string) (*types.RepositoryIndex, error) {
	if repositoryID == "" || localPath == "" {
		return nil, fmt.Errorf("%w: invalid parameters", types.ErrInvalidConfig)
	}

	// Create basic repository index metadata
	repoIndex := &types.RepositoryIndex{
		ID:          repositoryID,
		Name:        repositoryID,
		Path:        localPath,
		LastUpdated: mock_timeNow(),
		Files:       make(map[string]types.IndexedFile),
		Metadata:    make(map[string]interface{}),
		CommitHash:  "",
	}

	// Try to open as git repository
	repo, err := mock_gitPlainOpen(localPath)
	if err != nil {
		// Not a git repository, use filesystem metadata
		if info, statErr := mock_osStat(localPath); statErr == nil {
			repoIndex.Metadata["type"] = "directory"
			repoIndex.Metadata["last_modified"] = info.ModTime()
			repoIndex.Metadata["is_git_repo"] = false
		}
		return repoIndex, nil
	}

	// It's a git repository, get git metadata
	head, err := repo.Head()
	if err != nil {
		// Git repo but no HEAD (empty repo)
		repoIndex.Metadata["type"] = "git_repository"
		repoIndex.Metadata["is_git_repo"] = true
		repoIndex.Metadata["status"] = "empty_repository"
		return repoIndex, nil
	}

	// Get commit object
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		// Git repo with HEAD but can't get commit
		repoIndex.Metadata["type"] = "git_repository"
		repoIndex.Metadata["is_git_repo"] = true
		repoIndex.Metadata["commit_hash"] = head.Hash().String()
		repoIndex.CommitHash = head.Hash().String()
		return repoIndex, nil
	}

	// Full git repository with commit info
	repoIndex.Metadata["type"] = "git_repository"
	repoIndex.Metadata["is_git_repo"] = true
	repoIndex.Metadata["commit_hash"] = head.Hash().String()
	repoIndex.Metadata["commit_message"] = commit.Message
	repoIndex.Metadata["commit_author"] = commit.Author.Name
	repoIndex.Metadata["commit_date"] = commit.Author.When
	repoIndex.CommitHash = head.Hash().String()

	return repoIndex, nil
}

// ************************************************************************************************
// ListFiles returns all files in the repository that match the indexing configuration.
// It respects include/exclude patterns and file size limits.
//
// Returns:
//   - []string: List of file paths relative to repository root.
//   - error: An error if file listing fails.
//
// Example usage:
//
//	files, err := manager.ListFiles("/path/to/repo", &indexingConfig)
//	if err != nil {
//		return fmt.Errorf("failed to list files: %w", err)
//	}
func (m *Manager) ListFiles(localPath string, indexingConfig types.IndexingConfig) ([]string, error) {
	if localPath == "" {
		return nil, fmt.Errorf("%w: local path is empty", types.ErrInvalidPath)
	}

	var files []string

	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		// Skip .git directory
		if strings.Contains(relPath, ".git") {
			return nil
		}

		// Check if file matches indexing criteria
		if m.shouldIndexFile(relPath, info, indexingConfig) {
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk repository files\n>    %w", err)
	}

	return files, nil
}

// ************************************************************************************************
// shouldIndexFile determines if a file should be indexed based on configuration.
//
// Returns:
//   - bool: True if the file should be indexed.
func (m *Manager) shouldIndexFile(relPath string, info os.FileInfo, config types.IndexingConfig) bool {
	// Check if indexing is enabled
	if !config.Enabled {
		return false
	}

	// Check file size limit if specified
	if config.MaxFileSize != "" {
		// Simple size check - can be enhanced with proper parsing
		if info.Size() > 1024*1024 { // Default 1MB limit
			return false
		}
	}

	// Check exclude patterns
	for _, pattern := range config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
			return false
		}
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return false
		}
	}

	// Check include patterns
	if len(config.IncludePatterns) > 0 {
		for _, pattern := range config.IncludePatterns {
			if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
				return true
			}
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return true
			}
		}
		return false // No include pattern matched
	}

	return true // No include patterns specified, file passes exclude checks
}

// ************************************************************************************************
// GetFileContent reads the content of a file in the repository.
//
// Returns:
//   - string: The file content.
//   - error: An error if reading fails.
//
// Example usage:
//
//	content, err := manager.GetFileContent("/path/to/repo", "src/main.go")
//	if err != nil {
//		return fmt.Errorf("failed to read file: %w", err)
//	}
func (m *Manager) GetFileContent(localPath, relPath string) (string, error) {
	if localPath == "" || relPath == "" {
		return "", fmt.Errorf("%w: invalid parameters", types.ErrInvalidPath)
	}

	fullPath := filepath.Join(localPath, relPath)
	content, err := mock_osReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file content\n>    %w", err)
	}

	return string(content), nil
}

// ************************************************************************************************
// CleanupRepository removes a local repository directory.
// This is useful for cleaning up cloned repositories that are no longer needed.
//
// Returns:
//   - error: An error if cleanup fails.
//
// Example usage:
//
//	err := manager.CleanupRepository("my-repo")
//	if err != nil {
//		return fmt.Errorf("failed to cleanup repository: %w", err)
//	}
func (m *Manager) CleanupRepository(alias string) error {
	if alias == "" {
		return fmt.Errorf("%w: alias is empty", types.ErrInvalidConfig)
	}

	localPath := filepath.Join(m.workDir, alias)
	
	if err := mock_osRemoveAll(localPath); err != nil {
		return fmt.Errorf("failed to remove repository directory\n>    %w", err)
	}

	return nil
}