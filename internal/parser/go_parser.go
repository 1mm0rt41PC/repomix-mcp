// ************************************************************************************************
// Package parser provides Go AST parsing functionality for the repomix-mcp application.
// It extracts Go language constructs (functions, structs, variables, constants, types)
// from Go source files and generates structured representations for AI consumption.
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"repomix-mcp/pkg/types"
)

// ************************************************************************************************
// GoParser handles Go AST parsing and code structure extraction.
type GoParser struct {
	fileSet *token.FileSet
}

// ************************************************************************************************
// GoConstruct represents a parsed Go language construct.
type GoConstruct struct {
	Type       string            `json:"type"`       // "func", "struct", "var", "const", "type", "interface"
	Name       string            `json:"name"`       // Construct name
	Signature  string            `json:"signature"`  // Full signature/declaration
	Package    string            `json:"package"`    // Package name
	File       string            `json:"file"`       // Source file path
	Line       int               `json:"line"`       // Line number
	Exported   bool              `json:"exported"`   // Whether construct is exported (public)
	Receiver   string            `json:"receiver"`   // Method receiver (for methods)
	Parameters []string          `json:"parameters"` // Function parameters
	Returns    []string          `json:"returns"`    // Function return types
	Fields     []string          `json:"fields"`     // Struct fields
	Methods    []string          `json:"methods"`    // Interface methods
	Metadata   map[string]string `json:"metadata"`   // Additional metadata
}

// ************************************************************************************************
// GoFileAnalysis represents analysis of a single Go file.
type GoFileAnalysis struct {
	FilePath    string        `json:"filePath"`
	PackageName string        `json:"packageName"`
	Constructs  []GoConstruct `json:"constructs"`
}

// ************************************************************************************************
// GoPackageAnalysis represents the complete analysis of a Go package.
type GoPackageAnalysis struct {
	PackageName  string                   `json:"packageName"`
	Path         string                   `json:"path"`
	Files        []string                 `json:"files"`
	Constructs   map[string][]GoConstruct `json:"constructs"`   // Organized by type
	ExportedOnly map[string][]GoConstruct `json:"exportedOnly"` // Only exported constructs by type
	Summary      map[string]int           `json:"summary"`      // Count by construct type
}

// ************************************************************************************************
// NewGoParser creates a new Go parser instance.
func NewGoParser() *GoParser {
	return &GoParser{
		fileSet: token.NewFileSet(),
	}
}

