package cmd

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/ast/astutil"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var moduleName string

// newModuleCmd represents the new-module command
var newModuleCmd = &cobra.Command{
	Use:   "new-module",
	Short: "Scaffold a new application module",
	Long: `Creates a new module with boilerplate for a module definition, a page-rendering handler,
and automatically registers it with the application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if moduleName == "" {
			log.Fatal("Module name is required: --name=<module-name>")
		}

		if err := generateModule(moduleName); err != nil {
			log.Fatalf("Failed to generate module: %v", err)
		}

		errModules := updateModulesFile(moduleName)
		errDeps := updateDependenciesFile(moduleName)

		if errModules != nil || errDeps != nil {
			log.Println("Automatic file updates failed. Please add the following manually:")
			if errModules != nil {
				log.Printf(" - modules.go error: %v", errModules)
			}
			if errDeps != nil {
				log.Printf(" - dependencies.go error: %v", errDeps)
			}
			printNextSteps(moduleName) // Fallback to printing instructions
		} else {
			printSuccessMessage(moduleName)
		}
	},
}

func init() {
	rootCmd.AddCommand(newModuleCmd)
	newModuleCmd.Flags().StringVarP(&moduleName, "name", "n", "", "The name of the new module (e.g., 'inventory')")
}

type TemplateData struct {
	Name       string
	PascalName string
}

func generateModule(name string) error {
	caser := cases.Title(language.English)
	data := TemplateData{
		Name:       name,
		PascalName: caser.String(name),
	}

	moduleDir := filepath.Join("internal", "modules", name)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	// Generate module.go
	if err := generateFile(filepath.Join(moduleDir, "module.go"), moduleTemplate, data); err != nil {
		return err
	}

	// Generate handler.go
	if err := generateFile(filepath.Join(moduleDir, "handler.go"), handlerTemplate, data); err != nil {
		return err
	}

	return nil
}

func generateFile(path string, tmpl string, data TemplateData) error {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

func updateModulesFile(name string) error {
	modulesPath := "internal/app/modules.go"
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, modulesPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", modulesPath, err)
	}

	newImportPath := fmt.Sprintf("github.com/nfrund/goby/internal/modules/%s", name)
	astutil.AddImport(fset, node, newImportPath)

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "NewModules" {
			return true
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			ret, ok := n.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			compLit, ok := ret.Results[0].(*ast.CompositeLit)
			if !ok {
				return false
			}
			newElement := &ast.CallExpr{
				Fun: &ast.SelectorExpr{X: ast.NewIdent(name), Sel: ast.NewIdent("New")},
				Args: []ast.Expr{
					ast.NewIdent(fmt.Sprintf("%sDeps(deps)", name)),
				},
			}
			compLit.Elts = append([]ast.Expr{newElement}, compLit.Elts...)
			return false
		})
		return false
	})

	return writeASTToFile(fset, node, modulesPath)
}

func updateDependenciesFile(name string) error {
	depsPath := "internal/app/dependencies.go"
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, depsPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", depsPath, err)
	}

	newImportPath := fmt.Sprintf("github.com/nfrund/goby/internal/modules/%s", name)
	astutil.AddImport(fset, node, newImportPath)

	funcName := fmt.Sprintf("%sDeps", name)
	newFunc := &ast.FuncDecl{
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: fmt.Sprintf("// %s creates the dependency struct for the %s module.", funcName, name)},
			},
		},
		Name: ast.NewIdent(funcName),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("deps")}, Type: ast.NewIdent("Dependencies")}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.SelectorExpr{X: ast.NewIdent(name), Sel: ast.NewIdent("Dependencies")}}}},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CompositeLit{
							Type: &ast.SelectorExpr{X: ast.NewIdent(name), Sel: ast.NewIdent("Dependencies")},
							Elts: []ast.Expr{
								&ast.KeyValueExpr{
									Key:   ast.NewIdent("Renderer"),
									Value: &ast.SelectorExpr{X: ast.NewIdent("deps"), Sel: ast.NewIdent("Renderer")},
								},
							},
						},
					},
				},
			},
		},
	}

	node.Decls = append(node.Decls, newFunc)
	return writeASTToFile(fset, node, depsPath)
}

func printSuccessMessage(name string) {
	data := TemplateData{Name: name}
	fmt.Printf("✅ Successfully created module '%s' in internal/modules/%s/\n", name, name)
	fmt.Println("✅ Automatically updated application files:")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Print("\n1. Added dependency helper to 'internal/app/dependencies.go':\n\n")
	fmt.Printf(`
func %sDeps(deps Dependencies) %s.Dependencies {
	return %s.Dependencies{
		Renderer: deps.Renderer,
	}
}
`, data.Name, data.Name, data.Name)
	fmt.Print("\n2. Registered the new module in 'internal/app/modules.go':\n\n")
	fmt.Printf(`
%s.New(%sDeps(deps)),
`, data.Name, data.Name)
	fmt.Println("\n-----------------------------------------------------------------")
	fmt.Println("Ready to start building your new module!")
}

func printNextSteps(name string) {
	data := TemplateData{Name: name}
	fmt.Printf("✅ Successfully created module '%s' in internal/modules/%s/\n\n", name, name)
	fmt.Println("Next steps:")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Print("\n1. Add the dependency helper to 'internal/app/dependencies.go':\n\n")
	fmt.Printf(`
import "github.com/nfrund/goby/internal/modules/%s"

func %sDeps(deps Dependencies) %s.Dependencies {
	return %s.Dependencies{
		Renderer: deps.Renderer,
	}
}
`, data.Name, data.Name, data.Name, data.Name)
	fmt.Print("\n2. Register the new module in 'internal/app/modules.go':\n\n")
	fmt.Printf(`
import "github.com/nfrund/goby/internal/modules/%s"

inventory.New(inventoryDeps(deps)),
`, data.Name)
	fmt.Println("-----------------------------------------------------------------")
}

func writeASTToFile(fset *token.FileSet, node *ast.File, filename string) error {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return fmt.Errorf("failed to format AST: %w", err)
	}
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filename, err)
	}
	return nil
}

const moduleTemplate = `package {{.Name}}

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
)

type {{.PascalName}}Module struct {
	module.BaseModule
	renderer rendering.Renderer
}

type Dependencies struct{
	Renderer rendering.Renderer
}

func New(deps Dependencies) *{{.PascalName}}Module {
	return &{{.PascalName}}Module{
		renderer: deps.Renderer,
	}
}

func (m *{{.PascalName}}Module) Name() string {
	return "{{.Name}}"
}

func (m *{{.PascalName}}Module) Register(reg *registry.Registry) error {
	slog.Info("Registering {{.PascalName}}Module")
	return nil
}

func (m *{{.PascalName}}Module) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	slog.Info("Booting {{.PascalName}}Module: Setting up routes...")
	handler := NewHandler(m.renderer)
	g.GET("", handler.Get)
	return nil
}
`

const handlerTemplate = `package {{.Name}}

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
)

type Handler struct{
	renderer rendering.Renderer
}

func NewHandler(renderer rendering.Renderer) *Handler {
	return &Handler{
		renderer: renderer,
	}
}

func (h *Handler) Get(c echo.Context) error {
	pageContent := page("{{.Name}}")
	finalComponent := templ.Component(layouts.Base("{{.PascalName}}", view.GetFlashData(c).Messages, pageContent))
	return c.Render(http.StatusOK, "", finalComponent)
}

func page(name string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte("Hello from the " + name + " module!"))
		return err
	})
}
`
