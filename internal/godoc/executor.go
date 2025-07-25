// ************************************************************************************************
// Package godoc command execution functionality for Go module documentation retrieval.
// This file contains the core command execution logic for running go commands
// and parsing their output to extract documentation information.
package godoc

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ************************************************************************************************
// executeCommandWithLogging wraps command execution with verbose logging.
// It logs the command being executed and its stdout/stderr output when verbose mode is enabled.
//
// Returns:
//   - stdout: Standard output from the command
//   - stderr: Standard error from the command (if available separately)
//   - error: Command execution error
func (g *GoDocRetriever) executeCommandWithLogging(cmd *exec.Cmd, operation string) (stdout []byte, stderr []byte, err error) {
	// Build command string for logging
	cmdStr := cmd.Path
	if len(cmd.Args) > 1 {
		cmdStr = strings.Join(cmd.Args, " ")
	}

	if g.verbose {
		log.Printf("[CMD] %s", cmdStr)
	}

	// Execute command and capture output
	if cmd.Stderr == nil {
		// Use CombinedOutput when stderr is not set separately
		combined, err := cmd.CombinedOutput()

		if g.verbose {
			if err != nil {
				// Command failed - log the combined output as stderr
				log.Printf("[CMD STDERR] %s", strings.TrimSpace(string(combined)))
			} else {
				// Command succeeded - log as stdout
				if len(combined) > 0 {
					log.Printf("[CMD STDOUT] %s", strings.TrimSpace(string(combined)))
				} else {
					log.Printf("[CMD STDOUT] (no output)")
				}
			}
		}

		return combined, nil, err
	} else {
		// Use separate stdout/stderr when possible
		stdout, err := cmd.Output()

		if g.verbose {
			if err != nil {
				// Try to get stderr from ExitError
				if exitError, ok := err.(*exec.ExitError); ok {
					stderr = exitError.Stderr
					log.Printf("[CMD STDERR] %s", strings.TrimSpace(string(stderr)))
				} else {
					log.Printf("[CMD STDERR] %s", err.Error())
				}
			}

			if len(stdout) > 0 {
				log.Printf("[CMD STDOUT] %s", strings.TrimSpace(string(stdout)))
			} else {
				log.Printf("[CMD STDOUT] (no output)")
			}
		}

		return stdout, stderr, err
	}
}

// ************************************************************************************************
// executeGoCommands runs the complete sequence of Go commands to fetch module documentation.
// This includes module initialization, getting the target module, and documentation extraction.
//
// Returns:
//   - *GoModuleInfo: Complete module information with documentation.
//   - error: An error if any command fails.
func (g *GoDocRetriever) executeGoCommands(modulePath, tempDir string) (*GoModuleInfo, error) {
	if g.verbose {
		log.Printf("Executing Go commands for module %s in directory %s", modulePath, tempDir)
	}

	// Initialize the result structure
	moduleInfo := &GoModuleInfo{
		ModulePath:  modulePath,
		CachedAt:    mock_timeNow(),
		PackageList: []string{},
		Examples:    make(map[string]string),
	}

	// Step 1: Initialize Go module
	if err := g.initGoModule(tempDir); err != nil {
		return nil, fmt.Errorf("failed to initialize Go module: %w", err)
	}

	// Step 2: Get the target module
	version, err := g.getModule(modulePath, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get module %s: %w", modulePath, err)
	}
	moduleInfo.Version = version

	// Step 3: Get Go version
	goVersion, err := g.getGoVersion()
	if err != nil {
		if g.verbose {
			log.Printf("Warning: failed to get Go version: %v", err)
		}
	} else {
		moduleInfo.GoVersion = goVersion
	}

	// Step 4: Extract basic documentation
	basicDocs, err := g.runGoDoc(modulePath, tempDir, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic documentation: %w", err)
	}
	moduleInfo.Documentation = basicDocs

	// Step 5: Extract comprehensive documentation
	allDocs, err := g.runGoDoc(modulePath, tempDir, true)
	if err != nil {
		// Don't fail if comprehensive docs fail, just log it
		if g.verbose {
			log.Printf("Warning: failed to get comprehensive documentation for %s: %v", modulePath, err)
		}
		moduleInfo.ErrorInfo = fmt.Sprintf("Failed to get comprehensive docs: %v", err)
	} else {
		moduleInfo.AllDocs = allDocs
	}

	// Step 6: Try to get package list
	packages, err := g.listPackages(modulePath, tempDir)
	if err != nil {
		if g.verbose {
			log.Printf("Warning: failed to list packages for %s: %v", modulePath, err)
		}
	} else {
		moduleInfo.PackageList = packages
	}

	if g.verbose {
		log.Printf("Successfully executed Go commands for module %s", modulePath)
	}

	return moduleInfo, nil
}

