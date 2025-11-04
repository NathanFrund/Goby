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

var (
	moduleName  string
	minimalMode bool
)

// newModuleCmd represents the new-module command
var newModuleCmd = &cobra.Command{
	Use:   "new-module",
	Short: "Scaffold a new application module",
	Long: `Creates a new module with boilerplate for a module definition, a page-rendering handler,
and automatically registers it with the application.

By default, generates a full-featured module with pubsub integration, background services,
and topic management. Use --minimal flag to generate a simpler module with only basic dependencies.`,
	Run: func(cmd *cobra.Command, args []string) {
		if moduleName == "" {
			log.Fatal("Module name is required: --name=<module-name>")
		}

		if err := generateModule(moduleName, minimalMode); err != nil {
			log.Fatalf("Failed to generate module: %v", err)
		}

		errModules := updateModulesFile(moduleName)
		errDeps := updateDependenciesFile(moduleName, minimalMode)

		if errModules != nil || errDeps != nil {
			log.Println("Automatic file updates failed. Please add the following manually:")
			if errModules != nil {
				log.Printf(" - modules.go error: %v", errModules)
			}
			if errDeps != nil {
				log.Printf(" - dependencies.go error: %v", errDeps)
			}
			printNextSteps(moduleName, minimalMode) // Fallback to printing instructions
		} else {
			printSuccessMessage(moduleName, minimalMode)
		}
	},
}

func init() {
	rootCmd.AddCommand(newModuleCmd)
	newModuleCmd.Flags().StringVarP(&moduleName, "name", "n", "", "The name of the new module (e.g., 'inventory')")
	newModuleCmd.Flags().BoolVar(&minimalMode, "minimal", false, "Generate a minimal module with only basic dependencies (Renderer only)")
}

type TemplateData struct {
	Name       string
	PascalName string
}

func generateModule(name string, minimal bool) error {
	caser := cases.Title(language.English)
	data := TemplateData{
		Name:       name,
		PascalName: caser.String(name),
	}

	moduleDir := filepath.Join("internal", "modules", name)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	// Choose template based on minimal mode
	var moduleTempl, handlerTempl string
	if minimal {
		moduleTempl = minimalModuleTemplate
		handlerTempl = minimalHandlerTemplate
	} else {
		moduleTempl = moduleTemplate
		handlerTempl = handlerTemplate
	}

	// Generate module.go
	if err := generateFile(filepath.Join(moduleDir, "module.go"), moduleTempl, data); err != nil {
		return err
	}

	// Generate handler.go
	if err := generateFile(filepath.Join(moduleDir, "handler.go"), handlerTempl, data); err != nil {
		return err
	}

	// Generate additional files only for full mode
	if !minimal {
		// Generate subscriber.go
		if err := generateFile(filepath.Join(moduleDir, "subscriber.go"), subscriberTemplate, data); err != nil {
			return err
		}

		// Generate topics directory and topics.go
		topicsDir := filepath.Join(moduleDir, "topics")
		if err := os.MkdirAll(topicsDir, 0755); err != nil {
			return fmt.Errorf("failed to create topics directory: %w", err)
		}

		if err := generateFile(filepath.Join(topicsDir, "topics.go"), topicsTemplate, data); err != nil {
			return err
		}
	}

	// Generate README.md (different template based on mode)
	var readmeTempl string
	if minimal {
		readmeTempl = minimalReadmeTemplate
	} else {
		readmeTempl = readmeTemplate
	}

	if err := generateFile(filepath.Join(moduleDir, "README.md"), readmeTempl, data); err != nil {
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

func updateDependenciesFile(name string, minimal bool) error {
	depsPath := "internal/app/dependencies.go"
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, depsPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", depsPath, err)
	}

	newImportPath := fmt.Sprintf("github.com/nfrund/goby/internal/modules/%s", name)
	astutil.AddImport(fset, node, newImportPath)

	funcName := fmt.Sprintf("%sDeps", name)

	// Create dependency fields based on minimal mode
	var depFields []ast.Expr
	if minimal {
		// Minimal mode: only Renderer
		depFields = []ast.Expr{
			&ast.KeyValueExpr{
				Key:   ast.NewIdent("Renderer"),
				Value: &ast.SelectorExpr{X: ast.NewIdent("deps"), Sel: ast.NewIdent("Renderer")},
			},
		}
	} else {
		// Full mode: all dependencies
		depFields = []ast.Expr{
			&ast.KeyValueExpr{
				Key:   ast.NewIdent("Renderer"),
				Value: &ast.SelectorExpr{X: ast.NewIdent("deps"), Sel: ast.NewIdent("Renderer")},
			},
			&ast.KeyValueExpr{
				Key:   ast.NewIdent("Publisher"),
				Value: &ast.SelectorExpr{X: ast.NewIdent("deps"), Sel: ast.NewIdent("Publisher")},
			},
			&ast.KeyValueExpr{
				Key:   ast.NewIdent("Subscriber"),
				Value: &ast.SelectorExpr{X: ast.NewIdent("deps"), Sel: ast.NewIdent("Subscriber")},
			},
			&ast.KeyValueExpr{
				Key:   ast.NewIdent("TopicMgr"),
				Value: &ast.SelectorExpr{X: ast.NewIdent("deps"), Sel: ast.NewIdent("TopicMgr")},
			},
		}
	}

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
							Elts: depFields,
						},
					},
				},
			},
		},
	}

	node.Decls = append(node.Decls, newFunc)
	return writeASTToFile(fset, node, depsPath)
}

func printSuccessMessage(name string, minimal bool) {
	data := TemplateData{Name: name}

	if minimal {
		fmt.Printf("✅ Successfully created minimal module '%s' in internal/modules/%s/\n", name, name)
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
		fmt.Println("📋 Next steps:")
		fmt.Println("  • Customize HTTP handlers in internal/modules/" + name + "/handler.go")
		fmt.Println("  • Add more routes and functionality as needed")
		fmt.Println("  • Consider upgrading to full mode for pubsub integration")
		fmt.Println("\n🚀 Ready to start building your minimal module!")
	} else {
		fmt.Printf("✅ Successfully created full-featured module '%s' in internal/modules/%s/\n", name, name)
		fmt.Println("✅ Automatically updated application files:")
		fmt.Println("-----------------------------------------------------------------")
		fmt.Print("\n1. Added dependency helper to 'internal/app/dependencies.go':\n\n")
		fmt.Printf(`
func %sDeps(deps Dependencies) %s.Dependencies {
	return %s.Dependencies{
		Renderer:   deps.Renderer,
		Publisher:  deps.Publisher,
		Subscriber: deps.Subscriber,
		TopicMgr:   deps.TopicMgr,
	}
}
`, data.Name, data.Name, data.Name)
		fmt.Print("\n2. Registered the new module in 'internal/app/modules.go':\n\n")
		fmt.Printf(`
%s.New(%sDeps(deps)),
`, data.Name, data.Name)
		fmt.Println("\n-----------------------------------------------------------------")
		fmt.Println("📋 Next steps:")
		fmt.Println("  • Implement topics in internal/modules/" + name + "/topics/topics.go")
		fmt.Println("  • Add message handlers in internal/modules/" + name + "/subscriber.go")
		fmt.Println("  • Customize HTTP handlers in internal/modules/" + name + "/handler.go")
		fmt.Println("  • See existing chat/wargame modules for examples")
		fmt.Println("\n🚀 Ready to start building your new module!")
	}
}

