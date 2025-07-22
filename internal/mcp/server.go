// ************************************************************************************************
// Package mcp provides Model Context Protocol (MCP) server implementation for the repomix-mcp application.
// It implements a JSON-RPC 2.0 compliant MCP server that exposes repository indexing capabilities
// as MCP tools, following the official MCP specification.
package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"repomix-mcp/pkg/types"
)

// ************************************************************************************************
// Server implements the MCP server functionality.
// It provides JSON-RPC 2.0 compliant endpoints for MCP protocol communication.
type Server struct {
	config       *types.Config
	cache        CacheInterface
	searchEngine SearchInterface
	repositories map[string]*types.RepositoryIndex
}

// ************************************************************************************************
// CacheInterface defines the interface for cache operations.
type CacheInterface interface {
	GetRepository(id string) (*types.RepositoryIndex, error)
	StoreRepository(repo *types.RepositoryIndex) error
	ListRepositories() ([]string, error)
}

// ************************************************************************************************
// SearchInterface defines the interface for search operations.
type SearchInterface interface {
	Search(query types.SearchQuery) ([]types.SearchResult, error)
}

// ************************************************************************************************
// NewServer creates a new MCP server instance.
//
// Returns:
//   - *Server: The MCP server instance.
//   - error: An error if initialization fails.
//
// Example usage:
//
//	server, err := NewServer(config, cache, searchEngine)
//	if err != nil {
//		return fmt.Errorf("failed to create server: %w", err)
//	}
func NewServer(config *types.Config, cache CacheInterface, searchEngine SearchInterface) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	return &Server{
		config:       config,
		cache:        cache,
		searchEngine: searchEngine,
		repositories: make(map[string]*types.RepositoryIndex),
	}, nil
}

// ************************************************************************************************
// Start starts the MCP server on the configured port.
// It sets up HTTP handlers for the MCP JSON-RPC 2.0 endpoint.
//
// Returns:
//   - error: An error if server startup fails.
//
// Example usage:
//
//	err := server.Start()
//	if err != nil {
//		return fmt.Errorf("failed to start server: %w", err)
//	}
func (s *Server) Start() error {
	// Set up the main MCP endpoint for JSON-RPC 2.0
	http.HandleFunc("/mcp", s.handleMCPEndpoint)
	
	// Add health check endpoint
	http.HandleFunc("/health", s.handleHealth)

	// Start server
	address := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	log.Printf("Starting MCP server on %s", address)
	log.Printf("MCP endpoint available at: http://%s/mcp", address)

	return http.ListenAndServe(address, nil)
}

// ************************************************************************************************
// handleMCPEndpoint handles the main MCP endpoint for JSON-RPC 2.0 protocol.
func (s *Server) handleMCPEndpoint(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, MCP-Protocol-Version")

	// Handle preflight requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST requests for JSON-RPC
	if r.Method != http.MethodPost {
		s.sendJSONRPCError(w, nil, -32600, "Invalid Request", "Only POST method is allowed")
		return
	}

	// Parse JSON-RPC request
	var jsonRPCReq types.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonRPCReq); err != nil {
		s.sendJSONRPCError(w, nil, -32700, "Parse error", fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	// Validate JSON-RPC version
	if jsonRPCReq.JsonRPC != "2.0" {
		s.sendJSONRPCError(w, jsonRPCReq.ID, -32600, "Invalid Request", "JSON-RPC version must be 2.0")
		return
	}

	// Add verbose logging
	log.Printf("Received JSON-RPC request: method=%s, id=%v", jsonRPCReq.Method, jsonRPCReq.ID)

	// Route to appropriate handler
	switch jsonRPCReq.Method {
	case "initialize":
		s.handleInitialize(w, jsonRPCReq)
	case "initialized":
		s.handleInitialized(w, jsonRPCReq)
	case "notifications/initialized":
		s.handleInitialized(w, jsonRPCReq)
	case "tools/list":
		s.handleToolsList(w, jsonRPCReq)
	case "tools/call":
		s.handleToolsCall(w, jsonRPCReq)
	case "ping":
		s.handlePing(w, jsonRPCReq)
	default:
		s.sendJSONRPCError(w, jsonRPCReq.ID, -32601, "Method not found", fmt.Sprintf("Unknown method: %s", jsonRPCReq.Method))
	}
}

// ************************************************************************************************
// handleInitialize handles the MCP initialize request.
func (s *Server) handleInitialize(w http.ResponseWriter, req types.JSONRPCRequest) {
	log.Printf("Handling initialize request")
	
	result := types.MCPInitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		ServerInfo: map[string]interface{}{
			"name":    "repomix-mcp",
			"version": "1.0.0",
		},
	}

	s.sendJSONRPCResult(w, req.ID, result)
}

