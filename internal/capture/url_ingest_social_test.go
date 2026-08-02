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

func TestURLFromHTML_socialOEmbed(t *testing.T) {
	t.Parallel()
	oembedHTML, err := os.ReadFile("social/testdata/x_oembed.html")
	if err != nil {
		t.Fatal(err)
	}
	oembedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"html":        string(oembedHTML),
			"author_name": "Test Author",
		})
	}))
	defer oembedSrv.Close()

	dir := t.TempDir()
	v, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default().Capture
	cfg.SocialOEmbed = true
	c := New(v, KeywordTagger{}, cfg)
	c.client = &http.Client{Transport: oembedRoundTrip(oembedSrv.URL)}

	link := "https://x.com/testauthor/status/463440424141459456"
	res, err := c.URLFromHTML(context.Background(), link, []byte("<html></html>"), Options{Title: "My Title"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry.Type != vault.TypeSocial {
		t.Fatalf("type = %q want social", res.Entry.Type)
	}
	if !strings.Contains(res.Entry.Body, "Hello from the oEmbed test post") {
		t.Fatalf("body = %q", res.Entry.Body)
	}
	if res.Entry.Extra["platform"] != "x" {
		t.Fatalf("platform = %v", res.Entry.Extra["platform"])
	}
	extracted, err := os.ReadFile(filepath.Join(res.Entry.Dir, "extracted.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(extracted), "Hello from the oEmbed test post") {
		t.Fatalf("extracted = %q", extracted)
	}
}

func TestURLFromHTML_socialFallbackReadability(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	v, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default().Capture
	cfg.SocialOEmbed = true
	c := New(v, KeywordTagger{}, cfg)
	c.client = &http.Client{Transport: oembedRoundTrip("")} // will fail oEmbed

	link := "https://x.com/testauthor/status/463440424141459456"
	html := `<!DOCTYPE html><html><head><title>T</title></head><body><article><p>Fallback readability text.</p></article></body></html>`
	res, err := c.URLFromHTML(context.Background(), link, []byte(html), Options{Title: "Fallback Title"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Entry.Body, "Captured excerpt") {
		t.Fatalf("body = %q", res.Entry.Body)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("expected social fallback warning")
	}
}

func TestPreviewURL_socialOEmbed(t *testing.T) {
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

	dir := t.TempDir()
	v, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default().Capture
	cfg.SocialOEmbed = true
	c := New(v, KeywordTagger{}, cfg)
	c.client = &http.Client{Transport: oembedRoundTrip(oembedSrv.URL)}

	prev, err := c.PreviewURL(context.Background(), "https://twitter.com/u/status/123", Options{})
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

func oembedRoundTrip(oembedBase string) roundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		if oembedBase == "" {
			return nil, http.ErrServerClosed
		}
		if strings.Contains(req.URL.String(), "publish.twitter.com") ||
			strings.Contains(req.URL.String(), "graph.facebook.com") {
			u := oembedBase
			req2, err := http.NewRequestWithContext(req.Context(), req.Method, u, nil)
			if err != nil {
				return nil, err
			}
			return http.DefaultTransport.RoundTrip(req2)
		}
		return http.DefaultTransport.RoundTrip(req)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
