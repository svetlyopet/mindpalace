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

	post, err := Fetch(context.Background(), client, PlatformX, "https://twitter.com/u/status/99", false)
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

func TestOEmbedEndpoint_scriptFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		platform      Platform
		rawURL        string
		includeScript bool
		wantParam     string
		wantAbsent    string
	}{
		{name: "x omits script", platform: PlatformX, rawURL: "https://x.com/u/status/1", wantParam: "omit_script=true"},
		{name: "x keeps script", platform: PlatformX, rawURL: "https://x.com/u/status/1", includeScript: true, wantAbsent: "omit_script"},
		{name: "facebook omits script", platform: PlatformFacebook, rawURL: "https://www.facebook.com/page/posts/1", wantParam: "omitscript=true"},
		{name: "facebook keeps script", platform: PlatformFacebook, rawURL: "https://www.facebook.com/page/posts/1", includeScript: true, wantAbsent: "omitscript"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := oEmbedEndpoint(tt.platform, tt.rawURL, tt.includeScript)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantParam != "" && !strings.Contains(got, tt.wantParam) {
				t.Fatalf("endpoint %q missing %q", got, tt.wantParam)
			}
			if tt.wantAbsent != "" && strings.Contains(got, tt.wantAbsent) {
				t.Fatalf("endpoint %q still has %q", got, tt.wantAbsent)
			}
			if !strings.Contains(got, "url=") {
				t.Fatalf("endpoint %q missing url", got)
			}
		})
	}
}

func TestFetch_defaultOmitsScript(t *testing.T) {
	t.Parallel()
	const html = `<blockquote class="twitter-tweet"><p>Hi</p></blockquote>`
	gotURL := recordOEmbedFetch(t, html, false)
	if !strings.Contains(gotURL, "omit_script=true") {
		t.Fatalf("url = %q", gotURL)
	}
}

func TestFetch_includeScriptStoresHTML(t *testing.T) {
	t.Parallel()
	const html = `<blockquote class="twitter-tweet"><p>Hi</p></blockquote><script src="https://platform.twitter.com/widgets.js"></script>`
	post, gotURL := fetchRecorded(t, html, true)
	if strings.Contains(gotURL, "omit_script") {
		t.Fatalf("url = %q", gotURL)
	}
	if !strings.Contains(post.EmbedHTML, "<script") {
		t.Fatalf("embed = %q", post.EmbedHTML)
	}
	if !strings.Contains(post.Text, "Hi") {
		t.Fatalf("text = %q", post.Text)
	}
}

func recordOEmbedFetch(t *testing.T, html string, includeScript bool) string {
	t.Helper()
	post, gotURL := fetchRecorded(t, html, includeScript)
	if includeScript {
		return gotURL
	}
	if post.EmbedHTML != "" {
		t.Fatalf("embed = %q, want empty", post.EmbedHTML)
	}
	return gotURL
}

func fetchRecorded(t *testing.T, html string, includeScript bool) (*Post, string) {
	t.Helper()
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(oEmbedResponse{HTML: html, AuthorName: "A"})
	}))
	t.Cleanup(srv.Close)
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotURL = req.URL.String()
		req2, err := http.NewRequestWithContext(req.Context(), req.Method, srv.URL, nil)
		if err != nil {
			return nil, err
		}
		return http.DefaultTransport.RoundTrip(req2)
	})}
	post, err := Fetch(context.Background(), client, PlatformX, "https://twitter.com/u/status/99", includeScript)
	if err != nil {
		t.Fatal(err)
	}
	return post, gotURL
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
	_, err := Fetch(context.Background(), client, PlatformX, "https://twitter.com/u/status/1", false)
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
