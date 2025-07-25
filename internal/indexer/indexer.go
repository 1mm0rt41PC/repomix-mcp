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
	"repomix-mcp/internal/parser"
)

// ************************************************************************************************
// IndexingStrategy defines the strategy to use for indexing a repository.
type IndexingStrategy int

const (
	// StrategyRepomix uses the standard repomix CLI tool for indexing.
	StrategyRepomix IndexingStrategy = iota
	
	// StrategyGoNative uses Go AST parsing for Go projects.
	StrategyGoNative
)

// String returns a string representation of the indexing strategy.
func (s IndexingStrategy) String() string {
	switch s {
	case StrategyRepomix:
		return "repomix"
	case StrategyGoNative:
		return "go_native"
	default:
		return "unknown"
	}
}

// ************************************************************************************************
// Indexer manages repository content indexing with multiple strategies.
// It provides functionality to run repomix on repositories or use Go-specific
// parsing for Go projects, then parse the output into structured data.
type Indexer struct {
	repomixPath string
	tempDir     string
	goParser    *parser.GoParser
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
		goParser:    parser.NewGoParser(),
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
// DetermineIndexingStrategy determines the best indexing strategy for a repository.
// It checks for Go projects and returns the appropriate strategy.
//
// Returns:
//   - IndexingStrategy: The recommended indexing strategy.
//
// Example usage:
//
//	strategy := indexer.DetermineIndexingStrategy("/path/to/repo")
func (i *Indexer) DetermineIndexingStrategy(localPath string) IndexingStrategy {
	// Check if this is a Go project by looking for go.mod
	goModPath := filepath.Join(localPath, "go.mod")
	if _, err := mock_osStat(goModPath); err == nil {
		return StrategyGoNative
	}

	// Fallback: check for significant number of Go files
	goFileCount := 0
	filepath.Walk(localPath, func(path string, info mock_osFileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			goFileCount++
		}
		return nil
	})

	// Use Go native strategy if we find 3+ Go files
	if goFileCount >= 3 {
		return StrategyGoNative
	}

	// Default to repomix strategy
	return StrategyRepomix
}

// IndexRepository indexes a repository using the appropriate strategy.
// It automatically detects whether to use repomix or Go-native parsing.
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

	// Determine indexing strategy
	strategy := i.DetermineIndexingStrategy(localPath)

	switch strategy {
	case StrategyGoNative:
		return i.indexRepositoryWithGo(repositoryID, localPath, config)
	case StrategyRepomix:
		return i.indexRepositoryWithRepomix(repositoryID, localPath, config)
	default:
		return nil, fmt.Errorf("unknown indexing strategy: %s", strategy.String())
	}
}

// indexRepositoryWithGo indexes a Go repository using Go AST parsing.
func (i *Indexer) indexRepositoryWithGo(repositoryID, localPath string, config types.IndexingConfig) (*types.RepositoryIndex, error) {
	// Use Go parser for indexing
	repoIndex, err := i.goParser.ParseRepository(repositoryID, localPath, config)
	if err != nil {
		// Fallback to repomix if Go parsing fails
		fmt.Printf("Go parsing failed for %s, falling back to repomix: %v\n", repositoryID, err)
		return i.indexRepositoryWithRepomix(repositoryID, localPath, config)
	}

	// Write .repomix.xml file to repository directory
	xmlFilePath := filepath.Join(localPath, ".repomix.xml")
	if xmlFile, exists := repoIndex.Files[".repomix.xml"]; exists {
		if err := mock_osWriteFile(xmlFilePath, []byte(xmlFile.Content), 0644); err != nil {
			// Log error but don't fail indexing
			fmt.Printf("Warning: failed to write .repomix.xml to %s: %v\n", xmlFilePath, err)
		}
	}

	// Discover and add README files from all subfolders
	readmeFiles, err := i.findReadmeFiles(localPath, repositoryID)
	if err != nil {
		// Log error but don't fail indexing
		fmt.Printf("Warning: failed to discover README files: %v\n", err)
	} else {
		// Add README files to repository index
		for _, readmeFile := range readmeFiles {
			repoIndex.Files[readmeFile.Path] = readmeFile
		}
		
		// Update metadata
		repoIndex.Metadata["readme_count"] = len(readmeFiles)
		fmt.Printf("Added %d README files to repository index\n", len(readmeFiles))
	}

	return repoIndex, nil
}

