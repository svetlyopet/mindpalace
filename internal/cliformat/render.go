package cliformat

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/index"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
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

func RenderSearchHits(app *clictx.App, hits []dto.SearchHit) error {
	if app.JSON {
		return json.NewEncoder(os.Stdout).Encode(hits)
	}
	return RenderResults(app, SearchHitsToResults(hits))
}

func SearchHitsToResults(hits []dto.SearchHit) []search.Result {
	out := make([]search.Result, 0, len(hits))
	for _, h := range hits {
		created, _ := time.Parse(time.RFC3339, h.Created)
		out = append(out, search.Result{
			Meta: index.EntryMeta{
				ID:      h.ID,
				Title:   h.Title,
				Created: created,
				Type:    vault.Type(h.Type),
				Source:  h.Source,
				Tags:    h.Tags,
			},
			Score:     h.Score,
			Fragments: h.Fragments,
		})
	}
	return out
}
