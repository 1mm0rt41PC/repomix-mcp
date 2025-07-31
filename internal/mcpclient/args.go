// ************************************************************************************************
// Package mcpclient - Argument parsing utilities for MCP client operations.
// This file handles parsing of command-line arguments into MCP tool parameters.
package mcpclient

import (
	"fmt"
	"strconv"
	"strings"
)

// ************************************************************************************************
// ParseArguments parses command-line arguments string into a map suitable for MCP tool calls.
// It supports the format: "key=value,key2=value2,key3=value3"
// 
// The function performs automatic type conversion:
// - "true"/"false" -> boolean
// - Numeric strings -> numbers (int or float64)
// - Everything else -> string
//
// Parameters:
//   - argsString: The arguments string to parse
//
// Returns:
//   - map[string]interface{}: Parsed arguments map
//   - error: An error if parsing fails
//
// Example usage:
//
//	args, err := ParseArguments("libraryName=golang,tokens=5000,includeNonExported=true")
//	// Result: {"libraryName": "golang", "tokens": 5000, "includeNonExported": true}
func ParseArguments(argsString string) (map[string]interface{}, error) {
	if argsString == "" {
		return make(map[string]interface{}), nil
	}

	result := make(map[string]interface{})
	
	// Split by comma, but handle escaped commas
	pairs := splitArguments(argsString)
	
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Find the first '=' to split key and value
		eqIndex := strings.Index(pair, "=")
		if eqIndex == -1 {
			return nil, fmt.Errorf("invalid argument format '%s': missing '=' separator", pair)
		}

		key := strings.TrimSpace(pair[:eqIndex])
		value := strings.TrimSpace(pair[eqIndex+1:])

		if key == "" {
			return nil, fmt.Errorf("invalid argument: empty key in '%s'", pair)
		}

		// Convert value to appropriate type
		convertedValue := convertValue(value)
		result[key] = convertedValue
	}

	return result, nil
}

// ************************************************************************************************
// FormatArguments formats a map of arguments back into the command-line string format.
// This is useful for displaying parsed arguments or debugging.
//
// Parameters:
//   - args: The arguments map to format
//
// Returns:
//   - string: Formatted arguments string
//
// Example usage:
//
//	formatted := FormatArguments(map[string]interface{}{
//		"libraryName": "golang",
//		"tokens": 5000,
//		"includeNonExported": true,
//	})
//	// Result: "libraryName=golang,tokens=5000,includeNonExported=true"
func FormatArguments(args map[string]interface{}) string {
	if len(args) == 0 {
		return ""
	}

	var pairs []string
	for key, value := range args {
		pairs = append(pairs, fmt.Sprintf("%s=%v", key, value))
	}

	return strings.Join(pairs, ",")
}

// ************************************************************************************************
// ValidateRequiredArguments checks if all required arguments are present in the provided map.
//
// Parameters:
//   - args: The arguments map to validate
//   - required: List of required argument names
//
// Returns:
//   - error: An error if any required arguments are missing
//
// Example usage:
//
//	err := ValidateRequiredArguments(args, []string{"libraryName", "tokens"})
func ValidateRequiredArguments(args map[string]interface{}, required []string) error {
	var missing []string

	for _, requiredArg := range required {
		if _, exists := args[requiredArg]; !exists {
			missing = append(missing, requiredArg)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required arguments: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ************************************************************************************************
// Private helper functions

// splitArguments splits the arguments string by comma, handling escaped commas.
func splitArguments(argsString string) []string {
	var parts []string
	var current strings.Builder
	escaped := false

	for i, char := range argsString {
		switch char {
		case '\\':
			if i+1 < len(argsString) && argsString[i+1] == ',' {
				// Escaped comma
				escaped = true
				continue
			}
			current.WriteRune(char)
		case ',':
			if escaped {
				current.WriteRune(char)
				escaped = false
			} else {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
			escaped = false
		}
	}

	// Add the last part
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// convertValue converts a string value to the appropriate Go type.
func convertValue(value string) interface{} {
	// Remove surrounding quotes if present
	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
		(value[0] == '\'' && value[len(value)-1] == '\'')) {
		value = value[1 : len(value)-1]
	}

	// Convert boolean values
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	}

	// Try to convert to integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try to convert to float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Return as string
	return value
}

// ************************************************************************************************
// ArgumentBuilder provides a fluent interface for building MCP tool arguments.
type ArgumentBuilder struct {
	args map[string]interface{}
}

// NewArgumentBuilder creates a new argument builder.
func NewArgumentBuilder() *ArgumentBuilder {
	return &ArgumentBuilder{
		args: make(map[string]interface{}),
	}
}

// Add adds a key-value pair to the arguments.
func (ab *ArgumentBuilder) Add(key string, value interface{}) *ArgumentBuilder {
	ab.args[key] = value
	return ab
}

// AddString adds a string argument.
func (ab *ArgumentBuilder) AddString(key, value string) *ArgumentBuilder {
	ab.args[key] = value
	return ab
}

// AddInt adds an integer argument.
func (ab *ArgumentBuilder) AddInt(key string, value int) *ArgumentBuilder {
	ab.args[key] = value
	return ab
}

// AddBool adds a boolean argument.
func (ab *ArgumentBuilder) AddBool(key string, value bool) *ArgumentBuilder {
	ab.args[key] = value
	return ab
}

// Build returns the constructed arguments map.
func (ab *ArgumentBuilder) Build() map[string]interface{} {
	return ab.args
}

// Clear clears all arguments.
func (ab *ArgumentBuilder) Clear() *ArgumentBuilder {
	ab.args = make(map[string]interface{})
	return ab
}