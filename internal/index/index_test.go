package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/svetlyopet/mindpalace/internal/index"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestRebuildRemovesDeletedEntry(t *testing.T) {
	dir := t.TempDir()
	if _, err := vault.Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("editor: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := &vault.Entry{
		ID:      "ixdel1",
		Title:   "Indexed",
		Created: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC),
		Type:    vault.TypeNote,
		Body:    "hello",
	}
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	ix, err := index.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })

	if err := ix.Put(e); err != nil {
		t.Fatal(err)
	}
	if _, ok := ix.Get(e.ID); !ok {
		t.Fatal("expected entry in index before delete")
	}
	if err := v.Delete(e.ID); err != nil {
		t.Fatal(err)
	}
	stats, err := ix.Rebuild(context.Background(), v)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Removed != 1 {
		t.Fatalf("Rebuild Removed = %d, want 1", stats.Removed)
	}
	if _, ok := ix.Get(e.ID); ok {
		t.Fatal("expected entry removed from index after rebuild")
	}
}
