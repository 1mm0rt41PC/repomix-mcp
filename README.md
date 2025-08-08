# Repomix-MCP

[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/doc/go1.24)
[![Coverage](https://img.shields.io/badge/Coverage-0%25-red.svg)](coverage.html)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v2.1.1-blue.svg)](https://github.com/1mm0rt41PC/repomix-mcp/releases)
[![Windows](https://img.shields.io/badge/Windows-Compatible-success.svg)](https://github.com/1mm0rt41PC/repomix-mcp)
[![Linux](https://img.shields.io/badge/Linux-Compatible-success.svg)](https://github.com/1mm0rt41PC/repomix-mcp)

A Context7-compatible repository indexing and MCP server for internal private repositories. This application provides the same functionality as Context7 but for your own private repositories, using repomix as the CLI indexer and serving content through an MCP (Model Context Protocol) server.

## Overview

Repomix-MCP bridges the gap between your private repositories and AI tools by providing:

- **Repository Indexing**: Automated indexing of both local and remote repositories using repomix
- **Context7 Compatibility**: Drop-in replacement for Context7 with the same MCP tools
- **Efficient Caching**: BadgerDB-based caching for fast content retrieval
- **Authentication Support**: SSH keys and tokens for private repository access
- **Search Capabilities**: Semantic search across indexed repository content
- **Easy Configuration**: JSON-based configuration with sensible defaults

## Features

- 🔄 **Automatic Repository Management**: Clone, update, and track changes in repositories
- 🔍 **Intelligent Search**: Content-based search with relevance scoring and filtering
- 📝 **Optimized Content Processing**: Uses repomix with compression, comment removal, and empty line removal for cleaner output
- 🛡️ **Secure Authentication**: Support for SSH keys and access tokens
- ⚡ **Fast Caching**: BadgerDB storage for quick content retrieval
- 🔌 **MCP Integration**: Standard Model Context Protocol for AI tool compatibility
- 📱 **Independent MCP Client**: Built-in command-line client for testing and interaction
- 🎨 **Colorized JSON Output**: Syntax highlighting for improved readability
- 🧠 **Smart Auto-Content**: Automatic documentation inclusion for single-match resolutions
- 📊 **Comprehensive Logging**: Detailed logging and error reporting
- 🔧 **Flexible Configuration**: Support for multiple repository types and indexing rules
- 🎯 **Smart Go Analysis**: Advanced Go AST parsing with configurable export filtering (`includeNonExported`)

## Installation

### Prerequisites

- Go 1.24 or later
- [Repomix CLI](https://github.com/yamadashy/repomix) installed and available in PATH
- Git (for repository operations)

### Building from Source

```bash
git clone <repository-url>
cd repomix-mcp
go mod download
go build -o repomix-mcp ./cmd/repomix-mcp
```

### Binary Installation

Download the latest binary from the [releases page](https://github.com/user/repomix-mcp/releases).

## Quick Start

### 1. Create Configuration

Generate an example configuration file:

```bash
./repomix-mcp config example config.json
```

Edit the configuration to add your repositories:

```json
{
  "repositories": {
    "my-repo": {
      "type": "remote",
      "url": "git@github.com:org/private-repo.git",
      "auth": {
        "type": "ssh",
        "keyPath": "~/.ssh/id_rsa"
      },
      "indexing": {
        "enabled": true,
        "excludePatterns": ["*.log", "node_modules"],
        "includePatterns": ["*.go", "*.md", "*.json"],
        "includeNonExported": false
      }
    }
  },
  "cache": {
    "path": "~/.repomix-mcp",
    "maxSize": "1GB",
    "ttl": "24h"
  },
  "server": {
    "port": 8080,
    "host": "localhost",
    "logLevel": "info"
  }
}
```

### 2. Validate Setup

Check your configuration and dependencies:

```bash
./repomix-mcp validate -c config.json
```

### 3. Index Repositories

Index all configured repositories:

```bash
./repomix-mcp index -c config.json
```

Or index a specific repository:

```bash
./repomix-mcp index my-repo -c config.json
```

### 4. Start MCP Server

Start the server to serve content to AI tools:

```bash
./repomix-mcp serve -c config.json
```

The server will be available at `http://localhost:8080/mcp` (or your configured host/port).

## Configuration

### Repository Types

#### Local Repository
```json
{
  "type": "local",
  "path": "/path/to/local/repo",
  "auth": {"type": "none"}
}
```

#### Local Repository with Glob Pattern
```json
{
  "type": "local",
  "path": "C:\\Projects\\*",
  "auth": {"type": "none"}
}
```

**Glob Pattern Support**: Local repositories support glob patterns to automatically discover and index multiple directories. When a glob pattern like `C:\Projects\*` is used, the application will:

1. Expand the pattern to find all matching directories
2. Create separate repository entries for each discovered directory
3. Index each directory individually with separate repomix calls
4. Generate unique aliases based on the directory names

**Note**: Directories don't need to be git repositories. The application can index any directory containing code, whether it's a git repository, a folder with multiple projects, or just a collection of source files.

**Supported Glob Patterns**:
- `*` - Matches any sequence of characters (except path separators)
- `?` - Matches any single character
- `[]` - Character classes (e.g., `[abc]` matches a, b, or c)
- `**` - Recursive directory matching
- `{}` - Alternatives (e.g., `{a,b}` matches a or b)

**Examples**:
- `C:\Projects\*` - All direct subdirectories in C:\Projects
- `~/workspaces/*/*` - All subdirectories two levels deep in workspaces
- `C:\Code\{web,api}\*` - All subdirectories in either web or api folders
- `/home/user/repos/**` - All directories recursively under repos

#### Remote Repository with SSH
```json
{
  "type": "remote",
  "url": "git@github.com:org/repo.git",
  "auth": {
    "type": "ssh",
    "keyPath": "~/.ssh/id_rsa"
  }
}
```

#### Remote Repository with Token
```json
{
  "type": "remote",
  "url": "https://github.com/org/repo.git",
  "auth": {
    "type": "token",
    "token": "your-access-token",
    "username": "your-username"
  }
}
```

### Indexing Configuration

Control what gets indexed:

```json
{
  "indexing": {
    "enabled": true,
    "excludePatterns": [
      "*.log", "node_modules", ".git",
      "vendor", "target", "build"
    ],
    "includePatterns": [
      "*.go", "*.js", "*.py", "*.md",
      "*.json", "*.yaml"
    ],
    "maxFileSize": "1MB",
    "includeNonExported": false
  }
}
```

### Go Module Configuration

Configure Go module documentation retrieval and fallback behavior:

```json
{
  "goModule": {
    "enabled": true,
    "tempDirBase": "/tmp/repomix-mcp-gomod",
    "cacheTimeout": "24h",
    "commandTimeout": "30s",
    "maxRetries": 3,
    "maxConcurrent": 5
  }
}
```

#### Go Module Options

**`enabled`** (boolean, default: `true`):
- Enable Go module fallback for libraries not found in configured repositories
- When enabled, the system can automatically fetch and document Go modules from the internet
- Useful for resolving external Go dependencies and standard library documentation

**`tempDirBase`** (string, default: `/tmp/repomix-mcp-gomod`):
- Base directory for temporary Go module downloads
- The system creates subdirectories here for each Go module being processed
- Should be writable and have sufficient disk space for module downloads

**`cacheTimeout`** (string, default: `24h`):
- How long to cache Go module documentation before re-fetching
- Valid formats: "1h", "24h", "7d", etc.
- Longer timeouts reduce network usage but may miss updates

**`commandTimeout`** (string, default: `30s`):
- Timeout for individual Go commands (go mod download, go list, etc.)
- Prevents hanging operations on slow networks or large modules
- Valid formats: "10s", "1m", "5m", etc.

**`maxRetries`** (integer, default: `3`):
- Maximum number of retry attempts for failed Go operations
- Helps handle transient network issues or temporary module unavailability
- Set to 0 to disable retries

**`maxConcurrent`** (integer, default: `5`):
- Maximum number of concurrent Go module operations
- Limits resource usage when processing multiple modules simultaneously
- Balance between performance and system resource consumption

#### Configuration Examples

**Conservative Configuration (slower but more reliable):**
```json
{
  "goModule": {
    "enabled": true,
    "tempDirBase": "~/.repomix-mcp/gomod",
    "cacheTimeout": "72h",
    "commandTimeout": "60s",
    "maxRetries": 5,
    "maxConcurrent": 2
  }
}
```

**Performance-Optimized Configuration:**
```json
{
  "goModule": {
    "enabled": true,
    "tempDirBase": "/tmp/repomix-gomod",
    "cacheTimeout": "12h",
    "commandTimeout": "15s",
    "maxRetries": 2,
    "maxConcurrent": 10
  }
}
```

**Disable Go Module Fallback:**
```json
{
  "goModule": {
    "enabled": false
  }
}
```

#### How Go Module Fallback Works

When a library is requested via the `resolve-library-id` or `get-library-docs` tools:

1. **Local Search**: First searches configured repositories for matching libraries
2. **Go Module Detection**: If no local match is found, checks if the query looks like a Go module path
3. **Module Resolution**: Downloads and processes the Go module using `go mod download`
4. **Documentation Generation**: Runs repomix on the downloaded module to generate documentation
5. **Caching**: Stores the result in cache for future requests

**Example Go Module Paths:**
- `github.com/sirupsen/logrus`
- `golang.org/x/crypto/ssh`
- `google.golang.org/grpc`
- `github.com/gin-gonic/gin`

#### Security Considerations

**Important**: Go module fallback downloads code from the internet. Consider these security implications:

- **Network Access**: The server needs internet access to download modules
- **Disk Usage**: Downloaded modules consume disk space in `tempDirBase`
- **Execution**: Go commands are executed on the server (go mod download, go list)
- **Trust**: Only download modules from trusted sources

**Recommended Security Practices:**
- Use a dedicated temporary directory with appropriate permissions
- Monitor disk usage in the temporary directory
- Consider running in a sandboxed environment
- Regularly clean up old temporary files
- Use short cache timeouts for frequently updated modules

### Cache Configuration

Configure BadgerDB cache:

```json
{
  "cache": {
    "path": "~/.repomix-mcp",
    "maxSize": "1GB",
    "ttl": "24h"
  }
}
```

### Server Configuration

Configure the MCP server:

```json
{
  "server": {
    "port": 8080,
    "host": "localhost",
    "logLevel": "info"
  }
}
```

## MCP Server Integration

The server implements a fully compliant JSON-RPC 2.0 Model Context Protocol (MCP) server following the official MCP specification.

### MCP Endpoint

**Main Endpoint**: `http://localhost:8080/mcp`

This endpoint implements the official MCP JSON-RPC 2.0 protocol with proper initialization, tool discovery, and tool execution.

### Configuration for AI Clients

Add this to your MCP configuration:

```json
{
  "mcpServers": {
    "repomix-mcp": {
      "type": "streamable-http",
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

### MCP Protocol Flow

#### 1. Initialize
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {},
    "clientInfo": {"name": "client", "version": "1.0"}
  }
}
```

#### 2. List Tools
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

#### 3. Call Tools
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "resolve-library-id",
    "arguments": {
      "libraryName": "authentication"
    }
  }
}
```

### Available Tools

#### resolve-library-id (Enhanced)

**Enhanced Tool**: Resolves a general library name into a repository ID. **New**: If exactly one match is found, automatically includes the full documentation content (public/exported data only).

**Smart Behavior:**
- **Multiple matches**: Returns numbered list of repository IDs
- **Single match**: Returns repository ID + complete documentation content
- **No matches**: Returns error message

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "libraryName": {
      "type": "string",
      "description": "The name of the library to search for"
    },
    "tokens": {
      "type": "number",
      "description": "Maximum number of tokens to return for auto-included content (only applies when exactly one match is found)",
      "default": 10000
    }
  },
  "required": ["libraryName"]
}
```

**Example Response (Multiple Matches):**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Multiple repositories found for 'auth':\n\n1. auth-service\n2. auth-lib\n3. oauth-gateway\n\nUse get-library-docs with one of these IDs to retrieve documentation."
      }
    ],
    "isError": false
  }
}
```

**Example Response (Single Match with Auto-Content):**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Repository ID: auth-service\n\n# Repository: auth-service\n\n**Path:** /path/to/auth\n**Last Updated:** 2024-01-31 14:35:00\n\n## File: README.md\n\n# Authentication Service\n\nThis service handles JWT authentication...\n\n## File: main.go\n\npackage main\n\nimport (\n    \"github.com/gin-gonic/gin\"\n)\n\n// StartServer initializes the HTTP server\nfunc StartServer() {\n    // Implementation details...\n}\n\n[Full documentation content continues...]"
      }
    ],
    "isError": false
  }
}
```

#### get-library-docs

Fetches documentation for a repository using its ID.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "library-id": {
      "type": "string",
      "description": "Repository ID from resolve-library-id"
    },
    "topic": {
      "type": "string",
      "description": "Focus the docs on a specific topic"
    },
    "tokens": {
      "type": "number",
      "description": "Maximum number of tokens to return",
      "default": 10000
    },
    "includeNonExported": {
      "type": "boolean",
      "description": "Include non-exported constructs in Go projects (default: false)",
      "default": false
    }
  },
  "required": ["library-id"]
}
```

**New Feature: includeNonExported**

The `includeNonExported` parameter controls the level of detail in Go project documentation:

- **`false` (default)**: Only exported (public) constructs are included
  - Functions, types, variables, and constants that start with uppercase letters
  - Provides clean API documentation focused on public interfaces
  - Faster processing and smaller output

- **`true`**: All constructs (both exported and non-exported) are included
  - Complete codebase analysis including internal implementations
  - Useful for code reviews, architecture analysis, and refactoring
  - More comprehensive but larger output

**Usage Examples:**

```json
// API documentation (public interface only)
{
  "name": "get-library-docs",
  "arguments": {
    "library-id": "gomod:github.com/sirupsen/logrus",
    "includeNonExported": false,
    "tokens": 8000
  }
}

