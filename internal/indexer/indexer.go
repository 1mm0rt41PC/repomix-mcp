// ************************************************************************************************
// Package indexer provides repomix CLI integration for the repomix-mcp application.
// It handles execution of the repomix command-line tool to index repository content
// and process the output for storage in the cache system.
package indexer

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"repomix-mcp/pkg/types"
)

// ************************************************************************************************
// Indexer manages repomix CLI execution and content processing.
// It provides functionality to run repomix on repositories and parse the output
// into structured data for caching and search operations.
type Indexer struct {
	repomixPath string
	tempDir     string
}

// ************************************************************************************************
// NewIndexer creates a new indexer instance.
// It locates the repomix executable and prepares the indexer for operations.
//
// Returns:
//   - *Indexer: The indexer instance.
//   - error: An error if initialization fails.
//
// Example usage:
//
//	indexer, err := NewIndexer()
//	if err != nil {
//		return fmt.Errorf("failed to create indexer: %w", err)
//	}
func NewIndexer() (*Indexer, error) {
	// Try to find repomix in PATH
	repomixPath, err := mock_execLookPath("repomix")
	if err != nil {
		return nil, fmt.Errorf("%w: repomix not found in PATH\n>    %w", types.ErrRepomixNotFound, err)
	}

	// Create temp directory for repomix output
	tempDir, err := mock_osMkdirTemp("", "repomix-mcp-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory\n>    %w", err)
	}

	return &Indexer{
		repomixPath: repomixPath,
		tempDir:     tempDir,
	}, nil
}

// ************************************************************************************************
// Close cleans up the indexer resources.
// This method should be called when shutting down the indexer.
//
// Returns:
//   - error: An error if cleanup fails.
//
// Example usage:
//
//	defer indexer.Close()
func (i *Indexer) Close() error {
	if i.tempDir != "" {
		if err := mock_osRemoveAll(i.tempDir); err != nil {
			return fmt.Errorf("failed to cleanup temp directory\n>    %w", err)
		}
	}
	return nil
}

// ************************************************************************************************
// IndexRepository runs repomix on a repository and returns the indexed content.
// It executes the repomix CLI tool and processes the output into structured data.
//
// Returns:
//   - *types.RepositoryIndex: The indexed repository content.
//   - error: An error if indexing fails.
//
// Example usage:
//
//	index, err := indexer.IndexRepository("my-repo", "/path/to/repo", &config)
//	if err != nil {
//		return fmt.Errorf("failed to index repository: %w", err)
//	}
func (i *Indexer) IndexRepository(repositoryID, localPath string, config types.IndexingConfig) (*types.RepositoryIndex, error) {
	if repositoryID == "" || localPath == "" {
		return nil, fmt.Errorf("%w: invalid parameters", types.ErrInvalidConfig)
	}

	if !config.Enabled {
		return nil, fmt.Errorf("%w: indexing is disabled", types.ErrIndexingFailed)
	}

	// Create output file path
	outputFile := filepath.Join(i.tempDir, fmt.Sprintf("%s-output.xml", repositoryID))

	// Build repomix command
	args := []string{
		"--output", outputFile,
		"--style", "xml",
		"--compress",
		"--remove-comments",
		"--remove-empty-lines",
	}

	// Add include patterns
	if len(config.IncludePatterns) > 0 {
		args = append(args, "--include", strings.Join(config.IncludePatterns, ","))
	}

	// Add exclude patterns
	if len(config.ExcludePatterns) > 0 {
		excludePatterns := append(config.ExcludePatterns, ".git", ".git/**")
		args = append(args, "--ignore", strings.Join(excludePatterns, ","))
	}

	// Add repository path
	args = append(args, localPath)

	// Execute repomix
	cmd := mock_execCommand(i.repomixPath, args...)
	cmd.Dir = localPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: repomix execution failed: %s\n>    %w", types.ErrRepomixExecFailed, string(output), err)
	}

	// Read repomix output
	content, err := mock_osReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read repomix output\n>    %w", err)
	}

	// Parse repomix output into structured data
	repoIndex, err := i.parseRepomixOutput(repositoryID, localPath, string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse repomix output\n>    %w", err)
	}

	// Clean up output file
	mock_osRemove(outputFile)

	return repoIndex, nil
}

