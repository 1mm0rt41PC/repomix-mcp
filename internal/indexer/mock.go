package indexer

import (
	"os"
	"os/exec"
	"time"
)

// ************************************************************************************************
// Mock functions to allow easy and in depth unit test
var (
	// Mock for external package
	mock_execLookPath  = exec.LookPath
	mock_execCommand   = exec.Command
	mock_osMkdirTemp   = os.MkdirTemp
	mock_osRemoveAll   = os.RemoveAll
	mock_osReadFile    = os.ReadFile
	mock_osRemove      = os.Remove
	mock_osStat        = os.Stat
	mock_osIsNotExist  = os.IsNotExist
	mock_osWriteFile   = os.WriteFile
	mock_timeNow       = time.Now
)

// Type alias for os.FileInfo to use in mock functions
type mock_osFileInfo = os.FileInfo