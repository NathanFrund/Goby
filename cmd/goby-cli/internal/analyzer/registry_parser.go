package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ServiceInfo represents information about a registered service
type ServiceInfo struct {
	Key         string
	Type        string
	Description string
	Module      string
	FilePath    string
	LineNumber  int
}

// RegistryKeyInfo represents a registry key declaration
type RegistryKeyInfo struct {
	VarName     string
	KeyString   string
	Type        string
	Description string
	FilePath    string
	LineNumber  int
}

// RegistrySetInfo represents a registry.Set call
type RegistrySetInfo struct {
	KeyVar      string
	FilePath    string
	LineNumber  int
	FuncContext string
}

// RegistryParser analyzes Go source files to find registry patterns
type RegistryParser struct {
	fileSet *token.FileSet
}

// NewRegistryParser creates a new registry parser
func NewRegistryParser() *RegistryParser {
	return &RegistryParser{
		fileSet: token.NewFileSet(),
	}
}

// ParseDirectory recursively parses a directory for registry patterns
func (p *RegistryParser) ParseDirectory(rootPath string) ([]RegistryKeyInfo, []RegistrySetInfo, error) {
	var keys []RegistryKeyInfo
	var sets []RegistrySetInfo

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files and test files for now
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip vendor and node_modules directories
		if strings.Contains(path, "vendor/") || strings.Contains(path, "node_modules/") {
			return nil
		}

		fileKeys, fileSets, parseErr := p.parseFile(path)
		if parseErr != nil {
			// Log error but continue parsing other files
			return nil
		}

		keys = append(keys, fileKeys...)
		sets = append(sets, fileSets...)
		return nil
	})

	return keys, sets, err
}

// parseFile parses a single Go file for registry patterns
func (p *RegistryParser) parseFile(filePath string) ([]RegistryKeyInfo, []RegistrySetInfo, error) {
	src, err := parser.ParseFile(p.fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	var keys []RegistryKeyInfo
	var sets []RegistrySetInfo

	// Walk the AST to find registry patterns
	ast.Inspect(src, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			// Look for variable declarations that might be registry keys
			if node.Tok == token.VAR {
				for _, spec := range node.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						keyInfo := p.extractRegistryKey(valueSpec, node.Doc, filePath)
						if keyInfo != nil {
							keys = append(keys, *keyInfo)
						}
					}
				}
			}
		case *ast.CallExpr:
			// Look for registry.Set calls
			setInfo := p.extractRegistrySet(node, filePath)
			if setInfo != nil {
				sets = append(sets, *setInfo)
			}
		}
		return true
	})

	return keys, sets, nil
}

// extractRegistryKey extracts registry key information from variable declarations
func (p *RegistryParser) extractRegistryKey(valueSpec *ast.ValueSpec, doc *ast.CommentGroup, filePath string) *RegistryKeyInfo {
	// Check if this is a registry.Key declaration
	for i, name := range valueSpec.Names {
		if i < len(valueSpec.Values) {
			if callExpr, ok := valueSpec.Values[i].(*ast.CallExpr); ok {
				// Check if it's a registry.Key[T] call
				if p.isRegistryKeyCall(callExpr) {
					keyString := p.extractStringLiteral(callExpr)
					typeInfo := p.extractTypeFromRegistryKey(callExpr)
					description := p.extractDescription(doc)

					pos := p.fileSet.Position(name.Pos())

					return &RegistryKeyInfo{
						VarName:     name.Name,
						KeyString:   keyString,
						Type:        typeInfo,
						Description: description,
						FilePath:    filePath,
						LineNumber:  pos.Line,
					}
				}
			}
		}
	}
	return nil
}

