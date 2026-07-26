package web

import "embed"

// FS holds embedded UI assets (htmx, templates).
//
//go:embed static templates
var FS embed.FS
