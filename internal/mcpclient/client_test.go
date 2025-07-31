// ************************************************************************************************
// Package mcpclient - Unit tests for MCP client functionality.
// This file provides comprehensive testing for the MCP client implementation.
package mcpclient

import (
	"fmt"
	"testing"

	"repomix-mcp/pkg/types"
)

// ************************************************************************************************
// Test NewClient creation
func TestNewClient(t *testing.T) {
	tests := []struct {
		name          string
		serverAddress string
		expectError   bool
	}{
		{
			name:          "Valid HTTP address",
			serverAddress: "127.0.0.1:8080",
			expectError:   false,
		},
		{
			name:          "Valid HTTPS address",
			serverAddress: "https://server.com:443",
			expectError:   false,
		},
		{
			name:          "Empty address",
			serverAddress: "",
			expectError:   true,
		},
		{
			name:          "Invalid URL",
			serverAddress: ":/invalid-url",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.serverAddress)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for address %s, but got none", tt.serverAddress)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for address %s: %v", tt.serverAddress, err)
				return
			}
			
			if client == nil {
				t.Error("Expected client instance, got nil")
			}
		})
	}
}

// ************************************************************************************************
// Test argument parsing
func TestParseArguments(t *testing.T) {
	tests := []struct {
		name        string
		argsString  string
		expected    map[string]interface{}
		expectError bool
	}{
		{
			name:       "Empty string",
			argsString: "",
			expected:   map[string]interface{}{},
		},
		{
			name:       "Single string argument",
			argsString: "libraryName=golang",
			expected: map[string]interface{}{
				"libraryName": "golang",
			},
		},
		{
			name:       "Multiple arguments with types",
			argsString: "libraryName=golang,tokens=5000,includeNonExported=true",
			expected: map[string]interface{}{
				"libraryName":        "golang",
				"tokens":             5000,
				"includeNonExported": true,
			},
		},
		{
			name:       "Boolean false",
			argsString: "verbose=false",
			expected: map[string]interface{}{
				"verbose": false,
			},
		},
		{
			name:       "Float number",
			argsString: "ratio=3.14",
			expected: map[string]interface{}{
				"ratio": 3.14,
			},
		},
		{
			name:       "Quoted string",
			argsString: `name="test value"`,
			expected: map[string]interface{}{
				"name": "test value",
			},
		},
		{
			name:        "Missing equals sign",
			argsString:  "invalidarg",
			expectError: true,
		},
		{
			name:        "Empty key",
			argsString:  "=value",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseArguments(tt.argsString)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for args %s, but got none", tt.argsString)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for args %s: %v", tt.argsString, err)
				return
			}
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d arguments, got %d", len(tt.expected), len(result))
				return
			}
			
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key %s not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected %s=%v, got %s=%v", key, expectedValue, key, actualValue)
				}
			}
		})
	}
}

// ************************************************************************************************
// Test format arguments
func TestFormatArguments(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		expected string
	}{
		{
			name:     "Empty map",
			args:     map[string]interface{}{},
			expected: "",
		},
		{
			name: "Single argument",
			args: map[string]interface{}{
				"libraryName": "golang",
			},
			expected: "libraryName=golang",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatArguments(tt.args)
			
			if tt.expected == "" && result != "" {
				t.Errorf("Expected empty string, got %s", result)
			} else if tt.expected != "" && result == "" {
				t.Errorf("Expected %s, got empty string", tt.expected)
			}
			// Note: For multiple arguments, order is not guaranteed in maps
		})
	}
}

// ************************************************************************************************
// Test mock client functionality
func TestMockClient(t *testing.T) {
	client := NewMockClient("127.0.0.1:8080")
	
	// Test initial state
	if client.ConnectCalled {
		t.Error("ConnectCalled should be false initially")
	}
	
	if client.ListToolsCalled {
		t.Error("ListToolsCalled should be false initially")  
	}
	
	// Test Connect
	err := client.Connect()
	if err != nil {
		t.Errorf("Connect should not error: %v", err)
	}
	
	if !client.ConnectCalled {
		t.Error("ConnectCalled should be true after Connect()")
	}
	
	// Set up mock tools
	mockTools := CreateMockTools()
	client.SetMockTools(mockTools)
	
	// Test ListTools
	tools, err := client.ListTools()
	if err != nil {
		t.Errorf("ListTools should not error: %v", err)
	}
	
	if !client.ListToolsCalled {
		t.Error("ListToolsCalled should be true after ListTools()")
	}
	
	if len(tools) != len(mockTools) {
		t.Errorf("Expected %d tools, got %d", len(mockTools), len(tools))
	}
	
	// Test CallTool
	args := map[string]interface{}{
		"libraryName": "golang",
	}
	
	result, err := client.CallTool("resolve-library-id", args)
	if err != nil {
		t.Errorf("CallTool should not error: %v", err)
	}
	
	if result == nil {
		t.Error("CallTool result should not be nil")
	}
	
	if len(client.CallToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(client.CallToolCalls))
	}
	
	if client.CallToolCalls[0].ToolName != "resolve-library-id" {
		t.Errorf("Expected tool name 'resolve-library-id', got %s", client.CallToolCalls[0].ToolName)
	}
}

