// ************************************************************************************************
// Package search provides content search functionality for the repomix-mcp application.
// It handles searching through indexed repository content with support for text matching,
// filtering, and result ranking for efficient content discovery.
package search

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"repomix-mcp/pkg/types"
)

// ************************************************************************************************
// Engine provides search functionality for indexed repository content.
// It supports text-based searching with filtering and ranking capabilities
// to help users find relevant content across repositories.
type Engine struct {
	// Future: can add more sophisticated indexing like inverted indexes
}

// ************************************************************************************************
// NewEngine creates a new search engine instance.
//
// Returns:
//   - *Engine: The search engine instance.
//
// Example usage:
//
//	engine := NewEngine()
//	results, err := engine.Search(query, repositories)
func NewEngine() *Engine {
	return &Engine{}
}

// ************************************************************************************************
// Search performs a search across the provided repositories.
// It supports text matching, filtering, and result ranking.
//
// Returns:
//   - []types.SearchResult: Ranked search results.
//   - error: An error if search fails.
//
// Example usage:
//
//	results, err := engine.Search(query, repositories)
//	if err != nil {
//		return fmt.Errorf("search failed: %w", err)
//	}
func (e *Engine) Search(query types.SearchQuery, repositories map[string]*types.RepositoryIndex) ([]types.SearchResult, error) {
	if query.Query == "" {
		return nil, fmt.Errorf("%w: search query is empty", types.ErrInvalidSearchQuery)
	}

	var allResults []types.SearchResult

	// Search through all repositories or specific repository
	for repoID, repo := range repositories {
		// Skip if specific repository requested and this isn't it
		if query.RepositoryID != "" && query.RepositoryID != repoID {
			continue
		}

		// Search through files in this repository
		repoResults, err := e.searchRepository(query, repo)
		if err != nil {
			continue // Skip this repository on error, don't fail entire search
		}

		allResults = append(allResults, repoResults...)
	}

	// Sort results by score (highest first)
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// Apply result limit
	if query.MaxResults > 0 && len(allResults) > query.MaxResults {
		allResults = allResults[:query.MaxResults]
	}

	return allResults, nil
}

// ************************************************************************************************
// searchRepository searches within a single repository.
//
// Returns:
//   - []types.SearchResult: Search results from this repository.
//   - error: An error if repository search fails.
func (e *Engine) searchRepository(query types.SearchQuery, repo *types.RepositoryIndex) ([]types.SearchResult, error) {
	var results []types.SearchResult

	for _, file := range repo.Files {
		// Apply file pattern filter
		if query.FilePattern != "" {
			if matched, _ := mock_filepathMatch(query.FilePattern, file.Path); !matched {
				continue
			}
		}

		// Apply language filter
		if query.Language != "" && file.Language != query.Language {
			continue
		}

		// Search within file content
		fileResults := e.searchFile(query, file)
		results = append(results, fileResults...)
	}

	return results, nil
}

// ************************************************************************************************
// searchFile searches within a single file.
//
// Returns:
//   - []types.SearchResult: Search results from this file.
func (e *Engine) searchFile(query types.SearchQuery, file types.IndexedFile) []types.SearchResult {
	// Split content into lines for line-by-line search
	lines := strings.Split(file.Content, "\n")
	
	// Prepare search pattern
	searchPattern := strings.ToLower(query.Query)
	isRegex := false
	var regexPattern *regexp.Regexp
	
	// Check if query looks like a regex (starts and ends with /)
	if strings.HasPrefix(query.Query, "/") && strings.HasSuffix(query.Query, "/") && len(query.Query) > 2 {
		pattern := query.Query[1 : len(query.Query)-1]
		if compiled, err := regexp.Compile(pattern); err == nil {
			regexPattern = compiled
			isRegex = true
		}
	}

	matchCount := 0
	var bestMatch types.SearchResult

	// Search through each line
	for lineNum, line := range lines {
		var matched bool
		var highlightedLine string

		if isRegex && regexPattern != nil {
			// Regex search
			if regexPattern.MatchString(line) {
				matched = true
				highlightedLine = regexPattern.ReplaceAllStringFunc(line, func(match string) string {
					return fmt.Sprintf("**%s**", match)
				})
			}
		} else {
			// Simple text search (case-insensitive)
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, searchPattern) {
				matched = true
				// Highlight matches
				highlightedLine = e.highlightMatches(line, query.Query)
			}
		}

		if matched {
			matchCount++
			
			// Calculate score for this match
			score := e.calculateScore(query, file, line, lineNum)
			
			// Create search result
			result := types.SearchResult{
				File:        file,
				Score:       score,
				Snippet:     e.createSnippet(lines, lineNum, 2), // 2 lines context
				LineNumber:  lineNum + 1, // Convert to 1-based
				MatchCount:  1,
				Highlighted: highlightedLine,
			}

			// Keep track of best match for this file
			if score > bestMatch.Score {
				bestMatch = result
			}
		}
	}

	// Return best match with total match count
	if matchCount > 0 {
		bestMatch.MatchCount = matchCount
		return []types.SearchResult{bestMatch}
	}

	return nil
}

