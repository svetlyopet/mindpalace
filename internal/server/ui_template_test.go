package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func TestLibraryPageRender(t *testing.T) {
	s, _ := testServer(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.handleLibrary(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Mindpalace", "id=\"viewer\"", "id=\"global-search\"", "id=\"entry-list\"", "viewer-empty", "note-modal", "url-modal", "file-modal", "note-title", "note-tags", "url-panel-toggle", "file-panel-toggle", "url-allow-scripts", "social-allow-scripts", "resize-handle", "/static/app.js", "/static/capture-tags.js"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in library page HTML", want)
		}
	}
}

func TestEntryPageRenderSelectedRow(t *testing.T) {
	s, _ := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/entry/abc123")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	if !strings.Contains(body, "is-selected") {
		t.Fatal("expected selected entry row in sidebar")
	}
	if !strings.Contains(body, "entry-view") {
		t.Fatal("expected entry viewer markup")
	}
	if !strings.Contains(body, "btn-edit-tags-entry") {
		t.Fatal("expected edit tags button")
	}
	if !strings.Contains(body, "btn-delete-entry") {
		t.Fatal("expected delete button")
	}
}

func TestUIEntryViewerPartial(t *testing.T) {
	s, _ := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/ui/entry/abc123")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatal("partial should not include full layout")
	}
	if !strings.Contains(body, "entry-view") {
		t.Fatal("expected entry viewer partial")
	}
	if !strings.Contains(body, "btn-edit-tags-entry") {
		t.Fatal("expected edit tags button")
	}
	if !strings.Contains(body, "btn-delete-entry") {
		t.Fatal("expected delete button")
	}
}

func TestEntryViewerSandbox(t *testing.T) {
	s, _ := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	e, err := s.lib.Vault.Get("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.Dir, "source.html"), []byte("<html><body>safe</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	blocked := viewerHTML(t, ts.URL+"/ui/entry/abc123")
	if !strings.Contains(blocked, `sandbox=""`) {
		t.Fatalf("blocked viewer = %s", blocked)
	}
	if strings.Contains(blocked, "allow-scripts") {
		t.Fatalf("blocked viewer allows scripts: %s", blocked)
	}

	e.MarkAllowScripts()
	if err := vault.WriteEntry(e.Dir, e, s.lib.Vault.Cipher()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.Dir, "embed.html"), []byte("<html><body>embed</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	allowed := viewerHTML(t, ts.URL+"/ui/entry/abc123")
	if !strings.Contains(allowed, `sandbox="allow-scripts"`) {
		t.Fatalf("allowed viewer = %s", allowed)
	}
	if strings.Contains(allowed, "allow-same-origin") || strings.Contains(allowed, "allow-top-navigation") {
		t.Fatalf("sandbox is too open: %s", allowed)
	}
	if !strings.Contains(allowed, "/file/embed.html") {
		t.Fatalf("embed iframe missing: %s", allowed)
	}
}

func viewerHTML(t *testing.T, rawURL string) string {
	t.Helper()
	resp, err := http.Get(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestMissingEntryPageRedirectsHome(t *testing.T) {
	s, _ := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := noRedirectClient().Get(ts.URL + "/entry/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusFound)
	}
	if loc := resp.Header.Get("Location"); loc != "/" {
		t.Fatalf("Location=%q, want /", loc)
	}
}

func TestMissingUIEntryViewerHXRedirectsHome(t *testing.T) {
	s, _ := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/ui/entry/does-not-exist", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("HX-Request", "true")
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusOK)
	}
	if loc := resp.Header.Get("HX-Redirect"); loc != "/" {
		t.Fatalf("HX-Redirect=%q, want /", loc)
	}
}

func TestUnknownPathRedirectsHome(t *testing.T) {
	s, _ := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := noRedirectClient().Get(ts.URL + "/no-such-page")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusFound)
	}
	if loc := resp.Header.Get("Location"); loc != "/" {
		t.Fatalf("Location=%q, want /", loc)
	}
}

func TestMissingAPIEntryStillNotFound(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/entries/does-not-exist", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
