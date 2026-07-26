package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/library"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/testenv"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestAPIAuthAndEntries(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/entries")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/entries", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestAPIAssetTraversal(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/entries/abc123/files/../../etc/passwd", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestAPICaptureNote(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := `{"kind":"note","text":"server test note about golang","title":"Server test note","tags":[]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/capture", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var out struct {
		Entry struct {
			ID   string   `json:"id"`
			Tags []string `json:"tags"`
		} `json:"entry"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Entry.Tags) != 0 {
		t.Fatalf("tags = %v, want none (no auto-apply)", out.Entry.Tags)
	}
}

func TestAPICaptureNoteMissingTitle(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := `{"kind":"note","text":"no title note","tags":[]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/capture", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestAPICapturePreview(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := `{"kind":"note","text":"preview note about golang frameworks"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/capture/preview", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var out struct {
		Title          string   `json:"title"`
		SuggestedTags  []string `json:"suggested_tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Title == "" {
		t.Fatal("expected title")
	}
	if len(out.SuggestedTags) == 0 {
		t.Fatal("expected suggested tags")
	}
	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/entries", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var entries []map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries count = %d, preview must not create entries", len(entries))
	}
}

func TestAPICaptureUpload(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "snippet.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "golang snippet content"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteField("tags", "[]"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteField("title", "snippet.txt"); err != nil {
		t.Fatal(err)
	}
	w.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/capture/upload", &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d body %s", resp.StatusCode, readBody(resp))
	}
	var out struct {
		Entry struct {
			Type string   `json:"type"`
			Tags []string `json:"tags"`
		} `json:"entry"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Entry.Type != string(vault.TypeSnippet) {
		t.Fatalf("type = %q", out.Entry.Type)
	}
	if len(out.Entry.Tags) != 0 {
		t.Fatalf("tags = %v", out.Entry.Tags)
	}
}

func TestAPICaptureUploadPreview(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "ideas.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "# golang ideas\n"); err != nil {
		t.Fatal(err)
	}
	w.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/capture/upload/preview", &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/entries", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var entries []map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, preview must not create", len(entries))
	}
}

func readBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func TestAPIDeleteEntry(t *testing.T) {
	s, token := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/entries/abc123", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/entries/abc123", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete status = %d", resp.StatusCode)
	}
	if _, ok := s.lib.Index.Get("abc123"); ok {
		t.Fatal("expected entry removed from index")
	}
}

func TestAPICaptureWithSessionCookie(t *testing.T) {
	s, _ := testServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	sess, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/session", nil)
	resp, err := client.Do(sess)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("session status = %d", resp.StatusCode)
	}

	body := `{"kind":"note","text":"cookie auth note","title":"Cookie note"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/capture", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func testServer(t *testing.T) (*Server, string) {
	t.Helper()
	vi := testenv.TempVaultIndex(t, true)
	sr := search.New(vi.Index)
	cap := testenv.NewCapturer(vi.Vault, vi.Config)
	lib := library.New(vi.Vault, vi.Index, sr, cap)
	token := testenv.ServeToken(t, vi.Dir, vi.Config)
	return New(lib, vi.Config, token), token
}
