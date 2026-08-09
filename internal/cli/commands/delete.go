package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/dto"
	"golang.org/x/term"
)

var deleteYes bool

func NewDelete(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an entry and rebuild the search index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if rt.Remote() {
				e, err := rt.API.GetEntry(context.Background(), args[0])
				if err != nil {
					return err
				}
				if err := confirmDelete(e.ID, e.Title); err != nil {
					return err
				}
				res, err := rt.API.DeleteEntry(context.Background(), e.ID)
				if err != nil {
					return err
				}
				if rt.JSON {
					return json.NewEncoder(os.Stdout).Encode(res)
				}
				fmt.Printf("Deleted %s  %s\n", res.ID, res.Title)
				fmt.Printf("Reindexed %d entries, removed %d stale, took %s\n",
					res.Reindex.Indexed, res.Reindex.Removed, res.Reindex.Took)
				return nil
			}
			e, err := rt.Lib.GetEntry(args[0])
			if err != nil {
				return err
			}
			if err := confirmDelete(e.ID, e.Title); err != nil {
				return err
			}
			res, err := rt.Lib.DeleteEntry(context.Background(), e.ID)
			if err != nil {
				return err
			}
			if rt.JSON {
				return json.NewEncoder(os.Stdout).Encode(dto.DeleteFromLibrary(res.ID, res.Title, res.Reindex))
			}
			fmt.Printf("Deleted %s  %s\n", res.ID, res.Title)
			fmt.Printf("Reindexed %d entries, removed %d stale, took %s\n",
				res.Reindex.Indexed, res.Reindex.Removed, res.Reindex.Took.Round(time.Millisecond))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func confirmDelete(id, title string) error {
	if deleteYes {
		return nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("refusing to delete without --yes (non-interactive)")
	}
	fmt.Fprintf(os.Stderr, "Delete %q (%s)? [y/N]: ", title, id)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return nil
	default:
		return fmt.Errorf("delete cancelled")
	}
}
