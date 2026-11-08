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
	if _, err := os.Stat(filepath.Join(res.Entry.Dir, "embed.html")); !os.IsNotExist(err) {
		t.Fatalf("embed.html should be absent, stat err = %v", err)
	}
	if res.Entry.AllowsScripts() {
		t.Fatal("scripts should stay blocked")
	}
}

func TestAllowsScripts(t *testing.T) {
	t.Parallel()
	cfg := config.Default().Capture
	c := New(nil, nil, cfg)
	if c.allowsScripts(Options{}) {
		t.Fatal("default should block scripts")
	}
	if !c.allowsScripts(Options{AllowScripts: true}) {
		t.Fatal("option should allow scripts")
	}
	cfg.AllowScripts = true
	c = New(nil, nil, cfg)
	if !c.allowsScripts(Options{}) {
		t.Fatal("config should allow scripts")
	}
}

func TestSocial_allowScriptsWritesEmbed(t *testing.T) {
	t.Parallel()
	const htmlBody = `<blockquote class="twitter-tweet"><p>Hello from the oEmbed test post about golang.</p></blockquote><script async src="https://platform.twitter.com/widgets.js"></script>`
	oembedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"html":        htmlBody,
			"author_name": "Author",
			"author_url":  "https://x.com/testauthor",
		})
	}))
	defer oembedSrv.Close()

	var gotURL string
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default().Capture
	cfg.SocialOEmbed = true
	c := New(v, KeywordTagger{}, cfg)
	c.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Host, "publish.twitter.com") {
			gotURL = req.URL.String()
			req2, err := http.NewRequestWithContext(req.Context(), req.Method, oembedSrv.URL, nil)
			if err != nil {
				return nil, err
			}
			return http.DefaultTransport.RoundTrip(req2)
		}
		return http.DefaultTransport.RoundTrip(req)
	})}

	res, err := c.Social(context.Background(), "https://x.com/u/status/463440424141459456", Options{
		Title:        "My Title",
		Tags:         []string{"social"},
		TagsExplicit: true,
		AllowScripts: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(gotURL, "omit_script") {
		t.Fatalf("url = %q", gotURL)
	}
	if !res.Entry.AllowsScripts() {
		t.Fatal("entry should allow scripts")
	}
	data, err := os.ReadFile(filepath.Join(res.Entry.Dir, "embed.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<script") {
		t.Fatalf("embed.html = %s", data)
	}
	saved, err := vault.ReadEntry(res.Entry.Dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !saved.AllowsScripts() {
		t.Fatal("frontmatter lost allow_scripts")
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