// ************************************************************************************************
// ParseRepository analyzes a Go repository and extracts all language constructs.
// It scans for Go files, parses them, and organizes constructs by type.
func (p *GoParser) ParseRepository(repositoryID, localPath string, config types.IndexingConfig) (*types.RepositoryIndex, error) {
	if repositoryID == "" || localPath == "" {
		return nil, fmt.Errorf("%w: invalid parameters", types.ErrInvalidConfig)
	}

	// Check if this is a Go project
	if !p.isGoProject(localPath) {
		return nil, fmt.Errorf("not a Go project: no go.mod found in %s", localPath)
	}

	// Find all Go files (excluding test files)
	goFiles, err := p.findGoFiles(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Go files: %w", err)
	}

	if len(goFiles) == 0 {
		return nil, fmt.Errorf("no Go files found in repository")
	}

	// Parse all Go files and extract constructs
	fileAnalyses := make(map[string]*GoFileAnalysis)
	packageAnalyses := make(map[string]*GoPackageAnalysis)

	for _, goFile := range goFiles {
		constructs, pkg, err := p.parseGoFile(goFile, localPath)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to parse %s: %v\n", goFile, err)
			continue
		}

		// Create file analysis
		fileAnalyses[goFile] = &GoFileAnalysis{
			FilePath:    goFile,
			PackageName: pkg,
			Constructs:  constructs,
		}

		// Track package analysis
		if pkg != "" {
			if _, exists := packageAnalyses[pkg]; !exists {
				packageAnalyses[pkg] = &GoPackageAnalysis{
					PackageName:  pkg,
					Path:         filepath.Dir(goFile),
					Files:        make([]string, 0),
					Constructs:   make(map[string][]GoConstruct),
					ExportedOnly: make(map[string][]GoConstruct),
					Summary:      make(map[string]int),
				}
			}
			packageAnalyses[pkg].Files = append(packageAnalyses[pkg].Files, goFile)

			// Add constructs to package analysis
			for _, construct := range constructs {
				constructType := construct.Type

				// Add to all constructs
				if _, exists := packageAnalyses[pkg].Constructs[constructType]; !exists {
					packageAnalyses[pkg].Constructs[constructType] = make([]GoConstruct, 0)
				}
				packageAnalyses[pkg].Constructs[constructType] = append(packageAnalyses[pkg].Constructs[constructType], construct)

				// Add to exported-only if exported
				if construct.Exported {
					if _, exists := packageAnalyses[pkg].ExportedOnly[constructType]; !exists {
						packageAnalyses[pkg].ExportedOnly[constructType] = make([]GoConstruct, 0)
					}
					packageAnalyses[pkg].ExportedOnly[constructType] = append(packageAnalyses[pkg].ExportedOnly[constructType], construct)
				}
			}
		}
	}

	// Generate XML content
	xmlContent := p.generateRepomixXML(repositoryID, localPath, fileAnalyses, packageAnalyses, goFiles, config.IncludeNonExported)

	// Create repository index
	repoIndex := &types.RepositoryIndex{
		ID:          repositoryID,
		Name:        repositoryID,
		Path:        localPath,
		LastUpdated: time.Now(),
		Files:       make(map[string]types.IndexedFile),
		Metadata:    make(map[string]interface{}),
		CommitHash:  "", // Will be filled by repository manager
	}

	// Create a single indexed file containing the XML representation
	xmlFile := types.IndexedFile{
		Path:         ".repomix.xml",
		Content:      xmlContent,
		Hash:         p.calculateContentHash(xmlContent),
		Size:         int64(len(xmlContent)),
		ModTime:      time.Now(),
		Language:     "xml",
		RepositoryID: repositoryID,
		Metadata: map[string]string{
			"indexer_type":   "go_native",
			"go_files_count": fmt.Sprintf("%d", len(goFiles)),
			"packages_count": fmt.Sprintf("%d", len(packageAnalyses)),
		},
	}

	repoIndex.Files[".repomix.xml"] = xmlFile

	// Add metadata
	repoIndex.Metadata["indexer_type"] = "go_native"
	repoIndex.Metadata["file_count"] = len(goFiles)
	repoIndex.Metadata["packages_count"] = len(packageAnalyses)
	repoIndex.Metadata["indexed_at"] = time.Now().Format(time.RFC3339)
	repoIndex.Metadata["indexer_version"] = "repomix-mcp-go-v1.0.0"

	// Count constructs by type across all packages
	constructCounts := make(map[string]int)
	for _, pkgAnalysis := range packageAnalyses {
		for constructType, constructs := range pkgAnalysis.Constructs {
			constructCounts[constructType] += len(constructs)
		}
	}
	for constructType, count := range constructCounts {
		repoIndex.Metadata[fmt.Sprintf("%s_count", constructType)] = count
	}

	return repoIndex, nil
}

// ************************************************************************************************
// isGoProject checks if the given path contains a Go project.
func (p *GoParser) isGoProject(localPath string) bool {
	goModPath := filepath.Join(localPath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		return true
	}

	// Fallback: check for significant number of .go files
	goFiles, err := p.findGoFiles(localPath)
	if err != nil {
		return false
	}

	return len(goFiles) >= 3 // At least 3 Go files to consider it a Go project
}

// ************************************************************************************************
// findGoFiles recursively finds all Go files in the repository, excluding test files.
func (p *GoParser) findGoFiles(localPath string) ([]string, error) {
	var goFiles []string

	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and common ignore patterns
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for Go files, excluding test files
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			relPath, err := filepath.Rel(localPath, path)
			if err != nil {
				return err
			}
			goFiles = append(goFiles, relPath)
		}

		return nil
	})

	return goFiles, err
}

