package cliformat

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

var (
	searchTags   []string
	searchTypes  []string
	searchSince  string
	searchDomain string
	searchLimit  int
)

func BuildQuery(text string) (search.Query, error) {
	for _, t := range searchTypes {
		if !vault.Type(t).Valid() {
			return search.Query{}, clictx.Usagef("invalid type %q", t)
		}
	}
	return search.QueryFromListParams(text, searchTags, searchTypes, searchSince, searchDomain, searchLimit, time.Now())
}

func RegisterSearchFlags(searchCmd, listCmd *cobra.Command) {
	for _, c := range []*cobra.Command{searchCmd, listCmd} {
		c.Flags().StringSliceVar(&searchTags, "tag", nil, "filter by tag (AND)")
		c.Flags().StringSliceVar(&searchTypes, "type", nil, "filter by type")
		c.Flags().StringVar(&searchSince, "since", "", "created since (e.g. 2w, 2026-07-01)")
		c.Flags().StringVar(&searchDomain, "domain", "", "filter by source domain")
		c.Flags().IntVar(&searchLimit, "limit", 20, "max results")
	}
}
