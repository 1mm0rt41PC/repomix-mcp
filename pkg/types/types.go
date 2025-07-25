// ************************************************************************************************
// Package types provides shared data structures and interfaces for the repomix-mcp application.
// This package contains core types used across different components including repository
// configuration, cache management, and MCP server operations.
package types

import (
	"time"
)

// ************************************************************************************************
// RepositoryType defines the type of repository source.
type RepositoryType string

const (
	// RepositoryTypeLocal represents a local filesystem repository.
	RepositoryTypeLocal RepositoryType = "local"

	// RepositoryTypeRemote represents a remote Git repository.
	RepositoryTypeRemote RepositoryType = "remote"
)

// ************************************************************************************************
// AuthType defines the authentication method for repository access.
type AuthType string

const (
	// AuthTypeNone indicates no authentication is required.
	AuthTypeNone AuthType = "none"

	// AuthTypeSSH indicates SSH key-based authentication.
	AuthTypeSSH AuthType = "ssh"

	// AuthTypeToken indicates token-based authentication.
	AuthTypeToken AuthType = "token"
)

// ************************************************************************************************
// RepositoryAuth contains authentication configuration for repository access.
// It supports multiple authentication methods including SSH keys and access tokens.
type RepositoryAuth struct {
	Type     AuthType `json:"type" mapstructure:"type"`         // Authentication method
	KeyPath  string   `json:"keyPath" mapstructure:"keyPath"`   // Path to SSH private key
	Token    string   `json:"token" mapstructure:"token"`       // Access token for authentication
	Username string   `json:"username" mapstructure:"username"` // Username for token authentication
}

// ************************************************************************************************
// IndexingConfig defines configuration options for repository indexing.
// It controls which files are processed and how the indexing operation behaves.
type IndexingConfig struct {
	Enabled            bool     `json:"enabled" mapstructure:"enabled"`                       // Whether indexing is enabled
	ExcludePatterns    []string `json:"excludePatterns" mapstructure:"excludePatterns"`       // File patterns to exclude
	IncludePatterns    []string `json:"includePatterns" mapstructure:"includePatterns"`       // File patterns to include
	MaxFileSize        string   `json:"maxFileSize" mapstructure:"maxFileSize"`               // Maximum file size to index
	IncludeNonExported bool     `json:"includeNonExported" mapstructure:"includeNonExported"` // Include non-exported constructs (default: false)
}

// ************************************************************************************************
// RepositoryConfig represents configuration for a single repository.
// It contains all necessary information to clone, authenticate, and index a repository.
type RepositoryConfig struct {
	Type     RepositoryType `json:"type" mapstructure:"type"`         // Repository source type
	Path     string         `json:"path" mapstructure:"path"`         // Local path or remote URL
	URL      string         `json:"url" mapstructure:"url"`           // Git repository URL for remote repos
	Auth     RepositoryAuth `json:"auth" mapstructure:"auth"`         // Authentication configuration
	Indexing IndexingConfig `json:"indexing" mapstructure:"indexing"` // Indexing behavior configuration
	Branch   string         `json:"branch" mapstructure:"branch"`     // Git branch to index (default: main)
}

// ************************************************************************************************
// CacheConfig defines configuration for the BadgerDB cache system.
// It controls cache behavior, storage limits, and data retention policies.
type CacheConfig struct {
	Path    string `json:"path" mapstructure:"path"`       // Cache storage directory path
	MaxSize string `json:"maxSize" mapstructure:"maxSize"` // Maximum cache size
	TTL     string `json:"ttl" mapstructure:"ttl"`         // Time-to-live for cached entries
}