// Complete code analysis (all constructs)
{
  "name": "get-library-docs",
  "arguments": {
    "library-id": "my-go-project",
    "includeNonExported": true,
    "tokens": 15000
  }
}
```

**Example Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "# Repository: auth-service\n\n**Path:** /path/to/auth\n\n## File: README.md\n\n..."
      }
    ],
    "isError": false
  }
}
```

### Protocol Compliance

- ✅ **JSON-RPC 2.0**: Full compliance with JSON-RPC 2.0 specification
- ✅ **MCP 2024-11-05**: Compatible with VS Code and current MCP clients
- ✅ **Tool Discovery**: Proper `tools/list` implementation
- ✅ **Tool Execution**: Compliant `tools/call` implementation
- ✅ **Error Handling**: Standard JSON-RPC error responses
- ✅ **CORS Support**: Cross-origin headers for web clients

### Health Check

**Endpoint**: `GET /health`

Returns server status and capability information.

## MCP Client

Repomix-MCP includes a built-in **independent MCP client** for testing, debugging, and direct interaction with MCP servers. The client provides a complete command-line interface with colorized JSON output and smart features.

### Key Features

- 🎨 **Colorized JSON Output**: Syntax highlighting with custom ANSI colors (no external dependencies)
- 📱 **Independent Operation**: No config file or BadgerDB required - completely standalone
- 🔗 **HTTP/HTTPS Support**: Connect to any MCP-compatible server
- 🛠️ **Tool Discovery**: List available tools with detailed schemas
- ⚡ **Tool Execution**: Execute tools with flexible argument parsing
- 📊 **Multiple Output Formats**: JSON, table, and raw text output
- 🔍 **Verbose Debugging**: Detailed connection and execution information

