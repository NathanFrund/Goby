//go:build tools

package tools

// This file is a standard practice to track tool dependencies (like templ)
// which are used by 'go generate' but not imported by application code.
// This prevents 'go mod tidy' from deleting them.

import (
	_ "github.com/a-h/templ"
)
