package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func main() {
	// Define subcommands
	newModuleCmd := flag.NewFlagSet("new-module", flag.ExitOnError)
	moduleName := newModuleCmd.String("name", "", "The name of the new module (e.g., 'inventory')")

	if len(os.Args) < 2 {
		log.Println("Expected 'new-module' subcommand")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new-module":
		newModuleCmd.Parse(os.Args[2:])
		if *moduleName == "" {
			log.Fatal("Module name is required: --name=<module-name>")
		}
		if err := generateModule(*moduleName); err != nil {
			log.Fatalf("Failed to generate module: %v", err)
		}
		printNextSteps(*moduleName)
	default:
		log.Println("Expected 'new-module' subcommand")
		os.Exit(1)
	}
}

type TemplateData struct {
	Name       string
	PascalName string
}

func generateModule(name string) error {
	data := TemplateData{
		Name:       name,
		PascalName: strings.Title(name),
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

func printNextSteps(name string) {
	data := TemplateData{
		Name: name,
	}

	fmt.Printf("✅ Successfully created module '%s' in internal/modules/%s/\n\n", name, name)
	fmt.Println("Next steps:")
	fmt.Println("-----------------------------------------------------------------")

	// --- Step 1: Update dependencies.go ---
	fmt.Print("\n1. Add the dependency helper to 'internal/app/dependencies.go':\n\n")
	fmt.Printf(`
import "github.com/nfrund/goby/internal/modules/%s"

// %sDeps creates the dependency struct for the %s module.
func %sDeps(deps Dependencies) %s.Dependencies {
	return %s.Dependencies{
		Renderer: deps.Renderer,
	}
}
`, data.Name, data.Name, data.Name, data.Name, data.Name, data.Name)

	// --- Step 2: Update modules.go ---
	fmt.Print("\n2. Register the new module in 'internal/app/modules.go':\n\n")
	fmt.Printf(`
import "github.com/nfrund/goby/internal/modules/%s"

// Add to the NewModules function's return slice:
%s.New(%sDeps(deps)),
`, data.Name, data.Name, data.Name)
	fmt.Println("-----------------------------------------------------------------")
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

// {{.PascalName}}Module implements the module.Module interface.
type {{.PascalName}}Module struct {
	module.BaseModule
	renderer rendering.Renderer
}

// Dependencies holds all the services that the module requires.
type Dependencies struct{
	Renderer rendering.Renderer
}

// New creates a new instance of the module.
func New(deps Dependencies) *{{.PascalName}}Module {
	return &{{.PascalName}}Module{
		renderer: deps.Renderer,
	}
}

// Name returns the module's unique identifier.
func (m *{{.PascalName}}Module) Name() string {
	return "{{.Name}}"
}

// Register is called during application startup.
func (m *{{.PascalName}}Module) Register(reg *registry.Registry) error {
	slog.Info("Registering {{.PascalName}}Module")
	return nil
}

// Boot is called after all modules have been registered.
func (m *{{.PascalName}}Module) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	slog.Info("Booting {{.PascalName}}Module: Setting up routes...")
	handler := NewHandler(m.renderer)
	g.GET("", handler.Get)
	return nil
}
`

const handlerTemplate = `package {{.Name}}

import (
	"net/http"
	"context"
	"io"
	
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
)

// Handler manages the HTTP requests for the {{.Name}} module.
type Handler struct{
	renderer rendering.Renderer
}

// NewHandler creates a new handler.
func NewHandler(renderer rendering.Renderer) *Handler {
	return &Handler{
		renderer: renderer,
	}
}

// Get renders the main page for the {{.Name}} module.
func (h *Handler) Get(c echo.Context) error {
	pageContent := page("{{.Name}}")
	finalComponent := templ.Component(layouts.Base("{{.PascalName}}", view.GetFlashData(c).Messages, pageContent))
	return c.Render(http.StatusOK, "", finalComponent)
}

// page is an example placeholder component.
func page(name string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte("Hello from the " + name + " module!"))
		return err
	})
}
`
