package search

import (
	"net/url"
	"testing"
	"time"
)

func TestQueryFromURLIgnoresEmptyType(t *testing.T) {
	u, _ := url.Parse("/ui/entries?q=hello&type=")
	q, err := QueryFromURL(*u, fixedNow())
	if err != nil {
		t.Fatalf("empty type param: %v", err)
	}
	if q.Text != "hello" {
		t.Fatalf("q.Text = %q", q.Text)
	}
	if len(q.Types) != 0 {
		t.Fatalf("Types = %v", q.Types)
	}
}

func TestQueryFromURLIgnoresEmptyTag(t *testing.T) {
	u, _ := url.Parse("/ui/entries?q=Fixture&tag=&selected=")
	q, err := QueryFromURL(*u, fixedNow())
	if err != nil {
		t.Fatalf("empty tag param: %v", err)
	}
	if len(q.Tags) != 0 {
		t.Fatalf("Tags = %q, want none", q.Tags)
	}
}

func TestQueryFromURLSingleTagParam(t *testing.T) {
	u, _ := url.Parse("/?q=x&tag=foo&tag=bar")
	q, err := QueryFromURL(*u, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Tags) != 1 || q.Tags[0] != "foo" {
		t.Fatalf("Tags = %v, want [foo]", q.Tags)
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
}