// ************************************************************************************************
// parseGoFile parses a single Go file and extracts all constructs.
func (p *GoParser) parseGoFile(filePath, basePath string) ([]GoConstruct, string, error) {
	fullPath := filepath.Join(basePath, filePath)

	// Parse the Go file
	src, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	file, err := parser.ParseFile(p.fileSet, fullPath, src, parser.ParseComments)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse Go file: %w", err)
	}

	var constructs []GoConstruct
	packageName := file.Name.Name

	// Extract constructs using AST visitor
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			construct := p.extractFunction(node, filePath, packageName)
			constructs = append(constructs, construct)

		case *ast.GenDecl:
			// Handle type, var, const declarations
			for _, spec := range node.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					construct := p.extractType(s, node, filePath, packageName)
					constructs = append(constructs, construct)

				case *ast.ValueSpec:
					// Handle var and const
					constructs = append(constructs, p.extractValueSpec(s, node, filePath, packageName)...)
				}
			}
		}
		return true
	})

	return constructs, packageName, nil
}

// ************************************************************************************************
// extractFunction extracts function/method information from AST.
func (p *GoParser) extractFunction(fn *ast.FuncDecl, filePath, packageName string) GoConstruct {
	pos := p.fileSet.Position(fn.Pos())

	construct := GoConstruct{
		Type:     "func",
		Name:     fn.Name.Name,
		Package:  packageName,
		File:     filePath,
		Line:     pos.Line,
		Exported: ast.IsExported(fn.Name.Name),
		Metadata: make(map[string]string),
	}

	// Handle method receiver
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		construct.Type = "method"
		if recv := fn.Recv.List[0]; recv.Type != nil {
			construct.Receiver = p.typeToString(recv.Type)
		}
	}

	// Extract parameters
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			paramType := p.typeToString(param.Type)
			if len(param.Names) > 0 {
				for _, name := range param.Names {
					construct.Parameters = append(construct.Parameters, name.Name+" "+paramType)
				}
			} else {
				construct.Parameters = append(construct.Parameters, paramType)
			}
		}
	}

	// Extract return types
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			construct.Returns = append(construct.Returns, p.typeToString(result.Type))
		}
	}

	// Generate signature
	construct.Signature = p.generateFunctionSignature(construct)

	return construct
}

// ************************************************************************************************
// extractType extracts type declarations (struct, interface, type alias).
func (p *GoParser) extractType(ts *ast.TypeSpec, genDecl *ast.GenDecl, filePath, packageName string) GoConstruct {
	pos := p.fileSet.Position(ts.Pos())

	construct := GoConstruct{
		Name:     ts.Name.Name,
		Package:  packageName,
		File:     filePath,
		Line:     pos.Line,
		Exported: ast.IsExported(ts.Name.Name),
		Metadata: make(map[string]string),
	}

	switch t := ts.Type.(type) {
	case *ast.StructType:
		construct.Type = "struct"
		construct.Fields = p.extractStructFields(t)
		construct.Signature = p.generateStructSignature(construct)

	case *ast.InterfaceType:
		construct.Type = "interface"
		construct.Methods = p.extractInterfaceMethods(t)
		construct.Signature = p.generateInterfaceSignature(construct)

	default:
		construct.Type = "type"
		construct.Signature = fmt.Sprintf("type %s = %s", construct.Name, p.typeToString(ts.Type))
	}

	return construct
}

