package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
)

func NewTags(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "tags",
		Short: "List tags with counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if rt.Remote() {
				counts, err := rt.API.Tags(context.Background())
				if err != nil {
					return err
				}
				if rt.JSON {
					return json.NewEncoder(os.Stdout).Encode(counts)
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				for _, tc := range counts {
					fmt.Fprintf(w, "%s\t%d\n", tc.Tag, tc.Count)
				}
				return w.Flush()
			}
			counts := rt.Searcher.Tags()
			if rt.JSON {
				return json.NewEncoder(os.Stdout).Encode(counts)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, tc := range counts {
				fmt.Fprintf(w, "%s\t%d\n", tc.Tag, tc.Count)
			}
			return w.Flush()
		},
	}
}
