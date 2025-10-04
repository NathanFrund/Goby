package web

import "embed"

// FS contains all embedded web assets, including templates and static files.
// The patterns are relative to this file's directory (the 'web' directory).
//
//go:embed static/*
var FS embed.FS