// ************************************************************************************************
// calculateScore calculates a relevance score for a search match.
//
// Returns:
//   - float64: The relevance score (0.0 to 1.0).
func (e *Engine) calculateScore(query types.SearchQuery, file types.IndexedFile, line string, lineNum int) float64 {
	score := 0.0

	// Base score for any match
	score += 0.1

	// Boost for exact matches
	if strings.Contains(strings.ToLower(line), strings.ToLower(query.Query)) {
		score += 0.3
	}

	// Boost for matches in file name
	if strings.Contains(strings.ToLower(file.Path), strings.ToLower(query.Query)) {
		score += 0.2
	}

	// Boost for matches near the beginning of the file
	if lineNum < 50 {
		score += 0.1 * (50.0 - float64(lineNum)) / 50.0
	}

	// Boost for shorter files (more focused content)
	if file.Size < 10000 { // Less than 10KB
		score += 0.1
	}

	// Boost based on file type relevance
	if query.Language != "" && file.Language == query.Language {
		score += 0.2
	}

	// Boost for certain file types that are typically more important
	importantExtensions := []string{".md", ".go", ".js", ".py", ".java", ".cpp", ".c"}
	for _, ext := range importantExtensions {
		if strings.HasSuffix(strings.ToLower(file.Path), ext) {
			score += 0.1
			break
		}
	}

	// Normalize score to 0.0-1.0 range
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// ************************************************************************************************
// highlightMatches highlights search matches in a line of text.
//
// Returns:
//   - string: The line with highlighted matches.
func (e *Engine) highlightMatches(line, query string) string {
	// Simple case-insensitive highlighting
	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)
	
	if !strings.Contains(lowerLine, lowerQuery) {
		return line
	}

	// Find all matches and replace them with highlighted versions
	result := line
	searchLen := len(query)
	
	for {
		index := strings.Index(strings.ToLower(result), lowerQuery)
		if index == -1 {
			break
		}
		
		// Extract the actual match (preserving original case)
		match := result[index : index+searchLen]
		highlighted := fmt.Sprintf("**%s**", match)
		
		// Replace this occurrence
		result = result[:index] + highlighted + result[index+searchLen:]
		
		// Move past the highlighted portion to find next occurrence
		offset := index + len(highlighted)
		if offset >= len(result) {
			break
		}
		
		// Continue searching from after this match
		remaining := result[offset:]
		nextIndex := strings.Index(strings.ToLower(remaining), lowerQuery)
		if nextIndex == -1 {
			break
		}
		
		// Adjust the result to continue search
		result = result[:offset] + remaining
	}

	return result
}

// ************************************************************************************************
// createSnippet creates a context snippet around a matched line.
//
// Returns:
//   - string: The snippet with context lines.
func (e *Engine) createSnippet(lines []string, matchLine, contextLines int) string {
	start := matchLine - contextLines
	end := matchLine + contextLines + 1

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	snippet := strings.Join(lines[start:end], "\n")
	
	// Limit snippet length
	maxSnippetLength := 500
	if len(snippet) > maxSnippetLength {
		snippet = snippet[:maxSnippetLength] + "..."
	}

	return snippet
}

