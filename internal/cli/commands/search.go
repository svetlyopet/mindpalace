package commands

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/cliformat"
)

func NewSearch(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search with optional filters",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := cliformat.BuildQuery(strings.Join(args, " "))
			if err != nil {
				return err
			}
			results, err := rt.Searcher.Search(context.Background(), q)
			if err != nil {
				return err
			}
			return cliformat.RenderResults(rt.App, results)
		},
	}
}

func NewList(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List entries (filter without full-text query)",
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := cliformat.BuildQuery("")
			if err != nil {
				return err
			}
			results, err := rt.Searcher.Search(context.Background(), q)
			if err != nil {
				return err
			}
			return cliformat.RenderResults(rt.App, results)
		},
	}
}
