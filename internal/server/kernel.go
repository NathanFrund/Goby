package server

import (
	"github.com/nfrund/goby/internal/module"
	// "github.com/nfrund/goby/internal/modules/full-chat"
	"github.com/nfrund/goby/internal/modules/wargame"
)

// AppModules is the central registry of all application modules.
// The framework will iterate over this slice to register and boot each module.
var AppModules = []module.Module{
	&wargame.WargameModule{},
	// &fullchat.FullChatModule{}, // Skipping for now as requested
}
