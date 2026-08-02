package capture

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestURLFromHTML(t *testing.T) {
	dir := t.TempDir()
	v, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, KeywordTagger{}, config.Default().Capture)
	html := `<!DOCTYPE html><html><head><title>Test Page</title></head><body><article><h1>Hello capture</h1><p>Body text about golang.</p></article></body></html>`
	res, err := c.URLFromHTML(context.Background(), "https://example.com/article", []byte(html), Options{Title: "Test Page"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry.Title == "" {
		t.Fatal("expected title")
	}
	if !strings.Contains(res.Entry.Body, "Captured excerpt") {
		t.Fatal("expected excerpt in body")
	}
}

func TestURLFromHTMLWithThoughts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	v, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, KeywordTagger{}, config.Default().Capture)
	html := `<!DOCTYPE html><html><head><title>Test Page</title></head><body><article><h1>Hello capture</h1><p>Body text about golang.</p></article></body></html>`
	res, err := c.URLFromHTML(context.Background(), "https://example.com/article", []byte(html), Options{
		Title:    "Test Page",
		Thoughts: "read before meeting",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Entry.Body, "Captured excerpt") || !strings.Contains(res.Entry.Body, "## Thoughts") || !strings.Contains(res.Entry.Body, "read before meeting") {
		t.Fatalf("body = %q", res.Entry.Body)
	}
	plain, err := os.ReadFile(filepath.Join(res.Entry.Dir, "extracted.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plain), "read before meeting") {
		t.Fatalf("extracted.txt = %q", plain)
	}
}
