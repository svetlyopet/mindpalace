package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/capture"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestMergeTags(t *testing.T) {
	got := MergeTags([]string{"b", "a"}, []string{"c"}, []string{"b"})
	want := []string{"a", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestUpdateTags(t *testing.T) {
	lib := testLibrary(t)
	ctx := context.Background()
	e, err := lib.UpdateTags(ctx, "abc123", []string{"Gamma", "Work Project"}, []string{"Beta"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "gamma", "work project"}
	if len(e.Tags) != len(want) {
		t.Fatalf("tags = %v, want %v", e.Tags, want)
	}
	for i := range want {
		if e.Tags[i] != want[i] {
			t.Fatalf("tags = %v, want %v", e.Tags, want)
		}
	}
}

func TestDeleteEntry(t *testing.T) {
	lib := testLibrary(t)
	ctx := context.Background()
	res, err := lib.DeleteEntry(ctx, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if res.ID != "abc123" {
		t.Fatalf("id = %q", res.ID)
	}
	if _, err := lib.Vault.Get("abc123"); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("Get after delete: %v", err)
	}
}

func TestReadEntryFileTraversal(t *testing.T) {
	lib := testLibrary(t)
	e, _ := lib.Vault.Get("abc123")
	secret := filepath.Join(filepath.Dir(e.Dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := lib.ReadEntryFile("abc123", "../../secret.txt")
	if !errors.Is(err, ErrForbiddenPath) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetEntryNotFound(t *testing.T) {
	lib := testLibrary(t)
	_, err := lib.GetEntry("missing")
	if !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestReindex(t *testing.T) {
	lib := testLibrary(t)
	stats, err := lib.Reindex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Indexed < 1 {
		t.Fatalf("indexed = %d", stats.Indexed)
	}
}

func TestCommitCapture(t *testing.T) {
	lib := testLibrary(t)
	res, err := lib.Capturer.Note(context.Background(), "body", capture.Options{Title: "New", TagsExplicit: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.CommitCapture(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	got, err := lib.GetEntry(res.Entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "New" {
		t.Fatalf("title = %q", got.Title)
	}
}

func TestCaptureOptionsFromFields(t *testing.T) {
	opts := CaptureOptionsFromFields("T", []string{"a"}, true, vault.TypeArticle, true, "my take")
	if opts.Title != "T" || opts.Type != vault.TypeArticle || !opts.FullHTML || !opts.TagsExplicit {
		t.Fatalf("opts = %+v", opts)
	}
	if opts.Thoughts != "my take" {
		t.Fatalf("thoughts = %q", opts.Thoughts)
	}
}
