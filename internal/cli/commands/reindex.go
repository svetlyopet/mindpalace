package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
)

func NewReindex(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "reindex",
		Short: "Rebuild search index and metadata cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			stats, err := rt.Lib.Reindex(context.Background())
			if err != nil {
				return err
			}
			fmt.Printf("Reindexed %d entries, removed %d stale, took %s\n",
				stats.Indexed, stats.Removed, stats.Took.Round(1e6))
			return nil
		},
	}
}
