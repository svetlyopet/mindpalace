package social

import "testing"

func TestHandleFromProfileURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		url  string
		want string
	}{
		{"https://x.com/UK_Daniel_Card", "UK_Daniel_Card"},
		{"https://twitter.com/testauthor", "testauthor"},
		{"https://x.com/testauthor/status/1", ""},
		{"https://facebook.com/page", ""},
	}
	for _, tc := range tests {
		if got := handleFromProfileURL(tc.url); got != tc.want {
			t.Errorf("handleFromProfileURL(%q) = %q want %q", tc.url, got, tc.want)
		}
	}
}

func TestHandleFromOEmbedHTML(t *testing.T) {
	t.Parallel()
	html := `<blockquote>&mdash; Test Author (@testauthor) <a href="https://twitter.com/testauthor/status/1">March 1</a></blockquote>`
	if got := handleFromOEmbedHTML(html); got != "testauthor" {
		t.Fatalf("handle = %q", got)
	}
}

func TestBuildAuthorFromOEmbed_x(t *testing.T) {
	t.Parallel()
	html := `&mdash; Display (@handleuser)`
	a := BuildAuthorFromOEmbed(PlatformX, "Display", "https://x.com/handleuser", html)
	if a.DisplayName != "Display" {
		t.Fatalf("display = %q", a.DisplayName)
	}
	if a.Handle != "handleuser" {
		t.Fatalf("handle = %q", a.Handle)
	}
	if a.ProfileURL != "https://x.com/handleuser" {
		t.Fatalf("profile = %q", a.ProfileURL)
	}
}

func TestBuildAuthorFromOEmbed_facebook(t *testing.T) {
	t.Parallel()
	a := BuildAuthorFromOEmbed(PlatformFacebook, "Page Name", "https://www.facebook.com/mypage", "")
	if a.DisplayName != "Page Name" {
		t.Fatalf("display = %q", a.DisplayName)
	}
	if a.Handle != "" {
		t.Fatalf("handle = %q want empty", a.Handle)
	}
}

func TestAuthorExtraMap(t *testing.T) {
	t.Parallel()
	m := Author{
		DisplayName: "Name",
		Handle:      "user",
		ProfileURL:  "https://x.com/user",
		AvatarPath:  "assets/author-abc.jpg",
	}.AuthorExtraMap()
	if m["display_name"] != "Name" || m["handle"] != "user" || m["profile_url"] != "https://x.com/user" || m["avatar"] != "assets/author-abc.jpg" {
		t.Fatalf("map = %v", m)
	}
}