### Client Usage

#### Basic Commands

```bash
# List available tools from local server
./repomix-mcp client --mcp-srv 127.0.0.1:8080 --mcp-list

# List tools from remote HTTPS server
./repomix-mcp client --mcp-srv https://server.com:443 --mcp-list --verbose

# Execute a tool with arguments
./repomix-mcp client --mcp-use resolve-library-id --mcp-args="libraryName=golang"

# Execute with custom token limit (new feature)
./repomix-mcp client --mcp-use resolve-library-id --mcp-args="libraryName=gin,tokens=15000"

# Multiple arguments
./repomix-mcp client --mcp-use get-library-docs --mcp-args="library-id=my-repo,tokens=5000,topic=authentication"
```

#### Enhanced resolve-library-id Tool

The `resolve-library-id` tool now includes **smart auto-content inclusion**:

- **Multiple matches**: Returns numbered list of repository IDs
- **Single match**: Automatically includes full documentation content (public/exported only)
- **Configurable tokens**: Control content size with `tokens` parameter
- **Public/exported focus**: Clean, API-focused documentation

```bash
# Smart resolution with auto-content
./repomix-mcp client --mcp-use resolve-library-id --mcp-args="libraryName=myapi,tokens=12000"

# If only one repository matches "myapi", you get:
# Repository ID: myapi-v2
#
# # Repository: myapi-v2
# **Path:** /path/to/myapi
# **Last Updated:** 2024-01-31 14:35:00
#
# ## File: README.md
# [Full documentation content here...]
```

