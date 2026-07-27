package search

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/svetlyopet/mindpalace/internal/index"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

type Query struct {
	Text   string
	Tags   []string
	Types  []vault.Type
	Since  time.Time
	Until  time.Time
	Domain string
	Limit  int
}

type Result struct {
	Meta      index.EntryMeta
	Score     float64
	Fragments []string
}

type TagCount struct {
	Tag   string
	Count int
}

type Searcher struct {
	ix *index.Index
}

func New(ix *index.Index) *Searcher {
	return &Searcher{ix: ix}
}

func (s *Searcher) Search(ctx context.Context, q Query) ([]Result, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if strings.TrimSpace(q.Text) == "" {
		return s.searchMetaOnly(q), nil
	}
	return s.searchFullText(ctx, q)
}

func (s *Searcher) Tags() []TagCount {
	counts := make(map[string]int)
	for _, m := range s.ix.All() {
		for _, t := range m.Tags {
			counts[t]++
		}
	}
	out := make([]TagCount, 0, len(counts))
	for tag, n := range counts {
		out = append(out, TagCount{Tag: tag, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Tag < out[j].Tag
	})
	return out
}

func (s *Searcher) searchMetaOnly(q Query) []Result {
	var out []Result
	for _, m := range s.ix.All() {
		if !matchesFilters(m, q) {
			continue
		}
		out = append(out, Result{Meta: m, Score: 0})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Meta.Created.After(out[j].Meta.Created)
	})
	if len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out
}

func (s *Searcher) searchFullText(ctx context.Context, q Query) ([]Result, error) {
	_ = ctx
	filter := buildFilterQuery(q)
	res, err := s.runFullTextSearch(q, filter, 0)
	if err != nil {
		return nil, err
	}
	if res.Total == 0 {
		res, err = s.runFullTextSearch(q, filter, 1)
		if err != nil {
			return nil, err
		}
	}
	return hitsToResults(s.ix, res), nil
}

func (s *Searcher) runFullTextSearch(q Query, filter query.Query, fuzziness int) (*bleve.SearchResult, error) {
	match := buildTextQuery(q.Text, fuzziness)
	var bq bleve.SearchRequest
	if filter != nil {
		bq = *bleve.NewSearchRequest(bleve.NewConjunctionQuery(match, filter))
	} else {
		bq = *bleve.NewSearchRequest(match)
	}
	bq.Size = q.Limit
	bq.Highlight = bleve.NewHighlightWithStyle("html")
	bq.Fields = []string{"title", "body", "extracted", "tags"}
	return s.ix.Bleve().Search(&bq)
}

var fullTextSearchFields = []string{"title", "body", "extracted", "tags"}

func buildTextQuery(text string, fuzziness int) query.Query {
	parts := make([]query.Query, 0, len(fullTextSearchFields))
	for _, field := range fullTextSearchFields {
		mq := bleve.NewMatchQuery(text)
		mq.SetField(field)
		mq.Fuzziness = fuzziness
		parts = append(parts, mq)
	}
	dq := bleve.NewDisjunctionQuery(parts...)
	dq.SetMin(1)
	return dq
}

func hitsToResults(ix *index.Index, res *bleve.SearchResult) []Result {
	out := make([]Result, 0, len(res.Hits))
	for _, hit := range res.Hits {
		meta, ok := ix.Get(hit.ID)
		if !ok {
			continue
		}
		frags := collectFragments(hit)
		out = append(out, Result{
			Meta:      meta,
			Score:     hit.Score,
			Fragments: frags,
		})
	}
	return out
}

func collectFragments(hit *search.DocumentMatch) []string {
	var frags []string
	for _, f := range hit.Fragments {
		for _, s := range f {
			frags = append(frags, stripHTMLHighlight(s))
		}
	}
	return frags
}

func stripHTMLHighlight(s string) string {
	s = strings.ReplaceAll(s, "<mark>", "")
	s = strings.ReplaceAll(s, "</mark>", "")
	return s
}

func buildFilterQuery(q Query) query.Query {
	var parts []query.Query
	for _, tag := range q.Tags {
		tq := bleve.NewTermQuery(tag)
		tq.SetField("tags")
		parts = append(parts, tq)
	}
	for _, typ := range q.Types {
		tq := bleve.NewTermQuery(string(typ))
		tq.SetField("type")
		parts = append(parts, tq)
	}
	if !q.Since.IsZero() || !q.Until.IsZero() {
		min := q.Since
		if min.IsZero() {
			min = time.Unix(0, 0)
		}
		max := q.Until
		if max.IsZero() {
			max = time.Now().Add(24 * time.Hour)
		}
		rq := bleve.NewDateRangeQuery(min, max)
		rq.SetField("created")
		parts = append(parts, rq)
	}
	if q.Domain != "" {
		tq := bleve.NewTermQuery(strings.ToLower(q.Domain))
		tq.SetField("domain")
		parts = append(parts, tq)
	}
	if len(parts) == 0 {
		return nil
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return bleve.NewConjunctionQuery(parts...)
}

func matchesFilters(m index.EntryMeta, q Query) bool {
	for _, tag := range q.Tags {
		if !contains(m.Tags, tag) {
			return false
		}
	}
	if len(q.Types) > 0 {
		ok := false
		for _, t := range q.Types {
			if m.Type == t {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if !q.Since.IsZero() && m.Created.Before(q.Since) {
		return false
	}
	if !q.Until.IsZero() && m.Created.After(q.Until) {
		return false
	}
	if q.Domain != "" {
		if domainFromSource(m.Source) != strings.ToLower(q.Domain) {
			return false
		}
	}
	return true
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func domainFromSource(source string) string {
	if source == "" {
		return ""
	}
	u, err := url.Parse(source)
	if err != nil || u.Host == "" {
		return strings.ToLower(strings.TrimPrefix(source, "www."))
	}
	return strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
}

// ParseSince turns "2w", "3d", "6mo", "2026-07-01" into a time.Time (start of day for dates).
func ParseSince(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty since value")
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	if len(s) < 2 {
		return time.Time{}, fmt.Errorf("invalid since %q", s)
	}
	numStr := s[:len(s)-1]
	unit := s[len(s)-1:]
	if strings.HasSuffix(s, "mo") {
		numStr = s[:len(s)-2]
		unit = "mo"
	}
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return time.Time{}, fmt.Errorf("invalid since %q", s)
	}
	switch unit {
	case "d":
		return now.AddDate(0, 0, -n), nil
	case "w":
		return now.AddDate(0, 0, -7*n), nil
	case "mo":
		return now.AddDate(0, -n, 0), nil
	case "y":
		return now.AddDate(-n, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid since unit in %q", s)
	}
}
