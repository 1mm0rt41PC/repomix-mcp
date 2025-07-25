# Go Module Documentation Debugging

This document explains how to use the new command logging functionality to debug issues with Go module documentation retrieval in repomix-mcp.

## Overview

The Go module documentation functionality in repomix-mcp has been enhanced with comprehensive command logging to help debug issues when retrieving documentation for specific modules like `golang.org/x/tools/godoc`.

## Command Logging Format

When verbose mode is enabled (`-v` or `--verbose`), all Go commands executed during module documentation retrieval are logged with the following format:

- `[CMD] command arguments...` - The exact command being executed
- `[CMD STDOUT] output...` - Standard output from the command
- `[CMD STDERR] error output...` - Standard error from the command (if any)

## How to Enable Verbose Logging

### Using the MCP Server

To enable verbose logging when running the MCP server:

```bash
./repomix-mcp serve -v
```

This will enable verbose logging for:
- Cache operations during serving
- Go module documentation retrieval commands
- All Go command executions (go version, go mod init, go get, go doc, etc.)

### Example Output

When attempting to retrieve documentation for `golang.org/x/tools/godoc` with verbose logging enabled, you will see:

```
2025/07/25 23:40:25 Starting Go module documentation retrieval for: golang.org/x/tools/godoc
2025/07/25 23:40:25 [CMD] go version
2025/07/25 23:40:25 [CMD STDOUT] go version go1.24.1 windows/amd64
2025/07/25 23:40:25 Go version: go version go1.24.1 windows/amd64
2025/07/25 23:40:25 Created temp directory: C:\Users\...\Temp\repomix-mcp-godoc\gomod-707171379
2025/07/25 23:40:25 Executing Go commands for module golang.org/x/tools/godoc in directory C:\Users\...\Temp\repomix-mcp-godoc\gomod-707171379
2025/07/25 23:40:25 Initializing Go module in C:\Users\...\Temp\repomix-mcp-godoc\gomod-707171379
2025/07/25 23:40:25 [CMD] go mod init temp-docs
2025/07/25 23:40:25 [CMD STDOUT] go: creating new go.mod: module temp-docs
2025/07/25 23:40:25 Getting module: golang.org/x/tools/godoc
2025/07/25 23:40:25 [CMD] go get golang.org/x/tools/godoc
2025/07/25 23:40:28 [CMD STDOUT] go: added github.com/yuin/goldmark v1.4.13
go: added golang.org/x/tools v0.35.0
2025/07/25 23:40:28 [CMD] go version
2025/07/25 23:40:28 [CMD STDOUT] go version go1.24.1 windows/amd64
2025/07/25 23:40:28 Running: go doc golang.org/x/tools/godoc
2025/07/25 23:40:28 [CMD] doc golang.org/x/tools/godoc
2025/07/25 23:40:28 [CMD STDERR]
2025/07/25 23:40:28 Direct go doc approach failed, trying alternatives...
2025/07/25 23:40:28 [CMD] doc golang.org/x/tools/godoc
2025/07/25 23:40:28 [CMD STDERR]
2025/07/25 23:40:28 [CMD] doc godoc
2025/07/25 23:40:28 [CMD STDERR]
```

## Debugging Common Issues

### Issue: "go doc" Commands Returning Empty Output

**Symptoms:**
- `[CMD] doc golang.org/x/tools/godoc` shows empty stderr
- Multiple alternative approaches are tried
- Final error: "all documentation extraction approaches failed"

**Analysis:**
The logging shows that the module was successfully downloaded (`go get` succeeded), but the `go doc` command cannot find the package. This typically happens when:

1. The module path doesn't correspond to an actual Go package
2. The package is internal or has been moved/deprecated
3. The package doesn't have exportable documentation

**Solution:**
Check if the module path is correct and if the package actually exists and has documentation.

### Issue: Module Not Found

**Symptoms:**
- `[CMD] go get module-path` shows stderr with "not found" errors
- Download fails before reaching documentation extraction

**Analysis:**
The module path is invalid or the module doesn't exist.

### Issue: Permission or Network Errors

**Symptoms:**
- `[CMD] go get module-path` shows network-related errors in stderr
- Commands timeout or fail with permission errors

**Analysis:**
Network connectivity issues or authentication problems with Go module proxies.

## Testing the Functionality

You can test the command logging functionality using the included unit tests:

```bash
# Run all godoc tests with verbose output
go test -v ./internal/godoc

# Run specifically the problematic module test
go test -v ./internal/godoc -run TestProblematicGoModule

# Run command logging format tests
go test -v ./internal/godoc -run TestCommandLoggingFormat
```

## Configuration

The Go module documentation functionality is controlled by the `goModule` section in your configuration:

```json
{
  "goModule": {
    "enabled": true,
    "tempDirBase": "~/.repomix-mcp-godoc",
    "cacheTimeout": "1h",
    "commandTimeout": "60s",
    "maxRetries": 3,
    "maxConcurrent": 2
  }
}
```

When verbose logging is enabled, you'll see detailed information about:
- Temporary directory creation and cleanup
- Each Go command execution with full output
- Alternative approaches when primary methods fail
- Cache operations and storage

## Integration with MCP Tools

The command logging integrates seamlessly with the MCP tools. When using the `resolve-library-id` or `get-library-docs` tools through an AI client, if verbose mode is enabled, you'll see the complete command execution flow in the server logs.

Example MCP tool request that would trigger Go module documentation retrieval:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "resolve-library-id",
    "arguments": {
      "libraryName": "golang.org/x/tools/godoc"
    }
  }
}
```

With verbose logging enabled, this request would show all the command execution details in the server logs, making it easy to debug any issues with the Go module documentation retrieval process.