// ************************************************************************************************
// ServerConfig contains configuration for the MCP server.
// It defines network settings and operational parameters for the server.
type ServerConfig struct {
	Port     int    `json:"port" mapstructure:"port"`         // Server listening port
	LogLevel string `json:"logLevel" mapstructure:"logLevel"` // Logging verbosity level
	Host     string `json:"host" mapstructure:"host"`         // Server binding host

	// HTTPS Configuration
	HTTPSEnabled bool   `json:"httpsEnabled" mapstructure:"httpsEnabled"` // Enable HTTPS server
	HTTPSPort    int    `json:"httpsPort" mapstructure:"httpsPort"`       // HTTPS server port (default: 9443)
	CertPath     string `json:"certPath" mapstructure:"certPath"`         // Path to TLS certificate file
	KeyPath      string `json:"keyPath" mapstructure:"keyPath"`           // Path to TLS private key file
	AutoGenCert  bool   `json:"autoGenCert" mapstructure:"autoGenCert"`   // Auto-generate self-signed certificate
}

// ************************************************************************************************
// Config represents the complete application configuration.
// It combines repository definitions, cache settings, and server configuration.
type Config struct {
	Repositories map[string]RepositoryConfig `json:"repositories" mapstructure:"repositories"` // Repository definitions by alias
	Cache        CacheConfig                 `json:"cache" mapstructure:"cache"`               // Cache system configuration
	Server       ServerConfig                `json:"server" mapstructure:"server"`             // MCP server configuration
	GoModule     GoModuleConfig              `json:"goModule" mapstructure:"goModule"`         // Go module documentation configuration
}

// ************************************************************************************************
// IndexedFile represents a file that has been processed and stored in the cache.
// It contains metadata and content information for efficient retrieval.
type IndexedFile struct {
	Path         string            `json:"path"`         // Relative file path within repository
	Content      string            `json:"content"`      // File content
	Hash         string            `json:"hash"`         // Content hash for change detection
	Size         int64             `json:"size"`         // File size in bytes
	ModTime      time.Time         `json:"modTime"`      // Last modification time
	Language     string            `json:"language"`     // Detected programming language
	RepositoryID string            `json:"repositoryId"` // Repository identifier
	Metadata     map[string]string `json:"metadata"`     // Additional file metadata
}

// ************************************************************************************************
// RepositoryIndex contains all indexed files and metadata for a repository.
// It provides a complete view of the repository's indexed content.
type RepositoryIndex struct {
	ID          string                 `json:"id"`          // Unique repository identifier
	Name        string                 `json:"name"`        // Repository display name
	Path        string                 `json:"path"`        // Local repository path
	LastUpdated time.Time              `json:"lastUpdated"` // Last indexing timestamp
	Files       map[string]IndexedFile `json:"files"`       // Indexed files by path
	Metadata    map[string]interface{} `json:"metadata"`    // Repository metadata
	CommitHash  string                 `json:"commitHash"`  // Current Git commit hash
}

// ************************************************************************************************
// SearchResult represents a single search result with relevance scoring.
// It provides context and ranking information for search matches.
type SearchResult struct {
	File        IndexedFile `json:"file"`        // Matched file information
	Score       float64     `json:"score"`       // Relevance score (0.0 to 1.0)
	Snippet     string      `json:"snippet"`     // Content snippet showing match context
	LineNumber  int         `json:"lineNumber"`  // Line number of match
	MatchCount  int         `json:"matchCount"`  // Number of matches in file
	Highlighted string      `json:"highlighted"` // Highlighted match text
}

// ************************************************************************************************
// SearchQuery defines parameters for content search operations.
// It supports various search modes and filtering options.
type SearchQuery struct {
	Query        string `json:"query"`        // Search query string
	RepositoryID string `json:"repositoryId"` // Target repository (empty for all)
	FilePattern  string `json:"filePattern"`  // File name pattern filter
	Language     string `json:"language"`     // Programming language filter
	MaxResults   int    `json:"maxResults"`   // Maximum number of results
	Topic        string `json:"topic"`        // Topic filter for focused search
	Tokens       int    `json:"tokens"`       // Maximum tokens in response
}

