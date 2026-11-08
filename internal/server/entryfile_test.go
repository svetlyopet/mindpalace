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

func TestEntryFileSecurityHeaders(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	e, err := s.lib.Vault.Get("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.Dir, "source.html"), []byte("<html><body>safe</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/ui/entry/abc123/file/source.html", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	csp := resp.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'none'") {
		t.Fatalf("CSP = %q", csp)
	}
	if !strings.Contains(csp, "127.0.0.1") && !strings.Contains(csp, "style-src 'unsafe-inline' http") {
		t.Fatalf("CSP should allow server origin for sandboxed iframe stylesheets, got %q", csp)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff")
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestEntryFileAllowScriptsCSP(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	e, err := s.lib.Vault.Get("abc123")
	if err != nil {
		t.Fatal(err)
	}
	e.MarkAllowScripts()
	if err := vault.WriteEntry(e.Dir, e, s.lib.Vault.Cipher()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.Dir, "source.html"), []byte("<html><script>alert(1)</script></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.Dir, "embed.html"), []byte("<html><script src=\"https://platform.twitter.com/widgets.js\"></script></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	sourceCSP := fetchCSP(t, ts.URL+"/ui/entry/abc123/file/source.html", token)
	if strings.Contains(sourceCSP, "script-src 'none'") {
		t.Fatalf("source CSP = %q", sourceCSP)
	}
	if !strings.Contains(sourceCSP, "script-src 'unsafe-inline' https:") {
		t.Fatalf("source CSP = %q", sourceCSP)
	}

	embedCSP := fetchCSP(t, ts.URL+"/ui/entry/abc123/file/embed.html", token)
	if !strings.Contains(embedCSP, "https://platform.twitter.com") {
		t.Fatalf("embed CSP = %q", embedCSP)
	}
	if !strings.Contains(embedCSP, "https://connect.facebook.net") {
		t.Fatalf("embed CSP = %q", embedCSP)
	}
	if strings.Contains(embedCSP, "script-src 'none'") {
		t.Fatalf("embed CSP = %q", embedCSP)
	}
}

func TestEntryFileJavaScriptContentType(t *testing.T) {
	if got := entryFileContentType("app.js"); got != "text/javascript; charset=utf-8" {
		t.Fatalf("js = %q", got)
	}
	if got := entryFileContentType("mod.mjs"); got != "text/javascript; charset=utf-8" {
		t.Fatalf("mjs = %q", got)
	}
}

func fetchCSP(t *testing.T, rawURL, token string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	return resp.Header.Get("Content-Security-Policy")
}
