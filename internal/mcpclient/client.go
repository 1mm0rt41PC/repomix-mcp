// ************************************************************************************************
// Package mcpclient provides MCP (Model Context Protocol) client implementation for the repomix-mcp application.
// It implements a JSON-RPC 2.0 compliant MCP client that can connect to MCP servers and execute tools.
package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"repomix-mcp/pkg/types"
)

// ************************************************************************************************
// Client implements the MCP client functionality.
// It provides JSON-RPC 2.0 compliant communication with MCP servers.
type Client struct {
	serverAddress string
	httpClient    *http.Client
	verbose       bool
	initialized   bool
	sessionID     string
}

// ************************************************************************************************
// NewClient creates a new MCP client instance.
//
// Parameters:
//   - serverAddress: The MCP server address (e.g., "127.0.0.1:8080" or "https://server.com:443")
//
// Returns:
//   - *Client: The MCP client instance.
//   - error: An error if initialization fails.
//
// Example usage:
//
//	client, err := NewClient("127.0.0.1:8080")
//	if err != nil {
//		return fmt.Errorf("failed to create client: %w", err)
//	}
func NewClient(serverAddress string) (*Client, error) {
	if serverAddress == "" {
		return nil, fmt.Errorf("server address cannot be empty")
	}

	// Normalize server address
	if !strings.HasPrefix(serverAddress, "http://") && !strings.HasPrefix(serverAddress, "https://") {
		serverAddress = "http://" + serverAddress
	}

	// Validate URL
	_, err := url.Parse(serverAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid server address: %w", err)
	}

	client := &Client{
		serverAddress: serverAddress,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		verbose:     false,
		initialized: false,
	}

	return client, nil
}

// ************************************************************************************************
// SetVerbose sets the verbose logging mode for the client.
func (c *Client) SetVerbose(verbose bool) {
	c.verbose = verbose
}

// ************************************************************************************************
// Connect establishes a connection to the MCP server and initializes the session.
//
// Returns:
//   - error: An error if connection or initialization fails.
//
// Example usage:
//
//	err := client.Connect()
//	if err != nil {
//		return fmt.Errorf("failed to connect: %w", err)
//	}
func (c *Client) Connect() error {
	if c.verbose {
		log.Printf("Connecting to MCP server: %s", c.serverAddress)
	}

	// Test connection with a ping
	if err := c.ping(); err != nil {
		return fmt.Errorf("failed to ping server: %w", err)
	}

	// Initialize MCP session
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize MCP session: %w", err)
	}

	// Send initialized notification
	if err := c.sendInitialized(); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	c.initialized = true
	if c.verbose {
		log.Printf("Successfully connected and initialized MCP session")
	}

	return nil
}

// ************************************************************************************************
// ListTools retrieves the list of available tools from the MCP server.
//
// Returns:
//   - []types.MCPTool: List of available tools.
//   - error: An error if tool listing fails.
//
// Example usage:
//
//	tools, err := client.ListTools()
//	if err != nil {
//		return fmt.Errorf("failed to list tools: %w", err)
//	}
func (c *Client) ListTools() ([]types.MCPTool, error) {
	if !c.initialized {
		if err := c.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	if c.verbose {
		log.Printf("Requesting tools list from MCP server")
	}

	// Create tools/list request
	request := types.JSONRPCRequest{
		JsonRPC: "2.0",
		ID:      c.generateRequestID(),
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	// Send request
	response, err := c.sendJSONRPCRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send tools/list request: %w", err)
	}

	// Parse response
	if response.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s (code: %d)", response.Error.Message, response.Error.Code)
	}

	// Convert result to MCPToolsListResult
	var toolsResult types.MCPToolsListResult
	if err := c.convertResult(response.Result, &toolsResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools list response: %w", err)
	}

	if c.verbose {
		log.Printf("Retrieved %d tools from MCP server", len(toolsResult.Tools))
	}

	return toolsResult.Tools, nil
}

