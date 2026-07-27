package capture

import (
	"context"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

type stubTagger struct {
	tags []string
}

func (s stubTagger) SuggestTags(context.Context, string, string) ([]string, error) {
	return s.tags, nil
}

type stubPrompter struct {
	gotSuggested []string
	returnTags   []string
}

func (p *stubPrompter) PromptTags(_ context.Context, _ string, suggested []string) ([]string, error) {
	p.gotSuggested = append([]string(nil), suggested...)
	return p.returnTags, nil
}

func TestApplyTagsNoAutoMerge(t *testing.T) {
	t.Parallel()
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, stubTagger{tags: []string{"golang", "ideas"}}, config.Default().Capture)
	e := &vault.Entry{Title: "Test", Body: "content about golang"}
	res, err := c.applyTags(context.Background(), e, "content about golang", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Entry.Tags) != 0 {
		t.Fatalf("tags = %v, want none without prompter", res.Entry.Tags)
	}
	if len(res.SuggestedTags) != 2 {
		t.Fatalf("SuggestedTags = %v", res.SuggestedTags)
	}
}

func TestApplyTagsPrompterChoosesTags(t *testing.T) {
	t.Parallel()
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := &stubPrompter{returnTags: []string{"picked"}}
	c := New(v, stubTagger{tags: []string{"golang"}}, config.Default().Capture)
	e := &vault.Entry{Title: "Test", Body: "golang"}
	res, err := c.applyTags(context.Background(), e, "golang", Options{Prompter: p})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.gotSuggested) != 1 || p.gotSuggested[0] != "golang" {
		t.Fatalf("prompter suggested = %v", p.gotSuggested)
	}
	if len(res.Entry.Tags) != 1 || res.Entry.Tags[0] != "picked" {
		t.Fatalf("tags = %v", res.Entry.Tags)
	}
}

func TestApplyTagsExplicitSkipsSuggest(t *testing.T) {
	t.Parallel()
	v, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := &stubPrompter{returnTags: []string{"should-not-run"}}
	c := New(v, stubTagger{tags: []string{"golang"}}, config.Default().Capture)
	e := &vault.Entry{Title: "Test", Body: "body"}
	res, err := c.applyTags(context.Background(), e, "body", Options{TagsExplicit: true, Tags: []string{"manual"}, Prompter: p})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Entry.Tags) != 1 || res.Entry.Tags[0] != "manual" {
		t.Fatalf("tags = %v", res.Entry.Tags)
	}
	if p.gotSuggested != nil {
		t.Fatal("prompter should not run when tags explicit")
	}
}

func TestParseTagEditorText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "comments and blanks",
			in: "# tags\n\nwork\n\n# ignore\nideas\n",
			want: []string{"work", "ideas"},
		},
		{
			name: "normalize",
			in:   "Foo Bar\nUPPER\n",
			want: []string{"foo-bar", "upper"},
		},
		{
			name: "empty",
			in:   "# only comments\n",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseTagEditorText(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseTagEditorText() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ParseTagEditorText()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseTitleEditorText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "header and title", in: "# title\n\nMy note\n", want: "My note"},
		{name: "first line", in: "Hello world\n", want: "Hello world"},
		{name: "empty", in: "# only\n\n", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseTitleEditorText(tt.in); got != tt.want {
				t.Fatalf("ParseTitleEditorText() = %q, want %q", got, tt.want)
			}
		})
	}
}
