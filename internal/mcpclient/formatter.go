// ************************************************************************************************
// Package mcpclient - Output formatting utilities for MCP client operations.
// This file handles formatting of MCP responses for display to users.
package mcpclient

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"repomix-mcp/pkg/types"
)

// ANSI color codes for JSON syntax highlighting
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

// ************************************************************************************************
// OutputFormat defines the supported output formats for MCP client results.
type OutputFormat string

const (
	// OutputFormatJSON formats output as pretty-printed JSON
	OutputFormatJSON OutputFormat = "json"
	
	// OutputFormatTable formats output as human-readable tables
	OutputFormatTable OutputFormat = "table"
	
	// OutputFormatRaw formats output as raw text
	OutputFormatRaw OutputFormat = "raw"
)

// ************************************************************************************************
// FormatToolsList formats a list of MCP tools according to the specified output format.
//
// Parameters:
//   - tools: List of MCP tools to format
//   - format: Output format (json, table, raw)
//
// Returns:
//   - string: Formatted output
//   - error: An error if formatting fails
//
// Example usage:
//
//	output, err := FormatToolsList(tools, OutputFormatJSON)
func FormatToolsList(tools []types.MCPTool, format OutputFormat) (string, error) {
	switch format {
	case OutputFormatJSON:
		return formatToolsListJSON(tools)
	case OutputFormatTable:
		return formatToolsListTable(tools)
	case OutputFormatRaw:
		return formatToolsListRaw(tools)
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}

// ************************************************************************************************
// FormatToolResult formats an MCP tool execution result according to the specified output format.
//
// Parameters:
//   - toolName: Name of the executed tool
//   - result: Tool execution result
//   - format: Output format (json, table, raw)
//
// Returns:
//   - string: Formatted output
//   - error: An error if formatting fails
//
// Example usage:
//
//	output, err := FormatToolResult("resolve-library-id", result, OutputFormatJSON)
func FormatToolResult(toolName string, result *types.MCPToolCallResult, format OutputFormat) (string, error) {
	switch format {
	case OutputFormatJSON:
		return formatToolResultJSON(toolName, result)
	case OutputFormatTable:
		return formatToolResultTable(toolName, result)
	case OutputFormatRaw:
		return formatToolResultRaw(result)
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}

// ************************************************************************************************
// Private formatting functions for tools list

// formatToolsListJSON formats tools list as pretty JSON with syntax highlighting.
func formatToolsListJSON(tools []types.MCPTool) (string, error) {
	output := map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal tools to JSON: %w", err)
	}

	// Apply JSON syntax highlighting
	highlighted := highlightJSON(string(data))
	return highlighted, nil
}

// formatToolsListTable formats tools list as a human-readable table.
func formatToolsListTable(tools []types.MCPTool) (string, error) {
	if len(tools) == 0 {
		return "No tools available.\n", nil
	}

	var output strings.Builder
	
	// Create a string builder that acts like tabwriter
	var tableBuilder strings.Builder
	w := tabwriter.NewWriter(&tableBuilder, 0, 0, 2, ' ', 0)
	
	// Write header
	fmt.Fprintf(w, "NAME\tDESCRIPTION\tREQUIRED PARAMETERS\n")
	fmt.Fprintf(w, "----\t-----------\t-------------------\n")

	// Write tools
	for _, tool := range tools {
		requiredParams := extractRequiredParams(tool.InputSchema)
		fmt.Fprintf(w, "%s\t%s\t%s\n", 
			tool.Name, 
			truncateString(tool.Description, 50), 
			strings.Join(requiredParams, ", "))
	}

	w.Flush()
	
	output.WriteString(fmt.Sprintf("Available MCP Tools (%d):\n\n", len(tools)))
	output.WriteString(tableBuilder.String())
	output.WriteString("\n")

	return output.String(), nil
}

// formatToolsListRaw formats tools list as raw text.
func formatToolsListRaw(tools []types.MCPTool) (string, error) {
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("Available Tools (%d):\n", len(tools)))
	for i, tool := range tools {
		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, tool.Name))
		output.WriteString(fmt.Sprintf("   Description: %s\n", tool.Description))
		
		requiredParams := extractRequiredParams(tool.InputSchema)
		if len(requiredParams) > 0 {
			output.WriteString(fmt.Sprintf("   Required: %s\n", strings.Join(requiredParams, ", ")))
		}
		output.WriteString("\n")
	}

	return output.String(), nil
}

// ************************************************************************************************
// Private formatting functions for tool results

// formatToolResultJSON formats tool result as pretty JSON with syntax highlighting.
func formatToolResultJSON(toolName string, result *types.MCPToolCallResult) (string, error) {
	output := map[string]interface{}{
		"tool":     toolName,
		"success":  !result.IsError,
		"result":   result,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool result to JSON: %w", err)
	}

	// Apply JSON syntax highlighting
	highlighted := highlightJSON(string(data))
	return highlighted, nil
}

// formatToolResultTable formats tool result as a human-readable table.
func formatToolResultTable(toolName string, result *types.MCPToolCallResult) (string, error) {
	var output strings.Builder
	
	// Header
	output.WriteString(fmt.Sprintf("Tool Execution Result: %s\n", toolName))
	output.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	// Status
	status := "SUCCESS"
	if result.IsError {
		status = "ERROR"
	}
	output.WriteString(fmt.Sprintf("Status: %s\n", status))
	output.WriteString(fmt.Sprintf("Content Items: %d\n\n", len(result.Content)))
	
	// Content
	if len(result.Content) > 0 {
		output.WriteString("Content:\n")
		output.WriteString(strings.Repeat("-", 20) + "\n")
		
		for i, content := range result.Content {
			if len(result.Content) > 1 {
				output.WriteString(fmt.Sprintf("Item %d (%s):\n", i+1, content.Type))
			}
			
			switch content.Type {
			case "text":
				output.WriteString(content.Text)
			default:
				output.WriteString(fmt.Sprintf("[%s content]", content.Type))
			}
			
			if i < len(result.Content)-1 {
				output.WriteString("\n" + strings.Repeat("-", 20) + "\n")
			}
		}
	}
	
	output.WriteString("\n")
	return output.String(), nil
}