// ************************************************************************************************
// extractValueSpec extracts variable and constant declarations.
func (p *GoParser) extractValueSpec(vs *ast.ValueSpec, genDecl *ast.GenDecl, filePath, packageName string) []GoConstruct {
	var constructs []GoConstruct
	pos := p.fileSet.Position(vs.Pos())

	constructType := "var"
	if genDecl.Tok == token.CONST {
		constructType = "const"
	}

	for i, name := range vs.Names {
		construct := GoConstruct{
			Type:     constructType,
			Name:     name.Name,
			Package:  packageName,
			File:     filePath,
			Line:     pos.Line,
			Exported: ast.IsExported(name.Name),
			Metadata: make(map[string]string),
		}

		// Generate signature
		var typeStr string
		if vs.Type != nil {
			typeStr = p.typeToString(vs.Type)
		}

		var valueStr string
		if vs.Values != nil && i < len(vs.Values) {
			valueStr = p.nodeToString(vs.Values[i])
		}

		if constructType == "const" {
			if valueStr != "" {
				construct.Signature = fmt.Sprintf("const %s = %s", construct.Name, valueStr)
			} else {
				construct.Signature = fmt.Sprintf("const %s %s", construct.Name, typeStr)
			}
		} else {
			if typeStr != "" && valueStr != "" {
				construct.Signature = fmt.Sprintf("var %s %s = %s", construct.Name, typeStr, valueStr)
			} else if typeStr != "" {
				construct.Signature = fmt.Sprintf("var %s %s", construct.Name, typeStr)
			} else if valueStr != "" {
				construct.Signature = fmt.Sprintf("var %s = %s", construct.Name, valueStr)
			} else {
				construct.Signature = fmt.Sprintf("var %s", construct.Name)
			}
		}

		constructs = append(constructs, construct)
	}

	return constructs
}

// ************************************************************************************************
// extractStructFields extracts field information from a struct type.
func (p *GoParser) extractStructFields(st *ast.StructType) []string {
	var fields []string

	if st.Fields != nil {
		for _, field := range st.Fields.List {
			fieldType := p.typeToString(field.Type)

			if len(field.Names) > 0 {
				for _, name := range field.Names {
					tagStr := ""
					if field.Tag != nil {
						tagStr = " " + field.Tag.Value
					}
					fields = append(fields, fmt.Sprintf("%s %s%s", name.Name, fieldType, tagStr))
				}
			} else {
				// Embedded field
				fields = append(fields, fieldType)
			}
		}
	}

	return fields
}

// ************************************************************************************************
// extractInterfaceMethods extracts method signatures from an interface type.
func (p *GoParser) extractInterfaceMethods(it *ast.InterfaceType) []string {
	var methods []string

	if it.Methods != nil {
		for _, method := range it.Methods.List {
			if len(method.Names) > 0 {
				// Method
				methodName := method.Names[0].Name
				methodType := p.typeToString(method.Type)
				methods = append(methods, fmt.Sprintf("%s%s", methodName, methodType))
			} else {
				// Embedded interface
				methods = append(methods, p.typeToString(method.Type))
			}
		}
	}

	return methods
}

// ************************************************************************************************
// Helper methods for generating signatures and converting types to strings.

func (p *GoParser) generateFunctionSignature(construct GoConstruct) string {
	var sig strings.Builder

	sig.WriteString("func ")

	if construct.Receiver != "" {
		sig.WriteString(fmt.Sprintf("(%s) ", construct.Receiver))
	}

	sig.WriteString(construct.Name)
	sig.WriteString("(")
	sig.WriteString(strings.Join(construct.Parameters, ", "))
	sig.WriteString(")")

	if len(construct.Returns) > 0 {
		if len(construct.Returns) == 1 {
			sig.WriteString(" " + construct.Returns[0])
		} else {
			sig.WriteString(" (" + strings.Join(construct.Returns, ", ") + ")")
		}
	}

	return sig.String()
}

func (p *GoParser) generateStructSignature(construct GoConstruct) string {
	return fmt.Sprintf("type %s struct", construct.Name)
}

func (p *GoParser) generateInterfaceSignature(construct GoConstruct) string {
	return fmt.Sprintf("type %s interface", construct.Name)
}

func (p *GoParser) typeToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + p.typeToString(t.Elt)
		}
		return "[" + p.nodeToString(t.Len) + "]" + p.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + p.typeToString(t.Key) + "]" + p.typeToString(t.Value)
	case *ast.ChanType:
		switch t.Dir {
		case ast.RECV:
			return "<-chan " + p.typeToString(t.Value)
		case ast.SEND:
			return "chan<- " + p.typeToString(t.Value)
		default:
			return "chan " + p.typeToString(t.Value)
		}
	case *ast.FuncType:
		return p.funcTypeToString(t)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.SelectorExpr:
		return p.typeToString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

