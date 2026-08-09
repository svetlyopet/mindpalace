package commands

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
)

func NewTag(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "tag <id> [+tag ...] [-tag ...]",
		Short: "Add or remove tags",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var add, remove []string
			for _, arg := range args[1:] {
				if strings.HasPrefix(arg, "+") {
					add = append(add, strings.TrimPrefix(arg, "+"))
				} else if strings.HasPrefix(arg, "-") {
					remove = append(remove, strings.TrimPrefix(arg, "-"))
				} else {
					return clictx.Usagef("tag changes must be +tag or -tag")
				}
			}
			if rt.Remote() {
				_, err := rt.API.UpdateTags(cmd.Context(), args[0], add, remove)
				return err
			}
			_, err := rt.Lib.UpdateTags(cmd.Context(), args[0], add, remove)
			return err
		},
	}
}
