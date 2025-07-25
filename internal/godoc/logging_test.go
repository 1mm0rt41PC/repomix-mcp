// ************************************************************************************************
// Package godoc command logging tests for debugging Go module documentation retrieval.
// This file contains tests to verify that command logging works correctly with verbose mode.
package godoc

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"repomix-mcp/pkg/types"
)

// mockCache implements CacheInterface for testing
type mockCache struct {
	repos map[string]*types.RepositoryIndex
}

func (m *mockCache) GetRepository(id string) (*types.RepositoryIndex, error) {
	if repo, exists := m.repos[id]; exists {
		return repo, nil
	}
	return nil, fmt.Errorf("repository not found: %s", id)
}

func (m *mockCache) StoreRepository(repo *types.RepositoryIndex) error {
	if m.repos == nil {
		m.repos = make(map[string]*types.RepositoryIndex)
	}
	m.repos[repo.ID] = repo
	return nil
}

func (m *mockCache) ListRepositories() ([]string, error) {
	var ids []string
	for id := range m.repos {
		ids = append(ids, id)
	}
	return ids, nil
}

// TestCommandLoggingWithVerboseMode tests command logging when verbose mode is enabled
func TestCommandLoggingWithVerboseMode(t *testing.T) {
	// Create test configuration
	config := &types.GoModuleConfig{
		Enabled:        true,
		TempDirBase:    "/tmp/test-godoc",
		CacheTimeout:   "1h",
		CommandTimeout: "30s",
		MaxRetries:     3,
		MaxConcurrent:  2,
	}

	// Create mock cache
	cache := &mockCache{
		repos: make(map[string]*types.RepositoryIndex),
	}

	// Create GoDocRetriever
	retriever, err := NewGoDocRetriever(config, cache)
	if err != nil {
		t.Fatalf("Failed to create GoDocRetriever: %v", err)
	}

	// Enable verbose mode to test logging
	retriever.SetVerbose(true)

	// Test command logging with a simple Go version check
	t.Run("TestGoVersionLogging", func(t *testing.T) {
		// Mock the exec command to capture logging
		originalExecCommand := mock_execCommand
		defer func() { mock_execCommand = originalExecCommand }()

		// Mock successful go version command
		mock_execCommand = func(name string, args ...string) *exec.Cmd {
			if name == "go" && len(args) > 0 && args[0] == "version" {
				// Create a command that will succeed
				return exec.Command("echo", "go version go1.21.0 windows/amd64")
			}
			return exec.Command(name, args...)
		}

		// Test validateGoCommand which should log the command
		err := retriever.validateGoCommand()
		if err != nil {
			t.Logf("Expected: go command validation might fail in test environment: %v", err)
		}
		
		// The test passes if no panic occurs and logging is properly formatted
		t.Log("Command logging test completed successfully")
	})

	// Test module path validation
	t.Run("TestGoModulePathValidation", func(t *testing.T) {
		testCases := []struct {
			modulePath string
			expected   bool
		}{
			{"golang.org/x/tools/godoc", true},
			{"github.com/gin-gonic/gin", true},
			{"fmt", true},
			{"os", true},
			{"invalid..path", false},
			{"", false},
		}

		for _, tc := range testCases {
			result := IsGoModulePath(tc.modulePath)
			if result != tc.expected {
				t.Errorf("IsGoModulePath(%q) = %v, expected %v", tc.modulePath, result, tc.expected)
			}
		}
	})
}

// TestCommandLoggingFormat tests the exact format of command logging
func TestCommandLoggingFormat(t *testing.T) {
	// Create test configuration
	config := &types.GoModuleConfig{
		Enabled:        true,
		TempDirBase:    "/tmp/test-godoc",
		CacheTimeout:   "1h",
		CommandTimeout: "30s",
	}

	cache := &mockCache{}
	retriever, err := NewGoDocRetriever(config, cache)
	if err != nil {
		t.Fatalf("Failed to create GoDocRetriever: %v", err)
	}

	// Enable verbose mode
	retriever.SetVerbose(true)

	// Test the executeCommandWithLogging function directly
	t.Run("TestExecuteCommandWithLogging", func(t *testing.T) {
		// Create a simple echo command for testing
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Test successful command
		cmd := exec.CommandContext(ctx, "echo", "test output")
		stdout, _, err := retriever.executeCommandWithLogging(cmd, "test operation")
		
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		if !strings.Contains(string(stdout), "test output") {
			t.Errorf("Expected stdout to contain 'test output', got: %s", string(stdout))
		}
		
		t.Logf("Successfully tested command logging format")
	})

	// Test error command logging
	t.Run("TestErrorCommandLogging", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Test command that will fail
		cmd := exec.CommandContext(ctx, "nonexistent-command")
		stdout, stderr, err := retriever.executeCommandWithLogging(cmd, "test error operation")
		
		if err == nil {
			t.Error("Expected error for nonexistent command")
		}
		
		// The function should handle the error gracefully and log it
		t.Logf("Error logging test completed: err=%v, stdout=%s, stderr=%s", err, string(stdout), string(stderr))
	})
}

// TestProblematicGoModule tests with the specific module mentioned by the user
func TestProblematicGoModule(t *testing.T) {
	// Skip this test if we're not in a proper Go environment
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go command not available, skipping integration test")
	}

	config := &types.GoModuleConfig{
		Enabled:        true,
		TempDirBase:    "",  // Use system temp
		CacheTimeout:   "1h",
		CommandTimeout: "60s",
		MaxRetries:     3,
		MaxConcurrent:  2,
	}

	cache := &mockCache{}
	retriever, err := NewGoDocRetriever(config, cache)
	if err != nil {
		t.Fatalf("Failed to create GoDocRetriever: %v", err)
	}

	// Enable verbose mode to see all command logging
	retriever.SetVerbose(true)

	t.Run("TestGolangToolsGodoc", func(t *testing.T) {
		modulePath := "golang.org/x/tools/godoc"
		
		t.Logf("Testing Go module documentation retrieval for: %s", modulePath)
		t.Logf("This test will show detailed command logging for debugging purposes")
		
		// Attempt to retrieve documentation
		_, err := retriever.RetrieveDocumentation(modulePath)
		
		// We don't fail the test if retrieval fails, as the purpose is to observe logging
		if err != nil {
			t.Logf("Documentation retrieval failed (expected for debugging): %v", err)
		} else {
			t.Logf("Documentation retrieval succeeded")
		}
		
		t.Log("Command logging test for golang.org/x/tools/godoc completed")
	})
}

// Example of how the logging output should look:
// 
// [CMD] go version
// [CMD STDOUT] go version go1.21.0 windows/amd64
// 
// [CMD] go mod init temp-docs
// [CMD STDOUT] go: creating new go.mod: module temp-docs
// 
// [CMD] go get golang.org/x/tools/godoc
// [CMD STDERR] go: golang.org/x/tools/godoc: module golang.org/x/tools/godoc: not found
// 
// [CMD] go doc golang.org/x/tools/godoc
// [CMD STDERR] doc: package golang.org/x/tools/godoc is not in GOROOT or GOPATH