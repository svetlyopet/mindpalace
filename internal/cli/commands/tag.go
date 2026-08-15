package commands

import (
	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
)

func NewTag(rt *clictx.Runtime) *cobra.Command {
	var add, remove []string
	cmd := &cobra.Command{
		Use:   "tag <id>",
		Short: "Add or remove tags",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if rt.Remote() {
				_, err := rt.API.UpdateTags(cmd.Context(), args[0], add, remove)
				return err
			}
			_, err := rt.Lib.UpdateTags(cmd.Context(), args[0], add, remove)
			return err
		},
	}
	cmd.Flags().StringSliceVar(&add, "add", nil, "tag to add (repeatable)")
	cmd.Flags().StringSliceVar(&remove, "remove", nil, "tag to remove (repeatable)")
	cmd.MarkFlagsOneRequired("add", "remove")
	return cmd
}
