//go:build tools

// Package tools pins dev/security CLI versions listed in go.mod (tool block).
// Run: make gosec, make govulncheck, or go run …/cmd/gosec | …/cmd/govulncheck.
package tools

import (
	_ "github.com/securego/gosec/v2"
)
