package capture

import (
	"context"
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
