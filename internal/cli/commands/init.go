package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func NewInit(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init [path]",
		Short:   "Create a new vault at a path",
		Example: "  mp vault init\n  mp vault init ~/my-vault",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := rt.VaultFlag
			if len(args) == 1 {
				root = args[0]
			}
			root, err := config.ResolveVaultRoot(root)
			if err != nil {
				return err
			}
			v, err := vault.Init(root)
			if err != nil {
				return err
			}
			if err := config.WriteDefault(v.Root()); err != nil {
				return err
			}
			fmt.Printf("Vault initialized at %s\n", v.Root())
			return nil
		},
	}
	clictx.MarkSkipVaultOpen(cmd)
	return cmd
}
