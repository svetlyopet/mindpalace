//go:build e2e

package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/dto"
)

func TestCLICaptureNoteStdinAndType(t *testing.T) {
	t.Parallel()
	dir := initVault(t)

	out := mustRunStdin(t, dir, "piped note body\n", "add", "note", "--title", "Piped", "--tags", "piped")
	id := createdID(t, out)
	entry := decodeJSON[dto.Entry](t, mustRun(t, dir, "--json", "show", id))
	if entry.Title != "Piped" || !strings.Contains(entry.Body, "piped note body") {
		t.Fatalf("stdin note = %+v", entry)
	}
	if !containsTag(entry.Tags, "piped") {
		t.Fatalf("tags = %v", entry.Tags)
	}

	out = mustRun(t, dir, "add", "note", "-m", "snippet body", "--title", "Snippet note", "--tags", "snip", "--type", "snippet")
	id = createdID(t, out)
	entry = decodeJSON[dto.Entry](t, mustRun(t, dir, "--json", "show", id))
	if entry.Type != "snippet" || entry.Title != "Snippet note" {
		t.Fatalf("typed note = %+v", entry)
	}
}

func TestCLICaptureFile(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	file := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(file, []byte("file-body-marker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := mustRun(t, dir, "add", "file", file, "--title", "Imported file", "--tags", "docs", "--thoughts", "file-thoughts-marker")
	entry := decodeJSON[dto.Entry](t, mustRun(t, dir, "--json", "show", createdID(t, out)))
	if entry.Type != "snippet" || entry.Title != "Imported file" {
		t.Fatalf("file entry = %+v", entry)
	}
	if !strings.Contains(entry.Body, "file-body-marker") || !strings.Contains(entry.Body, "file-thoughts-marker") {
		t.Fatalf("body = %q", entry.Body)
	}
	if !strings.Contains(entry.Body, "## Thoughts") {
		t.Fatalf("body = %q, want thoughts section", entry.Body)
	}
}

func TestCLICaptureURL(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(articleHTML))
	}))
	t.Cleanup(page.Close)

	out := mustRun(t, dir, "add", "url", page.URL, "--title", "Local article", "--tags", "web", "--thoughts", "meeting-notes-marker")
	entry := decodeJSON[dto.Entry](t, mustRun(t, dir, "--json", "show", createdID(t, out)))
	if entry.Type != "article" || entry.Title != "Local article" || entry.Source != page.URL {
		t.Fatalf("url entry = %+v", entry)
	}
	if !strings.Contains(entry.Body, "zephyrquartz") || !strings.Contains(entry.Body, "meeting-notes-marker") {
		t.Fatalf("body = %q", entry.Body)
	}
	hits := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "search", "--domain", "127.0.0.1", "zephyrquartz"))
	if len(hits) != 1 || hits[0].ID != entry.ID {
		t.Fatalf("domain hits = %+v", hits)
	}

	out = mustRun(t, dir, "add", "url", page.URL+"/full", "--title", "Full article", "--tags", "web", "--full")
	full := decodeJSON[dto.Entry](t, mustRun(t, dir, "--json", "show", createdID(t, out)))
	info, err := os.Stat(filepath.Join(full.Dir, "source.html"))
	if err != nil || info.Size() == 0 {
		t.Fatalf("source.html missing: %v", err)
	}
}

func TestCLICaptureSocialRejectsNonSocialURL(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	_, errOut, code := runMP(t, dir, nil, "", "add", "social", "https://example.com/post", "--title", "Post", "--tags", "social")
	if code == 0 {
		t.Fatal("expected social capture to fail offline")
	}
	if !strings.Contains(errOut, "not a supported social post URL") {
		t.Fatalf("stderr = %q", errOut)
	}
}

func mustRunStdin(t *testing.T, vaultDir, stdin string, args ...string) string {
	t.Helper()
	out, errOut, code := runMP(t, vaultDir, nil, stdin, args...)
	if code != 0 {
		t.Fatalf("%v code=%d\nout=%q\nerr=%q", args, code, out, errOut)
	}
	return out
}

func containsTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}