// ************************************************************************************************
// handleInitialized handles the MCP initialized notification.
func (s *Server) handleInitialized(w http.ResponseWriter, req types.JSONRPCRequest) {
	log.Printf("Handling initialized notification")
	
	// For notifications (no ID), we don't send a JSON-RPC response
	// Just return HTTP 202 Accepted
	w.WriteHeader(http.StatusAccepted)
}

// ************************************************************************************************
// handleToolsList handles the tools/list request.
func (s *Server) handleToolsList(w http.ResponseWriter, req types.JSONRPCRequest) {
	log.Printf("Handling tools/list request")
	
	tools := []types.MCPTool{
		{
			Name:        "resolve-library-id",
			Description: "Resolves a general library name into a repository ID",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"libraryName": map[string]interface{}{
						"type":        "string",
						"description": "The name of the library to search for",
					},
				},
				"required": []string{"libraryName"},
			},
		},
		{
			Name:        "get-library-docs",
			Description: "Fetches documentation for a repository using its ID",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"context7CompatibleLibraryID": map[string]interface{}{
						"type":        "string",
						"description": "Repository ID from resolve-library-id",
					},
					"topic": map[string]interface{}{
						"type":        "string",
						"description": "Focus the docs on a specific topic",
					},
					"tokens": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of tokens to return",
						"default":     10000,
					},
				},
				"required": []string{"context7CompatibleLibraryID"},
			},
		},
	}

	result := types.MCPToolsListResult{
		Tools: tools,
	}

	s.sendJSONRPCResult(w, req.ID, result)
}

// ************************************************************************************************
// handleToolsCall handles the tools/call request.
func (s *Server) handleToolsCall(w http.ResponseWriter, req types.JSONRPCRequest) {
	log.Printf("Handling tools/call request")
	
	// Parse parameters
	var params types.MCPToolCallParams
	if err := s.parseParams(req.Params, &params); err != nil {
		s.sendJSONRPCError(w, req.ID, -32602, "Invalid params", fmt.Sprintf("Failed to parse parameters: %v", err))
		return
	}

	log.Printf("Tool call: name=%s, arguments=%+v", params.Name, params.Arguments)

	// Route to specific tool handler
	switch params.Name {
	case "resolve-library-id":
		s.handleResolveLibraryID(w, req.ID, params.Arguments)
	case "get-library-docs":
		s.handleGetLibraryDocs(w, req.ID, params.Arguments)
	default:
		s.sendJSONRPCError(w, req.ID, -32602, "Invalid params", fmt.Sprintf("Unknown tool: %s", params.Name))
	}
}

// ************************************************************************************************
// handlePing handles the ping request.
func (s *Server) handlePing(w http.ResponseWriter, req types.JSONRPCRequest) {
	log.Printf("Handling ping request")
	s.sendJSONRPCResult(w, req.ID, map[string]interface{}{})
}

// ************************************************************************************************
// handleResolveLibraryID handles the resolve-library-id tool.
func (s *Server) handleResolveLibraryID(w http.ResponseWriter, id interface{}, arguments map[string]interface{}) {
	// Extract library name
	libraryName, ok := arguments["libraryName"].(string)
	if !ok || libraryName == "" {
		s.sendToolError(w, id, "libraryName parameter is required and must be a string")
		return
	}

	log.Printf("Resolving library: %s", libraryName)

	// Find matching repositories
	matches := s.findRepositoryMatches(libraryName)
	if len(matches) == 0 {
		s.sendToolError(w, id, fmt.Sprintf("No repository found for library: %s", libraryName))
		return
	}

	// Return the best match as text content
	bestMatch := matches[0]
	result := types.MCPToolCallResult{
		Content: []types.MCPContent{
			{
				Type: "text",
				Text: bestMatch,
			},
		},
		IsError: false,
	}

	s.sendJSONRPCResult(w, id, result)
}

// ************************************************************************************************
// handleGetLibraryDocs handles the get-library-docs tool.
func (s *Server) handleGetLibraryDocs(w http.ResponseWriter, id interface{}, arguments map[string]interface{}) {
	// Extract library ID
	libraryID, ok := arguments["context7CompatibleLibraryID"].(string)
	if !ok || libraryID == "" {
		s.sendToolError(w, id, "context7CompatibleLibraryID parameter is required and must be a string")
		return
	}

	// Extract optional parameters
	topic, _ := arguments["topic"].(string)
	
	// Handle tokens parameter (can be number or string)
	tokens := 10000 // Default value
	if tokensParam, exists := arguments["tokens"]; exists {
		switch v := tokensParam.(type) {
		case float64:
			tokens = int(v)
		case int:
			tokens = v
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				tokens = parsed
			}
		}
	}

	// Ensure minimum token count
	if tokens < 1000 {
		tokens = 1000
	}

	log.Printf("Getting library docs: id=%s, topic=%s, tokens=%d", libraryID, topic, tokens)

	// Get repository documentation
	docs, err := s.getRepositoryDocs(libraryID, topic, tokens)
	if err != nil {
		s.sendToolError(w, id, err.Error())
		return
	}

	result := types.MCPToolCallResult{
		Content: []types.MCPContent{
			{
				Type: "text",
				Text: docs,
			},
		},
		IsError: false,
	}

	s.sendJSONRPCResult(w, id, result)
}

