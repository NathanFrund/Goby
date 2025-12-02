package app

import (
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/announcer"
	"github.com/nfrund/goby/internal/modules/examples/chat"
	"github.com/nfrund/goby/internal/modules/examples/profile"
	"github.com/nfrund/goby/internal/modules/examples/wargame"
)

// NewModules creates and returns the list of all active modules for the application.
// This is the single source of truth for which features are enabled.
func NewModules(deps Dependencies) []module.Module {
	return []module.Module{
		chat.New(chatDeps(deps)),
		wargame.New(wargameDeps(deps)),
		profile.New(profileDeps(deps)),
		announcer.New(announcerDeps(deps)),
	}
}