// ************************************************************************************************
// CallTool executes a specific tool on the MCP server.
//
// Parameters:
//   - toolName: The name of the tool to execute.
//   - arguments: The arguments to pass to the tool.
//
// Returns:
//   - *types.MCPToolCallResult: The tool execution result.
//   - error: An error if tool execution fails.
//
// Example usage:
//
//	args := map[string]interface{}{
//		"libraryName": "golang",
//	}
//	result, err := client.CallTool("resolve-library-id", args)
func (c *Client) CallTool(toolName string, arguments map[string]interface{}) (*types.MCPToolCallResult, error) {
	if !c.initialized {
		if err := c.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	if c.verbose {
		log.Printf("Calling tool '%s' with arguments: %+v", toolName, arguments)
	}

	// Create tools/call request
	params := types.MCPToolCallParams{
		Name:      toolName,
		Arguments: arguments,
	}

	request := types.JSONRPCRequest{
		JsonRPC: "2.0",
		ID:      c.generateRequestID(),
		Method:  "tools/call",
		Params:  params,
	}

	// Send request
	response, err := c.sendJSONRPCRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send tools/call request: %w", err)
	}

	// Parse response
	if response.Error != nil {
		return nil, fmt.Errorf("tools/call error: %s (code: %d)", response.Error.Message, response.Error.Code)
	}

	// Convert result to MCPToolCallResult
	var toolResult types.MCPToolCallResult
	if err := c.convertResult(response.Result, &toolResult); err != nil {
		return nil, fmt.Errorf("failed to parse tool call response: %w", err)
	}

	if c.verbose {
		log.Printf("Tool '%s' executed successfully, isError: %v", toolName, toolResult.IsError)
	}

	return &toolResult, nil
}

// ************************************************************************************************
// Close closes the client connection and cleans up resources.
func (c *Client) Close() error {
	if c.verbose && c.initialized {
		log.Printf("Closing MCP client connection")
	}
	
	c.initialized = false
	return nil
}

// ************************************************************************************************
// Private helper methods

// ping sends a ping request to test server connectivity.
func (c *Client) ping() error {
	request := types.JSONRPCRequest{
		JsonRPC: "2.0",
		ID:      c.generateRequestID(),
		Method:  "ping",
		Params:  map[string]interface{}{},
	}

	response, err := c.sendJSONRPCRequest(request)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("ping error: %s (code: %d)", response.Error.Message, response.Error.Code)
	}

	return nil
}

// initialize sends the MCP initialize request.
func (c *Client) initialize() error {
	initRequest := types.MCPInitializeRequest{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		ClientInfo: map[string]interface{}{
			"name":    "repomix-mcp-client",
			"version": "1.0.0",
		},
	}

	request := types.JSONRPCRequest{
		JsonRPC: "2.0",
		ID:      c.generateRequestID(),
		Method:  "initialize",
		Params:  initRequest,
	}

	response, err := c.sendJSONRPCRequest(request)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("initialize error: %s (code: %d)", response.Error.Message, response.Error.Code)
	}

	// Parse initialize result
	var initResult types.MCPInitializeResult
	if err := c.convertResult(response.Result, &initResult); err != nil {
		return fmt.Errorf("failed to parse initialize response: %w", err)
	}

	if c.verbose {
		log.Printf("MCP session initialized with protocol version: %s", initResult.ProtocolVersion)
	}

	return nil
}

// sendInitialized sends the initialized notification.
func (c *Client) sendInitialized() error {
	notification := types.JSONRPCRequest{
		JsonRPC: "2.0",
		Method:  "notifications/initialized",
		Params:  map[string]interface{}{},
	}

	// For notifications, we don't expect a response
	return c.sendJSONRPCNotification(notification)
}

// sendJSONRPCRequest sends a JSON-RPC request and returns the response.
func (c *Client) sendJSONRPCRequest(request types.JSONRPCRequest) (*types.JSONRPCResponse, error) {
	// Marshal request
	reqData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if c.verbose {
		log.Printf("Sending JSON-RPC request: %s", string(reqData))
	}

	// Create HTTP request
	url := c.serverAddress + "/mcp"
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("MCP-Protocol-Version", "2024-11-05")

	// Send HTTP request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.verbose {
		log.Printf("Received JSON-RPC response: %s", string(respData))
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Parse JSON-RPC response
	var jsonRPCResp types.JSONRPCResponse
	if err := json.Unmarshal(respData, &jsonRPCResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON-RPC response: %w", err)
	}

	return &jsonRPCResp, nil
}

// sendJSONRPCNotification sends a JSON-RPC notification (no response expected).
func (c *Client) sendJSONRPCNotification(notification types.JSONRPCRequest) error {
	// Marshal notification
	reqData, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if c.verbose {
		log.Printf("Sending JSON-RPC notification: %s", string(reqData))
	}

	// Create HTTP request
	url := c.serverAddress + "/mcp"
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("MCP-Protocol-Version", "2024-11-05")

	// Send HTTP request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// For notifications, we just check that we got a success status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("HTTP error for notification: %d %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// convertResult converts an interface{} result to a target struct.
func (c *Client) convertResult(result interface{}, target interface{}) error {
	// Convert through JSON marshaling/unmarshaling
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target: %w", err)
	}

	return nil
}

// generateRequestID generates a unique request ID.
func (c *Client) generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}