// ************************************************************************************************
// handleHealth handles health check requests.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":           "healthy",
		"repositories":     len(s.repositories),
		"cache_available":  s.cache != nil,
		"search_available": s.searchEngine != nil,
		"protocol":         "MCP JSON-RPC 2.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ************************************************************************************************
// sendJSONRPCResult sends a successful JSON-RPC response.
func (s *Server) sendJSONRPCResult(w http.ResponseWriter, id interface{}, result interface{}) {
	response := types.JSONRPCResponse{
		JsonRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON-RPC response: %v", err)
	}
}

// ************************************************************************************************
// sendJSONRPCError sends an error JSON-RPC response.
func (s *Server) sendJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	response := types.JSONRPCResponse{
		JsonRPC: "2.0",
		ID:      id,
		Error: &types.JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON-RPC error response: %v", err)
	}
}

// ************************************************************************************************
// sendToolError sends a tool execution error.
func (s *Server) sendToolError(w http.ResponseWriter, id interface{}, message string) {
	result := types.MCPToolCallResult{
		Content: []types.MCPContent{
			{
				Type: "text",
				Text: message,
			},
		},
		IsError: true,
	}

	s.sendJSONRPCResult(w, id, result)
}

// ************************************************************************************************
// parseParams parses JSON-RPC parameters into a struct.
func (s *Server) parseParams(params interface{}, target interface{}) error {
	if params == nil {
		return fmt.Errorf("params is nil")
	}

	// Convert to JSON and back to parse into target struct
	jsonData, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal params: %w", err)
	}

	return nil
}

// ************************************************************************************************
// findRepositoryMatches finds repositories matching a library name.
func (s *Server) findRepositoryMatches(libraryName string) []string {
	var matches []string
	
	// Get repositories from cache
	if s.cache != nil {
		repoIDs, err := s.cache.ListRepositories()
		if err == nil {
			for _, repoID := range repoIDs {
				// Simple string matching (case-insensitive)
				if strings.Contains(strings.ToLower(repoID), strings.ToLower(libraryName)) ||
					strings.Contains(strings.ToLower(libraryName), strings.ToLower(repoID)) {
					matches = append(matches, repoID)
				}
			}
		}
	}

	// Also check in-memory repositories
	for repoID := range s.repositories {
		if strings.Contains(strings.ToLower(repoID), strings.ToLower(libraryName)) ||
			strings.Contains(strings.ToLower(libraryName), strings.ToLower(repoID)) {
			// Avoid duplicates
			found := false
			for _, match := range matches {
				if match == repoID {
					found = true
					break
				}
			}
			if !found {
				matches = append(matches, repoID)
			}
		}
	}

	return matches
}

// ************************************************************************************************
// getRepositoryDocs retrieves documentation for a repository.
func (s *Server) getRepositoryDocs(libraryID, topic string, tokens int) (string, error) {
	// Try to get from cache first
	if s.cache != nil {
		repo, err := s.cache.GetRepository(libraryID)
		if err == nil {
			return s.extractDocumentation(repo, topic, tokens), nil
		}
	}

	// Try in-memory repositories
	if repo, exists := s.repositories[libraryID]; exists {
		return s.extractDocumentation(repo, topic, tokens), nil
	}

	return "", fmt.Errorf("repository not found: %s", libraryID)
}

