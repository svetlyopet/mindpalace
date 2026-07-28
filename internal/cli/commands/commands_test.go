package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/capture"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/cliformat"
	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/library"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/testenv"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func testRuntime(t *testing.T) (*clictx.Runtime, string) {
	t.Helper()
	vi := testenv.TempVaultIndex(t, true)
	sr := search.New(vi.Index)
	cap := testenv.NewCapturer(vi.Vault, vi.Config)
	lib := library.New(vi.Vault, vi.Index, sr, cap)
	app := &clictx.App{
		Root:     vi.Dir,
		Config:   vi.Config,
		Vault:    vi.Vault,
		Index:    vi.Index,
		Searcher: sr,
		Capturer: cap,
		Lib:      lib,
	}
	return &clictx.Runtime{App: app}, vi.Dir
}

func withStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestInitCreatesVault(t *testing.T) {
	dir := t.TempDir()
	rt := &clictx.Runtime{App: &clictx.App{}}
	cmd := NewInit(rt)
	if err := cmd.RunE(cmd, []string{dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.Open(dir); err != nil {
		t.Fatal(err)
	}
}

func TestAddNote(t *testing.T) {
	rt, _ := testRuntime(t)
	add := NewAdd(rt)
	addNote := NewAddNote(rt)
	addURL := NewAddURL(rt)
	addFile := NewAddFile(rt)
	ConfigureAddFlags(add, addNote, addURL)
	add.AddCommand(addNote)
	addMessage = "note body"
	addTitle = "CLI note"
	addTags = []string{"cli"}
	if err := add.PersistentFlags().Set("tags", "cli"); err != nil {
		t.Fatal(err)
	}
	if err := add.PersistentFlags().Set("title", "CLI note"); err != nil {
		t.Fatal(err)
	}
	cmd := addNote
	out := withStdout(t, func() {
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "Created") {
		t.Fatalf("out = %q", out)
	}
	_ = addFile
}

func TestSearchJSON(t *testing.T) {
	rt, _ := testRuntime(t)
	rt.JSON = true
	searchCmd := NewSearch(rt)
	listCmd := NewList(rt)
	cliformat.RegisterSearchFlags(searchCmd, listCmd)
	out := withStdout(t, func() {
		if err := searchCmd.RunE(searchCmd, []string{"Fixture"}); err != nil {
			t.Fatal(err)
		}
	})
	var hits []dto.SearchHit
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &hits); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if len(hits) == 0 {
		t.Fatal("expected hits")
	}
}

func TestList(t *testing.T) {
	rt, _ := testRuntime(t)
	searchCmd := NewSearch(rt)
	listCmd := NewList(rt)
	cliformat.RegisterSearchFlags(searchCmd, listCmd)
	out := withStdout(t, func() {
		if err := listCmd.RunE(listCmd, nil); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "abc123") {
		t.Fatalf("out = %q", out)
	}
}

func TestShowJSON(t *testing.T) {
	rt, _ := testRuntime(t)
	rt.JSON = true
	cmd := NewShow(rt)
	out := withStdout(t, func() {
		if err := cmd.RunE(cmd, []string{"abc123"}); err != nil {
			t.Fatal(err)
		}
	})
	var e dto.Entry
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &e); err != nil {
		t.Fatal(err)
	}
	if e.Title != "Fixture" {
		t.Fatalf("title = %q", e.Title)
	}
}

func TestTagCommand(t *testing.T) {
	rt, _ := testRuntime(t)
	cmd := NewTag(rt)
	if err := cmd.RunE(cmd, []string{"abc123", "+gamma", "-beta"}); err != nil {
		t.Fatal(err)
	}
	e, err := rt.Lib.GetEntry("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Tags) != 2 {
		t.Fatalf("tags = %v", e.Tags)
	}
}

func TestDeleteYes(t *testing.T) {
	rt, _ := testRuntime(t)
	cmd := NewDelete(rt)
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"abc123"}); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Lib.GetEntry("abc123"); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("Get: %v", err)
	}
}

func TestReindex(t *testing.T) {
	rt, _ := testRuntime(t)
	cmd := NewReindex(rt)
	out := withStdout(t, func() {
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "Reindexed") {
		t.Fatalf("out = %q", out)
	}
}

func TestTagsCommand(t *testing.T) {
	rt, _ := testRuntime(t)
	cmd := NewTags(rt)
	out := withStdout(t, func() {
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "alpha") {
		t.Fatalf("out = %q", out)
	}
}

func TestTagInvalidArg(t *testing.T) {
	rt, _ := testRuntime(t)
	cmd := NewTag(rt)
	err := cmd.RunE(cmd, []string{"abc123", "bad"})
	if err == nil {
		t.Fatal("expected usage error")
	}
}

func TestCommitCaptureViaLibrary(t *testing.T) {
	rt, _ := testRuntime(t)
	res, err := rt.Capturer.Note(t.Context(), "x", capture.Options{Title: "T", TagsExplicit: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Lib.CommitCapture(t.Context(), res); err != nil {
		t.Fatal(err)
	}
}
