package commands

import (
	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/cli/input"
	"github.com/svetlyopet/mindpalace/internal/clictx"
)

func NewEdit(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit entry.md in your editor (config, $EDITOR, or vim)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := rt.Vault.Get(args[0])
			if err != nil {
				return err
			}
			path := e.Dir + "/entry.md"
			ed, err := rt.Config.EditorCommand()
			if err != nil {
				return err
			}
			if err := input.RunEditor(ed, path); err != nil {
				return err
			}
			if rt.Remote() {
				// Server fsnotify watcher refreshes the index.
				return nil
			}
			return rt.Lib.ReindexEntryAfterEdit(cmd.Context(), e.ID)
		},
	}
	clictx.MarkVaultOnly(cmd)
	return cmd
}
