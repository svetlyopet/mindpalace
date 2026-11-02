//go:build e2e

package e2e_test

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

	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/testenv"
)

func TestAPIAuth(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	srv := startServe(t, dir)

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{name: "no token", token: "", want: http.StatusUnauthorized},
		{name: "bad token", token: "not-the-token", want: http.StatusUnauthorized},
		{name: "bearer", token: srv.Token, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := apiDo(t, srv, tt.token, http.MethodGet, "/api/entries", nil)
			if status != tt.want {
				t.Fatalf("status=%d body=%s", status, body)
			}
		})
	}

	status, body := apiDo(t, srv, "", http.MethodOptions, "/api/entries", nil)
	if status != http.StatusNoContent {
		t.Fatalf("options status=%d body=%s", status, body)
	}
	status, body = apiDo(t, srv, "", http.MethodGet, "/api/session", nil)
	if status != http.StatusNoContent {
		t.Fatalf("session status=%d body=%s", status, body)
	}
	status, body = apiDo(t, srv, "", http.MethodGet, "/", nil)
	if status != http.StatusOK || !strings.Contains(string(body), "Mindpalace") {
		t.Fatalf("library status=%d body=%s", status, body)
	}
	status, body = apiDo(t, srv, "", http.MethodPost, "/api/unlock", []byte(`{"password":""}`))
	if status != http.StatusNoContent {
		t.Fatalf("plaintext unlock status=%d body=%s", status, body)
	}
	status, body = apiDo(t, srv, srv.Token, http.MethodPost, "/api/lock", nil)
	if status != http.StatusNoContent {
		t.Fatalf("plaintext lock status=%d body=%s", status, body)
	}
	status, body = apiDo(t, srv, srv.Token, http.MethodGet, "/api/entries", nil)
	if status != http.StatusOK {
		t.Fatalf("entries after plaintext lock status=%d body=%s", status, body)
	}
}