// ************************************************************************************************
// Test mock client error simulation
func TestMockClientErrors(t *testing.T) {
	client := NewMockClient("127.0.0.1:8080")
	
	// Test connect error
	expectedError := fmt.Errorf("connection failed")
	client.SetConnectError(expectedError)
	
	err := client.Connect()
	if err != expectedError {
		t.Errorf("Expected connect error %v, got %v", expectedError, err)
	}
	
	// Reset and test list tools error
	client.Reset()
	client.SetListToolsError(expectedError)
	
	// Need to connect first
	_ = client.Connect()
	
	_, err = client.ListTools()
	if err != expectedError {
		t.Errorf("Expected list tools error %v, got %v", expectedError, err)
	}
	
	// Reset and test call tool error
	client.Reset()
	client.SetCallToolError(expectedError)
	
	// Need to connect first
	_ = client.Connect()
	
	_, err = client.CallTool("test-tool", map[string]interface{}{})
	if err != expectedError {
		t.Errorf("Expected call tool error %v, got %v", expectedError, err)
	}
}

// ************************************************************************************************
// Test argument builder
func TestArgumentBuilder(t *testing.T) {
	builder := NewArgumentBuilder()
	
	args := builder.
		AddString("libraryName", "golang").
		AddInt("tokens", 5000).
		AddBool("includeNonExported", true).
		Build()
	
	expected := map[string]interface{}{
		"libraryName":        "golang",
		"tokens":             5000,
		"includeNonExported": true,
	}
	
	if len(args) != len(expected) {
		t.Errorf("Expected %d arguments, got %d", len(expected), len(args))
	}
	
	for key, expectedValue := range expected {
		if actualValue, exists := args[key]; !exists {
			t.Errorf("Expected key %s not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("Expected %s=%v, got %s=%v", key, expectedValue, key, actualValue)
		}
	}
	
	// Test clear
	builder.Clear()
	clearedArgs := builder.Build()
	
	if len(clearedArgs) != 0 {
		t.Errorf("Expected 0 arguments after clear, got %d", len(clearedArgs))
	}
}

// ************************************************************************************************
// Test validate required arguments
func TestValidateRequiredArguments(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]interface{}
		required    []string
		expectError bool
	}{
		{
			name: "All required present",
			args: map[string]interface{}{
				"libraryName": "golang",
				"tokens":      5000,
			},
			required:    []string{"libraryName"},
			expectError: false,
		},
		{
			name: "Missing required argument",
			args: map[string]interface{}{
				"tokens": 5000,
			},
			required:    []string{"libraryName"},
			expectError: true,
		},
		{
			name: "Multiple missing arguments",
			args: map[string]interface{}{
				"extra": "value",
			},
			required:    []string{"libraryName", "tokens"},
			expectError: true,
		},
		{
			name:        "No required arguments",
			args:        map[string]interface{}{"any": "value"},
			required:    []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequiredArguments(tt.args, tt.required)
			
			if tt.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// ************************************************************************************************
// Test tools list formatting
func TestFormatToolsList(t *testing.T) {
	tools := CreateMockTools()
	
	// Test JSON format
	jsonOutput, err := FormatToolsList(tools, OutputFormatJSON)
	if err != nil {
		t.Errorf("JSON formatting should not error: %v", err)
	}
	
	if jsonOutput == "" {
		t.Error("JSON output should not be empty")
	}
	
	// Test table format
	tableOutput, err := FormatToolsList(tools, OutputFormatTable)
	if err != nil {
		t.Errorf("Table formatting should not error: %v", err)
	}
	
	if tableOutput == "" {
		t.Error("Table output should not be empty")
	}
	
	// Test raw format
	rawOutput, err := FormatToolsList(tools, OutputFormatRaw)
	if err != nil {
		t.Errorf("Raw formatting should not error: %v", err)
	}
	
	if rawOutput == "" {
		t.Error("Raw output should not be empty")
	}
	
	// Test invalid format
	_, err = FormatToolsList(tools, "invalid")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

// ************************************************************************************************
// Test tool result formatting
func TestFormatToolResult(t *testing.T) {
	result := &types.MCPToolCallResult{
		Content: []types.MCPContent{
			{
				Type: "text",
				Text: "Test result content",
			},
		},
		IsError: false,
	}
	
	// Test JSON format
	jsonOutput, err := FormatToolResult("test-tool", result, OutputFormatJSON)
	if err != nil {
		t.Errorf("JSON formatting should not error: %v", err)
	}
	
	if jsonOutput == "" {
		t.Error("JSON output should not be empty")
	}
	
	// Test table format
	tableOutput, err := FormatToolResult("test-tool", result, OutputFormatTable)
	if err != nil {
		t.Errorf("Table formatting should not error: %v", err)
	}
	
	if tableOutput == "" {
		t.Error("Table output should not be empty")
	}
	
	// Test raw format
	rawOutput, err := FormatToolResult("test-tool", result, OutputFormatRaw)
	if err != nil {
		t.Errorf("Raw formatting should not error: %v", err)
	}
	
	if rawOutput == "" {
		t.Error("Raw output should not be empty")
	}
}