// indexRepositoryWithRepomix indexes a repository using the repomix CLI tool.
func (i *Indexer) indexRepositoryWithRepomix(repositoryID, localPath string, config types.IndexingConfig) (*types.RepositoryIndex, error) {
	// Create output file path
	outputFile := filepath.Join(i.tempDir, fmt.Sprintf("%s-output.xml", repositoryID))

	// Build repomix command
	args := []string{
		"--output", outputFile,
		"--style", "xml",
		"--remove-comments",
		"--remove-empty-lines",
	}
	
	// Add compression only if we don't want non-exported items
	// Compression tends to filter out non-public elements
	if !config.IncludeNonExported {
		args = append(args, "--compress")
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

	// Discover and add README files from all subfolders
	readmeFiles, err := i.findReadmeFiles(localPath, repositoryID)
	if err != nil {
		// Log error but don't fail indexing
		fmt.Printf("Warning: failed to discover README files: %v\n", err)
	} else {
		// Add README files to repository index
		for _, readmeFile := range readmeFiles {
			repoIndex.Files[readmeFile.Path] = readmeFile
		}
		
		// Update metadata
		repoIndex.Metadata["readme_count"] = len(readmeFiles)
		fmt.Printf("Added %d README files to repository index\n", len(readmeFiles))
	}

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

// ************************************************************************************************
// findReadmeFiles recursively discovers README files in a repository and returns them as IndexedFiles.
// It searches for various README file patterns in all subfolders with configurable depth limits.
//
// Returns:
//   - []types.IndexedFile: List of discovered README files.
//   - error: An error if discovery fails.
//
// Example usage:
//
//	readmeFiles, err := indexer.findReadmeFiles("/path/to/repo", "repo-id")
//	if err != nil {
//		return fmt.Errorf("failed to find README files: %w", err)
//	}
func (i *Indexer) findReadmeFiles(localPath, repositoryID string) ([]types.IndexedFile, error) {
	if localPath == "" || repositoryID == "" {
		return nil, fmt.Errorf("%w: invalid parameters", types.ErrInvalidConfig)
	}

	var readmeFiles []types.IndexedFile
	maxDepth := 10 // Maximum folder depth to search
	maxFileSize := int64(5 * 1024 * 1024) // 5MB maximum file size

	// README file patterns to search for
	readmePatterns := []string{
		"README.md", "readme.md", "Readme.md", "ReadMe.md",
		"README.txt", "readme.txt", "Readme.txt", "ReadMe.txt",
		"README.rst", "readme.rst", "Readme.rst", "ReadMe.rst",
		"README", "readme", "Readme", "ReadMe",
		"README.adoc", "readme.adoc", "Readme.adoc",
		"README.org", "readme.org", "Readme.org",
	}

	// Walk the directory tree
	err := filepath.Walk(localPath, func(path string, info mock_osFileInfo, err error) error {
		if err != nil {
			// Log error but continue processing
			fmt.Printf("Warning: error accessing %s: %v\n", path, err)
			return nil
		}

		// Skip directories
		if info.IsDir() {
			// Check depth limit
			relPath, relErr := filepath.Rel(localPath, path)
			if relErr != nil {
				return nil
			}
			
			depth := strings.Count(relPath, string(filepath.Separator))
			if depth > maxDepth {
				return filepath.SkipDir
			}

			// Skip hidden directories and common ignore patterns
			dirName := info.Name()
			if strings.HasPrefix(dirName, ".") ||
			   dirName == "node_modules" ||
			   dirName == "vendor" ||
			   dirName == "__pycache__" ||
			   dirName == "target" ||
			   dirName == "build" ||
			   dirName == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches README patterns
		fileName := info.Name()
		isReadme := false
		for _, pattern := range readmePatterns {
			if fileName == pattern {
				isReadme = true
				break
			}
		}

		if !isReadme {
			return nil
		}

		// Check file size
		if info.Size() > maxFileSize {
			fmt.Printf("Warning: skipping large README file %s (%d bytes)\n", path, info.Size())
			return nil
		}

		// Calculate relative path from repository root
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			fmt.Printf("Warning: failed to calculate relative path for %s: %v\n", path, err)
			return nil
		}

		// Read file content
		content, err := mock_osReadFile(path)
		if err != nil {
			fmt.Printf("Warning: failed to read README file %s: %v\n", path, err)
			return nil
		}

		// Create indexed file
		indexedFile := types.IndexedFile{
			Path:         relPath,
			Content:      string(content),
			Hash:         i.calculateContentHash(string(content)),
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			Language:     i.detectLanguage(relPath),
			RepositoryID: repositoryID,
			Metadata: map[string]string{
				"file_type":      "readme",
				"subfolder_path": filepath.Dir(relPath),
				"original_name":  fileName,
			},
		}

		// Add folder depth for prioritization
		folderDepth := strings.Count(relPath, string(filepath.Separator))
		indexedFile.Metadata["folder_depth"] = fmt.Sprintf("%d", folderDepth)

		readmeFiles = append(readmeFiles, indexedFile)
		
		fmt.Printf("Discovered README file: %s (size: %d bytes)\n", relPath, info.Size())
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	fmt.Printf("Found %d README files in repository %s\n", len(readmeFiles), repositoryID)
	return readmeFiles, nil
}