// ************************************************************************************************
// extractDocumentation extracts and formats documentation from a repository.
func (s *Server) extractDocumentation(repo *types.RepositoryIndex, topic string, tokens int) string {
	log.Printf("Starting extractDocumentation: repo=%s, topic='%s', tokens=%d", repo.Name, topic, tokens)
	
	var docs strings.Builder
	
	// Add repository header
	docs.WriteString(fmt.Sprintf("# Repository: %s\n\n", repo.Name))
	docs.WriteString(fmt.Sprintf("**Path:** %s\n", repo.Path))
	docs.WriteString(fmt.Sprintf("**Last Updated:** %s\n", repo.LastUpdated.Format("2006-01-02 15:04:05")))
	if repo.CommitHash != "" {
		docs.WriteString(fmt.Sprintf("**Commit:** %s\n", repo.CommitHash))
	}
	docs.WriteString("\n")

	// Collect and prioritize files
	var priorityFiles []types.IndexedFile
	var otherFiles []types.IndexedFile

	for _, file := range repo.Files {
		// Skip if topic is specified and file doesn't contain it
		if topic != "" && !strings.Contains(strings.ToLower(file.Content), strings.ToLower(topic)) {
			continue
		}

		// Prioritize documentation files
		fileName := strings.ToLower(file.Path)
		if strings.Contains(fileName, "readme") ||
		   strings.Contains(fileName, "doc") ||
		   strings.HasSuffix(fileName, ".md") ||
		   strings.Contains(fileName, "changelog") ||
		   strings.Contains(fileName, "license") {
			priorityFiles = append(priorityFiles, file)
		} else {
			otherFiles = append(otherFiles, file)
		}
	}

	log.Printf("File categorization: priority=%d, other=%d, total=%d", len(priorityFiles), len(otherFiles), len(repo.Files))

	// Add priority files first
	currentTokens := len(docs.String())
	log.Printf("Initial token count: %d", currentTokens)
	
	for i, file := range priorityFiles {
		log.Printf("Processing priority file %d/%d: %s (content length: %d)", i+1, len(priorityFiles), file.Path, len(file.Content))
		
		if currentTokens >= tokens {
			log.Printf("Token limit reached, skipping remaining priority files")
			break
		}
		
		docs.WriteString(fmt.Sprintf("\n## File: %s\n\n", file.Path))
		
		// Safe truncation with bounds checking
		content := file.Content
		contentLength := len(content)
		remainingTokens := tokens - currentTokens
		
		log.Printf("Token calculation: current=%d, remaining=%d, content=%d", currentTokens, remainingTokens, contentLength)
		
		if contentLength > remainingTokens {
			// Calculate safe truncation point
			truncateLength := remainingTokens - 100 // Reserve 100 chars for truncation message
			if truncateLength <= 0 {
				log.Printf("No space left for content, skipping file: %s", file.Path)
				continue
			}
			if truncateLength > contentLength {
				truncateLength = contentLength
			}
			
			log.Printf("Truncating content from %d to %d characters", contentLength, truncateLength)
			content = content[:truncateLength] + "\n\n[Content truncated...]"
		}
		
		docs.WriteString(content)
		docs.WriteString("\n")
		currentTokens = len(docs.String())
		log.Printf("Updated token count after file %s: %d", file.Path, currentTokens)
	}

	// Add other files if we still have token budget
	for i, file := range otherFiles {
		log.Printf("Processing other file %d/%d: %s (content length: %d)", i+1, len(otherFiles), file.Path, len(file.Content))
		
		if currentTokens >= tokens {
			log.Printf("Token limit reached, skipping remaining other files")
			break
		}
		
		docs.WriteString(fmt.Sprintf("\n## File: %s\n\n", file.Path))
		
		// Safe truncation with bounds checking
		content := file.Content
		contentLength := len(content)
		remainingTokens := tokens - currentTokens
		
		log.Printf("Token calculation: current=%d, remaining=%d, content=%d", currentTokens, remainingTokens, contentLength)
		
		if contentLength > remainingTokens {
			// Calculate safe truncation point
			truncateLength := remainingTokens - 100 // Reserve 100 chars for truncation message
			if truncateLength <= 0 {
				log.Printf("No space left for content, skipping file: %s", file.Path)
				continue
			}
			if truncateLength > contentLength {
				truncateLength = contentLength
			}
			
			log.Printf("Truncating content from %d to %d characters", contentLength, truncateLength)
			content = content[:truncateLength] + "\n\n[Content truncated...]"
		}
		
		docs.WriteString(content)
		docs.WriteString("\n")
		currentTokens = len(docs.String())
		log.Printf("Updated token count after file %s: %d", file.Path, currentTokens)
	}

	// Add summary if we truncated
	finalLength := len(docs.String())
	if finalLength >= tokens {
		docs.WriteString(fmt.Sprintf("\n---\n**Note:** Documentation truncated to %d tokens. Repository contains %d total files.\n", tokens, len(repo.Files)))
	}

	log.Printf("Documentation extraction completed: final length=%d, target=%d", finalLength, tokens)
	return docs.String()
}

// ************************************************************************************************
// UpdateRepository updates a repository in the server.
func (s *Server) UpdateRepository(repo *types.RepositoryIndex) error {
	if repo == nil {
		return fmt.Errorf("repository cannot be nil")
	}

	s.repositories[repo.ID] = repo
	log.Printf("Updated repository in MCP server: %s", repo.ID)
	return nil
}

// ************************************************************************************************
// Stop gracefully stops the MCP server.
func (s *Server) Stop() error {
	log.Printf("MCP server stopped")
	return nil
}