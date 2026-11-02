//go:build e2e

package e2e_test

import (
	"os"
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/dto"
)

func TestCLIEditAndOpen(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	out := mustRun(t, dir, "add", "note", "-m", "original body", "--title", "Editable", "--tags", "edit")
	id := createdID(t, out)

	editor := editorStub(t, "zephyrrewrite")
	if _, errOut, code := runMP(t, dir, []string{"EDITOR=" + editor}, "", "edit", id); code != 0 {
		t.Fatalf("edit code=%d err=%q", code, errOut)
	}
	entry := decodeJSON[dto.Entry](t, mustRun(t, dir, "--json", "show", id))
	if !strings.Contains(entry.Body, "zephyrrewrite") {
		t.Fatalf("body = %q", entry.Body)
	}
	hits := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "search", "zephyrrewrite"))
	if len(hits) != 1 || hits[0].ID != id {
		t.Fatalf("search after edit = %+v", hits)
	}

	pathEnv, logPath := openStubDir(t)
	if _, errOut, code := runMP(t, dir, []string{"PATH=" + pathEnv}, "", "open", id); code != 0 {
		t.Fatalf("open code=%d err=%q", code, errOut)
	}
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logged), entry.Dir) {
		t.Fatalf("open log = %q, want dir %q", logged, entry.Dir)
	}
}

func TestCLIDeleteConfirmation(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	out := mustRun(t, dir, "add", "note", "-m", "delete me", "--title", "Doomed", "--tags", "gone")
	id := createdID(t, out)

	_, errOut, code := runMP(t, dir, nil, "", "delete", id)
	if code == 0 || !strings.Contains(errOut, "refusing to delete without --yes") {
		t.Fatalf("non-interactive delete code=%d err=%q", code, errOut)
	}

	ptyOut, code := runMPPty(t, dir, nil, "n\n", "delete", id)
	if code == 0 || !strings.Contains(ptyOut, "delete cancelled") {
		t.Fatalf("cancel delete code=%d out=%q", code, ptyOut)
	}
	hits := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "search", "Doomed"))
	if len(hits) != 1 {
		t.Fatalf("entry removed after cancel: %+v", hits)
	}

	ptyOut, code = runMPPty(t, dir, nil, "y\n", "delete", id)
	if code != 0 || !strings.Contains(ptyOut, "Deleted") {
		t.Fatalf("confirm delete code=%d out=%q", code, ptyOut)
	}
	hits = decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "search", "Doomed"))
	if len(hits) != 0 {
		t.Fatalf("entry still searchable: %+v", hits)
	}
}

func TestCLIUsageAndHelp(t *testing.T) {
	t.Parallel()
	dir := initVault(t)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "note missing title", args: []string{"add", "note", "-m", "body", "--tags", "t"}, want: "title is required"},
		{name: "note missing tags", args: []string{"add", "note", "-m", "body", "--title", "T"}, want: "tags are required"},
		{name: "invalid type", args: []string{"add", "note", "-m", "body", "--title", "T", "--tags", "t", "--type", "bogus"}, want: "invalid type"},
		{name: "wildcard bind", args: []string{"serve", "--addr", "0.0.0.0:1"}, want: "binds all interfaces"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errOut, code := runMP(t, dir, nil, "", tt.args...)
			if code == 0 || !strings.Contains(errOut, tt.want) {
				t.Fatalf("code=%d err=%q", code, errOut)
			}
		})
	}

	out, errOut, code := runMPArgs(t, []string{"--version"}, nil, "")
	if code != 0 || !strings.Contains(out, "version") {
		t.Fatalf("version code=%d out=%q err=%q", code, out, errOut)
	}
	out, errOut, code = runMPArgs(t, []string{"--help"}, nil, "")
	if code != 0 || !strings.Contains(out, "add") {
		t.Fatalf("help code=%d out=%q err=%q", code, out, errOut)
	}

	envDir := t.TempDir()
	out, errOut, code = runMPArgs(t, []string{"vault", "init"}, []string{"MINDPALACE_VAULT=" + envDir}, "")
	if code != 0 {
		t.Fatalf("init via env code=%d out=%q err=%q", code, out, errOut)
	}
	if _, err := os.Stat(envDir + "/config.yaml"); err != nil {
		t.Fatal(err)
	}
}
