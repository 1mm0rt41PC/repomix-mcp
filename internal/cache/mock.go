package cache

import (
	"os"
	"time"
)

// ************************************************************************************************
// Mock functions to allow easy and in depth unit test
var (
	// Mock for external package
	mock_osMkdirAll        = os.MkdirAll
	mock_timeParseDuration = time.ParseDuration
	mock_osUserHomeDir     = os.UserHomeDir
	mock_osStat            = os.Stat
	mock_osIsNotExist      = os.IsNotExist
	mock_timeNow           = time.Now
)