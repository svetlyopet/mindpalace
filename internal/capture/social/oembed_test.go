package social

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetch_xOEmbed(t *testing.T) {
	t.Parallel()
	const html = `<blockquote class="twitter-tweet"><p>Network fetch test</p></blockquote>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(oEmbedResponse{
			HTML:       html,
			AuthorName: "Net Author",
		})
	}))
	defer srv.Close()

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		u := srv.URL + "/oembed"
		req2, err := http.NewRequestWithContext(req.Context(), req.Method, u, nil)
		if err != nil {
			return nil, err
		}
		return http.DefaultTransport.RoundTrip(req2)
	})}

	post, err := Fetch(context.Background(), client, PlatformX, "https://twitter.com/u/status/99")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(post.Text, "Network fetch test") {
		t.Fatalf("text = %q", post.Text)
	}
	if post.Author.DisplayName != "Net Author" {
		t.Fatalf("author = %q", post.Author.DisplayName)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetch_emptyHTML(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(oEmbedResponse{HTML: "  "})
	}))
	defer srv.Close()
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		u := srv.URL
		req2, err := http.NewRequestWithContext(req.Context(), req.Method, u, nil)
		if err != nil {
			return nil, err
		}
		return http.DefaultTransport.RoundTrip(req2)
	})}
	_, err := Fetch(context.Background(), client, PlatformX, "https://twitter.com/u/status/1")
	if err == nil {
		t.Fatal("expected error for empty html")
	}
}

func TestFetchOEmbedJSON_httpError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer srv.Close()
	_, err := fetchOEmbedJSON(context.Background(), srv.Client(), srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtractPostID(t *testing.T) {
	t.Parallel()
	if got := extractPostID(PlatformX, "https://twitter.com/u/status/463440424141459456"); got != "463440424141459456" {
		t.Fatalf("post id = %q", got)
	}
	if got := extractPostID(PlatformFacebook, "https://www.facebook.com/watch?v=999"); got != "999" {
		t.Fatalf("post id = %q", got)
	}
}

func TestParseOGImage(t *testing.T) {
	t.Parallel()
	html := `<html><head><meta property="og:image" content="https://cdn.example/poster.jpg"></head></html>`
	if got := parseOGImage(html); got != "https://cdn.example/poster.jpg" {
		t.Fatalf("og:image = %q", got)
	}
}

func TestSaveMedia_downloadsImage(t *testing.T) {
	t.Parallel()
	const imgBytes = "\x89PNG\r\n\x1a\nfake"
	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(imgBytes))
	}))
	defer imgSrv.Close()

	post := &Post{
		Text:   "caption",
		Images: []MediaRef{{URL: imgSrv.URL + "/photo.png", Alt: "pic"}},
	}
	dir := t.TempDir()
	body, warns := SaveMedia(context.Background(), imgSrv.Client(), "entry1", dir, post)
	if len(warns) > 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if !strings.Contains(body, "/ui/entry/entry1/file/assets/") {
		t.Fatalf("body = %q", body)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "assets"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("assets = %d want 1", len(entries))
	}
}

func TestEnrichAuthorAvatar(t *testing.T) {
	t.Parallel()
	const avatarURL = "https://cdn.example/avatar.jpg"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<meta property="og:image" content="` + avatarURL + `">`))
	}))
	defer srv.Close()

	post := &Post{Author: Author{ProfileURL: srv.URL}}
	warns := EnrichAuthorAvatar(context.Background(), srv.Client(), post)
	if len(warns) > 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if post.Author.AvatarURL != avatarURL {
		t.Fatalf("avatar = %q", post.Author.AvatarURL)
	}
}

func TestSaveAuthorAvatar(t *testing.T) {
	t.Parallel()
	const imgBytes = "\x89PNG\r\n\x1a\nfake"
	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(imgBytes))
	}))
	defer imgSrv.Close()

	dir := t.TempDir()
	path, warns := SaveAuthorAvatar(context.Background(), imgSrv.Client(), dir, imgSrv.URL+"/avatar.png")
	if len(warns) > 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if !strings.HasPrefix(path, "assets/author-") {
		t.Fatalf("path = %q", path)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "assets"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("assets = %d want 1", len(entries))
	}
	if !strings.HasPrefix(entries[0].Name(), "author-") {
		t.Fatalf("filename = %q", entries[0].Name())
	}
}