func (p *GoParser) funcTypeToString(ft *ast.FuncType) string {
	var sig strings.Builder
	sig.WriteString("func(")

	if ft.Params != nil {
		var params []string
		for _, param := range ft.Params.List {
			paramType := p.typeToString(param.Type)
			params = append(params, paramType)
		}
		sig.WriteString(strings.Join(params, ", "))
	}

	sig.WriteString(")")

	if ft.Results != nil && len(ft.Results.List) > 0 {
		var results []string
		for _, result := range ft.Results.List {
			results = append(results, p.typeToString(result.Type))
		}
		if len(results) == 1 {
			sig.WriteString(" " + results[0])
		} else {
			sig.WriteString(" (" + strings.Join(results, ", ") + ")")
		}
	}

	return sig.String()
}

func (p *GoParser) nodeToString(node ast.Node) string {
	if node == nil {
		return "nil"
	}

	switch n := node.(type) {
	case *ast.Ident:
		return n.Name
	case *ast.BasicLit:
		return n.Value
	case *ast.BinaryExpr:
		return p.nodeToString(n.X) + " " + n.Op.String() + " " + p.nodeToString(n.Y)
	case *ast.UnaryExpr:
		return n.Op.String() + p.nodeToString(n.X)
	case *ast.CallExpr:
		// Handle function calls like errors.New("message")
		funcName := p.nodeToString(n.Fun)
		args := make([]string, 0, len(n.Args))
		for _, arg := range n.Args {
			args = append(args, p.nodeToString(arg))
		}
		return funcName + "(" + strings.Join(args, ", ") + ")"
	case *ast.FuncLit:
		// Handle anonymous functions like func(x int) error { ... }
		return p.funcTypeToString(n.Type)
	case *ast.SelectorExpr:
		return p.nodeToString(n.X) + "." + n.Sel.Name
	case *ast.CompositeLit:
		// Handle composite literals like []string{"a", "b"}
		typeName := ""
		if n.Type != nil {
			typeName = p.typeToString(n.Type)
		}
		if len(n.Elts) == 0 {
			return typeName + "{}"
		}
		// For complex composite literals, show abbreviated form
		if len(n.Elts) > 3 {
			return typeName + "{...}"
		}
		elts := make([]string, 0, len(n.Elts))
		for _, elt := range n.Elts {
			elts = append(elts, p.nodeToString(elt))
		}
		return typeName + "{" + strings.Join(elts, ", ") + "}"
	case *ast.ArrayType:
		return "[]" + p.typeToString(n.Elt)
	case *ast.MapType:
		return "map[" + p.typeToString(n.Key) + "]" + p.typeToString(n.Value)
	case *ast.StarExpr:
		return "&" + p.nodeToString(n.X)
	case *ast.KeyValueExpr:
		return p.nodeToString(n.Key) + ": " + p.nodeToString(n.Value)
	case *ast.IndexExpr:
		return p.nodeToString(n.X) + "[" + p.nodeToString(n.Index) + "]"
	case *ast.SliceExpr:
		low := ""
		high := ""
		if n.Low != nil {
			low = p.nodeToString(n.Low)
		}
		if n.High != nil {
			high = p.nodeToString(n.High)
		}
		return p.nodeToString(n.X) + "[" + low + ":" + high + "]"
	case *ast.TypeAssertExpr:
		return p.nodeToString(n.X) + ".(" + p.typeToString(n.Type) + ")"
	case *ast.ParenExpr:
		return "(" + p.nodeToString(n.X) + ")"
	default:
		// For complex expressions, show the type name instead of "..."
		return fmt.Sprintf("<%T>", n)
	}
}

