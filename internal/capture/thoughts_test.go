package capture

import (
	"strings"
	"testing"
)

func TestAppendThoughts_empty(t *testing.T) {
	t.Parallel()
	if got := AppendThoughts("hello", ""); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if got := AppendThoughts("", ""); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendThoughts_appends(t *testing.T) {
	t.Parallel()
	got := AppendThoughts("## Post\n\ntext", "my note")
	for _, part := range []string{"## Post", "## Thoughts", "my note"} {
		if !strings.Contains(got, part) {
			t.Fatalf("got %q missing %q", got, part)
		}
	}
}

func TestAppendThoughts_onlyThoughts(t *testing.T) {
	t.Parallel()
	got := AppendThoughts("", "solo thought")
	want := "## Thoughts\n\nsolo thought"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMergeIndexText(t *testing.T) {
	t.Parallel()
	if got := MergeIndexText("ocr text", "comment"); got != "ocr text\n\ncomment" {
		t.Fatalf("got %q", got)
	}
	if got := MergeIndexText("", "comment"); got != "comment" {
		t.Fatalf("got %q", got)
	}
}
