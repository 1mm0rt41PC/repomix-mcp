package search

import (
	"path/filepath"
)

// ************************************************************************************************
// Mock functions to allow easy and in depth unit test
var (
	// Mock for external package
	mock_filepathMatch = filepath.Match
)