#### Output Formats

```bash
# Colorized JSON (default)
./repomix-mcp client --mcp-list --format json

# Human-readable table
./repomix-mcp client --mcp-list --format table

# Raw text output
./repomix-mcp client --mcp-list --format raw
```

#### Verbose Mode

```bash
# Detailed debugging information
./repomix-mcp client --mcp-list --verbose

# Output includes:
# MCP CLIENT CONNECTION:
# ------------------------------
# Server: 127.0.0.1:8080
# Status: CONNECTED
#
# [Detailed request/response information]
```

### Client Configuration

The client accepts these command-line flags:

| Flag | Description | Default |
|------|-------------|---------|
| `--mcp-srv` | MCP server address | `127.0.0.1:8080` |
| `--mcp-list` | List available tools | `false` |
| `--mcp-use` | Tool name to execute | `""` |
| `--mcp-args` | Tool arguments (key=value,key2=value2) | `""` |
| `--format` | Output format (json, table, raw) | `json` |
| `--verbose` | Show detailed information | `false` |

### JSON Syntax Highlighting

The client features **built-in JSON syntax highlighting** without external dependencies:

- 🟣 **Purple**: Property keys (`"name":`)
- 🔷 **Cyan**: String values (`"resolve-library-id"`)
- 🔵 **Blue**: Numbers (`4`, `10000`)
- 🟢 **Green**: `true` values
- 🔴 **Red**: `false` values
- 🟣 **Purple**: `null` values
- 🟡 **Yellow**: Structural characters `{}`
- ⚪ **White**: Punctuation `:,`