// ************************************************************************************************
// initGoModule initializes a new Go module in the temporary directory.
func (g *GoDocRetriever) initGoModule(tempDir string) error {
	ctx, cancel := g.createCommandContext()
	defer cancel()

	cmd := mock_execCommandContext(ctx, "go", "mod", "init", "temp-docs")
	cmd.Dir = tempDir

	if g.verbose {
		log.Printf("Initializing Go module in %s", tempDir)
	}

	stdout, stderr, err := g.executeCommandWithLogging(cmd, "go mod init")
	if err != nil {
		if len(stderr) > 0 {
			return fmt.Errorf("go mod init failed: %s", string(stderr))
		}
		return fmt.Errorf("go mod init failed: %s", string(stdout))
	}

	return nil
}

// ************************************************************************************************
// getModule fetches the specified Go module using `go get`.
func (g *GoDocRetriever) getModule(modulePath, tempDir string) (string, error) {
	ctx, cancel := g.createCommandContext()
	defer cancel()

	cmd := mock_execCommandContext(ctx, "go", "get", modulePath)
	cmd.Dir = tempDir

	if g.verbose {
		log.Printf("Getting module: %s", modulePath)
	}

	stdout, stderr, err := g.executeCommandWithLogging(cmd, "go get")
	if err != nil {
		if len(stderr) > 0 {
			return "", fmt.Errorf("go get %s failed: %s", modulePath, string(stderr))
		}
		return "", fmt.Errorf("go get %s failed: %s", modulePath, string(stdout))
	}

	// Try to extract version from output
	outputStr := string(stdout)
	if strings.Contains(outputStr, "@") {
		// Look for version information in the output
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, modulePath) && strings.Contains(line, "@") {
				parts := strings.Split(line, "@")
				if len(parts) > 1 {
					return strings.TrimSpace(parts[1]), nil
				}
			}
		}
	}

	return "latest", nil
}

// ************************************************************************************************
// runGoDoc executes `go doc` command to extract documentation.
func (g *GoDocRetriever) runGoDoc(modulePath, tempDir string, allDocs bool) (string, error) {
	ctx, cancel := g.createCommandContext()
	defer cancel()

	args := []string{"go", "doc"}
	if allDocs {
		args = append(args, "-all")
	}
	args = append(args, modulePath)

	cmd := mock_execCommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = tempDir

	command := "go doc"
	if allDocs {
		command = "go doc -all"
	}

	if g.verbose {
		log.Printf("Running: %s %s", command, modulePath)
	}

	stdout, _, err := g.executeCommandWithLogging(cmd, "go doc")
	if err != nil {
		// Log the failure and try alternative approaches
		if g.verbose {
			log.Printf("Direct go doc approach failed, trying alternatives...")
		}
		return g.tryAlternativeDocApproaches(modulePath, tempDir, allDocs)
	}

	result := strings.TrimSpace(string(stdout))
	if result == "" {
		if g.verbose {
			log.Printf("go doc returned empty output, trying alternatives...")
		}
		return g.tryAlternativeDocApproaches(modulePath, tempDir, allDocs)
	}

	return result, nil
}