func printNextSteps(name string, minimal bool) {
	data := TemplateData{Name: name}

	if minimal {
		fmt.Printf("✅ Successfully created minimal module '%s' in internal/modules/%s/\n\n", name, name)
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

%s.New(%sDeps(deps)),
`, data.Name, data.Name, data.Name)
		fmt.Println("\n-----------------------------------------------------------------")
		fmt.Println("📋 Additional steps:")
		fmt.Println("  • Customize HTTP handlers in internal/modules/" + name + "/handler.go")
		fmt.Println("  • Add more routes and functionality as needed")
		fmt.Println("  • Consider upgrading to full mode for pubsub integration")
		fmt.Println("-----------------------------------------------------------------")
	} else {
		fmt.Printf("✅ Successfully created full-featured module '%s' in internal/modules/%s/\n\n", name, name)
		fmt.Println("Next steps:")
		fmt.Println("-----------------------------------------------------------------")
		fmt.Print("\n1. Add the dependency helper to 'internal/app/dependencies.go':\n\n")
		fmt.Printf(`
import "github.com/nfrund/goby/internal/modules/%s"

func %sDeps(deps Dependencies) %s.Dependencies {
	return %s.Dependencies{
		Renderer:   deps.Renderer,
		Publisher:  deps.Publisher,
		Subscriber: deps.Subscriber,
		TopicMgr:   deps.TopicMgr,
	}
}
`, data.Name, data.Name, data.Name, data.Name)
		fmt.Print("\n2. Register the new module in 'internal/app/modules.go':\n\n")
		fmt.Printf(`
import "github.com/nfrund/goby/internal/modules/%s"

%s.New(%sDeps(deps)),
`, data.Name, data.Name, data.Name)
		fmt.Println("\n-----------------------------------------------------------------")
		fmt.Println("📋 Additional steps:")
		fmt.Println("  • Implement topics in internal/modules/" + name + "/topics/topics.go")
		fmt.Println("  • Add message handlers in internal/modules/" + name + "/subscriber.go")
		fmt.Println("  • Customize HTTP handlers in internal/modules/" + name + "/handler.go")
		fmt.Println("  • See existing chat/wargame modules for examples")
		fmt.Println("-----------------------------------------------------------------")
	}
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
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/{{.Name}}/topics"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/topicmgr"
)

// {{.PascalName}}Module implements the module.Module interface for the {{.Name}} module.
type {{.PascalName}}Module struct {
	module.BaseModule
	publisher  pubsub.Publisher
	subscriber pubsub.Subscriber
	renderer   rendering.Renderer
	topicMgr   *topicmgr.Manager
	
	// Database integration (uncomment as needed):
	// database  database.Database
	// itemStore stores.ItemStore
	// userStore stores.UserStore
	
	// Script engine integration (uncomment as needed):
	// scriptEngine script.ScriptEngine
	// scriptHelper *script.ModuleScriptHelper
	
	// Presence service integration (uncomment as needed):
	// presenceService *presence.Service
}

// Dependencies contains all the dependencies required by the {{.Name}} module.
// This struct is used for constructor injection to make dependencies explicit.
type Dependencies struct {
	// Core dependencies
	Renderer rendering.Renderer
	
	// Communication dependencies
	Publisher  pubsub.Publisher
	Subscriber pubsub.Subscriber
	TopicMgr   *topicmgr.Manager
	
	// Optional advanced dependencies (uncomment as needed):
	
	// Database integration (choose one approach):
	// Database database.Database                    // Raw database access
	// UserStore stores.UserStore                   // Store pattern for users
	// ItemStore stores.ItemStore                   // Store pattern for items
	
	// Advanced services:
	// ScriptEngine script.ScriptEngine             // For script execution
	// PresenceService *presence.Service            // For user presence tracking
	// CacheService cache.Service                   // For caching
	// EmailService email.Service                   // For email notifications
}

// New creates a new instance of {{.PascalName}}Module with the provided dependencies.
func New(deps Dependencies) *{{.PascalName}}Module {
	return &{{.PascalName}}Module{
		publisher:  deps.Publisher,
		subscriber: deps.Subscriber,
		renderer:   deps.Renderer,
		topicMgr:   deps.TopicMgr,
		
		// Database integration (uncomment as needed):
		// database:  deps.Database,
		// itemStore: deps.ItemStore,
		// userStore: deps.UserStore,
		
		// Script engine integration (uncomment as needed):
		// scriptEngine: deps.ScriptEngine,
		// scriptHelper: script.NewModuleScriptHelper(deps.ScriptEngine, "{{.Name}}", getScriptConfig()),
		
		// Presence service integration (uncomment as needed):
		// presenceService: deps.PresenceService,
	}
}

// Name returns the module name.
func (m *{{.PascalName}}Module) Name() string {
	return "{{.Name}}"
}

// Register registers the {{.Name}} module's services and topics with the registry.
func (m *{{.PascalName}}Module) Register(reg *registry.Registry) error {
	slog.Info("Registering {{.PascalName}}Module")
	
	// Register topics for this module
	if err := m.registerTopics(); err != nil {
		return fmt.Errorf("failed to register {{.Name}} topics: %w", err)
	}

	// Register any message handlers (subscriptions are set up in Boot)
	if err := m.registerHandlers(); err != nil {
		return fmt.Errorf("failed to register {{.Name}} message handlers: %w", err)
	}

	return nil
}

// registerTopics registers all topics used by this module.
func (m *{{.PascalName}}Module) registerTopics() error {
	// Register module topics with the topic manager
	if err := topics.RegisterTopics(); err != nil {
		return fmt.Errorf("failed to register topics: %w", err)
	}
	
	slog.Debug("Successfully registered {{.Name}} module topics")
	return nil
}

// registerHandlers sets up any module-specific message handler configurations.
// Note: Actual subscriptions are started in Boot() to ensure proper lifecycle management.
func (m *{{.PascalName}}Module) registerHandlers() error {
	// TODO: Add any handler registration logic here
	// This is typically used for registering handler metadata or validation
	// Actual message subscriptions are set up in the Boot method
	
	// Script engine integration example (uncomment as needed):
	// if m.scriptHelper != nil {
	// 	// Register embedded scripts for this module
	// 	provider := &{{.PascalName}}ScriptProvider{}
	// 	m.scriptHelper.RegisterEmbeddedScripts(provider)
	// 	slog.Info("Registered {{.Name}} embedded scripts")
	// }
	
	slog.Debug("{{.PascalName}}Module message handlers registered")
	return nil
}

// Boot sets up the routes and starts background services for the {{.Name}} module.
func (m *{{.PascalName}}Module) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	slog.Info("Booting {{.PascalName}}Module: Setting up routes and background services...")

	// --- Start Background Services ---
	
	// Create and start the {{.Name}} subscriber in a goroutine
	{{.Name}}Subscriber := NewSubscriber(m.subscriber, m.publisher, m.renderer)
	go {{.Name}}Subscriber.Start(ctx)
	
	// --- Register HTTP Handlers ---
	
	handler := NewHandler(m.publisher, m.renderer)
	
	// Advanced integration examples (uncomment as needed):
	// handler := NewHandlerWithDatabase(m.publisher, m.renderer, m.database, m.itemStore, m.userStore)
	// handler := NewHandlerWithPresence(m.publisher, m.renderer, m.presenceService)
	// handler := NewHandlerFull(m.publisher, m.renderer, m.database, m.presenceService, m.scriptExecutor)

	// Public routes (no authentication required)
	g.GET("/public", handler.GetPublic)
	g.GET("/status", handler.GetStatus)

	// Protected routes (require authentication)
	// The authentication middleware is typically added at the router group level
	// in the application's route setup. If you need to add it here, you would do:
	// protected := g.Group("", middleware.RequireAuth())
	g.GET("", handler.Get)
	g.POST("/action", handler.PostAction)
	
	// Advanced integration routes (uncomment as needed):
	// g.GET("/presence", handler.GetPresence)           // Presence service
	// g.POST("/presence", handler.PostPresenceUpdate)   // Presence updates
	// g.POST("/script-action", handler.PostScriptAction) // Script engine
	// g.GET("/items", handler.GetItems)                 // Database integration
	// g.POST("/items", handler.PostItem)                // Database with transactions
	// g.GET("/items/:id", handler.GetItemByID)          // Database queries

	slog.Info("{{.PascalName}}Module boot completed successfully")
	return nil
}

// Shutdown is called on application termination to gracefully shut down the module.
func (m *{{.PascalName}}Module) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down {{.PascalName}}Module...")
	
	// TODO: Add any cleanup logic here
	// - Cancel background goroutines (they should respect the context)
	// - Close any open resources
	// - Wait for pending operations to complete
	
	slog.Info("{{.PascalName}}Module shutdown completed")
	return nil
}

// Script engine integration helpers (uncomment as needed):

// getScriptConfig returns the script configuration for this module.
// func getScriptConfig() *script.ModuleScriptConfig {
// 	return &script.ModuleScriptConfig{
// 		MessageHandlers: map[string]string{
// 			topics.TopicExampleEvent.Name():  "event_processor",
// 			topics.TopicClientAction.Name(): "action_processor",
// 		},
// 		EndpointScripts: map[string]string{
// 			"/script-action": "action_handler",
// 		},
// 		DefaultLimits: script.GetDefaultSecurityLimits(),
// 		AutoExtract:   false,
// 	}
// }

// getExposedFunctions returns functions available to scripts in this module.
// func getExposedFunctions() map[string]interface{} {
// 	return map[string]interface{}{
// 		"log": func(message string) {
// 			slog.Info("Script log", "module", "{{.Name}}", "message", message)
// 		},
// 		"getCurrentTime": func() string {
// 			return time.Now().Format(time.RFC3339)
// 		},
// 		"validateData": func(data map[string]interface{}) bool {
// 			// Add your validation logic here
// 			return data != nil && len(data) > 0
// 		},
// 	}
// }

// {{.PascalName}}ScriptProvider implements the EmbeddedScriptProvider interface.
// type {{.PascalName}}ScriptProvider struct{}

// func (p *{{.PascalName}}ScriptProvider) GetEmbeddedScripts() map[string]string {
// 	return map[string]string{
// 		"event_processor": ` + "`" + `
// 			// Process {{.Name}} events
// 			function processEvent(event) {
// 				log("Processing event: " + event.action);
// 				
// 				// Add your event processing logic here
// 				if (event.action === "example_action") {
// 					return {
// 						processed: true,
// 						timestamp: getCurrentTime(),
// 						skip_processing: false
// 					};
// 				}
// 				
// 				return { processed: false };
// 			}
// 		` + "`" + `,
// 		"action_processor": ` + "`" + `
// 			// Process client actions
// 			function processAction(action) {
// 				log("Processing action: " + action.action);
// 				
// 				// Validate action data
// 				if (!validateData(action.data)) {
// 					return { error: "Invalid action data" };
// 				}
// 				
// 				// Add your action processing logic here
// 				return {
// 					result: "success",
// 					timestamp: getCurrentTime()
// 				};
// 			}
// 		` + "`" + `,
// 	}
// }

// func (p *{{.PascalName}}ScriptProvider) GetModuleName() string {
// 	return "{{.Name}}"
// }
`

const handlerTemplate = `package {{.Name}}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/modules/{{.Name}}/topics"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
)

// UserContextKey is the key used to store the authenticated user in the request context.
// This is set by the authentication middleware.
const UserContextKey = middleware.UserContextKey

// Common errors for the {{.Name}} module
var (
	ErrUnauthorized   = errors.New("authentication required")
	ErrInvalidRequest = errors.New("invalid request data")
	ErrNotFound       = errors.New("resource not found")
)

// Handler handles HTTP requests for the {{.Name}} module.
type Handler struct {
	publisher pubsub.Publisher
	renderer  rendering.Renderer
	
	// Database integration (uncomment as needed):
	// database  database.Database    // Raw database access
	// itemStore stores.ItemStore     // Store pattern for items
	// userStore stores.UserStore     // Store pattern for users
	
	// Script engine integration (uncomment as needed):
	// scriptExecutor *script.ScriptExecutor
	
	// Presence service integration (uncomment as needed):
	// presenceService *presence.Service
}

// NewHandler creates a new handler instance with the required dependencies.
func NewHandler(publisher pubsub.Publisher, renderer rendering.Renderer) *Handler {
	return &Handler{
		publisher: publisher,
		renderer:  renderer,
		
		// Database integration (uncomment as needed):
		// database:  deps.Database,
		// itemStore: deps.ItemStore,
		// userStore: deps.UserStore,
		
		// Script engine integration (uncomment as needed):
		// scriptExecutor: scriptExecutor,
		
		// Presence service integration (uncomment as needed):
		// presenceService: presenceService,
	}
}

// getUserDisplayName returns the best available display name for the user.
// Checks name and email in order, returning the first non-empty value.
// Returns an empty string if neither is available.
func getUserDisplayName(user *domain.User) string {
	switch {
	case user == nil:
		return ""
	case user.Name != nil && *user.Name != "":
		return *user.Name
	case user.Email != "":
		return user.Email
	default:
		return ""
	}
}

// getCurrentUser retrieves the authenticated user from the request context.
// Returns ErrUnauthorized if no user is found.
func (h *Handler) getCurrentUser(c echo.Context) (*domain.User, error) {
	user, ok := c.Get(UserContextKey).(*domain.User)
	if !ok || user == nil {
		return nil, ErrUnauthorized
	}
	return user, nil
}

// validateRequest performs basic request validation.
func (h *Handler) validateRequest(c echo.Context, requiredFields ...string) error {
	// Check Content-Type for POST/PUT requests
	if c.Request().Method == http.MethodPost || c.Request().Method == http.MethodPut {
		contentType := c.Request().Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/x-www-form-urlencoded") && 
		   !strings.Contains(contentType, "multipart/form-data") &&
		   !strings.Contains(contentType, "application/json") {
			return fmt.Errorf("unsupported content type: %s", contentType)
		}
	}

	// Validate required form fields
	for _, field := range requiredFields {
		if value := c.FormValue(field); value == "" {
			return fmt.Errorf("required field '%s' is missing or empty", field)
		}
	}

	return nil
}

// handleError provides consistent error handling across handlers.
func (h *Handler) handleError(c echo.Context, err error, defaultStatus int) error {
	// Log the error with request context for debugging
	slog.Error("Handler error", 
		"error", err, 
		"path", c.Request().URL.Path, 
		"method", c.Request().Method,
		"status", defaultStatus)
	
	switch {
	case errors.Is(err, ErrUnauthorized):
		return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
	case errors.Is(err, ErrInvalidRequest):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "Resource not found")
	default:
		return echo.NewHTTPError(defaultStatus, err.Error())
	}
}

// Get handles GET /{{.Name}} requests.
// This is an example of a protected route that requires authentication.
func (h *Handler) Get(c echo.Context) error {
	// Get the current user (requires authentication)
	user, err := h.getCurrentUser(c)
	if err != nil {
		return h.handleError(c, err, http.StatusUnauthorized)
	}

	// Get the best available display name (name or email)
	displayName := getUserDisplayName(user)
	pageContent := page("{{.Name}}", displayName)
	finalComponent := templ.Component(layouts.Base("{{.PascalName}}", view.GetFlashData(c).Messages, pageContent))
	return c.Render(http.StatusOK, "", finalComponent)
}

// GetPublic handles GET /{{.Name}}/public requests.
// This is an example of a public route that doesn't require authentication.
func (h *Handler) GetPublic(c echo.Context) error {
	// This route is public, but we can still check if there's a user
	user, _ := h.getCurrentUser(c)
	
	// Get the best available display name (name or email) if user is logged in
	displayName := ""
	if user != nil {
		displayName = getUserDisplayName(user)
	}
	
	pageContent := page("Public {{.Name}}", displayName)
	finalComponent := templ.Component(layouts.Base("Public {{.PascalName}}", view.GetFlashData(c).Messages, pageContent))
	return c.Render(http.StatusOK, "", finalComponent)
}

// PostAction handles POST /{{.Name}}/action requests.
// This is an example of how to handle form submissions and publish events.
func (h *Handler) PostAction(c echo.Context) error {
	// Validate request
	if err := h.validateRequest(c, "action"); err != nil {
		return h.handleError(c, fmt.Errorf("%w: %v", ErrInvalidRequest, err), http.StatusBadRequest)
	}

	// Get the current user
	user, err := h.getCurrentUser(c)
	if err != nil {
		return h.handleError(c, err, http.StatusUnauthorized)
	}

	// Extract form data
	action := c.FormValue("action")
	data := c.FormValue("data")

	// Create event payload
	event := map[string]interface{}{
		"action":    action,
		"data":      data,
		"userID":    user.Email,
		"timestamp": "2024-01-01T00:00:00Z", // TODO: Use actual timestamp
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return h.handleError(c, fmt.Errorf("failed to marshal event: %w", err), http.StatusInternalServerError)
	}

	// Publish the event
	if err := h.publisher.Publish(c.Request().Context(), pubsub.Message{
		Topic:   topics.TopicExampleEvent.Name(),
		UserID:  user.Email,
		Payload: payload,
	}); err != nil {
		return h.handleError(c, fmt.Errorf("failed to publish event: %w", err), http.StatusInternalServerError)
	}

	// Return success response
	return c.NoContent(http.StatusOK)
}

// GetStatus handles GET /{{.Name}}/status requests.
// This is an example of a JSON API endpoint.
func (h *Handler) GetStatus(c echo.Context) error {
	// This could be a public endpoint for health checks
	status := map[string]interface{}{
		"module":  "{{.Name}}",
		"status":  "active",
		"version": "1.0.0",
	}

	return c.JSON(http.StatusOK, status)
}

// Example database integration methods (uncomment and modify as needed):

// GetItems handles GET /{{.Name}}/items requests.
// This demonstrates database integration using the store pattern.
// func (h *Handler) GetItems(c echo.Context) error {
// 	// Get the current user
// 	user, err := h.getCurrentUser(c)
// 	if err != nil {
// 		return h.handleError(c, err, http.StatusUnauthorized)
// 	}
//
// 	// Use store pattern to fetch items
// 	items, err := h.itemStore.GetByUserID(c.Request().Context(), user.ID)
// 	if err != nil {
// 		return h.handleError(c, fmt.Errorf("failed to fetch items: %w", err), http.StatusInternalServerError)
// 	}
//
// 	return c.JSON(http.StatusOK, map[string]interface{}{
// 		"items": items,
// 		"count": len(items),
// 	})
// }

// PostItem handles POST /{{.Name}}/items requests.
// This demonstrates database integration with transaction handling.
// func (h *Handler) PostItem(c echo.Context) error {
// 	// Validate request
// 	if err := h.validateRequest(c, "name"); err != nil {
// 		return h.handleError(c, fmt.Errorf("%w: %v", ErrInvalidRequest, err), http.StatusBadRequest)
// 	}
//
// 	// Get the current user
// 	user, err := h.getCurrentUser(c)
// 	if err != nil {
// 		return h.handleError(c, err, http.StatusUnauthorized)
// 	}
//
// 	// Extract form data
// 	name := c.FormValue("name")
// 	description := c.FormValue("description")
//
// 	// Start database transaction
// 	tx, err := h.database.BeginTx(c.Request().Context(), nil)
// 	if err != nil {
// 		return h.handleError(c, fmt.Errorf("failed to start transaction: %w", err), http.StatusInternalServerError)
// 	}
// 	defer tx.Rollback() // Will be ignored if tx.Commit() succeeds
//
// 	// Create item using transaction
// 	item := &domain.Item{
// 		Name:        name,
// 		Description: description,
// 		UserID:      user.ID,
// 		CreatedAt:   time.Now(),
// 	}
//
// 	if err := h.itemStore.CreateWithTx(c.Request().Context(), tx, item); err != nil {
// 		return h.handleError(c, fmt.Errorf("failed to create item: %w", err), http.StatusInternalServerError)
// 	}
//
// 	// Log the creation (could also publish an event)
// 	slog.Info("Item created", "itemID", item.ID, "userID", user.ID, "name", name)
//
// 	// Commit transaction
// 	if err := tx.Commit(); err != nil {
// 		return h.handleError(c, fmt.Errorf("failed to commit transaction: %w", err), http.StatusInternalServerError)
// 	}
//
// 	// Optionally publish an event about the item creation
// 	event := map[string]interface{}{
// 		"action":    "item_created",
// 		"itemID":    item.ID,
// 		"userID":    user.ID,
// 		"timestamp": item.CreatedAt.Format(time.RFC3339),
// 	}
//
// 	payload, _ := json.Marshal(event)
// 	h.publisher.Publish(c.Request().Context(), pubsub.Message{
// 		Topic:   topics.TopicStateUpdate.Name(),
// 		UserID:  user.Email,
// 		Payload: payload,
// 	})
//
// 	return c.JSON(http.StatusCreated, item)
// }

// GetItemByID handles GET /{{.Name}}/items/:id requests.
// This demonstrates database queries with error handling.
// func (h *Handler) GetItemByID(c echo.Context) error {
// 	// Get item ID from URL parameter
// 	itemID := c.Param("id")
// 	if itemID == "" {
// 		return h.handleError(c, ErrInvalidRequest, http.StatusBadRequest)
// 	}
//
// 	// Get the current user
// 	user, err := h.getCurrentUser(c)
// 	if err != nil {
// 		return h.handleError(c, err, http.StatusUnauthorized)
// 	}
//
// 	// Fetch item from database
// 	item, err := h.itemStore.GetByID(c.Request().Context(), itemID)
// 	if err != nil {
// 		if errors.Is(err, stores.ErrNotFound) {
// 			return h.handleError(c, ErrNotFound, http.StatusNotFound)
// 		}
// 		return h.handleError(c, fmt.Errorf("failed to fetch item: %w", err), http.StatusInternalServerError)
// 	}
//
// 	// Check if user owns the item
// 	if item.UserID != user.ID {
// 		return h.handleError(c, ErrNotFound, http.StatusNotFound) // Don't reveal existence
// 	}
//
// 	return c.JSON(http.StatusOK, item)
// }

// Script engine integration examples (uncomment and modify as needed):

// PostScriptAction handles POST /{{.Name}}/script-action requests.
// This demonstrates script engine integration for custom business logic.
// func (h *Handler) PostScriptAction(c echo.Context) error {
// 	// Validate request
// 	if err := h.validateRequest(c, "action", "data"); err != nil {
// 		return h.handleError(c, fmt.Errorf("%w: %v", ErrInvalidRequest, err), http.StatusBadRequest)
// 	}
//
// 	// Get the current user
// 	user, err := h.getCurrentUser(c)
// 	if err != nil {
// 		return h.handleError(c, err, http.StatusUnauthorized)
// 	}
//
// 	// Extract form data
// 	action := c.FormValue("action")
// 	data := c.FormValue("data")
//
// 	// Prepare script execution context
// 	scriptData := map[string]interface{}{
// 		"action": action,
// 		"data":   data,
// 		"userID": user.ID,
// 		"userEmail": user.Email,
// 	}
//
// 	// Define functions available to scripts
// 	exposedFunctions := map[string]interface{}{
// 		"log": func(message string) {
// 			slog.Info("Script log", "message", message, "userID", user.ID)
// 		},
// 		"publishEvent": func(eventType string, eventData map[string]interface{}) error {
// 			payload, _ := json.Marshal(eventData)
// 			return h.publisher.Publish(c.Request().Context(), pubsub.Message{
// 				Topic:   topics.TopicExampleEvent.Name(),
// 				UserID:  user.Email,
// 				Payload: payload,
// 			})
// 		},
// 		"getCurrentTime": func() string {
// 			return time.Now().Format(time.RFC3339)
// 		},
// 	}
//
// 	// Execute script (assuming you have a script executor)
// 	// result, err := h.scriptExecutor.ExecuteScript(c.Request().Context(), "action_processor", scriptData, exposedFunctions)
// 	// if err != nil {
// 	// 	slog.Error("Script execution failed", "error", err, "action", action)
// 	// 	return h.handleError(c, fmt.Errorf("script execution failed: %w", err), http.StatusInternalServerError)
// 	// }
//
// 	// Return script result
// 	// return c.JSON(http.StatusOK, map[string]interface{}{
// 	// 	"action": action,
// 	// 	"result": result,
// 	// 	"executedAt": time.Now().Format(time.RFC3339),
// 	// })
//
// 	// Fallback response when script engine is not configured
// 	return c.JSON(http.StatusOK, map[string]interface{}{
// 		"message": "Script engine not configured",
// 		"action":  action,
// 	})
// }

// Presence service integration examples (uncomment and modify as needed):

// GetPresence handles GET /{{.Name}}/presence requests.
// This demonstrates presence service integration for real-time user tracking.
// func (h *Handler) GetPresence(c echo.Context) error {
// 	// Get current online users from presence service
// 	onlineUsers := h.presenceService.GetOnlineUsers()
// 	
// 	// Filter users relevant to this module (optional)
// 	moduleUsers := make([]string, 0)
// 	for _, userID := range onlineUsers {
// 		// Add your filtering logic here
// 		// For example, check if user has access to this module
// 		moduleUsers = append(moduleUsers, userID)
// 	}
// 	
// 	return c.JSON(http.StatusOK, map[string]interface{}{
// 		"online_users": moduleUsers,
// 		"count":        len(moduleUsers),
// 		"timestamp":    time.Now().Format(time.RFC3339),
// 	})
// }

// PostPresenceUpdate handles POST /{{.Name}}/presence requests.
// This demonstrates updating user presence status.
// func (h *Handler) PostPresenceUpdate(c echo.Context) error {
// 	// Get the current user
// 	user, err := h.getCurrentUser(c)
// 	if err != nil {
// 		return h.handleError(c, err, http.StatusUnauthorized)
// 	}
// 	
// 	// Extract presence data
// 	status := c.FormValue("status")     // e.g., "active", "away", "busy"
// 	activity := c.FormValue("activity") // e.g., "viewing_items", "editing"
// 	
// 	// Update user presence with module-specific activity
// 	presenceData := map[string]interface{}{
// 		"module":   "{{.Name}}",
// 		"status":   status,
// 		"activity": activity,
// 		"timestamp": time.Now().Format(time.RFC3339),
// 	}
// 	
// 	if err := h.presenceService.UpdateUserPresence(user.Email, presenceData); err != nil {
// 		slog.Error("Failed to update user presence", "error", err, "userID", user.Email)
// 		return h.handleError(c, fmt.Errorf("failed to update presence: %w", err), http.StatusInternalServerError)
// 	}
// 	
// 	// Optionally publish a presence update event
// 	event := map[string]interface{}{
// 		"action":   "presence_updated",
// 		"userID":   user.Email,
// 		"status":   status,
// 		"activity": activity,
// 		"module":   "{{.Name}}",
// 		"timestamp": time.Now().Format(time.RFC3339),
// 	}
// 	
// 	payload, _ := json.Marshal(event)
// 	h.publisher.Publish(c.Request().Context(), pubsub.Message{
// 		Topic:   topics.TopicStateUpdate.Name(),
// 		UserID:  user.Email,
// 		Payload: payload,
// 	})
// 	
// 	return c.JSON(http.StatusOK, map[string]interface{}{
// 		"message": "Presence updated successfully",
// 		"status":  status,
// 	})
// }

// page is an example template function that shows how to use the user's name.
// In a real application, you would use a proper templ component.
func page(name string, userName string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		greeting := "Hello"
		if userName != "" {
			greeting += ", " + userName
		}
		_, err := w.Write([]byte(greeting + "! Welcome to the " + name + " module!"))
		return err
	})
}
`

const subscriberTemplate = `package {{.Name}}

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nfrund/goby/internal/modules/{{.Name}}/topics"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	wsTopics "github.com/nfrund/goby/internal/websocket"
)

// Subscriber handles background message processing for the {{.Name}} module.
// It listens for messages on various topics and processes them accordingly.
type Subscriber struct {
	subscriber pubsub.Subscriber
	publisher  pubsub.Publisher
	renderer   rendering.Renderer
	
	// Database integration (uncomment as needed):
	// database  database.Database
	// itemStore stores.ItemStore
	// userStore stores.UserStore
	
	// Script engine integration (uncomment as needed):
	// scriptExecutor *script.ScriptExecutor
	// exposedFunctions map[string]interface{}
	
	// Presence service integration (uncomment as needed):
	// presenceService *presence.Service
}

// NewSubscriber creates a new subscriber service for the {{.Name}} module.
func NewSubscriber(sub pubsub.Subscriber, pub pubsub.Publisher, renderer rendering.Renderer) *Subscriber {
	return &Subscriber{
		subscriber: sub,
		publisher:  pub,
		renderer:   renderer,
		
		// Database integration (uncomment as needed):
		// database:  database,
		// itemStore: itemStore,
		// userStore: userStore,
		
		// Script engine integration (uncomment as needed):
		// scriptExecutor: scriptExecutor,
		// exposedFunctions: getExposedFunctions(),
		
		// Presence service integration (uncomment as needed):
		// presenceService: presenceService,
	}
}

// Start begins listening for {{.Name}}-related messages.
// This method blocks until the provided context is canceled.
func (s *Subscriber) Start(ctx context.Context) {
	slog.Info("Starting {{.Name}} module subscriber")

	// Listen for example events from this module
	go func() {
		err := s.subscriber.Subscribe(ctx, topics.TopicExampleEvent.Name(), s.handleExampleEvent)
		if err != nil && err != context.Canceled {
			slog.Error("{{.PascalName}} example event subscriber stopped with error", "error", err)
		}
	}()

	// Listen for client-initiated messages (from WebSocket clients)
	go func() {
		err := s.subscriber.Subscribe(ctx, topics.TopicClientAction.Name(), s.handleClientAction)
		if err != nil && err != context.Canceled {
			slog.Error("{{.PascalName}} client action subscriber stopped with error", "error", err)
		}
	}()

	// Listen for WebSocket client connections to send welcome messages
	go func() {
		err := s.subscriber.Subscribe(ctx, wsTopics.TopicClientReady.Name(), s.handleClientConnect)
		if err != nil && err != context.Canceled {
			slog.Error("{{.PascalName}} client connect subscriber stopped with error", "error", err)
		}
	}()

	slog.Info("{{.PascalName}} module subscriber started successfully")
}

// handleExampleEvent processes example events for this module.
func (s *Subscriber) handleExampleEvent(ctx context.Context, msg pubsub.Message) error {
	// Parse the message payload
	var event struct {
		Action    string ` + "`" + `json:"action"` + "`" + `
		Data      string ` + "`" + `json:"data"` + "`" + `
		UserID    string ` + "`" + `json:"userID,omitempty"` + "`" + `
		Timestamp string ` + "`" + `json:"timestamp,omitempty"` + "`" + `
	}

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		slog.Error("Failed to unmarshal {{.Name}} example event", "error", err)
		return nil // Don't stop the subscriber for a bad message
	}

	slog.Info("Processing {{.Name}} example event", 
		"action", event.Action, 
		"userID", event.UserID,
		"data", event.Data)

	// TODO: Add your event processing logic here
	// Example: Render a component and broadcast it
	// component := components.ExampleEvent(event.Action, event.Data, event.UserID)
	// renderedHTML, err := s.renderer.RenderComponent(ctx, component)
	// if err != nil {
	// 	slog.Error("Failed to render {{.Name}} event component", "error", err)
	// 	return err
	// }

	// Database integration example:
	// if event.Action == "item_created" {
	// 	// Update database based on the event
	// 	if err := s.itemStore.UpdateStatus(ctx, event.Data, "processed"); err != nil {
	// 		slog.Error("Failed to update item status", "error", err)
	// 		// Don't return error - continue processing
	// 	}
	// }

	// Script engine integration example:
	// if s.scriptExecutor != nil {
	// 	// Execute event processor script
	// 	scriptResult, err := s.scriptExecutor.ExecuteMessageHandler(ctx, msg.Topic, &msg, s.exposedFunctions)
	// 	if err != nil {
	// 		slog.Error("Script execution failed for {{.Name}} event", "error", err)
	// 		// Continue with normal processing even if script fails
	// 	} else if scriptResult != nil {
	// 		slog.Info("Event processor script executed successfully",
	// 			"execution_time", scriptResult.Metrics.ExecutionTime,
	// 			"result", scriptResult.Result)
	// 		
	// 		// Use script result to influence processing
	// 		if result, ok := scriptResult.Result.(map[string]interface{}); ok {
	// 			if shouldSkip, exists := result["skip_processing"]; exists && shouldSkip == true {
	// 				slog.Info("Script indicated to skip further processing")
	// 				return nil
	// 			}
	// 		}
	// 	}
	// }

	// Broadcast to all HTML clients
	// return s.publisher.Publish(ctx, pubsub.Message{
	// 	Topic:   wsTopics.TopicHTMLBroadcast.Name(),
	// 	Payload: renderedHTML,
	// })

	return nil
}

// handleClientAction processes actions initiated by WebSocket clients.
func (s *Subscriber) handleClientAction(ctx context.Context, msg pubsub.Message) error {
	// Parse the client action
	var action struct {
		Action string                 ` + "`" + `json:"action"` + "`" + `
		Data   map[string]interface{} ` + "`" + `json:"data,omitempty"` + "`" + `
		UserID string                 ` + "`" + `json:"userID,omitempty"` + "`" + `
	}

	if err := json.Unmarshal(msg.Payload, &action); err != nil {
		slog.Error("Failed to unmarshal {{.Name}} client action", "error", err)
		return nil // Don't stop the subscriber for a bad message
	}

	slog.Info("Processing {{.Name}} client action", 
		"action", action.Action, 
		"userID", action.UserID)

	// TODO: Add your client action processing logic here
	// Example actions might include:
	// - "create_item": Create a new item
	// - "update_status": Update something's status
	// - "delete_item": Remove an item

	switch action.Action {
	case "example_action":
		// Process the example action
		slog.Debug("Processing example action for {{.Name}}", "data", action.Data)
		
		// You might publish a response or update event
		// responseEvent := map[string]interface{}{
		// 	"type": "action_completed",
		// 	"action": action.Action,
		// 	"result": "success",
		// 	"userID": action.UserID,
		// }
		// 
		// payload, _ := json.Marshal(responseEvent)
		// return s.publisher.Publish(ctx, pubsub.Message{
		// 	Topic:   topics.TopicExampleEvent.Name(),
		// 	Payload: payload,
		// })

	default:
		slog.Warn("Unknown {{.Name}} client action", "action", action.Action)
	}

	return nil
}

// handleClientConnect sends a welcome message to newly connected clients.
func (s *Subscriber) handleClientConnect(ctx context.Context, msg pubsub.Message) error {
	var readyEvent struct {
		Endpoint string ` + "`" + `json:"endpoint"` + "`" + `
		UserID   string ` + "`" + `json:"userID"` + "`" + `
	}

	if err := json.Unmarshal(msg.Payload, &readyEvent); err != nil {
		slog.Error("Failed to unmarshal WebSocket ready event", "error", err)
		return nil // Don't stop the subscriber for a bad message
	}

	// Only send welcome messages to HTML clients
	if readyEvent.Endpoint == "html" && readyEvent.UserID != "" {
		slog.Debug("Sending {{.Name}} welcome message", "userID", readyEvent.UserID)

		// Presence service integration example:
		// if s.presenceService != nil {
		// 	// Update user presence to indicate they're active in this module
		// 	presenceData := map[string]interface{}{
		// 		"module":    "{{.Name}}",
		// 		"status":    "active",
		// 		"activity":  "connected",
		// 		"timestamp": time.Now().Format(time.RFC3339),
		// 	}
		// 	
		// 	if err := s.presenceService.UpdateUserPresence(readyEvent.UserID, presenceData); err != nil {
		// 		slog.Error("Failed to update user presence on connect", "error", err, "userID", readyEvent.UserID)
		// 	}
		// }

		// TODO: Create and render a welcome component
		// welcomeComponent := components.WelcomeMessage("Welcome to {{.Name}}, " + readyEvent.UserID + "!")
		// renderedHTML, err := s.renderer.RenderComponent(ctx, welcomeComponent)
		// if err != nil {
		// 	slog.Error("Failed to render {{.Name}} welcome message", "error", err, "userID", readyEvent.UserID)
		// 	return err
		// }

		// Send the welcome message directly to the user
		// directMsg := pubsub.Message{
		// 	Topic:   wsTopics.TopicHTMLDirect.Name(),
		// 	Payload: renderedHTML,
		// 	Metadata: map[string]string{
		// 		"recipient_id": readyEvent.UserID,
		// 	},
		// }
		// return s.publisher.Publish(ctx, directMsg)
	}

	return nil
}
`
const topicsTemplate = `package topics

import "github.com/nfrund/goby/internal/topicmgr"

// Module topics for the {{.Name}} system
// These topics handle {{.Name}} events, actions, and communication

var (
	// TopicExampleEvent represents example events for this module
	TopicExampleEvent = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "{{.Name}}.event.example",
		Module:      "{{.Name}}",
		Description: "Example event for the {{.Name}} module",
		Pattern:     "{{.Name}}.event.example",
		Example:     ` + "`" + `{"action":"example_action","data":"example data","userID":"user123","timestamp":"2024-01-01T00:00:00Z"}` + "`" + `,
		Metadata: map[string]interface{}{
			"event_type": "example",
			"payload_fields": []string{"action", "data", "userID", "timestamp"},
		},
	})

	// TopicClientAction represents actions initiated by clients
	TopicClientAction = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "client.{{.Name}}.action",
		Module:      "{{.Name}}",
		Description: "Client-initiated actions for the {{.Name}} module",
		Pattern:     "client.{{.Name}}.action",
		Example:     ` + "`" + `{"action":"create_item","data":{"name":"example"},"userID":"user123"}` + "`" + `,
		Metadata: map[string]interface{}{
			"source": "client",
			"action_type": "user_initiated",
			"payload_fields": []string{"action", "data", "userID"},
		},
	})

	// TopicStateUpdate represents state changes in this module
	TopicStateUpdate = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "{{.Name}}.state.update",
		Module:      "{{.Name}}",
		Description: "State update events for the {{.Name}} module",
		Pattern:     "{{.Name}}.state.update",
		Example:     ` + "`" + `{"entityID":"item123","field":"status","oldValue":"pending","newValue":"completed","timestamp":"2024-01-01T00:00:00Z"}` + "`" + `,
		Metadata: map[string]interface{}{
			"event_type": "state_change",
			"payload_fields": []string{"entityID", "field", "oldValue", "newValue", "timestamp"},
		},
	})

	// TODO: Add more topics as needed for your module
	// Examples:
	// - TopicItemCreated for when items are created
	// - TopicItemDeleted for when items are deleted
	// - TopicUserJoined for when users join
	// - TopicNotification for sending notifications
)

// RegisterTopics registers all {{.Name}} module topics with the topic manager
func RegisterTopics() error {
	manager := topicmgr.Default()
	
	topics := []topicmgr.Topic{
		TopicExampleEvent,
		TopicClientAction,
		TopicStateUpdate,
		// TODO: Add your additional topics here
	}
	
	for _, topic := range topics {
		if err := manager.Register(topic); err != nil {
			return err
		}
	}
	
	return nil
}

// MustRegisterTopics registers all {{.Name}} module topics and panics on error
func MustRegisterTopics() {
	if err := RegisterTopics(); err != nil {
		panic("failed to register {{.Name}} module topics: " + err.Error())
	}
}
`
const minimalModuleTemplate = `package {{.Name}}

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
)

// {{.PascalName}}Module implements the module.Module interface for the {{.Name}} module.
type {{.PascalName}}Module struct {
	module.BaseModule
	renderer rendering.Renderer
}

// Dependencies contains all the dependencies required by the {{.Name}} module.
// This is a minimal configuration with only basic rendering capabilities.
type Dependencies struct {
	// Core dependencies
	Renderer rendering.Renderer
	
	// To upgrade to full pubsub integration, uncomment and add:
	// Publisher  pubsub.Publisher
	// Subscriber pubsub.Subscriber
	// TopicMgr   *topicmgr.Manager
}

// New creates a new instance of {{.PascalName}}Module with the provided dependencies.
func New(deps Dependencies) *{{.PascalName}}Module {
	return &{{.PascalName}}Module{
		renderer: deps.Renderer,
	}
}

// Name returns the module name.
func (m *{{.PascalName}}Module) Name() string {
	return "{{.Name}}"
}

// Register registers the {{.Name}} module's services with the registry.
func (m *{{.PascalName}}Module) Register(reg *registry.Registry) error {
	slog.Info("Registering {{.PascalName}}Module (minimal mode)")
	
	// TODO: Add any service registration logic here
	// For pubsub integration, you would register topics and handlers here
	
	return nil
}

// Boot sets up the routes for the {{.Name}} module.
func (m *{{.PascalName}}Module) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	slog.Info("Booting {{.PascalName}}Module (minimal mode): Setting up routes...")

	// --- Register HTTP Handlers ---
	
	handler := NewHandler(m.renderer)

	// Public routes (no authentication required)
	g.GET("/public", handler.GetPublic)
	g.GET("/status", handler.GetStatus)

	// Protected routes (require authentication)
	// The authentication middleware is typically added at the router group level
	// in the application's route setup. If you need to add it here, you would do:
	// protected := g.Group("", middleware.RequireAuth())
	g.GET("", handler.Get)

	slog.Info("{{.PascalName}}Module boot completed successfully")
	return nil
}

// Shutdown is called on application termination to gracefully shut down the module.
func (m *{{.PascalName}}Module) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down {{.PascalName}}Module...")
	
	// TODO: Add any cleanup logic here
	// - Close any open resources
	// - Wait for pending operations to complete
	
	slog.Info("{{.PascalName}}Module shutdown completed")
	return nil
}
`

const minimalHandlerTemplate = `package {{.Name}}

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
)

// UserContextKey is the key used to store the authenticated user in the request context.
// This is set by the authentication middleware.
const UserContextKey = middleware.UserContextKey

// Common errors for the {{.Name}} module
var (
	ErrUnauthorized   = errors.New("authentication required")
	ErrInvalidRequest = errors.New("invalid request data")
	ErrNotFound       = errors.New("resource not found")
)

// Handler handles HTTP requests for the {{.Name}} module.
type Handler struct {
	renderer rendering.Renderer
}

// NewHandler creates a new handler instance with the required dependencies.
func NewHandler(renderer rendering.Renderer) *Handler {
	return &Handler{
		renderer: renderer,
	}
}

// getUserDisplayName returns the best available display name for the user.
// Checks name and email in order, returning the first non-empty value.
// Returns an empty string if neither is available.
func getUserDisplayName(user *domain.User) string {
	switch {
	case user == nil:
		return ""
	case user.Name != nil && *user.Name != "":
		return *user.Name
	case user.Email != "":
		return user.Email
	default:
		return ""
	}
}

// getCurrentUser retrieves the authenticated user from the request context.
// Returns ErrUnauthorized if no user is found.
func (h *Handler) getCurrentUser(c echo.Context) (*domain.User, error) {
	user, ok := c.Get(UserContextKey).(*domain.User)
	if !ok || user == nil {
		return nil, ErrUnauthorized
	}
	return user, nil
}

// handleError provides consistent error handling across handlers.
func (h *Handler) handleError(c echo.Context, err error, defaultStatus int) error {
	// Log the error with request context for debugging
	slog.Error("Handler error", 
		"error", err, 
		"path", c.Request().URL.Path, 
		"method", c.Request().Method,
		"status", defaultStatus)
	
	switch {
	case errors.Is(err, ErrUnauthorized):
		return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
	case errors.Is(err, ErrInvalidRequest):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "Resource not found")
	default:
		return echo.NewHTTPError(defaultStatus, err.Error())
	}
}

// Get handles GET /{{.Name}} requests.
// This is an example of a protected route that requires authentication.
func (h *Handler) Get(c echo.Context) error {
	// Get the current user (requires authentication)
	user, err := h.getCurrentUser(c)
	if err != nil {
		return h.handleError(c, err, http.StatusUnauthorized)
	}

	// Get the best available display name (name or email)
	displayName := getUserDisplayName(user)
	pageContent := page("{{.Name}}", displayName)
	finalComponent := templ.Component(layouts.Base("{{.PascalName}}", view.GetFlashData(c).Messages, pageContent))
	return c.Render(http.StatusOK, "", finalComponent)
}

// GetPublic handles GET /{{.Name}}/public requests.
// This is an example of a public route that doesn't require authentication.
func (h *Handler) GetPublic(c echo.Context) error {
	// This route is public, but we can still check if there's a user
	user, _ := h.getCurrentUser(c)
	
	// Get the best available display name (name or email) if user is logged in
	displayName := ""
	if user != nil {
		displayName = getUserDisplayName(user)
	}
	
	pageContent := page("Public {{.Name}}", displayName)
	finalComponent := templ.Component(layouts.Base("Public {{.PascalName}}", view.GetFlashData(c).Messages, pageContent))
	return c.Render(http.StatusOK, "", finalComponent)
}

// GetStatus handles GET /{{.Name}}/status requests.
// This is an example of a JSON API endpoint.
func (h *Handler) GetStatus(c echo.Context) error {
	// This could be a public endpoint for health checks
	status := map[string]interface{}{
		"module":  "{{.Name}}",
		"status":  "active",
		"version": "1.0.0",
		"mode":    "minimal",
	}

	return c.JSON(http.StatusOK, status)
}

// page is an example template function that shows how to use the user's name.
// In a real application, you would use a proper templ component.
func page(name string, userName string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		greeting := "Hello"
		if userName != "" {
			greeting += ", " + userName
		}
		_, err := w.Write([]byte(greeting + "! Welcome to the " + name + " module! (Minimal Mode)"))
		return err
	})
}
`
const readmeTemplate = `# {{.PascalName}} Module

This module provides {{.Name}} functionality for the Goby application.

## Overview

The {{.PascalName}} module is a full-featured module with pubsub integration, background message processing, and comprehensive HTTP handlers. It follows the established patterns from the chat and wargame modules.

## Generated Files

- ` + "`" + `module.go` + "`" + ` - Main module implementation with lifecycle management
- ` + "`" + `handler.go` + "`" + ` - HTTP request handlers with pubsub integration
- ` + "`" + `subscriber.go` + "`" + ` - Background message processing service
- ` + "`" + `topics/topics.go` + "`" + ` - Topic definitions and registration

## Features

### Core Functionality
- ✅ Module lifecycle management (Register, Boot, Shutdown)
- ✅ HTTP handlers with authentication support
- ✅ Background message processing
- ✅ Topic management and registration
- ✅ Structured logging with slog
- ✅ Proper error handling and recovery

### Communication
- ✅ Pubsub message publishing and subscribing
- ✅ WebSocket integration for real-time updates
- ✅ Topic-based message routing
- ✅ Client action processing

### Advanced Features (Commented Examples)
- 🔧 Database integration (raw access and store patterns)
- 🔧 Script engine integration for custom business logic
- 🔧 Presence service integration for user tracking
- 🔧 Caching and email service integration

## Quick Start

### 1. Implement Your Topics

Edit ` + "`" + `topics/topics.go` + "`" + ` to define your module-specific topics:

` + "```" + `go
// Add your custom topics
TopicItemCreated = topicmgr.DefineModule(topicmgr.TopicConfig{
    Name:        "{{.Name}}.item.created",
    Module:      "{{.Name}}",
    Description: "Published when a new item is created",
    // ... rest of configuration
})
` + "```" + `

### 2. Add Message Handlers

Edit ` + "`" + `subscriber.go` + "`" + ` to process your messages:

` + "```" + `go
// Add to Start() method
go func() {
    err := s.subscriber.Subscribe(ctx, topics.TopicItemCreated.Name(), s.handleItemCreated)
    if err != nil && err != context.Canceled {
        slog.Error("Item created subscriber stopped with error", "error", err)
    }
}()

// Implement the handler
func (s *Subscriber) handleItemCreated(ctx context.Context, msg pubsub.Message) error {
    // Your processing logic here
    return nil
}
` + "```" + `

### 3. Customize HTTP Handlers

Edit ` + "`" + `handler.go` + "`" + ` to add your routes and logic:

` + "```" + `go
// Add to Boot() method in module.go
g.POST("/items", handler.PostCreateItem)
g.GET("/items/:id", handler.GetItem)

// Implement in handler.go
func (h *Handler) PostCreateItem(c echo.Context) error {
    // Your handler logic here
    return nil
}
` + "```" + `

## Architecture

### Module Structure
` + "```" + `
{{.Name}}/
├── module.go          # Main module with lifecycle management
├── handler.go         # HTTP handlers with pubsub integration  
├── subscriber.go      # Background message processing
├── topics/
│   └── topics.go      # Topic definitions and registration
└── README.md          # This documentation
` + "```" + `

### Message Flow
1. **HTTP Request** → Handler → Publishes Message → Topic
2. **Topic** → Subscriber → Processes Message → May Publish Response
3. **WebSocket Client** → Publishes Action → Topic → Subscriber → Processes

### Dependencies
- ` + "`" + `Renderer` + "`" + ` - For rendering HTML components
- ` + "`" + `Publisher` + "`" + ` - For publishing messages to topics
- ` + "`" + `Subscriber` + "`" + ` - For subscribing to topic messages
- ` + "`" + `TopicMgr` + "`" + ` - For topic registration and management

## Common Patterns

### Publishing Messages from Handlers

` + "```" + `go
func (h *Handler) PostAction(c echo.Context) error {
    // Create event payload
    event := map[string]interface{}{
        "action": "item_created",
        "data":   formData,
        "userID": user.Email,
    }
    
    payload, _ := json.Marshal(event)
    return h.publisher.Publish(c.Request().Context(), pubsub.Message{
        Topic:   topics.TopicItemCreated.Name(),
        UserID:  user.Email,
        Payload: payload,
    })
}
` + "```" + `

### Processing Messages in Subscriber

` + "```" + `go
func (s *Subscriber) handleMessage(ctx context.Context, msg pubsub.Message) error {
    var event MyEvent
    if err := json.Unmarshal(msg.Payload, &event); err != nil {
        slog.Error("Failed to unmarshal event", "error", err)
        return nil // Don't stop subscriber for bad messages
    }
    
    // Process the event
    // Optionally publish response or update
    return nil
}
` + "```" + `

### Error Handling

` + "```" + `go
func (h *Handler) handleError(c echo.Context, err error, defaultStatus int) error {
    switch {
    case errors.Is(err, ErrUnauthorized):
        return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
    case errors.Is(err, ErrInvalidRequest):
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    default:
        return echo.NewHTTPError(defaultStatus, err.Error())
    }
}
` + "```" + `

## Advanced Integration

### Database Integration

Uncomment database-related code in the templates and add to Dependencies:

` + "```" + `go
type Dependencies struct {
    // ... existing dependencies
    Database  database.Database    // Raw database access
    ItemStore stores.ItemStore     // Store pattern
}
` + "```" + `

### Script Engine Integration

For custom business logic execution:

` + "```" + `go
type Dependencies struct {
    // ... existing dependencies
    ScriptEngine script.ScriptEngine
}

// In module.go
scriptHelper := script.NewModuleScriptHelper(deps.ScriptEngine, "{{.Name}}", getScriptConfig())
` + "```" + `

### Presence Service Integration

For real-time user tracking:

` + "```" + `go
type Dependencies struct {
    // ... existing dependencies
    PresenceService *presence.Service
}
` + "```" + `

## Testing

### Unit Tests
Create tests for your handlers and business logic:

` + "```" + `go
func TestHandler_PostAction(t *testing.T) {
    // Test implementation
}
` + "```" + `

### Integration Tests
Test the full message flow:

` + "```" + `go
func TestMessageFlow(t *testing.T) {
    // Test pubsub integration
}
` + "```" + `

## Troubleshooting

### Common Issues

**Module not starting:**
- Check that topics are registered correctly in ` + "`" + `topics.go` + "`" + `
- Verify dependencies are properly injected
- Check logs for registration errors

**Messages not processing:**
- Ensure subscriber is started in ` + "`" + `Boot()` + "`" + ` method
- Check topic names match between publisher and subscriber
- Verify message format matches expected structure

**HTTP handlers not working:**
- Check route registration in ` + "`" + `Boot()` + "`" + ` method
- Verify authentication middleware is properly configured
- Check error handling and logging

### Debugging

Enable debug logging:
` + "```" + `go
slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})))
` + "```" + `

### Performance

- Use structured logging instead of fmt.Printf
- Handle errors gracefully in subscribers (don't stop processing)
- Consider message batching for high-volume scenarios
- Monitor goroutine usage in background services

## Examples

See the existing ` + "`" + `chat` + "`" + ` and ` + "`" + `wargame` + "`" + ` modules for comprehensive examples of:
- Complex message processing patterns
- Advanced pubsub integration
- Background service management
- Error handling and recovery
- Script engine integration (wargame)
- Presence service integration (chat)

## Contributing

When extending this module:
1. Follow the established patterns from existing modules
2. Add comprehensive error handling and logging
3. Include tests for new functionality
4. Update this README with new features
5. Consider backward compatibility

## License

This module is part of the Goby application.
`

const minimalReadmeTemplate = `# {{.PascalName}} Module (Minimal)

This is a minimal module providing basic {{.Name}} functionality for the Goby application.

## Overview

The {{.PascalName}} module is generated in minimal mode with only basic HTTP handlers and rendering capabilities. This provides a simple starting point that can be upgraded to full pubsub integration when needed.

## Generated Files

- ` + "`" + `module.go` + "`" + ` - Basic module implementation
- ` + "`" + `handler.go` + "`" + ` - Simple HTTP request handlers

## Features

### Core Functionality
- ✅ Basic module lifecycle management
- ✅ HTTP handlers with authentication support
- ✅ Template rendering
- ✅ Error handling
- ✅ Status endpoint for health checks

### Minimal Dependencies
- ` + "`" + `Renderer` + "`" + ` - For rendering HTML components

## Quick Start

### 1. Add Your Routes

Edit the ` + "`" + `Boot()` + "`" + ` method in ` + "`" + `module.go` + "`" + `:

` + "```" + `go
// Add your routes
g.GET("/items", handler.GetItems)
g.POST("/items", handler.PostItem)
g.GET("/items/:id", handler.GetItem)
` + "```" + `

### 2. Implement Handlers

Add your handlers to ` + "`" + `handler.go` + "`" + `:

` + "```" + `go
func (h *Handler) GetItems(c echo.Context) error {
    // Your implementation here
    return c.JSON(http.StatusOK, items)
}
` + "```" + `

### 3. Add Business Logic

Extend the module with your specific functionality:

` + "```" + `go
// Add fields to Handler struct
type Handler struct {
    renderer rendering.Renderer
    // Add your services here
}
` + "```" + `

## Upgrading to Full Mode

To upgrade this minimal module to full pubsub integration:

### 1. Update Dependencies

In ` + "`" + `module.go` + "`" + `, uncomment and add:

` + "```" + `go
type Dependencies struct {
    Renderer   rendering.Renderer
    // Uncomment these for full mode:
    Publisher  pubsub.Publisher
    Subscriber pubsub.Subscriber
    TopicMgr   *topicmgr.Manager
}
` + "```" + `

### 2. Add Topics Package

Create ` + "`" + `topics/topics.go` + "`" + ` with your topic definitions.

### 3. Add Subscriber Service

Create ` + "`" + `subscriber.go` + "`" + ` for background message processing.

### 4. Update Dependencies Injection

Update ` + "`" + `internal/app/dependencies.go` + "`" + `:

` + "```" + `go
func {{.Name}}Deps(deps Dependencies) {{.Name}}.Dependencies {
    return {{.Name}}.Dependencies{
        Renderer:   deps.Renderer,
        Publisher:  deps.Publisher,
        Subscriber: deps.Subscriber,
        TopicMgr:   deps.TopicMgr,
    }
}
` + "```" + `

### 5. Regenerate with Full Mode

Alternatively, regenerate the module without the ` + "`" + `--minimal` + "`" + ` flag:

` + "```" + `bash
goby-cli new-module --name={{.Name}}
` + "```" + `

## Architecture

### Simple Structure
` + "```" + `
{{.Name}}/
├── module.go          # Basic module implementation
├── handler.go         # HTTP handlers
└── README.md          # This documentation
` + "```" + `

### Request Flow
1. **HTTP Request** → Handler → Process → Return Response

## Common Patterns

### Basic Handler Pattern

` + "```" + `go
func (h *Handler) GetResource(c echo.Context) error {
    // Get current user if needed
    user, err := h.getCurrentUser(c)
    if err != nil {
        return h.handleError(c, err, http.StatusUnauthorized)
    }
    
    // Process request
    result := processRequest(user)
    
    // Return response
    return c.JSON(http.StatusOK, result)
}
` + "```" + `

### Template Rendering

` + "```" + `go
func (h *Handler) GetPage(c echo.Context) error {
    pageContent := myPageComponent(data)
    finalComponent := templ.Component(layouts.Base("Title", nil, pageContent))
    return c.Render(http.StatusOK, "", finalComponent)
}
` + "```" + `

### Error Handling

` + "```" + `go
func (h *Handler) handleError(c echo.Context, err error, defaultStatus int) error {
    switch {
    case errors.Is(err, ErrUnauthorized):
        return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
    case errors.Is(err, ErrInvalidRequest):
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    default:
        return echo.NewHTTPError(defaultStatus, err.Error())
    }
}
` + "```" + `

## Testing

### Handler Tests

` + "```" + `go
func TestHandler_GetStatus(t *testing.T) {
    handler := NewHandler(mockRenderer)
    
    req := httptest.NewRequest(http.MethodGet, "/status", nil)
    rec := httptest.NewRecorder()
    c := echo.New().NewContext(req, rec)
    
    err := handler.GetStatus(c)
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, rec.Code)
}
` + "```" + `

## Troubleshooting

### Common Issues

**Routes not working:**
- Check route registration in ` + "`" + `Boot()` + "`" + ` method
- Verify handler method signatures
- Check for typos in route paths

**Authentication issues:**
- Ensure authentication middleware is configured at the application level
- Check ` + "`" + `getCurrentUser()` + "`" + ` implementation
- Verify user context is properly set

**Template rendering issues:**
- Check that renderer is properly injected
- Verify template component implementation
- Check for import issues with templ components

### Debugging

Add logging to your handlers:

` + "```" + `go
func (h *Handler) MyHandler(c echo.Context) error {
    slog.Info("Processing request", "path", c.Request().URL.Path)
    // ... handler logic
}
` + "```" + `

## Examples

For more advanced patterns, see:
- ` + "`" + `chat` + "`" + ` module - Full pubsub integration
- ` + "`" + `wargame` + "`" + ` module - Advanced features

## Contributing

When extending this minimal module:
1. Keep it simple initially
2. Add comprehensive error handling
3. Include tests for new functionality
4. Consider upgrading to full mode for complex features
5. Update this README with changes

## License

This module is part of the Goby application.
`
