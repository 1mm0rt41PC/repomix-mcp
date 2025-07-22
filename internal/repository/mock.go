package repository

import (
	"os"
	"time"

	"github.com/go-git/go-git/v5"
)

// ************************************************************************************************
// Mock functions to allow easy and in depth unit test
var (
	// Mock for external package
	mock_osUserHomeDir  = os.UserHomeDir
	mock_osMkdirAll     = os.MkdirAll
	mock_osStat         = os.Stat
	mock_osIsNotExist   = os.IsNotExist
	mock_osReadFile     = os.ReadFile
	mock_osRemoveAll    = os.RemoveAll
	mock_timeNow        = time.Now
	mock_gitPlainOpen   = git.PlainOpen
	mock_gitPlainClone  = git.PlainClone
)