// ************************************************************************************************
// SearchByTopic performs a topic-focused search across repositories.
// This is useful for finding content related to specific topics or concepts.
//
// Returns:
//   - []types.SearchResult: Topic-focused search results.
//   - error: An error if search fails.
//
// Example usage:
//
//	results, err := engine.SearchByTopic("authentication", repositories)
//	if err != nil {
//		return fmt.Errorf("topic search failed: %w", err)
//	}
func (e *Engine) SearchByTopic(topic string, repositories map[string]*types.RepositoryIndex) ([]types.SearchResult, error) {
	if topic == "" {
		return nil, fmt.Errorf("%w: topic is empty", types.ErrInvalidSearchQuery)
	}

	// Create a query focused on the topic
	query := types.SearchQuery{
		Query:      topic,
		MaxResults: 50, // Default limit for topic searches
	}

	// Perform the search
	results, err := e.Search(query, repositories)
	if err != nil {
		return nil, fmt.Errorf("failed to search by topic\n>    %w", err)
	}

	// Additional filtering and boosting for topic-specific results
	var topicResults []types.SearchResult
	for _, result := range results {
		// Boost results that have topic in filename or path
		if strings.Contains(strings.ToLower(result.File.Path), strings.ToLower(topic)) {
			result.Score += 0.3
		}

		// Boost results in documentation files
		if strings.Contains(strings.ToLower(result.File.Path), "doc") ||
			strings.Contains(strings.ToLower(result.File.Path), "readme") ||
			strings.HasSuffix(strings.ToLower(result.File.Path), ".md") {
			result.Score += 0.2
		}

		topicResults = append(topicResults, result)
	}

	// Re-sort by updated scores
	sort.Slice(topicResults, func(i, j int) bool {
		return topicResults[i].Score > topicResults[j].Score
	})

	return topicResults, nil
}

// ************************************************************************************************
// GetSuggestions provides search suggestions based on indexed content.
// This can be used to help users discover content or refine their searches.
//
// Returns:
//   - []string: List of search suggestions.
//   - error: An error if suggestion generation fails.
//
// Example usage:
//
//	suggestions, err := engine.GetSuggestions("auth", repositories)
//	if err != nil {
//		return fmt.Errorf("failed to get suggestions: %w", err)
//	}
func (e *Engine) GetSuggestions(prefix string, repositories map[string]*types.RepositoryIndex) ([]string, error) {
	if len(prefix) < 2 {
		return nil, fmt.Errorf("%w: prefix too short", types.ErrInvalidSearchQuery)
	}

	suggestions := make(map[string]int) // suggestion -> frequency
	lowerPrefix := strings.ToLower(prefix)

	// Extract words from file content that start with the prefix
	for _, repo := range repositories {
		for _, file := range repo.Files {
			// Split content into words
			words := strings.Fields(file.Content)
			for _, word := range words {
				// Clean word (remove punctuation)
				cleanWord := strings.ToLower(regexp.MustCompile(`[^\w]`).ReplaceAllString(word, ""))
				
				if len(cleanWord) > len(prefix) && strings.HasPrefix(cleanWord, lowerPrefix) {
					suggestions[cleanWord]++
				}
			}

			// Also check file paths
			pathParts := strings.Split(file.Path, "/")
			for _, part := range pathParts {
				cleanPart := strings.ToLower(part)
				if len(cleanPart) > len(prefix) && strings.HasPrefix(cleanPart, lowerPrefix) {
					suggestions[cleanPart]++
				}
			}
		}
	}

	// Convert to sorted list
	type suggestion struct {
		word  string
		count int
	}

	var sortedSuggestions []suggestion
	for word, count := range suggestions {
		sortedSuggestions = append(sortedSuggestions, suggestion{word, count})
	}

	// Sort by frequency (descending)
	sort.Slice(sortedSuggestions, func(i, j int) bool {
		return sortedSuggestions[i].count > sortedSuggestions[j].count
	})

	// Return top 10 suggestions
	result := make([]string, 0, 10)
	for i, s := range sortedSuggestions {
		if i >= 10 {
			break
		}
		result = append(result, s.word)
	}

	return result, nil
}