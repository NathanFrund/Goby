// Package modules contains all self-contained application features.
//
// Each subdirectory is a module that implements the [module.Module] interface.
// Modules are registered in internal/app/modules.go and are loaded at startup.
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
// Then manually add to internal/app/modules.go:
//
//	import "github.com/nfrund/goby/internal/modules/myidea"
//
//	func NewModules(deps Dependencies) []module.Module {
//	    return []module.Module{
//	        myidea.New(),
//	        // ... other modules
//	    }
//	}
//
// # Module Interface
//
// All modules must implement:
//
//	type Module interface {
//	    Name() string
//	    Register(reg *Registry) error
//	    Boot(ctx context.Context, router *echo.Group, reg *Registry) error
//	    Shutdown(ctx context.Context) error
//	}
//
// Embed [module.BaseModule] to get default no-op implementations.
package modules
