package capture

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestSocial_unsupportedURL(t *testing.T) {
	t.Parallel()
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default().Capture
	cfg.SocialOEmbed = true
	c := New(v, KeywordTagger{}, cfg)
	_, err = c.Social(context.Background(), "https://example.com/article", Options{Title: "T"})
	if err == nil || !strings.Contains(err.Error(), "not a supported social post URL") {
		t.Fatalf("err = %v", err)
	}
}

func TestSocial_success(t *testing.T) {
	t.Parallel()
	oembedHTML, err := os.ReadFile("social/testdata/x_oembed.html")
	if err != nil {
		t.Fatal(err)
	}
	oembedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"html":        string(oembedHTML),
			"author_name": "Author",
			"author_url":  "https://x.com/testauthor",
		})
	}))
	defer oembedSrv.Close()

	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default().Capture
	cfg.SocialOEmbed = true
	c := New(v, KeywordTagger{}, cfg)
	c.client = &http.Client{Transport: oembedRoundTrip(oembedSrv.URL)}

	link := "https://x.com/u/status/463440424141459456"
	res, err := c.Social(context.Background(), link, Options{Title: "My Title", Tags: []string{"social"}, TagsExplicit: true, Thoughts: "worth revisiting"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry.Type != vault.TypeSocial {
		t.Fatalf("type = %q", res.Entry.Type)
	}
	if !strings.Contains(res.Entry.Body, "Hello from the oEmbed test post") {
		t.Fatalf("body = %q", res.Entry.Body)
	}
	extracted, err := os.ReadFile(filepath.Join(res.Entry.Dir, "extracted.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(extracted), "Hello from the oEmbed test post") {
		t.Fatalf("extracted = %q", extracted)
	}
	if res.Entry.Extra["author_name"] != "Author" {
		t.Fatalf("author_name = %v", res.Entry.Extra["author_name"])
	}
	author, ok := res.Entry.Extra["author"].(map[string]any)
	if !ok {
		t.Fatalf("missing author extra")
	}
	if author["handle"] != "testauthor" {
		t.Fatalf("author handle = %v", author["handle"])
	}
	if !strings.Contains(res.Entry.Body, "## Thoughts") || !strings.Contains(res.Entry.Body, "worth revisiting") {
		t.Fatalf("body = %q", res.Entry.Body)
	}
}

func TestPreviewSocial(t *testing.T) {
	t.Parallel()
	oembedHTML, err := os.ReadFile("social/testdata/x_oembed.html")
	if err != nil {
		t.Fatal(err)
	}
	oembedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"html":        string(oembedHTML),
			"author_name": "Preview Author",
		})
	}))
	defer oembedSrv.Close()

	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default().Capture
	cfg.SocialOEmbed = true
	c := New(v, KeywordTagger{}, cfg)
	c.client = &http.Client{Transport: oembedRoundTrip(oembedSrv.URL)}

	prev, err := c.PreviewSocial(context.Background(), "https://twitter.com/u/status/1", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if prev.Type != vault.TypeSocial {
		t.Fatalf("type = %q", prev.Type)
	}
	if !strings.Contains(prev.Title, "Preview Author") {
		t.Fatalf("title = %q", prev.Title)
	}
}
