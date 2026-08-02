package server

import (
	"testing"
	"time"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestSocialAuthorFromEntry_nested(t *testing.T) {
	t.Parallel()
	e := &vault.Entry{
		ID:   "e1",
		Type: vault.TypeSocial,
		Extra: map[string]any{
			"platform": "x",
			"author": map[string]any{
				"display_name": "Display",
				"handle":       "user",
				"profile_url":  "https://x.com/user",
				"avatar":       "assets/author-abc.jpg",
			},
		},
	}
	view := socialAuthorFromEntry(e)
	if view == nil {
		t.Fatal("expected view")
	}
	if view.DisplayName != "Display" || view.Handle != "user" || view.ProfileURL != "https://x.com/user" {
		t.Fatalf("view = %+v", view)
	}
	if view.AvatarURL != "/ui/entry/e1/file/assets/author-abc.jpg" {
		t.Fatalf("avatar url = %q", view.AvatarURL)
	}
}

func TestSocialAuthorFromEntry_legacy(t *testing.T) {
	t.Parallel()
	e := &vault.Entry{
		ID:   "e2",
		Type: vault.TypeSocial,
		Extra: map[string]any{
			"author_name": "Legacy Author",
			"author_url":  "https://www.facebook.com/page",
		},
	}
	view := socialAuthorFromEntry(e)
	if view == nil || view.DisplayName != "Legacy Author" || view.ProfileURL != "https://www.facebook.com/page" {
		t.Fatalf("view = %+v", view)
	}
}

func TestSocialAuthorFromEntry_nonSocial(t *testing.T) {
	t.Parallel()
	e := &vault.Entry{ID: "e3", Type: vault.TypeArticle, Created: time.Now()}
	if socialAuthorFromEntry(e) != nil {
		t.Fatal("expected nil for non-social entry")
	}
}
