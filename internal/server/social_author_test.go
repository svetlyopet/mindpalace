package server

import (
	"strings"
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
	if view.DisplayName != "Display" || view.Handle != "user" {
		t.Fatalf("view = %+v", view)
	}
	if !strings.Contains(string(view.NameLinkHTML), `href="https://x.com/user"`) {
		t.Fatalf("name link = %q", view.NameLinkHTML)
	}
	if !strings.Contains(string(view.NameLinkHTML), ">Display<") {
		t.Fatalf("name link = %q", view.NameLinkHTML)
	}
	if !strings.Contains(string(view.HandleLinkHTML), `href="https://x.com/user"`) {
		t.Fatalf("handle link = %q", view.HandleLinkHTML)
	}
	if !strings.Contains(string(view.HandleLinkHTML), ">@user<") {
		t.Fatalf("handle link = %q", view.HandleLinkHTML)
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
	if view == nil || view.DisplayName != "Legacy Author" {
		t.Fatalf("view = %+v", view)
	}
	if !strings.Contains(string(view.NameLinkHTML), `href="https://www.facebook.com/page"`) {
		t.Fatalf("name link = %q", view.NameLinkHTML)
	}
	if !strings.Contains(string(view.NameLinkHTML), ">Legacy Author<") {
		t.Fatalf("name link = %q", view.NameLinkHTML)
	}
}

func TestSocialAuthorFromEntry_nonSocial(t *testing.T) {
	t.Parallel()
	e := &vault.Entry{ID: "e3", Type: vault.TypeArticle, Created: time.Now()}
	if socialAuthorFromEntry(e) != nil {
		t.Fatal("expected nil for non-social entry")
	}
}

func TestSocialAuthorFromEntry_unsafeProfileURL(t *testing.T) {
	t.Parallel()
	e := &vault.Entry{
		ID:   "e4",
		Type: vault.TypeSocial,
		Extra: map[string]any{
			"platform": "x",
			"author": map[string]any{
				"display_name": "Display",
				"handle":       "user",
				"profile_url":  "javascript:alert(1)",
			},
		},
	}
	view := socialAuthorFromEntry(e)
	if view == nil {
		t.Fatal("expected view")
	}
	if view.DisplayName != "Display" || view.Handle != "user" {
		t.Fatalf("view = %+v", view)
	}
	if view.NameLinkHTML != "" || view.HandleLinkHTML != "" || view.ProfileLinkHTML != "" {
		t.Fatalf("expected no link HTML for unsafe profile URL, got name=%q handle=%q profile=%q",
			view.NameLinkHTML, view.HandleLinkHTML, view.ProfileLinkHTML)
	}
}

func TestSocialAuthorFromEntry_facebookViewAccount(t *testing.T) {
	t.Parallel()
	e := &vault.Entry{
		ID:   "e5",
		Type: vault.TypeSocial,
		Extra: map[string]any{
			"platform":   "facebook",
			"author_url": "https://www.facebook.com/page",
		},
	}
	view := socialAuthorFromEntry(e)
	if view == nil {
		t.Fatal("expected view")
	}
	if view.NameLinkHTML != "" {
		t.Fatalf("expected no name link without display name, got %q", view.NameLinkHTML)
	}
	if !strings.Contains(string(view.ProfileLinkHTML), `href="https://www.facebook.com/page"`) {
		t.Fatalf("profile link = %q", view.ProfileLinkHTML)
	}
	if !strings.Contains(string(view.ProfileLinkHTML), ">View account<") {
		t.Fatalf("profile link = %q", view.ProfileLinkHTML)
	}
}
