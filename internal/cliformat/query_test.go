package cliformat

import (
	"testing"

	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/testenv"
)

func resetSearchFlags() {
	searchTags = nil
	searchTypes = nil
	searchSince = ""
	searchDomain = ""
	searchLimit = 20
}

func TestBuildQueryInvalidType(t *testing.T) {
	resetSearchFlags()
	searchTypes = []string{"not-a-type"}
	_, err := BuildQuery("hello")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildQueryWithTag(t *testing.T) {
	resetSearchFlags()
	searchTags = []string{"alpha"}
	q, err := BuildQuery("hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Tags) != 1 || q.Tags[0] != "alpha" {
		t.Fatalf("tags = %v", q.Tags)
	}
	if q.Limit != 20 {
		t.Fatalf("limit = %d", q.Limit)
	}
}

func TestRenderResultsSmoke(t *testing.T) {
	vi := testenv.TempVaultIndex(t, true)
	sr := search.New(vi.Index)
	results, err := sr.Search(t.Context(), search.Query{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	hits := dto.SearchHitsFrom(results)
	if len(hits) == 0 || hits[0].ID != "abc123" {
		t.Fatalf("hits = %+v", hits)
	}
	app := &clictx.App{JSON: false}
	if err := RenderResults(app, results); err != nil {
		t.Fatal(err)
	}
}
