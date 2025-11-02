package app

import (
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/chat"
	"github.com/nfrund/goby/internal/modules/wargame"
)

// NewModules creates and returns the list of all active modules for the application.
// This is the single source of truth for which features are enabled.
func NewModules(deps Dependencies) []module.Module {
	return []module.Module{
		// Add new application modules here.
		wargame.New(wargameDeps(deps)),
		chat.New(chatDeps(deps)),
	}
}