### Server Testing Workflow

Use the client to test your MCP server:

```bash
# 1. Start your server
./repomix-mcp serve -c config.json &

# 2. Test connection and list tools
./repomix-mcp client --mcp-list --verbose

# 3. Test tool resolution
./repomix-mcp client --mcp-use resolve-library-id --mcp-args="libraryName=test"

# 4. Test documentation retrieval
./repomix-mcp client --mcp-use get-library-docs --mcp-args="library-id=your-repo,includeNonExported=true"

# 5. Test with different output formats
./repomix-mcp client --mcp-list --format table
```

### Advanced Usage Examples

```bash
# Test Go module fallback
./repomix-mcp client --mcp-use resolve-library-id --mcp-args="libraryName=github.com/gin-gonic/gin"

# Get detailed documentation with non-exported constructs
./repomix-mcp client --mcp-use get-library-docs --mcp-args="library-id=gomod:github.com/sirupsen/logrus,includeNonExported=true,tokens=20000"

# Focus on specific topic
./repomix-mcp client --mcp-use get-library-docs --mcp-args="library-id=my-auth-service,topic=jwt,tokens=8000"

# Test against remote server
./repomix-mcp client --mcp-srv https://api.company.com:9443 --mcp-list --verbose
```

### Troubleshooting Client Issues

