//go:build e2e

package e2e_test

import (
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/dto"
)

func TestCLISearchAndListFilters(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	noteOut := mustRun(t, dir, "add", "note", "-m", "alpha beta filter body", "--title", "Filter note", "--tags", "alpha,beta", "--type", "note")
	noteID := createdID(t, noteOut)
	snipOut := mustRun(t, dir, "add", "note", "-m", "snippet only body", "--title", "Filter snippet", "--tags", "alpha", "--type", "snippet")
	snipID := createdID(t, snipOut)

	all := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "list"))
	if len(all) < 2 {
		t.Fatalf("list = %+v", all)
	}

	byType := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "search", "--type", "snippet", "snippet"))
	if len(byType) != 1 || byType[0].ID != snipID {
		t.Fatalf("type hits = %+v", byType)
	}

	bothTags := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "list", "--tag", "alpha", "--tag", "beta"))
	if len(bothTags) != 1 || bothTags[0].ID != noteID {
		t.Fatalf("AND tag hits = %+v", bothTags)
	}

	recent := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "list", "--since", "2w"))
	if len(recent) < 2 {
		t.Fatalf("since 2w = %+v", recent)
	}
	future := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "list", "--since", "2099-01-01"))
	if len(future) != 0 {
		t.Fatalf("future since = %+v", future)
	}

	limited := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "list", "--limit", "1"))
	if len(limited) != 1 {
		t.Fatalf("limit = %+v", limited)
	}

	if _, errOut, code := runMP(t, dir, nil, "", "tag", noteID, "--remove", "beta"); code != 0 {
		t.Fatalf("tag remove code=%d err=%q", code, errOut)
	}
	after := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "list", "--tag", "beta"))
	if len(after) != 0 {
		t.Fatalf("tag beta after remove = %+v", after)
	}

	text := mustRun(t, dir, "tags")
	if !strings.Contains(text, "alpha") {
		t.Fatalf("tags text = %q", text)
	}
	counts := decodeJSON[[]dto.TagCount](t, mustRun(t, dir, "--json", "tags"))
	if !tagCountHas(counts, "alpha") {
		t.Fatalf("tag counts = %+v", counts)
	}

	shown := stripANSI(mustRun(t, dir, "show", noteID))
	if !strings.Contains(shown, "Filter note") || !strings.Contains(shown, "alpha beta filter body") {
		t.Fatalf("show = %q", shown)
	}
}

func tagCountHas(counts []dto.TagCount, tag string) bool {
	for _, c := range counts {
		if c.Tag == tag && c.Count > 0 {
			return true
		}
	}
	return false
}