func TestAPIEntries(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	srv := startServe(t, dir)
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(articleHTML))
	}))
	t.Cleanup(page.Close)

	bad := []struct {
		name string
		body string
		want string
	}{
		{name: "invalid kind", body: `{"kind":"nope","title":"T","tags":["t"]}`, want: "invalid kind"},
		{name: "invalid type", body: `{"kind":"note","text":"body","title":"T","tags":["t"],"type":"bogus"}`, want: "invalid type"},
		{name: "social url", body: `{"kind":"social","url":"https://example.com/post","title":"T","tags":["t"]}`, want: "not a supported social post URL"},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			status, body := apiDo(t, srv, srv.Token, http.MethodPost, "/api/capture", []byte(tt.body))
			if status != http.StatusBadRequest || !strings.Contains(string(body), tt.want) {
				t.Fatalf("status=%d body=%s", status, body)
			}
		})
	}

	status, body := apiDo(t, srv, srv.Token, http.MethodPost, "/api/capture/preview", []byte(`{"kind":"note","text":"golang databases zephyrquartz","title":"Preview title"}`))
	if status != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", status, body)
	}
	var preview struct {
		Title         string   `json:"title"`
		SuggestedTags []string `json:"suggested_tags"`
	}
	if err := json.Unmarshal(body, &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Title != "Preview title" || !containsTag(preview.SuggestedTags, "golang") {
		t.Fatalf("preview = %+v", preview)
	}
	if n := len(decodeEntries(t, srv)); n != 0 {
		t.Fatalf("preview created %d entries", n)
	}

	note := captureEntry(t, srv, map[string]any{
		"kind": "note", "text": "server note body", "title": "Server note", "tags": []string{"note"},
	})
	if note.Type != "note" || note.Title != "Server note" {
		t.Fatalf("note = %+v", note)
	}

	urlEntry := captureEntry(t, srv, map[string]any{
		"kind": "url", "url": page.URL, "title": "Server article", "tags": []string{"web"}, "thoughts": "meeting-notes-marker",
	})
	if urlEntry.Type != "article" || urlEntry.Source != page.URL || !strings.Contains(urlEntry.Body, "zephyrquartz") {
		t.Fatalf("url entry = %+v", urlEntry)
	}

	htmlEntry := captureEntry(t, srv, map[string]any{
		"kind": "html", "url": "https://example.com/html-fixture", "html": articleHTML,
		"title": "HTML fixture", "tags": []string{"html"}, "full": true,
	})
	if htmlEntry.Type != "article" || !strings.Contains(htmlEntry.Body, "zephyrquartz") {
		t.Fatalf("html entry = %+v", htmlEntry)
	}

	status, fileBody := apiDo(t, srv, srv.Token, http.MethodGet, "/api/entries/"+urlEntry.ID+"/files/extracted.txt", nil)
	if status != http.StatusOK || !strings.Contains(string(fileBody), "zephyrquartz") {
		t.Fatalf("file status=%d body=%s", status, fileBody)
	}
	assertTraversalRejected(t, srv, urlEntry.ID)

	hits := getEntries(t, srv, "/api/entries?q=zephyrquartz&type=article&tag=web&domain=127.0.0.1&since=2000-01-01&limit=20")
	if len(hits) != 1 || hits[0].ID != urlEntry.ID {
		t.Fatalf("filtered hits = %+v", hits)
	}
	if future := getEntries(t, srv, "/api/entries?since=2099-01-01"); len(future) != 0 {
		t.Fatalf("future hits = %+v", future)
	}
	if limited := getEntries(t, srv, "/api/entries?limit=1"); len(limited) != 1 {
		t.Fatalf("limit hits = %+v", limited)
	}

	status, body = apiDo(t, srv, srv.Token, http.MethodGet, "/api/tags", nil)
	if status != http.StatusOK {
		t.Fatalf("tags status=%d body=%s", status, body)
	}
	counts := decodeJSON[[]dto.TagCount](t, string(body))
	if !tagCountHas(counts, "web") {
		t.Fatalf("tags = %+v", counts)
	}

	status, body = apiDo(t, srv, srv.Token, http.MethodPost, "/api/entries/"+note.ID+"/tags", []byte(`{"add":["extra"],"remove":["note"]}`))
	if status != http.StatusOK {
		t.Fatalf("update tags status=%d body=%s", status, body)
	}
	updated := decodeJSON[dto.Entry](t, string(body))
	if containsTag(updated.Tags, "note") || !containsTag(updated.Tags, "extra") {
		t.Fatalf("tags = %v", updated.Tags)
	}

	status, body = apiDo(t, srv, srv.Token, http.MethodGet, "/api/entries/"+note.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("get status=%d body=%s", status, body)
	}
	status, body = apiDo(t, srv, srv.Token, http.MethodDelete, "/api/entries/"+note.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", status, body)
	}
	var deleted dto.DeleteResponse
	if err := json.Unmarshal(body, &deleted); err != nil {
		t.Fatal(err)
	}
	if deleted.ID != note.ID || deleted.Title != note.Title {
		t.Fatalf("delete = %+v", deleted)
	}
	status, _ = apiDo(t, srv, srv.Token, http.MethodGet, "/api/entries/"+note.ID, nil)
	if status != http.StatusNotFound {
		t.Fatalf("get after delete status=%d", status)
	}
	status, _ = apiDo(t, srv, srv.Token, http.MethodGet, "/api/entries/missing", nil)
	if status != http.StatusNotFound {
		t.Fatalf("missing status=%d", status)
	}
}

