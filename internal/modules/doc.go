// Package modules contains all self-contained application features.
//
// Each subdirectory is a module that implements the [module.Module] interface.
// Modules are automatically discovered and registered - no manual edits needed!
//
// # Auto-Discovery & Code Generation
//
// The Goby framework uses code generation (genmodules.go) to automatically:
//   - Discover all modules in internal/modules/ and internal/modules/examples/
//   - Generate type-safe dependency wiring (internal/app/dependencies.gen.go)
//   - Generate the module list (internal/app/modules.gen.go)
//
// Simply create a new module directory with a module.go file, and it will be
// automatically included in the next build. Run "go run genmodules.go" to
// regenerate the code manually.
//
// # Module Types
//
// Goby supports three module creation patterns:
//
// 1. Quick Prototype (--quick)
//    For rapid prototyping with minimal overhead.
//    - No dependency wiring required
//    - Single file with inline handlers
//    - Raw HTML responses
//
//    Usage: goby-cli new-module --quick --name=myidea
//
// 2. Minimal Module (--minimal)
//    For simple pages that need templ rendering.
//    - Uses Renderer dependency
//    - Separate handler.go file
//    - Integrates with layouts system
//
//    Usage: goby-cli new-module --minimal --name=mypage
//
// 3. Full Module (default)
//    For features needing pubsub, background services, topics.
//    - Full dependency injection
//    - Subscriber for message processing
//    - Topic definitions
//
//    Usage: goby-cli new-module --name=myfeature
//
// # Quick Start
//
// To quickly prototype a new page:
//
//	goby-cli new-module --quick --name=myidea
//
// The module will be automatically discovered and wired on the next build.
//
// # Module Interface
//
// All modules must implement:
//
//	type Module interface {
//	    Name() string
//	    Register(reg *registry.Registry) error
//	    Boot(ctx context.Context, router *echo.Group, reg *registry.Registry) error
//	    Shutdown(ctx context.Context) error
//	}
//
// Embed [module.BaseModule] to get default no-op implementations.
package modules
