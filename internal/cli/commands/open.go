package commands

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
)

func NewOpen(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open <id>",
		Short: "Open entry source URL or directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := rt.Vault.Get(args[0])
			if err != nil {
				return err
			}
			target := e.Dir
			if e.Source != "" {
				target = e.Source
			}
			return openTarget(target)
		},
	}
	clictx.MarkVaultOnly(cmd)
	return cmd
}

func openTarget(target string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", target)
	case "linux":
		c = exec.Command("xdg-open", target)
	default:
		return fmt.Errorf("open: unsupported OS %s", runtime.GOOS)
	}
	return c.Run()
}
