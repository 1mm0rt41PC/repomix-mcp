# MCP Client Usage Examples

This document provides comprehensive examples of using the `repomix-mcp client` command to interact with MCP servers.

## Overview

The MCP client allows you to connect to Model Context Protocol (MCP) servers and execute tools remotely. It supports:

- **Server Discovery**: Connect to MCP servers via HTTP/HTTPS
- **Tool Listing**: Discover available tools with their schemas
- **Tool Execution**: Execute tools with parsed arguments
- **Multiple Output Formats**: JSON, table, and raw text formats

## Basic Usage

### Connect and List Available Tools

```bash
# Connect to local MCP server and list tools
repomix-mcp client --mcp-srv 127.0.0.1:8080 --mcp-list

# Connect to remote HTTPS server
repomix-mcp client --mcp-srv https://mcp-server.example.com:443 --mcp-list

# List tools with table format for better readability
repomix-mcp client --mcp-srv 127.0.0.1:8080 --mcp-list --format table

# Verbose output for debugging connections
repomix-mcp client --mcp-srv 127.0.0.1:8080 --mcp-list --verbose
```

### Execute Tools

```bash
# Execute resolve-library-id tool
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use resolve-library-id \
  --mcp-args="libraryName=golang"

# Execute get-library-docs with multiple arguments
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use get-library-docs \
  --mcp-args="library-id=golang-project,tokens=5000,topic=authentication"

# Execute refresh tool to clear cache
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use refresh \
  --mcp-args="repositoryID=my-repo,force=true"

# Get README content
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use get-readme \
  --mcp-args="library-id=golang-project,format=markdown"
```

## Command Line Arguments

### Required Arguments

- `--mcp-srv`: MCP server address (default: `127.0.0.1:8080`)

### Action Arguments (choose one)

- `--mcp-list`: List available tools from the server
- `--mcp-use <tool-name>`: Execute a specific tool

### Optional Arguments

- `--mcp-args`: Tool arguments in `key=value,key2=value2` format
- `--format`: Output format (`json`, `table`, `raw`) - default: `json`
- `--verbose`: Show detailed connection and execution information

## Argument Format

The `--mcp-args` parameter accepts arguments in a comma-separated format with automatic type conversion:

### Supported Data Types

```bash
# String arguments
--mcp-args="libraryName=golang,description=A programming language"

# Integer arguments  
--mcp-args="tokens=5000,maxResults=100"

# Float arguments
--mcp-args="ratio=3.14,threshold=0.5"

# Boolean arguments
--mcp-args="includeNonExported=true,verbose=false"

# Quoted strings (for values with special characters)
--mcp-args='name="My Project",description="A project with spaces"'

# Escaped commas in values
--mcp-args="tags=go\\,programming\\,language,version=1.0"
```

### Common Tool Arguments

#### resolve-library-id
```bash
--mcp-args="libraryName=<library-name>"
```

#### get-library-docs
```bash
--mcp-args="library-id=<repo-id>,tokens=10000,topic=<optional-topic>,includeNonExported=false"
```

#### refresh
```bash
--mcp-args="repositoryID=<optional-repo-id>,force=false"
```

#### get-readme
```bash
--mcp-args="library-id=<repo-id>,format=markdown"
```

## Output Formats

### JSON Format (Default)

Provides structured output with complete metadata:

```json
{
  "tools": [
    {
      "name": "resolve-library-id",
      "description": "Resolves a general library name into a repository ID",
      "inputSchema": {
        "type": "object",
        "properties": {
          "libraryName": {
            "type": "string",
            "description": "The name of the library to search for"
          }
        },
        "required": ["libraryName"]
      }
    }
  ],
  "count": 1
}
```

### Table Format

Human-readable tabular output:

```
Available MCP Tools (4):

NAME                  DESCRIPTION                                        REQUIRED PARAMETERS
----                  -----------                                        -------------------
resolve-library-id    Resolves a general library name into a reposit... libraryName
get-library-docs      Fetches documentation for a repository using i... library-id
refresh               Force refresh global cache for all or specific... 
get-readme            Extract and return README content if it exists    library-id
```

### Raw Format

Simple text output:

```
Available Tools (4):
1. resolve-library-id
   Description: Resolves a general library name into a repository ID
   Required: libraryName

2. get-library-docs
   Description: Fetches documentation for a repository using its ID
   Required: library-id
```

## Complete Workflow Examples

### Discover and Use a Library

```bash
# Step 1: List available tools
repomix-mcp client --mcp-srv 127.0.0.1:8080 --mcp-list --format table

# Step 2: Resolve library name to repository ID
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use resolve-library-id \
  --mcp-args="libraryName=golang" \
  --format raw

# Step 3: Get documentation using the resolved ID
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use get-library-docs \
  --mcp-args="library-id=golang-project,tokens=15000,topic=concurrency" \
  --format raw
```

### Cache Management

```bash
# Refresh all repository caches
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use refresh \
  --mcp-args="force=true"

# Refresh specific repository
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use refresh \
  --mcp-args="repositoryID=golang-project,force=true"
```

