package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	
	"repomix-mcp/pkg/types"
)

func TestGoParser_ParseRepository(t *testing.T) {
	// Create a temporary test directory
	tempDir, err := os.MkdirTemp("", "go_parser_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test go.mod file
	goModContent := `module test-repo

go 1.21

require (
	github.com/example/dep v1.0.0
)
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create a test Go file with various constructs
	testGoContent := `package main

import (
	"fmt"
	"log"
)

// Constants
const (
	Version = "1.0.0"
	MaxRetries = 3
)

const SingleConst = "test"

// Variables
var (
	GlobalVar string
	counter   int
)

var singleVar = "hello"

// Types
type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
	Age  int    ` + "`json:\"age\"`" + `
}

type StringAlias = string

type Handler interface {
	Handle(data []byte) error
	Close() error
}

// Functions
func main() {
	fmt.Println("Hello, World!")
}

func processData(data []byte, count int) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty data")
	}
	return string(data), nil
}

// Methods
func (u *User) GetDisplayName() string {
	return fmt.Sprintf("%s (%d)", u.Name, u.Age)
}

func (u User) IsAdult() bool {
	return u.Age >= 18
}

// Unexported constructs
func internalFunc() {
	log.Println("internal")
}

type internalStruct struct {
	data string
}
`

	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(testGoContent), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Create parser and test
	parser := NewGoParser()
	
	// Test with default config (includeNonExported = false)
	config := types.IndexingConfig{
		Enabled:           true,
		IncludeNonExported: false,
	}
	
	repoIndex, err := parser.ParseRepository("test-repo", tempDir, config)
	if err != nil {
		t.Fatalf("ParseRepository failed: %v", err)
	}

	// Verify basic repository information
	if repoIndex.ID != "test-repo" {
		t.Errorf("Expected repository ID 'test-repo', got '%s'", repoIndex.ID)
	}

	if repoIndex.Path != tempDir {
		t.Errorf("Expected repository path '%s', got '%s'", tempDir, repoIndex.Path)
	}

	// Verify .repomix.xml file was created
	xmlFile, exists := repoIndex.Files[".repomix.xml"]
	if !exists {
		t.Fatal("Expected .repomix.xml file to be created")
	}

	if xmlFile.Language != "xml" {
		t.Errorf("Expected XML language, got '%s'", xmlFile.Language)
	}

	// Verify XML content contains expected structures
	xmlContent := xmlFile.Content
	
	expectedPatterns := []string{
		"<file_summary>",
		"<directory_structure>",
		"<files>",
		`<file path="main.go" package="main">`,
		`<package name="main">`,
		"const Version = \"1.0.0\"",
		"const SingleConst = \"test\"",
		"var GlobalVar string",
		"var singleVar = \"hello\"",
		"type User struct",
		"type StringAlias = string",
		"type Handler interface",
		"func main()",
		"func processData(data []byte, count int) (string, error)",
		"func (*User) GetDisplayName() string",
		"func (User) IsAdult() bool",
		"func internalFunc()",
		"type internalStruct struct",
		"main.go",
		"// Package: main",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(xmlContent, pattern) {
			t.Errorf("Expected XML content to contain '%s'", pattern)
		}
	}

	// Verify metadata
	if indexerType, ok := repoIndex.Metadata["indexer_type"].(string); !ok || indexerType != "go_native" {
		t.Errorf("Expected indexer_type 'go_native', got '%v'", repoIndex.Metadata["indexer_type"])
	}

	if fileCount, ok := repoIndex.Metadata["file_count"].(int); !ok || fileCount != 1 {
		t.Errorf("Expected file_count 1, got '%v'", repoIndex.Metadata["file_count"])
	}
}

func TestGoParser_isGoProject(t *testing.T) {
	parser := NewGoParser()

	// Test with go.mod file
	tempDir, err := os.MkdirTemp("", "go_project_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	if !parser.isGoProject(tempDir) {
		t.Error("Expected directory with go.mod to be detected as Go project")
	}

	// Test without go.mod but with Go files
	tempDir2, err := os.MkdirTemp("", "go_project_test2_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir2)

	// Create multiple Go files
	for i := 0; i < 4; i++ {
		filename := filepath.Join(tempDir2, fmt.Sprintf("file%d.go", i))
		if err := os.WriteFile(filename, []byte("package main"), 0644); err != nil {
			t.Fatalf("Failed to write Go file: %v", err)
		}
	}

	if !parser.isGoProject(tempDir2) {
		t.Error("Expected directory with 4+ Go files to be detected as Go project")
	}

	// Test with insufficient Go files
	tempDir3, err := os.MkdirTemp("", "go_project_test3_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir3)

	// Create only 2 Go files
	for i := 0; i < 2; i++ {
		filename := filepath.Join(tempDir3, fmt.Sprintf("file%d.go", i))
		if err := os.WriteFile(filename, []byte("package main"), 0644); err != nil {
			t.Fatalf("Failed to write Go file: %v", err)
		}
	}

	if parser.isGoProject(tempDir3) {
		t.Error("Expected directory with only 2 Go files to NOT be detected as Go project")
	}
}

func TestGoParser_findGoFiles(t *testing.T) {
	parser := NewGoParser()

	tempDir, err := os.MkdirTemp("", "find_go_files_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory structure with Go files
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	testFiles := []string{
		"main.go",
		"helper.go",
		"main_test.go",        // Should be excluded
		"subdir/sub.go",
		"subdir/sub_test.go",  // Should be excluded
		"README.md",           // Should be excluded
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", file, err)
		}
		if err := os.WriteFile(fullPath, []byte("package main"), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", file, err)
		}
	}

	goFiles, err := parser.findGoFiles(tempDir)
	if err != nil {
		t.Fatalf("findGoFiles failed: %v", err)
	}

	expectedFiles := []string{"main.go", "helper.go", "subdir/sub.go"}
	if len(goFiles) != len(expectedFiles) {
		t.Errorf("Expected %d Go files, got %d: %v", len(expectedFiles), len(goFiles), goFiles)
	}

	for _, expected := range expectedFiles {
		found := false
		for _, actual := range goFiles {
			if filepath.ToSlash(actual) == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s in Go files list", expected)
		}
	}

	// Verify test files are excluded
	for _, goFile := range goFiles {
		if strings.HasSuffix(goFile, "_test.go") {
			t.Errorf("Test file %s should be excluded", goFile)
		}
	}
}

func TestGoParser_generateRepomixXML(t *testing.T) {
	parser := NewGoParser()

	// Create test file analyses
	fileAnalyses := map[string]*GoFileAnalysis{
		"main.go": {
			FilePath:    "main.go",
			PackageName: "main",
			Constructs: []GoConstruct{
				{
					Type:      "const",
					Name:      "Version",
					Signature: "const Version = \"1.0.0\"",
					Package:   "main",
					File:      "main.go",
					Line:      5,
					Exported:  true,
				},
				{
					Type:      "func",
					Name:      "main",
					Signature: "func main()",
					Package:   "main",
					File:      "main.go",
					Line:      10,
					Exported:  false,
				},
				{
					Type:      "struct",
					Name:      "User",
					Signature: "type User struct",
					Package:   "main",
					File:      "main.go",
					Line:      15,
					Exported:  true,
					Fields:    []string{"ID int", "Name string"},
				},
			},
		},
	}

	// Create test package analyses
	packageAnalyses := map[string]*GoPackageAnalysis{
		"main": {
			PackageName: "main",
			Path:        "/path/to/repo",
			Files:       []string{"main.go"},
			Constructs: map[string][]GoConstruct{
				"const": {fileAnalyses["main.go"].Constructs[0]},
				"func":  {fileAnalyses["main.go"].Constructs[1]},
				"struct": {fileAnalyses["main.go"].Constructs[2]},
			},
			ExportedOnly: map[string][]GoConstruct{
				"const": {fileAnalyses["main.go"].Constructs[0]},
				"struct": {fileAnalyses["main.go"].Constructs[2]},
			},
			Summary: make(map[string]int),
		},
	}

	goFiles := []string{"main.go", "helper.go"}

	// Test with includeNonExported = false (default behavior)
	xml := parser.generateRepomixXML("test-repo", "/path/to/repo", fileAnalyses, packageAnalyses, goFiles, false)

	// Verify XML structure with new format
	expectedElements := []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<repository>",
		"<file_summary>",
		"<purpose>",
		"<file_format>",
		"<usage_guidelines>",
		"<notes>",
		"<directory_structure>",
		"main.go",
		"helper.go",
		"<files>",
		`<file path="main.go" package="main">`,
		`<package name="main">`,
		"const Version = \"1.0.0\"",
		"func main()",
		"type User struct",
		"</repository>",
	}

	for _, element := range expectedElements {
		if !strings.Contains(xml, element) {
			t.Errorf("Expected XML to contain '%s'", element)
		}
	}

	// Verify struct fields are included
	if !strings.Contains(xml, "ID int") || !strings.Contains(xml, "Name string") {
		t.Error("Expected struct fields to be included in XML output")
	}

	// Verify package section only contains exported constructs
	if !strings.Contains(xml, `<package name="main">`) {
		t.Error("Expected package section to be included")
	}
}

func TestGoParser_IncludeNonExported(t *testing.T) {
	// Create a temporary test directory
	tempDir, err := os.MkdirTemp("", "go_parser_include_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create go.mod
	goModContent := `module test-repo
go 1.21
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create test file with mixed exported/unexported constructs
	testGoContent := `package main

// Exported constructs
const ExportedConstant = "public"
var ExportedVariable = "public var"
type ExportedStruct struct {
	PublicField string
}
func ExportedFunction() string {
	return "exported"
}

// Unexported constructs
const unexportedConstant = "private"
var unexportedVariable = "private var"
type unexportedStruct struct {
	privateField string
}
func unexportedFunction() string {
	return "unexported"
}

func main() {
	// Main function
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(testGoContent), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	parser := NewGoParser()

	// Test with includeNonExported = false
	t.Run("ExcludeNonExported", func(t *testing.T) {
		config := types.IndexingConfig{
			Enabled:           true,
			IncludeNonExported: false,
		}
		
		repoIndex, err := parser.ParseRepository("test-repo", tempDir, config)
		if err != nil {
			t.Fatalf("ParseRepository failed: %v", err)
		}

		xmlContent := repoIndex.Files[".repomix.xml"].Content

		// Exported constructs should be present
		exportedConstructs := []string{
			"ExportedConstant",
			"ExportedVariable",
			"ExportedStruct",
			"ExportedFunction",
		}
		for _, construct := range exportedConstructs {
			if !strings.Contains(xmlContent, construct) {
				t.Errorf("Expected exported construct '%s' to be present", construct)
			}
		}

		// Unexported constructs should be filtered out
		unexportedConstructs := []string{
			"unexportedConstant",
			"unexportedVariable",
			"unexportedStruct",
			"unexportedFunction",
		}
		for _, construct := range unexportedConstructs {
			if strings.Contains(xmlContent, construct) {
				t.Errorf("Expected unexported construct '%s' to be filtered out", construct)
			}
		}

		// Check notes section
		if !strings.Contains(xmlContent, "Only exported constructs are included") {
			t.Error("Expected notes to indicate only exported constructs are included")
		}
	})

	// Test with includeNonExported = true
	t.Run("IncludeNonExported", func(t *testing.T) {
		config := types.IndexingConfig{
			Enabled:           true,
			IncludeNonExported: true,
		}
		
		repoIndex, err := parser.ParseRepository("test-repo", tempDir, config)
		if err != nil {
			t.Fatalf("ParseRepository failed: %v", err)
		}

		xmlContent := repoIndex.Files[".repomix.xml"].Content

		// Both exported and unexported constructs should be present
		allConstructs := []string{
			"ExportedConstant",
			"ExportedVariable",
			"ExportedStruct",
			"ExportedFunction",
			"unexportedConstant",
			"unexportedVariable",
			"unexportedStruct",
			"unexportedFunction",
		}
		for _, construct := range allConstructs {
			if !strings.Contains(xmlContent, construct) {
				t.Errorf("Expected construct '%s' to be present when includeNonExported=true", construct)
			}
		}

		// Check notes section
		if !strings.Contains(xmlContent, "All constructs (both exported and unexported) are included") {
			t.Error("Expected notes to indicate all constructs are included")
		}

		// Check package section should say "all constructs"
		if !strings.Contains(xmlContent, "(all constructs)") {
			t.Error("Expected package section to indicate all constructs are included")
		}
	})
}