// ************************************************************************************************
// tryAlternativeDocApproaches tries different ways to get documentation when direct approach fails.
func (g *GoDocRetriever) tryAlternativeDocApproaches(modulePath, tempDir string, allDocs bool) (string, error) {
	alternatives := []string{
		modulePath,
		filepath.Base(modulePath), // Just the package name
	}

	for _, alt := range alternatives {
		if result, err := g.runGoDocDirect(alt, tempDir, allDocs); err == nil && result != "" {
			return result, nil
		}
	}

	return "", fmt.Errorf("all documentation extraction approaches failed for %s", modulePath)
}

// ************************************************************************************************
// runGoDocDirect runs go doc with a specific path.
func (g *GoDocRetriever) runGoDocDirect(path, tempDir string, allDocs bool) (string, error) {
	ctx, cancel := g.createCommandContext()
	defer cancel()

	args := []string{"doc"}
	if allDocs {
		args = append(args, "-all")
	}
	args = append(args, path)

	cmd := mock_execCommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = tempDir

	stdout, _, err := g.executeCommandWithLogging(cmd, "go doc direct")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(stdout)), nil
}

// ************************************************************************************************
// listPackages attempts to list packages in the module.
func (g *GoDocRetriever) listPackages(modulePath, tempDir string) ([]string, error) {
	ctx, cancel := g.createCommandContext()
	defer cancel()

	cmd := mock_execCommandContext(ctx, "go", "list", "-f", "{{.ImportPath}}", modulePath+"/...")
	cmd.Dir = tempDir

	if g.verbose {
		log.Printf("Listing packages for: %s", modulePath)
	}

	stdout, _, err := g.executeCommandWithLogging(cmd, "go list")
	if err != nil {
		// Try simpler approach
		if g.verbose {
			log.Printf("go list with template failed, trying simple approach...")
		}
		return g.listPackagesSimple(modulePath, tempDir)
	}

	outputStr := strings.TrimSpace(string(stdout))
	if outputStr == "" {
		return []string{}, nil
	}

	packages := strings.Split(outputStr, "\n")
	result := make([]string, 0, len(packages))
	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg != "" {
			result = append(result, pkg)
		}
	}

	return result, nil
}

// ************************************************************************************************
// listPackagesSimple tries a simpler approach to list packages.
func (g *GoDocRetriever) listPackagesSimple(modulePath, tempDir string) ([]string, error) {
	ctx, cancel := g.createCommandContext()
	defer cancel()

	cmd := mock_execCommandContext(ctx, "go", "list", modulePath)
	cmd.Dir = tempDir

	stdout, _, err := g.executeCommandWithLogging(cmd, "go list simple")
	if err != nil {
		return []string{}, err
	}

	outputStr := strings.TrimSpace(string(stdout))
	if outputStr == "" {
		return []string{}, nil
	}

	return []string{outputStr}, nil
}

// ************************************************************************************************
// getGoVersion gets the Go version being used.
func (g *GoDocRetriever) getGoVersion() (string, error) {
	ctx, cancel := g.createCommandContext()
	defer cancel()

	cmd := mock_execCommandContext(ctx, "go", "version")

	stdout, _, err := g.executeCommandWithLogging(cmd, "go version")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(stdout)), nil
}

// ************************************************************************************************
// createCommandContext creates a context with timeout for command execution.
func (g *GoDocRetriever) createCommandContext() (context.Context, context.CancelFunc) {
	timeout := 60 * time.Second // Default timeout

	if g.config.CommandTimeout != "" {
		if parsed, err := mock_timeParseDuration(g.config.CommandTimeout); err == nil {
			timeout = parsed
		}
	}

	return context.WithTimeout(context.Background(), timeout)
}
