// ************************************************************************************************
// Package godoc mock functions for testing and abstraction of system calls.
// This file provides mock-able interfaces for file system operations and command execution
// following the same pattern used throughout the repomix-mcp application.
package godoc

import (
	"os"
	"os/exec"
	"os/user"
	"time"
)

// ************************************************************************************************
// Mock functions for file system operations
// These follow the same pattern as internal/indexer/mock.go and internal/cache/mock.go

// mock_osTempDir returns the system temporary directory
var mock_osTempDir = os.TempDir

// mock_osUserHomeDir returns the current user's home directory
var mock_osUserHomeDir = func() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}

// mock_osMkdirAll creates a directory and all necessary parent directories
var mock_osMkdirAll = os.MkdirAll

// mock_osMkdirTemp creates a temporary directory
var mock_osMkdirTemp = os.MkdirTemp

// mock_osRemoveAll removes a directory and all its contents
var mock_osRemoveAll = os.RemoveAll

// mock_osStat returns file information
var mock_osStat = os.Stat

// mock_osIsNotExist checks if an error indicates a file doesn't exist
var mock_osIsNotExist = os.IsNotExist

// mock_osReadFile reads a file and returns its contents
var mock_osReadFile = os.ReadFile

// mock_osWriteFile writes data to a file
var mock_osWriteFile = os.WriteFile

// ************************************************************************************************
// Mock functions for command execution

// mock_execCommand creates a new command
var mock_execCommand = exec.Command

// mock_execCommandContext creates a new command with context
var mock_execCommandContext = exec.CommandContext

// mock_execLookPath searches for an executable in PATH
var mock_execLookPath = exec.LookPath

// ************************************************************************************************
// Mock functions for time operations

// mock_timeNow returns the current time
var mock_timeNow = time.Now

// mock_timeParseDuration parses a duration string
var mock_timeParseDuration = time.ParseDuration

// ************************************************************************************************
// Mock file info interface for compatibility
type mock_osFileInfo = os.FileInfo