// extractRegistrySet extracts registry.Set call information
func (p *RegistryParser) extractRegistrySet(callExpr *ast.CallExpr, filePath string) *RegistrySetInfo {
	if p.isRegistrySetCall(callExpr) {
		// Extract the key variable name (second argument)
		if len(callExpr.Args) >= 2 {
			if ident, ok := callExpr.Args[1].(*ast.Ident); ok {
				pos := p.fileSet.Position(callExpr.Pos())

				return &RegistrySetInfo{
					KeyVar:      ident.Name,
					FilePath:    filePath,
					LineNumber:  pos.Line,
					FuncContext: p.extractFunctionContext(callExpr),
				}
			}
		}
	}
	return nil
}

// isRegistryKeyCall checks if a call expression is registry.Key[T]
func (p *RegistryParser) isRegistryKeyCall(callExpr *ast.CallExpr) bool {
	if indexExpr, ok := callExpr.Fun.(*ast.IndexExpr); ok {
		if selExpr, ok := indexExpr.X.(*ast.SelectorExpr); ok {
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				return ident.Name == "registry" && selExpr.Sel.Name == "Key"
			}
		}
	}
	return false
}

// isRegistrySetCall checks if a call expression is registry.Set
func (p *RegistryParser) isRegistrySetCall(callExpr *ast.CallExpr) bool {
	// Handle both registry.Set and registry.Set[T] patterns
	switch fun := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		if ident, ok := fun.X.(*ast.Ident); ok {
			return ident.Name == "registry" && fun.Sel.Name == "Set"
		}
	case *ast.IndexExpr:
		if selExpr, ok := fun.X.(*ast.SelectorExpr); ok {
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				return ident.Name == "registry" && selExpr.Sel.Name == "Set"
			}
		}
	}
	return false
}

// extractStringLiteral extracts string literal from function arguments
func (p *RegistryParser) extractStringLiteral(callExpr *ast.CallExpr) string {
	if len(callExpr.Args) > 0 {
		if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
			if basicLit.Kind == token.STRING {
				// Remove quotes
				return strings.Trim(basicLit.Value, `"`)
			}
		}
	}
	return ""
}

// extractTypeFromRegistryKey extracts the type parameter from registry.Key[T]
func (p *RegistryParser) extractTypeFromRegistryKey(callExpr *ast.CallExpr) string {
	if indexExpr, ok := callExpr.Fun.(*ast.IndexExpr); ok {
		return p.typeToString(indexExpr.Index)
	}
	return ""
}

// typeToString converts an AST type expression to string
func (p *RegistryParser) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.typeToString(t.X)
	case *ast.SelectorExpr:
		return p.typeToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + p.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + p.typeToString(t.Key) + "]" + p.typeToString(t.Value)
	default:
		return "unknown"
	}
}

// extractDescription extracts description from comment group
func (p *RegistryParser) extractDescription(commentGroup *ast.CommentGroup) string {
	if commentGroup == nil {
		return ""
	}

	var description strings.Builder
	for _, comment := range commentGroup.List {
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		if description.Len() > 0 {
			description.WriteString(" ")
		}
		description.WriteString(text)
	}

	return description.String()
}

// extractFunctionContext attempts to determine the function context of a registry.Set call
func (p *RegistryParser) extractFunctionContext(callExpr *ast.CallExpr) string {
	// Walk up the AST to find the containing function
	// This is a simplified version - could be enhanced to traverse parent nodes
	return "initialization"
}

// ParseFileForContext parses a file with additional context extraction
func (p *RegistryParser) ParseFileForContext(filePath string) (*FileContext, error) {
	src, err := parser.ParseFile(p.fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	context := &FileContext{
		FilePath:    filePath,
		PackageName: src.Name.Name,
		Imports:     make(map[string]string),
	}

	// Extract imports
	for _, imp := range src.Imports {
		if imp.Path != nil {
			importPath := strings.Trim(imp.Path.Value, `"`)
			alias := ""
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				// Extract package name from path
				parts := strings.Split(importPath, "/")
				if len(parts) > 0 {
					alias = parts[len(parts)-1]
				}
			}
			context.Imports[alias] = importPath
		}
	}

	return context, nil
}

// FileContext provides additional context about a parsed file
type FileContext struct {
	FilePath    string
	PackageName string
	Imports     map[string]string
}
