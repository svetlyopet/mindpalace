package dto

import (
	"time"

	"github.com/svetlyopet/mindpalace/internal/index"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

type Entry struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Created string         `json:"created"`
	Type    string         `json:"type"`
	Source  string         `json:"source,omitempty"`
	Tags    []string       `json:"tags,omitempty"`
	Body    string         `json:"body,omitempty"`
	Dir     string         `json:"dir,omitempty"`
	Extra   map[string]any `json:"extra,omitempty"`
}

func EntryFromVault(e *vault.Entry) Entry {
	if e == nil {
		return Entry{}
	}
	return Entry{
		ID:      e.ID,
		Title:   e.Title,
		Created: e.Created.Format(time.RFC3339),
		Type:    string(e.Type),
		Source:  e.Source,
		Tags:    e.Tags,
		Body:    e.Body,
		Dir:     e.Dir,
		Extra:   e.Extra,
	}
}

type SearchHit struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Created   string   `json:"created"`
	Type      string   `json:"type"`
	Source    string   `json:"source,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Score     float64  `json:"score,omitempty"`
	Fragments []string `json:"fragments,omitempty"`
}

func SearchHitsFrom(results []search.Result) []SearchHit {
	out := make([]SearchHit, 0, len(results))
	for _, r := range results {
		out = append(out, SearchHit{
			ID:        r.Meta.ID,
			Title:     r.Meta.Title,
			Created:   r.Meta.Created.Format(time.RFC3339),
			Type:      string(r.Meta.Type),
			Source:    r.Meta.Source,
			Tags:      r.Meta.Tags,
			Score:     r.Score,
			Fragments: r.Fragments,
		})
	}
	return out
}

type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func TagCountsFrom(counts []search.TagCount) []TagCount {
	out := make([]TagCount, 0, len(counts))
	for _, c := range counts {
		out = append(out, TagCount{Tag: c.Tag, Count: c.Count})
	}
	return out
}

type DeleteResponse struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Reindex ReindexSummary `json:"reindex"`
}

type ReindexSummary struct {
	Indexed int    `json:"indexed"`
	Removed int    `json:"removed"`
	Took    string `json:"took"`
}

func DeleteFromLibrary(id, title string, stats index.RebuildStats) DeleteResponse {
	return DeleteResponse{
		ID:    id,
		Title: title,
		Reindex: ReindexSummary{
			Indexed: stats.Indexed,
			Removed: stats.Removed,
			Took:    stats.Took.Round(time.Millisecond).String(),
		},
	}
}

type CaptureResponse struct {
	Entry         Entry    `json:"entry"`
	Warnings      []string `json:"warnings,omitempty"`
	SuggestedTags []string `json:"suggested_tags,omitempty"`
}

func NewCaptureResponse(entry *vault.Entry, warnings, suggested []string) CaptureResponse {
	return CaptureResponse{
		Entry:         EntryFromVault(entry),
		Warnings:      warnings,
		SuggestedTags: suggested,
	}
}
