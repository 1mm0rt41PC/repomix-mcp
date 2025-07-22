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
)