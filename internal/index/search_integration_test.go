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
	if _, err := vault.Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("editor: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, _ := vault.Open(dir)
	e := &vault.Entry{ID: "s2", Title: "Golang Concurrency", Created: time.Now(), Type: vault.TypeNote, Body: "patterns"}
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	ix, _ := index.Open(dir)
	defer ix.Close()
	if err := ix.Put(e); err != nil {
		t.Fatal(err)
	}
	sr := search.New(ix)
	for _, qtext := range []string{"Golang Concurrency", "golang patterns", "missing word"} {
		res, _ := sr.Search(context.Background(), search.Query{Text: qtext, Limit: 10})
		t.Logf("q=%q hits=%d", qtext, len(res))
	}
}

func TestSearchFindsTagsMultiTagEntry(t *testing.T) {
	dir := t.TempDir()
	if _, err := vault.Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("editor: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, _ := vault.Open(dir)
	e := &vault.Entry{
		ID: "s3", Title: "Indexed note", Created: time.Now(), Type: vault.TypeNote,
		Tags: []string{"alpha", "beta"},
		Body: "uniquebodytoken",
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
	for _, qtext := range []string{"alpha", "beta", "uniquebodytoken"} {
		res, err := sr.Search(context.Background(), search.Query{Text: qtext, Limit: 10})
		if err != nil {
			t.Fatalf("q=%q err=%v", qtext, err)
		}
		if len(res) == 0 || res[0].Meta.ID != "s3" {
			t.Fatalf("q=%q: expected hit for s3, got %+v", qtext, res)
		}
	}
	res, err := sr.Search(context.Background(), search.Query{Text: "uniquebodytoken", Tags: []string{"alpha"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Meta.ID != "s3" {
		t.Fatalf("text+tag: got %+v", res)
	}
}

func TestSearchFindsSpacedTag(t *testing.T) {
	dir := t.TempDir()
	if _, err := vault.Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("editor: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, _ := vault.Open(dir)
	e := &vault.Entry{
		ID: "s4", Title: "Spaced tag note", Created: time.Now(), Type: vault.TypeNote,
		Tags: []string{"tag with whitespace"},
		Body: "spacedtagbody",
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
	res, err := sr.Search(context.Background(), search.Query{Tags: []string{"tag with whitespace"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Meta.ID != "s4" {
		t.Fatalf("spaced tag filter: got %+v", res)
	}
}