// formatToolResultRaw formats tool result as raw text (just the content).
func formatToolResultRaw(result *types.MCPToolCallResult) (string, error) {
	var output strings.Builder
	
	for i, content := range result.Content {
		if content.Type == "text" {
			output.WriteString(content.Text)
		} else {
			output.WriteString(fmt.Sprintf("[%s content]", content.Type))
		}
		
		if i < len(result.Content)-1 {
			output.WriteString("\n")
		}
	}
	
	return output.String(), nil
}

// ************************************************************************************************
// Helper functions

// extractRequiredParams extracts required parameter names from JSON schema.
func extractRequiredParams(schema map[string]interface{}) []string {
	required, ok := schema["required"]
	if !ok {
		return []string{}
	}
	
	requiredList, ok := required.([]interface{})
	if !ok {
		return []string{}
	}
	
	var params []string
	for _, param := range requiredList {
		if paramStr, ok := param.(string); ok {
			params = append(params, paramStr)
		}
	}
	
	return params
}

// truncateString truncates a string to the specified length with ellipsis.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	
	if maxLen <= 3 {
		return s[:maxLen]
	}
	
	return s[:maxLen-3] + "..."
}

// ************************************************************************************************
// FormatError formats an error message for display.
func FormatError(err error, verbose bool) string {
	if !verbose {
		return fmt.Sprintf("Error: %s", err.Error())
	}
	
	var output strings.Builder
	output.WriteString("ERROR DETAILS:\n")
	output.WriteString(strings.Repeat("=", 50) + "\n")
	output.WriteString(fmt.Sprintf("Message: %s\n", err.Error()))
	output.WriteString(fmt.Sprintf("Type: %T\n", err))
	
	return output.String()
}

// ************************************************************************************************
// JSON syntax highlighting functions

// highlightJSON applies basic ANSI color highlighting to JSON text
func highlightJSON(jsonStr string) string {
	// Parse JSON character by character to avoid interference
	result := ""
	inString := false
	escaped := false
	i := 0
	chars := []rune(jsonStr)
	
	for i < len(chars) {
		char := chars[i]
		
		if char == '"' && !escaped {
			if !inString {
				// Starting a string - check if it's a key
				inString = true
				// Look ahead to see if this is a key (followed by :)
				j := i + 1
				for j < len(chars) && chars[j] != '"' {
					if chars[j] == '\\' {
						j += 2 // Skip escaped character
					} else {
						j++
					}
				}
				if j < len(chars) {
					j++ // Skip closing quote
					for j < len(chars) && (chars[j] == ' ' || chars[j] == '\t') {
						j++ // Skip whitespace
					}
					if j < len(chars) && chars[j] == ':' {
						result += colorPurple + string(char)
					} else {
						result += colorCyan + string(char)
					}
				} else {
					result += colorCyan + string(char)
				}
			} else {
				// Ending a string
				result += string(char) + colorReset
				inString = false
			}
		} else if inString {
			result += string(char)
		} else {
			// Outside of strings - handle other elements
			switch char {
			case '{', '}', '[', ']':
				result += colorYellow + string(char) + colorReset
			case ':':
				result += colorWhite + string(char) + colorReset
			case ',':
				result += colorWhite + string(char) + colorReset
			default:
				// Check for keywords and numbers
				if char >= '0' && char <= '9' || char == '-' || char == '.' {
					// Start of a number
					numStart := i
					for i < len(chars) && (chars[i] >= '0' && chars[i] <= '9' || chars[i] == '-' || chars[i] == '.' || chars[i] == 'e' || chars[i] == 'E' || chars[i] == '+') {
						i++
					}
					number := string(chars[numStart:i])
					result += colorBlue + number + colorReset
					i-- // Back up one since the loop will increment
				} else if char == 't' && i+3 < len(chars) && string(chars[i:i+4]) == "true" {
					result += colorGreen + "true" + colorReset
					i += 3 // Skip ahead
				} else if char == 'f' && i+4 < len(chars) && string(chars[i:i+5]) == "false" {
					result += colorRed + "false" + colorReset
					i += 4 // Skip ahead
				} else if char == 'n' && i+3 < len(chars) && string(chars[i:i+4]) == "null" {
					result += colorPurple + "null" + colorReset
					i += 3 // Skip ahead
				} else {
					result += string(char)
				}
			}
		}
		
		// Handle escape sequences
		if char == '\\' && inString {
			escaped = !escaped
		} else {
			escaped = false
		}
		
		i++
	}
	
	return result
}

// ************************************************************************************************
// FormatConnectionInfo formats connection information for display.
func FormatConnectionInfo(serverAddress string, connected bool) string {
	var output strings.Builder
	
	output.WriteString("MCP CLIENT CONNECTION:\n")
	output.WriteString(strings.Repeat("-", 30) + "\n")
	output.WriteString(fmt.Sprintf("Server: %s\n", serverAddress))
	
	status := "DISCONNECTED"
	if connected {
		status = "CONNECTED"
	}
	output.WriteString(fmt.Sprintf("Status: %s\n", status))
	
	return output.String()
}