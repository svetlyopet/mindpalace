//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/testenv"
)

func runMP(t *testing.T, vaultDir string, extraEnv []string, stdin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	allArgs := append([]string{"--vault", vaultDir}, args...)
	cmd := exec.Command(mpBin, allArgs...)
	cmd.Env = append(os.Environ(), extraEnv...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return outBuf.String(), errBuf.String(), code
}

func TestCLIHappyPath(t *testing.T) {
	dir := t.TempDir()
	if out, errOut, code := runMP(t, dir, nil, "", "vault", "init", dir); code != 0 {
		t.Fatalf("init code=%d out=%q err=%q", code, out, errOut)
	}
	if _, errOut, code := runMP(t, dir, nil, "", "add", "note", "-m", "e2e body", "--title", "E2E note", "--tags", "e2e"); code != 0 {
		t.Fatalf("add code=%d err=%q", code, errOut)
	}
	out, errOut, code := runMP(t, dir, nil, "", "--json", "search", "e2e")
	if code != 0 {
		t.Fatalf("search code=%d err=%q", code, errOut)
	}
	var hits []dto.SearchHit
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &hits); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if len(hits) == 0 {
		t.Fatal("expected search hit")
	}
	id := hits[0].ID

	out, _, code = runMP(t, dir, nil, "", "--json", "show", id)
	if code != 0 {
		t.Fatalf("show code=%d", code)
	}
	var entry dto.Entry
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Title != "E2E note" {
		t.Fatalf("title = %q", entry.Title)
	}

	if _, errOut, code = runMP(t, dir, nil, "", "tag", id, "+extra"); code != 0 {
		t.Fatalf("tag code=%d err=%q", code, errOut)
	}
	out, _, code = runMP(t, dir, nil, "", "--json", "list", "--tag", "extra")
	if code != 0 {
		t.Fatal(code)
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected hit after tag")
	}

	if _, errOut, code = runMP(t, dir, nil, "", "reindex"); code != 0 {
		t.Fatalf("reindex code=%d err=%q", code, errOut)
	}

	out, _, code = runMP(t, dir, nil, "", "--json", "delete", id, "-y")
	if code != 0 {
		t.Fatalf("delete code=%d out=%q", code, out)
	}
	out, _, code = runMP(t, dir, nil, "", "--json", "search", "e2e")
	if code != 0 {
		t.Fatal(code)
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("expected empty search, got %v", hits)
	}
}

func TestCLIEncryptedVault(t *testing.T) {
	dir := t.TempDir()
	if _, errOut, code := runMP(t, dir, nil, "", "vault", "init", dir); code != 0 {
		t.Fatalf("init: %q", errOut)
	}
	if _, errOut, code := runMP(t, dir, nil, "", "add", "note", "-m", "secret", "--title", "Enc", "--tags", "sec"); code != 0 {
		t.Fatalf("add: %q", errOut)
	}
	// mp vault encrypt requires a TTY; encrypt in-process then exercise CLI unlock via env.
	testenv.EncryptVaultAt(t, dir, "secret")
	out, errOut, code := runMP(t, dir, []string{"MINDPALACE_PASSWORD=secret"}, "", "--json", "search", "Enc")
	if code != 0 {
		t.Fatalf("search locked unlock via env code=%d err=%q", code, errOut)
	}
	var hits []dto.SearchHit
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &hits); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if len(hits) == 0 {
		t.Fatal("expected hit with password env")
	}
}

func TestCLIFailures(t *testing.T) {
	dir := t.TempDir()
	_, errOut, code := runMP(t, dir, nil, "", "search", "x")
	if code == 0 {
		t.Fatal("expected failure on uninitialized vault")
	}
	if !strings.Contains(errOut, "mp vault init") {
		t.Fatalf("stderr = %q, want mp vault init hint", errOut)
	}
	dir2 := t.TempDir()
	runMP(t, dir2, nil, "", "vault", "init", dir2)
	_, _, code = runMP(t, dir2, nil, "", "delete", "bogus", "-y")
	if code == 0 {
		t.Fatal("expected failure deleting missing entry")
	}
}

func init() {
	_ = filepath.Separator
}