// ************************************************************************************************
// JSONRPCRequest represents a JSON-RPC 2.0 request message.
type JSONRPCRequest struct {
	JsonRPC string      `json:"jsonrpc"`          // JSON-RPC version (must be "2.0")
	ID      interface{} `json:"id,omitempty"`     // Request identifier (can be string, number, or null)
	Method  string      `json:"method"`           // Method name
	Params  interface{} `json:"params,omitempty"` // Method parameters
}

// ************************************************************************************************
// JSONRPCResponse represents a JSON-RPC 2.0 response message.
type JSONRPCResponse struct {
	JsonRPC string        `json:"jsonrpc"`          // JSON-RPC version (must be "2.0")
	ID      interface{}   `json:"id"`               // Request identifier (matches request ID)
	Result  interface{}   `json:"result,omitempty"` // Result data (on success)
	Error   *JSONRPCError `json:"error,omitempty"`  // Error information (on failure)
}

// ************************************************************************************************
// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int         `json:"code"`           // Error code
	Message string      `json:"message"`        // Error message
	Data    interface{} `json:"data,omitempty"` // Additional error data
}

// ************************************************************************************************
// JSONRPCNotification represents a JSON-RPC 2.0 notification message.
type JSONRPCNotification struct {
	JsonRPC string      `json:"jsonrpc"`          // JSON-RPC version (must be "2.0")
	Method  string      `json:"method"`           // Method name
	Params  interface{} `json:"params,omitempty"` // Method parameters
}

// ************************************************************************************************
// MCPInitializeRequest represents the MCP initialize request.
type MCPInitializeRequest struct {
	ProtocolVersion string                 `json:"protocolVersion"` // MCP protocol version
	Capabilities    map[string]interface{} `json:"capabilities"`    // Client capabilities
	ClientInfo      map[string]interface{} `json:"clientInfo"`      // Client information
}

// ************************************************************************************************
// MCPInitializeResult represents the MCP initialize response.
type MCPInitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"` // Server protocol version
	Capabilities    map[string]interface{} `json:"capabilities"`    // Server capabilities
	ServerInfo      map[string]interface{} `json:"serverInfo"`      // Server information
}

// ************************************************************************************************
// MCPToolsListResult represents the response to tools/list.
type MCPToolsListResult struct {
	Tools []MCPTool `json:"tools"` // Available tools
}

// ************************************************************************************************
// MCPTool represents a tool definition in MCP.
type MCPTool struct {
	Name        string                 `json:"name"`        // Tool name
	Description string                 `json:"description"` // Tool description
	InputSchema map[string]interface{} `json:"inputSchema"` // JSON Schema for inputs
}

// ************************************************************************************************
// MCPToolCallParams represents parameters for tools/call.
type MCPToolCallParams struct {
	Name      string                 `json:"name"`      // Tool name
	Arguments map[string]interface{} `json:"arguments"` // Tool arguments
}

// ************************************************************************************************
// MCPToolCallResult represents the result of tools/call.
type MCPToolCallResult struct {
	Content []MCPContent `json:"content"` // Response content
	IsError bool         `json:"isError"` // Whether this is an error result
}

// ************************************************************************************************
// MCPContent represents content in MCP responses.
type MCPContent struct {
	Type string `json:"type"` // Content type ("text", "image", etc.)
	Text string `json:"text"` // Text content (for type "text")
}

// Legacy types for backward compatibility
// ************************************************************************************************
// MCPRequest represents an incoming MCP tool request (legacy).
type MCPRequest struct {
	Tool       string                 `json:"tool"`       // MCP tool name
	Parameters map[string]interface{} `json:"parameters"` // Tool parameters
	RequestID  string                 `json:"requestId"`  // Unique request identifier
}

// ************************************************************************************************
// MCPResponse represents an MCP tool response (legacy).
type MCPResponse struct {
	Success   bool                   `json:"success"`   // Operation success status
	Data      interface{}            `json:"data"`      // Response data
	Error     string                 `json:"error"`     // Error message if failed
	RequestID string                 `json:"requestId"` // Corresponding request identifier
	Metadata  map[string]interface{} `json:"metadata"`  // Additional response metadata
}
