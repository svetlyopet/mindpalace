package search

import (
	"context"
	"testing"
	"time"

	"github.com/svetlyopet/mindpalace/internal/testenv"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestParseSince(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		in      string
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			name: "date",
			in:   "2026-07-01",
			check: func(t *testing.T, got time.Time) {
				want := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Fatalf("got %v want %v", got, want)
				}
			},
		},
		{
			name: "days",
			in:   "3d",
			check: func(t *testing.T, got time.Time) {
				want := now.AddDate(0, 0, -3)
				if got.Sub(want).Abs() > time.Second {
					t.Fatalf("got %v want ~%v", got, want)
				}
			},
		},
		{
			name: "weeks",
			in:   "2w",
			check: func(t *testing.T, got time.Time) {
				want := now.AddDate(0, 0, -14)
				if got.Sub(want).Abs() > time.Second {
					t.Fatalf("got %v want ~%v", got, want)
				}
			},
		},
		{
			name: "months",
			in:   "6mo",
			check: func(t *testing.T, got time.Time) {
				want := now.AddDate(0, -6, 0)
				if got.Sub(want).Abs() > 24*time.Hour {
					t.Fatalf("got %v want ~%v", got, want)
				}
			},
		},
		{
			name:    "empty",
			in:      "",
			wantErr: true,
		},
		{
			name:    "invalid",
			in:      "nope",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSince(tc.in, now)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, got)
		})
	}
}

func TestSearcherTagsOrdering(t *testing.T) {
	vi := testenv.TempVaultIndex(t, true)
	sr := New(vi.Index)
	counts := sr.Tags()
	if len(counts) < 2 {
		t.Fatalf("counts = %v", counts)
	}
	for i := 1; i < len(counts); i++ {
		if counts[i].Count > counts[i-1].Count {
			t.Fatalf("not sorted by count: %v", counts)
		}
		if counts[i].Count == counts[i-1].Count && counts[i].Tag < counts[i-1].Tag {
			t.Fatalf("tie-break tag order: %v", counts)
		}
	}
}

func TestSearcherMetaOnlyTagFilter(t *testing.T) {
	vi := testenv.TempVaultIndex(t, true)
	sr := New(vi.Index)
	res, err := sr.Search(context.Background(), Query{Tags: []string{"alpha"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Meta.ID != "abc123" {
		t.Fatalf("res = %+v", res)
	}
	res, err = sr.Search(context.Background(), Query{Tags: []string{"missing"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Fatalf("expected no hits, got %v", res)
	}
}

func TestSearcherFullText(t *testing.T) {
	vi := testenv.TempVaultIndex(t, false)
	e := &vault.Entry{
		ID: "ft01", Title: "Rust ownership", Created: time.Now().UTC(),
		Type: vault.TypeNote, Body: "borrow checker rules",
	}
	testenv.WriteEntry(t, vi.Vault, vi.Index, e)
	sr := New(vi.Index)
	res, err := sr.Search(context.Background(), Query{Text: "borrow", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 || res[0].Meta.ID != "ft01" {
		t.Fatalf("res = %+v", res)
	}
}
