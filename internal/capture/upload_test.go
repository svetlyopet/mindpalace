package capture

import (
	"context"
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestUploadTextSnippet(t *testing.T) {
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, KeywordTagger{}, config.Default().Capture)
	body := "package main\n\nfunc main() { println(\"golang\") }\n"
	res, err := c.Upload(context.Background(), "main.go", []byte(body), Options{Title: "main.go", TagsExplicit: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry.Type != vault.TypeSnippet {
		t.Fatalf("type = %q", res.Entry.Type)
	}
	if !strings.Contains(res.Entry.Body, "package main") {
		t.Fatal("expected body in entry")
	}
}

func TestUploadTooLarge(t *testing.T) {
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, KeywordTagger{}, config.Default().Capture)
	data := make([]byte, maxUploadSize+1)
	_, err = c.Upload(context.Background(), "big.bin", data, Options{})
	if err == nil {
		t.Fatal("expected size error")
	}
}

func TestPreviewUploadText(t *testing.T) {
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, KeywordTagger{}, config.Default().Capture)
	p, err := c.PreviewUpload(context.Background(), "notes.md", []byte("# Ideas\n\ngolang capture"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "notes.md" {
		t.Fatalf("title = %q", p.Title)
	}
	if p.Type != vault.TypeSnippet {
		t.Fatalf("type = %q", p.Type)
	}
}

func TestPreviewUploadNoAutoTagsOnEntry(t *testing.T) {
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, KeywordTagger{}, config.Default().Capture)
	res, err := c.Upload(context.Background(), "x.txt", []byte("golang golang frameworks"), Options{Title: "x.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Entry.Tags) != 0 {
		t.Fatalf("tags = %v, want none without explicit tags", res.Entry.Tags)
	}
}