// ************************************************************************************************
// parseRepomixOutput parses the repomix XML output into structured repository data.
// It extracts individual files and their content from the combined output.
//
// Returns:
//   - *types.RepositoryIndex: The parsed repository index.
//   - error: An error if parsing fails.
func (i *Indexer) parseRepomixOutput(repositoryID, localPath, content string) (*types.RepositoryIndex, error) {
	repoIndex := &types.RepositoryIndex{
		ID:          repositoryID,
		Name:        repositoryID,
		Path:        localPath,
		LastUpdated: mock_timeNow(),
		Files:       make(map[string]types.IndexedFile),
		Metadata:    make(map[string]interface{}),
		CommitHash:  "", // Will be filled by repository manager
	}

	// Parse the XML content
	files, err := i.extractFilesFromXML(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract files from XML\n>    %w", err)
	}

	// Process each file
	for _, file := range files {
		// Create indexed file
		indexedFile := types.IndexedFile{
			Path:         file.Path,
			Content:      file.Content,
			Hash:         i.calculateContentHash(file.Content),
			Size:         int64(len(file.Content)),
			ModTime:      mock_timeNow(),
			Language:     i.detectLanguage(file.Path),
			RepositoryID: repositoryID,
			Metadata:     make(map[string]string),
		}

		repoIndex.Files[file.Path] = indexedFile
	}

	// Add repository metadata
	repoIndex.Metadata["file_count"] = len(repoIndex.Files)
	repoIndex.Metadata["indexed_at"] = mock_timeNow().Format(time.RFC3339)
	repoIndex.Metadata["indexer_version"] = "repomix-mcp-v1.0.0"

	return repoIndex, nil
}

// ************************************************************************************************
// FileContent represents a file extracted from repomix output.
type FileContent struct {
	Path    string
	Content string
}

// ************************************************************************************************
// extractFilesFromXML extracts individual files from repomix XML output.
// It parses the structured XML format to identify file boundaries and content.
//
// Returns:
//   - []FileContent: List of extracted files.
//   - error: An error if extraction fails.
func (i *Indexer) extractFilesFromXML(content string) ([]FileContent, error) {
	var files []FileContent
	lines := strings.Split(content, "\n")
	
	var currentFile *FileContent
	var inFileBlock bool
	var fileContentLines []string

	for _, line := range lines {
		// Check for XML file tag pattern: <file path="path/to/file">
		if strings.Contains(line, "<file path=") {
			// Save previous file if exists
			if currentFile != nil {
				currentFile.Content = strings.Join(fileContentLines, "\n")
				files = append(files, *currentFile)
			}

			// Extract file path from XML attribute
			start := strings.Index(line, `path="`)
			if start != -1 {
				start += 6 // Skip 'path="'
				end := strings.Index(line[start:], `"`)
				if end != -1 {
					filePath := line[start : start+end]
					
					currentFile = &FileContent{
						Path:    filePath,
						Content: "",
					}
					fileContentLines = nil
					inFileBlock = true
				}
			}
			continue
		}

		// Check for end of file block
		if strings.Contains(line, "</file>") {
			inFileBlock = false
			continue
		}

		// Collect content within file blocks
		if inFileBlock && currentFile != nil {
			fileContentLines = append(fileContentLines, line)
		}
	}

	// Save last file if exists
	if currentFile != nil {
		currentFile.Content = strings.Join(fileContentLines, "\n")
		files = append(files, *currentFile)
	}

	return files, nil
}