### Documentation Retrieval

```bash
# Get README in markdown format
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use get-readme \
  --mcp-args="library-id=golang-project,format=markdown" \
  --format raw

# Get README as plain text
repomix-mcp client --mcp-srv 127.0.0.1:8080 \
  --mcp-use get-readme \
  --mcp-args="library-id=golang-project,format=text" \
  --format raw
```

## Error Handling

The client provides detailed error messages for common issues:

### Connection Errors
```bash
# Server unreachable
$ repomix-mcp client --mcp-srv 127.0.0.1:9999 --mcp-list
Error: failed to connect to MCP server: HTTP request failed: dial tcp 127.0.0.1:9999: connection refused
```

### Invalid Arguments
```bash
# Missing required argument
$ repomix-mcp client --mcp-use resolve-library-id --mcp-args=""
Error: failed to execute tool: tools/call error: missing required arguments: libraryName (code: -32602)

# Invalid argument format
$ repomix-mcp client --mcp-use resolve-library-id --mcp-args="invalid-format"
Error: failed to parse tool arguments: invalid argument format 'invalid-format': missing '=' separator
```

### Tool Execution Errors
```bash
# Unknown library
$ repomix-mcp client --mcp-use resolve-library-id --mcp-args="libraryName=nonexistent"
Error: tool execution failed
# (Tool result will contain error details)
```

## Advanced Usage

### Using with Shell Scripts

```bash
#!/bin/bash

# Function to resolve and get docs for a library
get_library_docs() {
    local library_name="$1"
    local server="${2:-127.0.0.1:8080}"
    
    echo "Resolving library: $library_name"
    
    # Resolve library ID
    repo_id=$(repomix-mcp client --mcp-srv "$server" \
        --mcp-use resolve-library-id \
        --mcp-args="libraryName=$library_name" \
        --format raw)
    
    if [ $? -ne 0 ]; then
        echo "Failed to resolve library: $library_name"
        return 1
    fi
    
    echo "Repository ID: $repo_id"
    echo "Fetching documentation..."
    
    # Get documentation
    repomix-mcp client --mcp-srv "$server" \
        --mcp-use get-library-docs \
        --mcp-args="library-id=$repo_id,tokens=20000" \
        --format raw
}

# Usage
get_library_docs "golang" "127.0.0.1:8080"
```

### JSON Processing with jq

```bash
# Extract tool names using jq
repomix-mcp client --mcp-srv 127.0.0.1:8080 --mcp-list --format json | \
  jq -r '.tools[].name'

# Get tool descriptions
repomix-mcp client --mcp-srv 127.0.0.1:8080 --mcp-list --format json | \
  jq -r '.tools[] | "\(.name): \(.description)"'

# Filter tools by required parameters
repomix-mcp client --mcp-srv 127.0.0.1:8080 --mcp-list --format json | \
  jq -r '.tools[] | select(.inputSchema.required | length > 0) | .name'
```

## Troubleshooting

### Enable Verbose Mode

For debugging connection and execution issues:

```bash
repomix-mcp client --mcp-srv 127.0.0.1:8080 --mcp-list --verbose
```

This will show:
- Connection establishment details
- JSON-RPC request/response messages
- Tool execution progress
- Detailed error information

### Common Issues

1. **Connection Timeout**: Increase server timeout or check network connectivity
2. **SSL Certificate Errors**: Use HTTP instead of HTTPS for local testing
3. **Invalid JSON-RPC**: Ensure server implements MCP protocol correctly
4. **Tool Not Found**: Verify tool name matches exactly (case-sensitive)
5. **Argument Type Mismatch**: Check tool schema for required argument types

### Testing Server Connectivity

```bash
# Test basic connectivity
curl -X POST http://127.0.0.1:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":"test","method":"ping","params":{}}'

# Test HTTPS with self-signed certificates
curl -k -X POST https://127.0.0.1:9443/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":"test","method":"ping","params":{}}'
```

## Integration Examples

### CI/CD Pipeline

```yaml
# GitHub Actions example
- name: Get Repository Documentation
  run: |
    repomix-mcp client --mcp-srv ${{ secrets.MCP_SERVER }} \
      --mcp-use get-library-docs \
      --mcp-args="library-id=${{ github.repository }},tokens=50000" \
      --format raw > docs/api-reference.md
```

### Development Workflow

```bash
# Add to your .bashrc or .zshrc
alias mcp-list='repomix-mcp client --mcp-list --format table'
alias mcp-docs='function _mcp_docs() { repomix-mcp client --mcp-use get-library-docs --mcp-args="library-id=$1,tokens=20000" --format raw; }; _mcp_docs'
alias mcp-resolve='function _mcp_resolve() { repomix-mcp client --mcp-use resolve-library-id --mcp-args="libraryName=$1" --format raw; }; _mcp_resolve'

# Usage:
# mcp-list
# mcp-resolve golang
# mcp-docs golang-project
```

This comprehensive guide covers all aspects of using the MCP client functionality integrated into the repomix-mcp CLI application.