package capture

import (
	"context"
	"os"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestNoteRequiresTitle(t *testing.T) {
	t.Parallel()
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, KeywordTagger{}, config.Default().Capture)
	_, err = c.Note(context.Background(), "body text", Options{})
	if err == nil {
		t.Fatal("expected error without title")
	}
	res, err := c.Note(context.Background(), "body text", Options{Title: "My note"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry.Title != "My note" {
		t.Fatalf("title = %q", res.Entry.Title)
	}
}

func TestFileRequiresTitle(t *testing.T) {
	t.Parallel()
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, KeywordTagger{}, config.Default().Capture)
	path := t.TempDir() + "/hello.txt"
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = c.File(context.Background(), path, Options{})
	if err == nil {
		t.Fatal("expected error without title")
	}
	res, err := c.File(context.Background(), path, Options{Title: "Snippet"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry.Title != "Snippet" {
		t.Fatalf("title = %q", res.Entry.Title)
	}
}