package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/svetlyopet/mindpalace/internal/cli"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func main() {
	if err := cli.Execute(); err != nil {
		code := 1
		if errors.Is(err, vault.ErrNotFound) {
			code = 3
		}
		var u cli.UsageError
		if errors.As(err, &u) && u.IsUsage() {
			code = 2
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