**Connection Refused**:
```bash
# Check if server is running
curl http://127.0.0.1:8080/health

# Try different port
./repomix-mcp client --mcp-srv 127.0.0.1:9080 --mcp-list
```

**Color Issues**:
```bash
# If colors don't display correctly, use raw format
./repomix-mcp client --mcp-list --format raw

# Or redirect to file to avoid color codes
./repomix-mcp client --mcp-list > output.json
```

**Authentication Errors**:
```bash
# Test with verbose mode for detailed error information
./repomix-mcp client --mcp-list --verbose
```

## Usage Examples

### AI Tool Integration

Use with AI tools that support MCP:

```bash
# With Claude Desktop or other MCP-compatible tools
# Add this to your MCP configuration:
{
  "servers": {
    "repomix-mcp": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "http://localhost:8080/mcp/resolve-library-id",
        "-H", "Content-Type: application/json",
        "-d", "{\"tool\":\"resolve-library-id\",\"parameters\":{\"libraryName\":\"$LIBRARY\"}}"
      ]
    }
  }
}
```

### Command Line Usage

```bash
# Validate configuration
./repomix-mcp validate

# Index specific repository (will expand globs automatically)
./repomix-mcp index my-api-service

# Index all repositories (expands all glob patterns)
./repomix-mcp index

# Start server in background
./repomix-mcp serve &

# Generate new example config
./repomix-mcp config example new-config.json
```

### Glob Pattern Examples

When you configure a repository with a glob pattern like this:

```json
{
  "repositories": {
    "my-projects": {
      "type": "local",
      "path": "C:\\Projects\\*"
    }
  }
}
```

And your directory structure is:
```
C:\Projects\
├── project-a\
├── project-b\
└── project-c\
```

The application will automatically:
1. **Discover**: Find all matching directories (`project-a`, `project-b`, `project-c`)
2. **Expand**: Create separate repository entries (`my-projects-project-a`, `my-projects-project-b`, `my-projects-project-c`)
3. **Index**: Run repomix separately on each directory
4. **Cache**: Store each repository separately for individual access via MCP tools

**Result**: You get 3 separate repomix calls and 3 separate cached repositories, each optimized and indexed individually.

### Scheduled Indexing

Set up a cron job for regular updates:

```bash
# Update repositories every hour
0 * * * * /path/to/repomix-mcp index -c /path/to/config.json
```

## Architecture

### Components

- **Configuration Manager**: Handles JSON configuration loading and validation
- **Repository Manager**: Manages Git operations (clone, pull, authentication)
- **Indexer**: Integrates with repomix CLI for content extraction
- **Cache System**: BadgerDB-based storage for indexed content
- **Search Engine**: Content search with relevance scoring
- **MCP Server**: HTTP server providing Context7-compatible tools

### Data Flow

