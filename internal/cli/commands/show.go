package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/cli/output"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/dto"
)

func NewShow(rt *clictx.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := rt.Lib.GetEntry(args[0])
			if err != nil {
				return err
			}
			if rt.JSON {
				return json.NewEncoder(os.Stdout).Encode(dto.EntryFromVault(e))
			}
			fmt.Printf("%s  %s\n", e.ID, e.Title)
			fmt.Printf("%s  %s\n", e.Created.Format(time.RFC3339), e.Type)
			if e.Source != "" {
				fmt.Println(e.Source)
			}
			if len(e.Tags) > 0 {
				fmt.Println("tags:", strings.Join(e.Tags, ", "))
			}
			fmt.Println()
			rendered, err := output.RenderMarkdown(e.Body, output.TerminalWidth(80))
			if err != nil {
				return err
			}
			fmt.Print(rendered)
			return nil
		},
	}
}
