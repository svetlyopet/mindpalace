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

func TestFullTextSearchEmptyWhenBleveRecreatedButMetaStale(t *testing.T) {
	dir := t.TempDir()
	vault.Init(dir)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("editor: \"\"\n"), 0o644)
	v, _ := vault.Open(dir)
	e := &vault.Entry{ID: "desync1", Title: "UniqueKeywordTitle", Created: time.Now(), Type: vault.TypeNote, Body: "UniquePayloadWord"}
	v.Create(e)
	ix, _ := index.Open(dir)
	ix.Put(e)
	ix.Close()

	bleveDir := vault.IndexDir(dir)
	os.RemoveAll(bleveDir)

	ix2, _ := index.Open(dir)
	defer ix2.Close()
	_ = ix2.Refresh(context.Background(), v)

	sr := search.New(ix2)
	metaList, _ := sr.Search(context.Background(), search.Query{Limit: 10})
	ft, _ := sr.Search(context.Background(), search.Query{Text: "UniqueKeywordTitle", Limit: 10})
	t.Logf("metaOnly=%d fullText=%d metaCount=%d", len(metaList), len(ft), len(ix2.All()))
	if len(metaList) == 0 {
		t.Fatal("expected meta-only list")
	}
	if len(ft) == 0 {
		t.Fatal("expected full-text hit after Refresh re-indexed bleve")
	}
}
