package config

import (
	"os"
)

// ************************************************************************************************
// Mock functions to allow easy and in depth unit test
var (
	// Mock for external package
	mock_osUserHomeDir = os.UserHomeDir
	mock_osStat        = os.Stat
	mock_osIsNotExist  = os.IsNotExist
	mock_osMkdirAll    = os.MkdirAll
	mock_osWriteFile   = os.WriteFile
	mock_osReadFile    = os.ReadFile
)