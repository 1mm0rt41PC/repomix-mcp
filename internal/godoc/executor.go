// ************************************************************************************************
// Package godoc command execution functionality for Go module documentation retrieval.
// This file contains the core command execution logic for running go commands
// and parsing their output to extract documentation information.
package godoc

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"
)

// ************************************************************************************************
// executeGoCommands runs the complete sequence of Go commands to fetch module documentation.
// This includes module initialization, getting the target module, vendoring, and documentation extraction.
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

	// Step 3: Vendor dependencies (optional, helps with some modules)
	if err := g.vendorModule(tempDir); err != nil {
		// Don't fail if vendoring fails, just log it
		if g.verbose {
			log.Printf("Warning: vendoring failed for %s: %v", modulePath, err)
		}
	}

	// Step 4: Get Go version
	goVersion, err := g.getGoVersion()
	if err != nil {
		if g.verbose {
			log.Printf("Warning: failed to get Go version: %v", err)
		}
	} else {
		moduleInfo.GoVersion = goVersion
	}

	// Step 5: Extract basic documentation
	basicDocs, err := g.runGoDoc(modulePath, tempDir, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic documentation: %w", err)
	}
	moduleInfo.Documentation = basicDocs

	// Step 6: Extract comprehensive documentation
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

	// Step 7: Try to get package list
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

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod init failed: %s", string(output))
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

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go get %s failed: %s", modulePath, string(output))
	}

	// Try to extract version from output
	outputStr := string(output)
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
// vendorModule runs `go mod vendor` to vendor dependencies.
func (g *GoDocRetriever) vendorModule(tempDir string) error {
	ctx, cancel := g.createCommandContext()
	defer cancel()

	cmd := mock_execCommandContext(ctx, "go", "mod", "vendor")
	cmd.Dir = tempDir

	if g.verbose {
		log.Printf("Vendoring dependencies in %s", tempDir)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod vendor failed: %s", string(output))
	}

	return nil
}

// ************************************************************************************************
// runGoDoc executes `go doc` command to extract documentation.
func (g *GoDocRetriever) runGoDoc(modulePath, tempDir string, allDocs bool) (string, error) {
	ctx, cancel := g.createCommandContext()
	defer cancel()

	args := []string{"doc"}
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

	output, err := cmd.Output()
	if err != nil {
		// Try alternative approaches if direct module path fails
		return g.tryAlternativeDocApproaches(modulePath, tempDir, allDocs)
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
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

	// If module path has vendor directory, try the vendored path
	vendorPath := filepath.Join(tempDir, "vendor", modulePath)
	if _, err := mock_osStat(vendorPath); err == nil {
		alternatives = append(alternatives, "./vendor/"+modulePath)
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

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
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

	output, err := cmd.Output()
	if err != nil {
		// Try simpler approach
		return g.listPackagesSimple(modulePath, tempDir)
	}

	outputStr := strings.TrimSpace(string(output))
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

	output, err := cmd.Output()
	if err != nil {
		return []string{}, err
	}

	outputStr := strings.TrimSpace(string(output))
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

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
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