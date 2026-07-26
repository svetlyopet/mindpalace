package cliformat

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/search"
)

func RenderResults(app *clictx.App, results []search.Result) error {
	if app.JSON {
		return json.NewEncoder(os.Stdout).Encode(dto.SearchHitsFrom(results))
	}
	for _, r := range results {
		line := fmt.Sprintf("%s  %s  %-10s  %s",
			r.Meta.ID,
			r.Meta.Created.Format("2006-01-02"),
			r.Meta.Type,
			r.Meta.Title,
		)
		if len(r.Meta.Tags) > 0 {
			line += "  [" + strings.Join(r.Meta.Tags, ", ") + "]"
		}
		fmt.Println(line)
		for _, f := range r.Fragments {
			fmt.Println("  …", f)
		}
	}
	return nil
}
