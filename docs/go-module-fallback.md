# Go Module Documentation Fallback

This document describes the Go module documentation fallback feature added to repomix-mcp, which enables automatic retrieval of Go module documentation when modules are not found in the local cache.

## Overview

The Go module fallback feature enhances the existing MCP tools (`resolve-library-id` and `get-library-docs`) with the ability to automatically fetch and cache documentation for Go modules using the `go doc` command when they are not available in the local repository cache.

## How It Works

### Workflow

1. **Library Resolution**: When `resolve-library-id` is called with a library name:
   - First checks existing repositories in cache and memory
   - If no match found, checks if the library name looks like a Go module path
   - If it matches Go module patterns, attempts to retrieve documentation
   - Creates a synthetic repository with ID format `gomod:<module-path>`
   - Returns the synthetic repository ID

2. **Documentation Retrieval**: When `get-library-docs` is called:
   - Detects if repository ID has `gomod:` prefix
   - If yes, extracts module path and retrieves documentation
   - If not cached, performs the temporary directory workflow:
     - Creates temporary directory
     - Runs `go mod init test`
     - Runs `go get <module-path>`
     - Runs `go mod vendor` (optional)
     - Executes `go doc <module-path>` and `go doc -all <module-path>`
     - Parses and formats the output
   - Caches results for future use
   - Returns formatted documentation

### Go Module Path Detection

The system recognizes the following patterns as Go module paths:

- **Standard library**: `fmt`, `net/http`, `encoding/json`, etc.
- **External modules**: `github.com/user/repo`, `golang.org/x/crypto`, etc.
- **Domain-based modules**: `example.com/path/to/module`

## Configuration

Add the following section to your `config.json`:

```json
{
  "goModule": {
    "enabled": true,
    "tempDirBase": "~/.repomix-mcp/godoc-temp",
    "cacheTimeout": "24h",
    "commandTimeout": "60s",
    "maxRetries": 3,
    "maxConcurrent": 5
  }
}
```

### Configuration Options

- **`enabled`**: Enable/disable Go module fallback (default: `false`)
- **`tempDirBase`**: Base directory for temporary Go modules (default: system temp + `/repomix-mcp-godoc`)
- **`cacheTimeout`**: How long to cache Go module docs (default: `24h`)
- **`commandTimeout`**: Timeout for individual Go commands (default: `60s`)
- **`maxRetries`**: Maximum retries for failed commands (default: `3`)
- **`maxConcurrent`**: Maximum concurrent Go operations (default: `5`)

## Examples

### Using with MCP Tools

1. **Resolve a Go module**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "resolve-library-id",
    "arguments": {
      "libraryName": "github.com/gin-gonic/gin"
    }
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "gomod:github.com/gin-gonic/gin"
      }
    ],
    "isError": false
  }
}
```

2. **Get documentation**:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get-library-docs",
    "arguments": {
      "context7CompatibleLibraryID": "gomod:github.com/gin-gonic/gin",
      "tokens": 10000
    }
  }
}
```

### Supported Go Module Types

- **Standard Library**: `fmt`, `os`, `net/http`, `encoding/json`
- **GitHub Modules**: `github.com/gin-gonic/gin`, `github.com/gorilla/mux`
- **Go Extended**: `golang.org/x/crypto/bcrypt`, `golang.org/x/net/html`
- **Other Domains**: `go.uber.org/zap`, `google.golang.org/grpc`

## Generated Documentation Structure

When a Go module is processed, the system creates a synthetic repository with the following files:

- **`go-doc.md`**: Basic documentation from `go doc`
- **`go-doc-all.md`**: Comprehensive documentation from `go doc -all`
- **`packages.txt`**: List of packages in the module (if available)
- **`example-*.go`**: Code examples (if found)

### Sample Repository Structure

```
gomod:github.com/gin-gonic/gin/
├── go-doc.md              # Basic documentation
├── go-doc-all.md          # Comprehensive documentation
├── packages.txt           # Package list
└── example-simple.go      # Code examples
```

## Error Handling

The system includes comprehensive error handling:

### Graceful Fallbacks

- If Go command is not available → Falls back to "not found" error
- If module doesn't exist → Clear error message about module availability
- If network issues → Retry mechanism with exponential backoff
- If timeout → Clean termination of hanging processes

### Security Measures

- **Input Validation**: Strict validation of module paths to prevent injection
- **Sandbox Execution**: Go commands run in isolated temporary directories
- **Resource Limits**: CPU and memory limits for Go operations
- **Path Traversal Protection**: Temp directories don't escape designated areas

## Performance Considerations

### Optimization Strategies

1. **Caching**: Both successful and failed attempts are cached
2. **Concurrent Processing**: Multiple Go doc requests can run in parallel
3. **Smart Cleanup**: Automatic cleanup of temporary directories
4. **Resource Management**: Limited concurrent operations prevent system overload

### Cache Strategy

- **TTL-based Expiration**: Configurable cache timeout
- **Synthetic Repository IDs**: Use `gomod:` prefix for easy identification
- **BadgerDB Integration**: Leverages existing cache infrastructure
- **Metadata Preservation**: Stores module version, Go version, and timestamps

## Testing

### Manual Testing

Use the provided test script:

```bash
python test_gomodule.py
```

This script tests various scenarios including:
- Standard library modules
- Popular external modules
- Invalid modules (error handling)

### Test Configuration

Use `test-gomodule-config.json` for testing:

```json
{
  "goModule": {
    "enabled": true,
    "tempDirBase": "~/.repomix-mcp-test/godoc-temp",
    "cacheTimeout": "1h",
    "commandTimeout": "30s",
    "maxRetries": 2,
    "maxConcurrent": 3
  }
}
```

## Troubleshooting

### Common Issues

1. **Go command not found**
   - Ensure Go is installed and in PATH
   - Check with `go version`

2. **Module not found**
   - Verify module path is correct
   - Check if module exists on pkg.go.dev

3. **Network timeouts**
   - Increase `commandTimeout` in configuration
   - Check internet connectivity

4. **Permission errors**
   - Ensure temp directory is writable
   - Check file system permissions

### Debug Mode

Enable debug logging by setting `logLevel` to `debug` in server configuration:

```json
{
  "server": {
    "logLevel": "debug"
  }
}
```

## Integration with AI Tools

The Go module fallback integrates seamlessly with existing AI workflows:

1. **Context7 Compatibility**: Uses same tool names and parameters
2. **Standard Response Format**: Returns documentation in expected format
3. **Caching Consistency**: Cached results work with existing cache tools
4. **Error Handling**: Provides clear error messages for AI consumption

## Future Enhancements

Potential improvements for future versions:

- **Package-level Documentation**: Support for specific package documentation
- **Version-specific Docs**: Retrieve docs for specific module versions
- **Example Extraction**: Better parsing of code examples
- **Parallel Processing**: Concurrent documentation retrieval
- **Smart Caching**: Cache invalidation based on module updates

---

This feature significantly enhances repomix-mcp's capability to provide comprehensive documentation for Go projects, making it a more complete solution for AI-assisted Go development.