1. **Configuration Loading**: Parse and validate JSON configuration
2. **Repository Preparation**: Clone/update repositories based on configuration
3. **Content Indexing**: Run repomix with optimized arguments to extract and structure content
4. **Cache Storage**: Store indexed content in BadgerDB for fast retrieval
5. **MCP Serving**: Serve content through standardized MCP tools
6. **Search & Retrieval**: Process search queries and return ranked results

### Repomix Integration

The application uses repomix with optimized arguments for better AI consumption:

- `--compress`: Intelligent code extraction focusing on essential signatures
- `--remove-comments`: Removes comments to reduce noise and token count
- `--remove-empty-lines`: Eliminates empty lines for cleaner output
- `--style xml`: Structured XML format for reliable parsing

This results in significantly smaller, cleaner output that's optimized for AI analysis while preserving the essential code structure and functionality.

## Development

### Project Structure

```
repomix-mcp/
├── cmd/repomix-mcp/     # Main application entry point
├── internal/            # Internal packages
│   ├── cache/          # BadgerDB cache implementation
│   ├── config/         # Configuration management
│   ├── indexer/        # Repomix CLI integration
│   ├── mcp/            # MCP server implementation
│   ├── mcpclient/      # MCP client implementation (NEW)
│   │   ├── client.go   # Core client logic
│   │   ├── formatter.go # JSON syntax highlighting
│   │   └── args.go     # Argument parsing
│   ├── repository/     # Git repository management
│   └── search/         # Search engine
├── pkg/types/          # Shared types and interfaces
├── configs/            # Example configurations
├── examples/           # Usage examples and documentation (NEW)
└── docs/              # Documentation
```

### Building

```bash
go build -o repomix-mcp ./cmd/repomix-mcp
```

### Testing

```bash
go test ./...
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Troubleshooting

### Common Issues

**"repomix not found in PATH"**
- Install repomix CLI: `npm install -g repomix`
- Verify installation: `repomix --version`

**"unknown option '--include-empty-directories=false'"**
- This indicates an outdated repomix CLI version
- Update repomix: `npm update -g repomix`
- The application has been updated to work with current repomix versions

**Authentication failures**
- Verify SSH key permissions: `chmod 600 ~/.ssh/id_rsa`
- Test Git access: `git clone <your-repo-url>`
- Check token permissions for private repositories

**Cache permission errors**
- Ensure cache directory is writable
- Check disk space availability
- Verify cache path in configuration

**Large repository indexing fails**
- Increase `maxFileSize` in indexing configuration
- Add exclusion patterns for large binary files
- Consider splitting large repositories

### Debug Mode

Enable debug logging:

```json
{
  "server": {
    "logLevel": "debug"
  }
}
```

### Health Check

Check server health:

```bash
curl http://localhost:8080/mcp/health
```

### MCP Client Issues

**JSON colors not displaying correctly**
- Terminal may not support ANSI colors
- Use raw format: `./repomix-mcp client --mcp-list --format raw`
- Or table format: `./repomix-mcp client --mcp-list --format table`

**Client connection fails**
- Verify server is running: `curl http://127.0.0.1:8080/health`
- Check firewall and port settings
- Try different server address: `--mcp-srv 127.0.0.1:9080`

**Tool execution errors**
- Use verbose mode for detailed debugging: `--verbose`
- Check argument format: `--mcp-args="key=value,key2=value2"`
- Verify tool name with: `--mcp-list`

**Auto-content not appearing**
- Feature only works with single repository matches
- Multiple matches will show list instead of content
- Use `get-library-docs` for specific repository content

**Performance issues with large repositories**
- Reduce token limit: `--mcp-args="libraryName=repo,tokens=5000"`
- Use `includeNonExported=false` for faster processing (default)
- Consider excluding large files in server configuration

## 📄 License

Licensed under the Apache License 2.0. See [LICENSE](LICENSE) for details.

## Support

- Create an issue for bug reports
- Submit feature requests through GitHub issues
- Check existing issues before creating new ones