func TestAPIUpload(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	srv := startServe(t, dir)

	status, body := postUpload(t, srv, "/api/capture/upload/preview", "ideas.md", "golang ideas\n", "Ideas", nil)
	if status != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", status, body)
	}
	var preview struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(body, &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Title != "Ideas" {
		t.Fatalf("preview title = %q", preview.Title)
	}
	if n := len(decodeEntries(t, srv)); n != 0 {
		t.Fatalf("preview created %d entries", n)
	}

	status, body = postUpload(t, srv, "/api/capture/upload", "snippet.txt", "upload-body-marker\n", "Uploaded", []string{"docs"})
	if status != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", status, body)
	}
	var created struct {
		Entry dto.Entry `json:"entry"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}
	if created.Entry.Title != "Uploaded" || created.Entry.Type != "snippet" || !containsTag(created.Entry.Tags, "docs") {
		t.Fatalf("upload entry = %+v", created.Entry)
	}
	if !strings.Contains(created.Entry.Body, "upload-body-marker") {
		t.Fatalf("body = %q", created.Entry.Body)
	}
}

func TestAPIEncryptedSession(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	mustRun(t, dir, "add", "note", "-m", "hidden", "--title", "Hidden", "--tags", "sec")
	testenv.EncryptVaultAt(t, dir, "secret")
	srv := startServe(t, dir)

	status, body := apiDo(t, srv, "", http.MethodGet, "/api/session", nil)
	if status != http.StatusUnauthorized || !strings.Contains(string(body), "locked") {
		t.Fatalf("session status=%d body=%s", status, body)
	}
	status, body = apiDo(t, srv, srv.Token, http.MethodGet, "/api/entries", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("locked entries status=%d body=%s", status, body)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	status, body, _ = apiDoHeaders(t, srv, "", http.MethodPost, "/api/unlock", []byte(`{"password":"wrong"}`), client)
	if status != http.StatusUnauthorized || !strings.Contains(string(body), "wrong password") {
		t.Fatalf("wrong password status=%d body=%s", status, body)
	}
	status, body, _ = apiDoHeaders(t, srv, "", http.MethodPost, "/api/unlock", []byte(`{"password":"secret"}`), client)
	if status != http.StatusNoContent {
		t.Fatalf("unlock status=%d body=%s", status, body)
	}
	status, body, _ = apiDoHeaders(t, srv, "", http.MethodGet, "/api/entries", nil, client)
	if status != http.StatusOK || !strings.Contains(string(body), "Hidden") {
		t.Fatalf("cookie entries status=%d body=%s", status, body)
	}
	status, body, _ = apiDoHeaders(t, srv, "", http.MethodPost, "/api/lock", nil, client)
	if status != http.StatusNoContent {
		t.Fatalf("lock status=%d body=%s", status, body)
	}
	status, body = apiDo(t, srv, "", http.MethodGet, "/api/session", nil)
	if status != http.StatusUnauthorized || !strings.Contains(string(body), "locked") {
		t.Fatalf("session after lock status=%d body=%s", status, body)
	}
}

func captureEntry(t *testing.T, srv *runningServer, payload map[string]any) dto.Entry {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	status, body := apiDo(t, srv, srv.Token, http.MethodPost, "/api/capture", raw)
	if status != http.StatusCreated {
		t.Fatalf("capture status=%d body=%s payload=%s", status, body, raw)
	}
	var created struct {
		Entry dto.Entry `json:"entry"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}
	return created.Entry
}

func getEntries(t *testing.T, srv *runningServer, path string) []dto.SearchHit {
	t.Helper()
	status, body := apiDo(t, srv, srv.Token, http.MethodGet, path, nil)
	if status != http.StatusOK {
		t.Fatalf("GET %s status=%d body=%s", path, status, body)
	}
	return decodeJSON[[]dto.SearchHit](t, string(body))
}

func decodeEntries(t *testing.T, srv *runningServer) []dto.SearchHit {
	t.Helper()
	return getEntries(t, srv, "/api/entries")
}

func postUpload(t *testing.T, srv *runningServer, path, name, contents, title string, tags []string) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, contents); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteField("title", title); err != nil {
		t.Fatal(err)
	}
	if tags != nil {
		raw, err := json.Marshal(tags)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.WriteField("tags", string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+srv.Token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, b
}

func assertTraversalRejected(t *testing.T, srv *runningServer, id string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/entries/"+id+"/files/../../config.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+srv.Token)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusForbidden, http.StatusNotFound, http.StatusBadRequest,
		http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
	default:
		t.Fatalf("traversal status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), srv.Token) {
		t.Fatal("traversal response included the API token")
	}
}
