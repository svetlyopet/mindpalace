package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