// ************************************************************************************************
// calculateContentHash generates a simple hash for content change detection.
//
// Returns:
//   - string: The content hash.
func (i *Indexer) calculateContentHash(content string) string {
	// Simple hash based on content length and first/last characters
	// In production, you might want to use a proper hash function like SHA256
	if len(content) == 0 {
		return "empty"
	}
	
	first := content[0]
	last := content[len(content)-1]
	
	return fmt.Sprintf("%d_%c_%c", len(content), first, last)
}

// ************************************************************************************************
// detectLanguage attempts to detect the programming language based on file extension.
//
// Returns:
//   - string: The detected language or "text" if unknown.
func (i *Indexer) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	languageMap := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".py":   "python",
		".java": "java",
		".cpp":  "cpp",
		".c":    "c",
		".cs":   "csharp",
		".php":  "php",
		".rb":   "ruby",
		".rs":   "rust",
		".kt":   "kotlin",
		".swift": "swift",
		".scala": "scala",
		".sh":   "bash",
		".ps1":  "powershell",
		".sql":  "sql",
		".html": "html",
		".css":  "css",
		".scss": "scss",
		".sass": "sass",
		".json": "json",
		".xml":  "xml",
		".yaml": "yaml",
		".yml":  "yaml",
		".toml": "toml",
		".ini":  "ini",
		".conf": "config",
		".md":   "markdown",
		".txt":  "text",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "text"
}

// ************************************************************************************************
// GetRepomixVersion returns the version of the repomix CLI tool.
//
// Returns:
//   - string: The repomix version.
//   - error: An error if version retrieval fails.
//
// Example usage:
//
//	version, err := indexer.GetRepomixVersion()
//	if err != nil {
//		return fmt.Errorf("failed to get repomix version: %w", err)
//	}
func (i *Indexer) GetRepomixVersion() (string, error) {
	cmd := mock_execCommand(i.repomixPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repomix version\n>    %w", err)
	}

	version := strings.TrimSpace(string(output))
	return version, nil
}

// ************************************************************************************************
// ValidateRepomix checks if repomix is available and working.
//
// Returns:
//   - error: An error if repomix is not available or working.
//
// Example usage:
//
//	err := indexer.ValidateRepomix()
//	if err != nil {
//		return fmt.Errorf("repomix validation failed: %w", err)
//	}
func (i *Indexer) ValidateRepomix() error {
	cmd := mock_execCommand(i.repomixPath, "--help")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%w: repomix help command failed\n>    %w", types.ErrRepomixNotFound, err)
	}

	return nil
}

// ************************************************************************************************
// IndexSingleFile indexes a single file using repomix (useful for incremental updates).
//
// Returns:
//   - *types.IndexedFile: The indexed file.
//   - error: An error if indexing fails.
//
// Example usage:
//
//	file, err := indexer.IndexSingleFile("/path/to/repo", "src/main.go")
//	if err != nil {
//		return fmt.Errorf("failed to index file: %w", err)
//	}
func (i *Indexer) IndexSingleFile(repositoryPath, filePath string) (*types.IndexedFile, error) {
	if repositoryPath == "" || filePath == "" {
		return nil, fmt.Errorf("%w: invalid parameters", types.ErrInvalidConfig)
	}

	fullPath := filepath.Join(repositoryPath, filePath)
	
	// Check if file exists
	if _, err := mock_osStat(fullPath); mock_osIsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", types.ErrFileNotFound, filePath)
	}

	// Read file content directly
	content, err := mock_osReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content\n>    %w", err)
	}

	// Get file info
	fileInfo, err := mock_osStat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info\n>    %w", err)
	}

	// Create indexed file
	indexedFile := &types.IndexedFile{
		Path:         filePath,
		Content:      string(content),
		Hash:         i.calculateContentHash(string(content)),
		Size:         fileInfo.Size(),
		ModTime:      fileInfo.ModTime(),
		Language:     i.detectLanguage(filePath),
		RepositoryID: "", // Will be set by caller
		Metadata:     make(map[string]string),
	}

	return indexedFile, nil
}