// ************************************************************************************************
// Package mcpclient - Mock implementations for testing MCP client functionality.
// This file provides mock MCP client and server implementations for unit testing.
package mcpclient

import (
	"fmt"
	"repomix-mcp/pkg/types"
)

// ************************************************************************************************
// MockClient provides a mock implementation of MCP client for testing.
type MockClient struct {
	serverAddress string
	verbose       bool
	initialized   bool
	
	// Mock responses
	mockTools       []types.MCPTool
	mockToolResults map[string]*types.MCPToolCallResult
	
	// Error simulation
	connectError    error
	listToolsError  error
	callToolError   error
	
	// Call tracking
	ConnectCalled   bool
	ListToolsCalled bool
	CallToolCalls   []MockToolCall
}

// ************************************************************************************************
// MockToolCall represents a recorded tool call for testing.
type MockToolCall struct {
	ToolName  string
	Arguments map[string]interface{}
}

// ************************************************************************************************
// NewMockClient creates a new mock MCP client.
func NewMockClient(serverAddress string) *MockClient {
	return &MockClient{
		serverAddress:   serverAddress,
		mockToolResults: make(map[string]*types.MCPToolCallResult),
		CallToolCalls:   make([]MockToolCall, 0),
	}
}

// ************************************************************************************************
// SetVerbose sets the verbose logging mode for the mock client.
func (m *MockClient) SetVerbose(verbose bool) {
	m.verbose = verbose
}

// ************************************************************************************************
// Connect simulates connecting to an MCP server.
func (m *MockClient) Connect() error {
	m.ConnectCalled = true
	
	if m.connectError != nil {
		return m.connectError
	}
	
	m.initialized = true
	return nil
}

// ************************************************************************************************
// ListTools returns the mock tools list.
func (m *MockClient) ListTools() ([]types.MCPTool, error) {
	m.ListToolsCalled = true
	
	if m.listToolsError != nil {
		return nil, m.listToolsError
	}
	
	if !m.initialized {
		return nil, fmt.Errorf("client not connected")
	}
	
	return m.mockTools, nil
}

// ************************************************************************************************
// CallTool simulates calling a tool and returns the mock result.
func (m *MockClient) CallTool(toolName string, arguments map[string]interface{}) (*types.MCPToolCallResult, error) {
	// Record the call
	m.CallToolCalls = append(m.CallToolCalls, MockToolCall{
		ToolName:  toolName,
		Arguments: arguments,
	})
	
	if m.callToolError != nil {
		return nil, m.callToolError
	}
	
	if !m.initialized {
		return nil, fmt.Errorf("client not connected")
	}
	
	// Return mock result if available
	if result, exists := m.mockToolResults[toolName]; exists {
		return result, nil
	}
	
	// Default success result
	return &types.MCPToolCallResult{
		Content: []types.MCPContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Mock result for tool: %s", toolName),
			},
		},
		IsError: false,
	}, nil
}

// ************************************************************************************************
// Close simulates closing the client connection.
func (m *MockClient) Close() error {
	m.initialized = false
	return nil
}

// ************************************************************************************************
// Mock configuration methods

// SetMockTools sets the tools that will be returned by ListTools.
func (m *MockClient) SetMockTools(tools []types.MCPTool) {
	m.mockTools = tools
}

// SetMockToolResult sets the result that will be returned for a specific tool.
func (m *MockClient) SetMockToolResult(toolName string, result *types.MCPToolCallResult) {
	m.mockToolResults[toolName] = result
}

// SetConnectError sets an error that will be returned by Connect.
func (m *MockClient) SetConnectError(err error) {
	m.connectError = err
}

// SetListToolsError sets an error that will be returned by ListTools.
func (m *MockClient) SetListToolsError(err error) {
	m.listToolsError = err
}

// SetCallToolError sets an error that will be returned by CallTool.
func (m *MockClient) SetCallToolError(err error) {
	m.callToolError = err
}

// ************************************************************************************************
// Test helper methods

// Reset clears all mock state and call tracking.
func (m *MockClient) Reset() {
	m.initialized = false
	m.mockTools = nil
	m.mockToolResults = make(map[string]*types.MCPToolCallResult)
	m.connectError = nil
	m.listToolsError = nil
	m.callToolError = nil
	m.ConnectCalled = false
	m.ListToolsCalled = false
	m.CallToolCalls = make([]MockToolCall, 0)
}

// GetCallCount returns the number of times a specific tool was called.
func (m *MockClient) GetCallCount(toolName string) int {
	count := 0
	for _, call := range m.CallToolCalls {
		if call.ToolName == toolName {
			count++
		}
	}
	return count
}

// GetLastCall returns the arguments from the last tool call, or nil if no calls were made.
func (m *MockClient) GetLastCall(toolName string) map[string]interface{} {
	for i := len(m.CallToolCalls) - 1; i >= 0; i-- {
		if m.CallToolCalls[i].ToolName == toolName {
			return m.CallToolCalls[i].Arguments
		}
	}
	return nil
}

// ************************************************************************************************
// CreateMockTools creates a set of standard mock tools for testing.
func CreateMockTools() []types.MCPTool {
	return []types.MCPTool{
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
					"library-id": map[string]interface{}{
						"type":        "string",
						"description": "Repository ID from resolve-library-id",
					},
					"tokens": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of tokens to return",
						"default":     10000,
					},
				},
				"required": []string{"library-id"},
			},
		},
		{
			Name:        "refresh",
			Description: "Force refresh global cache for all or specific repositories",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repositoryID": map[string]interface{}{
						"type":        "string",
						"description": "Target specific repository ID, empty for all repositories",
					},
					"force": map[string]interface{}{
						"type":        "boolean",
						"description": "Skip confirmation prompts",
						"default":     false,
					},
				},
				"required": []string{},
			},
		},
	}
}

// ************************************************************************************************
// CreateMockToolResults creates standard mock tool results for testing.
func CreateMockToolResults() map[string]*types.MCPToolCallResult {
	return map[string]*types.MCPToolCallResult{
		"resolve-library-id": {
			Content: []types.MCPContent{
				{
					Type: "text",
					Text: "golang-project",
				},
			},
			IsError: false,
		},
		"get-library-docs": {
			Content: []types.MCPContent{
				{
					Type: "text",
					Text: "# Repository: golang-project\n\nThis is mock documentation for the golang-project repository.\n\n## Files\n\n- main.go: Main application file\n- README.md: Project documentation",
				},
			},
			IsError: false,
		},
		"refresh": {
			Content: []types.MCPContent{
				{
					Type: "text",
					Text: "Successfully refreshed 5 repositories",
				},
			},
			IsError: false,
		},
	}
}