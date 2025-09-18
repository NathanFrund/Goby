package server

import (
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/modules/wargame"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/templates"
)

// registerModules initializes all application modules that have dependencies
// and returns them for the service locator. This is the central point for
// module registration.
func registerModules(htmlHub, dataHub *hub.Hub, renderer *templates.Renderer) map[string]any {
	// Initialize the wargame module
	wargameEngine := wargame.NewEngine(htmlHub, dataHub, renderer)

	// To add a new module, you would initialize it here.
	// anotherModuleEngine := anothermodule.NewEngine(...)

	return map[string]any{
		string(registry.WargameEngineKey): wargameEngine,
		// "anothermodule.engine": anotherModuleEngine,
	}
}
