package webtemplates

import "embed"

// Embed all HTML templates under this directory.
// Note: patterns are relative to this file's directory.
//
//go:embed layouts/*.html components/*.html partials/*.html pages/*.html
var FS embed.FS