func (p *GoParser) calculateContentHash(content string) string {
	// Simple hash based on content length and first/last characters
	if len(content) == 0 {
		return "empty"
	}

	first := content[0]
	last := content[len(content)-1]

	return fmt.Sprintf("go_%d_%c_%c", len(content), first, last)
}

// ************************************************************************************************
// generateRepomixXML generates XML output in repomix-compatible format for Go projects.
func (p *GoParser) generateRepomixXML(repositoryID, localPath string, fileAnalyses map[string]*GoFileAnalysis, packageAnalyses map[string]*GoPackageAnalysis, goFiles []string, includeNonExported bool) string {
	var xml strings.Builder

	// XML header
	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	xml.WriteString("<repository>\n")

	// File summary section
	xml.WriteString("<file_summary>\n")
	xml.WriteString("This file is a merged representation of a subset of the codebase, containing Go files with extracted language constructs.\n")
	xml.WriteString("The content has been processed where Go AST analysis extracted functions, structs, variables, constants, and types.\n\n")

	xml.WriteString("<purpose>\n")
	xml.WriteString("This file contains a Go-specific analysis of the repository's Go source code.\n")
	xml.WriteString("It is designed to be easily consumable by AI systems for Go code analysis,\n")
	xml.WriteString("code review, or other automated processes focusing on Go language constructs.\n")
	xml.WriteString("</purpose>\n\n")

	xml.WriteString("<file_format>\n")
	xml.WriteString("The content is organized as follows:\n")
	xml.WriteString("1. This summary section\n")
	xml.WriteString("2. Repository information\n")
	xml.WriteString("3. Directory structure\n")
	xml.WriteString("4. Individual file sections with constructs from each file\n")
	xml.WriteString("5. Package sections with exported constructs only\n")
	xml.WriteString("</file_format>\n\n")

	xml.WriteString("<usage_guidelines>\n")
	xml.WriteString("- This file should be treated as read-only. Any changes should be made to the\n")
	xml.WriteString("  original repository files, not this packed version.\n")
	xml.WriteString("- When processing this file, use the construct signatures to understand\n")
	xml.WriteString("  the codebase structure and relationships.\n")
	xml.WriteString("- Be aware that this file may contain sensitive information. Handle it with\n")
	xml.WriteString("  the same level of security as you would the original repository.\n")
	xml.WriteString("</usage_guidelines>\n\n")

	xml.WriteString("<notes>\n")
	xml.WriteString("- Test files (*_test.go) are excluded from this analysis\n")
	if includeNonExported {
		xml.WriteString("- All constructs (both exported and unexported) are included\n")
	} else {
		xml.WriteString("- Only exported constructs are included\n")
	}
	xml.WriteString("- Constructs are organized by type for easy navigation\n")
	xml.WriteString("- Line numbers and file locations are preserved for reference\n")
	xml.WriteString("- Go AST parsing ensures accurate construct extraction\n")
	xml.WriteString("</notes>\n\n")
	xml.WriteString("</file_summary>\n\n")

	// Directory structure
	xml.WriteString("<directory_structure>\n")
	sort.Strings(goFiles)
	for _, file := range goFiles {
		xml.WriteString(file + "\n")
	}
	xml.WriteString("</directory_structure>\n\n")

	// Individual file sections
	xml.WriteString("<files>\n")

	// Sort files for consistent output
	sortedFiles := make([]string, 0, len(fileAnalyses))
	for filePath := range fileAnalyses {
		sortedFiles = append(sortedFiles, filePath)
	}
	sort.Strings(sortedFiles)

	// Generate file-specific sections
	for _, filePath := range sortedFiles {
		fileAnalysis := fileAnalyses[filePath]

		// Group constructs by type for this file
		fileConstructsByType := make(map[string][]GoConstruct)
		for _, construct := range fileAnalysis.Constructs {
			// Filter by export status if includeNonExported is false
			if !includeNonExported && !construct.Exported {
				continue
			}
			constructType := construct.Type
			if _, exists := fileConstructsByType[constructType]; !exists {
				fileConstructsByType[constructType] = make([]GoConstruct, 0)
			}
			fileConstructsByType[constructType] = append(fileConstructsByType[constructType], construct)
		}
		if len(fileConstructsByType) == 0 {
			continue // Skip files with no constructs
		}
		xml.WriteString(fmt.Sprintf(`<file path="%s" package="%s">`+"\n", filePath, fileAnalysis.PackageName))
		xml.WriteString(fmt.Sprintf("// Package: %s\n", fileAnalysis.PackageName))
		xml.WriteString(fmt.Sprintf("// File: %s\n\n", filePath))

		// Sort construct types for consistent output
		constructTypes := []string{"const", "var", "type", "struct", "interface", "func", "method"}

		for _, constructType := range constructTypes {
			if constructs, exists := fileConstructsByType[constructType]; exists && len(constructs) > 0 {
				// Sort constructs by name for consistent output
				sort.Slice(constructs, func(i, j int) bool {
					return constructs[i].Name < constructs[j].Name
				})

				for _, construct := range constructs {
					xml.WriteString(construct.Signature)
					if constructType == "struct" && len(construct.Fields) > 0 {
						xml.WriteString(" {\n")
						for _, field := range construct.Fields {
							xml.WriteString(fmt.Sprintf("    %s\n", field))
						}
						xml.WriteString("}")
					} else if constructType == "interface" && len(construct.Methods) > 0 {
						xml.WriteString(" {\n")
						for _, method := range construct.Methods {
							xml.WriteString(fmt.Sprintf("    %s\n", method))
						}
						xml.WriteString("}")
					}
					xml.WriteString(fmt.Sprintf("  // %s:%d\n", construct.File, construct.Line))
				}
				xml.WriteString("\n")
			}
		}

		xml.WriteString("</file>\n\n")
	}

	// Package sections with exported constructs only
	sortedPackages := make([]string, 0, len(packageAnalyses))
	for packageName := range packageAnalyses {
		sortedPackages = append(sortedPackages, packageName)
	}
	sort.Strings(sortedPackages)

	for _, packageName := range sortedPackages {
		pkgAnalysis := packageAnalyses[packageName]
		xml.WriteString(fmt.Sprintf(`<package name="%s">`+"\n", packageName))
		if includeNonExported {
			xml.WriteString(fmt.Sprintf("// Package: %s (all constructs)\n\n", packageName))
		} else {
			xml.WriteString(fmt.Sprintf("// Package: %s (exported constructs only)\n\n", packageName))
		}

		// Sort construct types for consistent output
		constructTypes := []string{"const", "var", "type", "struct", "interface", "func", "method"}

		// Choose which construct collection to use
		var constructsToUse map[string][]GoConstruct
		if includeNonExported {
			constructsToUse = pkgAnalysis.Constructs
		} else {
			constructsToUse = pkgAnalysis.ExportedOnly
		}

		for _, constructType := range constructTypes {
			if constructs, exists := constructsToUse[constructType]; exists && len(constructs) > 0 {
				// Sort constructs by name for consistent output
				sort.Slice(constructs, func(i, j int) bool {
					return constructs[i].Name < constructs[j].Name
				})

				for _, construct := range constructs {
					xml.WriteString(construct.Signature)
					if constructType == "struct" && len(construct.Fields) > 0 {
						xml.WriteString(" {\n")
						for _, field := range construct.Fields {
							xml.WriteString(fmt.Sprintf("    %s\n", field))
						}
						xml.WriteString("}")
					} else if constructType == "interface" && len(construct.Methods) > 0 {
						xml.WriteString(" {\n")
						for _, method := range construct.Methods {
							xml.WriteString(fmt.Sprintf("    %s\n", method))
						}
						xml.WriteString("}")
					}
					xml.WriteString(fmt.Sprintf("  // %s:%d\n", construct.File, construct.Line))
				}
				xml.WriteString("\n")
			}
		}

		xml.WriteString("</package>\n\n")
	}

	xml.WriteString("</files>\n")
	xml.WriteString("</repository>\n")

	return xml.String()
}
