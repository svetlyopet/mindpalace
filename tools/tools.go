//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/securego/gosec/v2"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
