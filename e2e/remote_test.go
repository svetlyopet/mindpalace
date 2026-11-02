//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/svetlyopet/mindpalace/internal/dto"
)

func TestCLIWhileServeRunning(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	srv := startServe(t, dir)

	out := mustRun(t, dir, "add", "note", "-m", "remote body", "--title", "Remote note", "--tags", "remote")
	id := createdID(t, out)

	hits := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "search", "remote"))
	if len(hits) != 1 || hits[0].ID != id {
		t.Fatalf("search = %+v", hits)
	}
	listed := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "list", "--tag", "remote"))
	if len(listed) != 1 || listed[0].ID != id {
		t.Fatalf("list = %+v", listed)
	}
	shown := decodeJSON[dto.Entry](t, mustRun(t, dir, "--json", "show", id))
	apiEntry := getAPIEntry(t, srv, id)
	if shown.ID != apiEntry.ID || shown.Title != apiEntry.Title || shown.Body != apiEntry.Body {
		t.Fatalf("cli %+v api %+v", shown, apiEntry)
	}
	counts := decodeJSON[[]dto.TagCount](t, mustRun(t, dir, "--json", "tags"))
	if !tagCountHas(counts, "remote") {
		t.Fatalf("tags = %+v", counts)
	}

	if _, errOut, code := runMP(t, dir, nil, "", "tag", id, "--add", "extra"); code != 0 {
		t.Fatalf("tag code=%d err=%q", code, errOut)
	}
	apiEntry = getAPIEntry(t, srv, id)
	if !containsTag(apiEntry.Tags, "extra") {
		t.Fatalf("tags after remote tag = %v", apiEntry.Tags)
	}

	editor := editorStub(t, "zephyrrewrite")
	if _, errOut, code := runMP(t, dir, []string{"EDITOR=" + editor}, "", "edit", id); code != 0 {
		t.Fatalf("edit code=%d err=%q", code, errOut)
	}
	waitFor(t, 10*time.Second, func() bool {
		entry := getAPIEntry(t, srv, id)
		if !strings.Contains(entry.Body, "zephyrrewrite") {
			return false
		}
		found := getEntries(t, srv, "/api/entries?q=zephyrrewrite")
		return len(found) == 1 && found[0].ID == id
	})

	_, errOut, code := runMP(t, dir, nil, "", "reindex")
	if code == 0 || !strings.Contains(errOut, "mp serve is running") {
		t.Fatalf("reindex code=%d err=%q", code, errOut)
	}
	_, errOut, code = runMP(t, dir, nil, "", "vault", "encrypt")
	if code == 0 || !strings.Contains(errOut, "mp serve is running") {
		t.Fatalf("encrypt code=%d err=%q", code, errOut)
	}

	if _, errOut, code = runMP(t, dir, nil, "", "delete", id, "-y"); code != 0 {
		t.Fatalf("delete code=%d err=%q", code, errOut)
	}
	hits = decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "search", "Remote"))
	if len(hits) != 0 {
		t.Fatalf("search after delete = %+v", hits)
	}
	status, _ := apiDo(t, srv, srv.Token, http.MethodGet, "/api/entries/"+id, nil)
	if status != http.StatusNotFound {
		t.Fatalf("api get after delete status=%d", status)
	}
}

func getAPIEntry(t *testing.T, srv *runningServer, id string) dto.Entry {
	t.Helper()
	status, body := apiDo(t, srv, srv.Token, http.MethodGet, "/api/entries/"+id, nil)
	if status != http.StatusOK {
		t.Fatalf("get %s status=%d body=%s", id, status, body)
	}
	var entry dto.Entry
	if err := json.Unmarshal(body, &entry); err != nil {
		t.Fatal(err)
	}
	return entry
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if fn() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for condition")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
