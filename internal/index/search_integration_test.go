package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/svetlyopet/mindpalace/internal/index"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestSearchFindsTitleAndBody(t *testing.T) {
	dir := t.TempDir()
	if _, err := vault.Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("editor: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, _ := vault.Open(dir)
	e := &vault.Entry{
		ID: "s1", Title: "Golang Concurrency Patterns", Created: time.Now(), Type: vault.TypeNote,
		Body: "This note discusses goroutines and channels in depth.",
	}
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	ix, err := index.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Close()
	if err := ix.Put(e); err != nil {
		t.Fatal(err)
	}
	sr := search.New(ix)
	for _, qtext := range []string{"Golang", "goroutines", "Concurrency", "channels"} {
		res, err := sr.Search(context.Background(), search.Query{Text: qtext, Limit: 10})
		if err != nil {
			t.Fatalf("q=%q err=%v", qtext, err)
		}
		t.Logf("q=%q hits=%d", qtext, len(res))
		if len(res) == 0 {
			t.Fatalf("q=%q: expected hits", qtext)
		}
	}
}

func TestSearchMultiWord(t *testing.T) {
	dir := t.TempDir()
	vault.Init(dir)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("editor: \"\"\n"), 0o644)
	v, _ := vault.Open(dir)
	e := &vault.Entry{ID: "s2", Title: "Golang Concurrency", Created: time.Now(), Type: vault.TypeNote, Body: "patterns"}
	v.Create(e)
	ix, _ := index.Open(dir)
	defer ix.Close()
	ix.Put(e)
	sr := search.New(ix)
	for _, qtext := range []string{"Golang Concurrency", "golang patterns", "missing word"} {
		res, _ := sr.Search(context.Background(), search.Query{Text: qtext, Limit: 10})
		t.Logf("q=%q hits=%d", qtext, len(res))
	}
}
