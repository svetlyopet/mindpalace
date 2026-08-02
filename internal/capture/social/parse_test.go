package social

import (
	"os"
	"strings"
	"testing"
)

func TestParseOEmbedHTML_xPost(t *testing.T) {
	t.Parallel()
	html, err := os.ReadFile("testdata/x_oembed.html")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseOEmbedHTML(string(html))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(parsed.Text, "Hello from the oEmbed test post") {
		t.Fatalf("text = %q", parsed.Text)
	}
}

func TestParseOEmbedHTML_xPhoto(t *testing.T) {
	t.Parallel()
	html, err := os.ReadFile("testdata/x_oembed_photo.html")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseOEmbedHTML(string(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Images) != 2 {
		t.Fatalf("images = %d want 2", len(parsed.Images))
	}
	if parsed.Images[0].URL != "https://t.co/abc" {
		t.Fatalf("photo link url = %q", parsed.Images[0].URL)
	}
	if parsed.Images[1].URL != "https://pbs.twimg.com/media/photo.jpg" {
		t.Fatalf("image url = %q", parsed.Images[1].URL)
	}
}

func TestParseOEmbedHTML_xPhotoLinkOnly(t *testing.T) {
	t.Parallel()
	html := `<blockquote class="twitter-tweet"><p lang="en" dir="ltr">👀 <a href="https://t.co/UB24oCzQmn">pic.twitter.com/UB24oCzQmn</a></p></blockquote>`
	parsed, err := ParseOEmbedHTML(html)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Images) != 1 {
		t.Fatalf("images = %d want 1", len(parsed.Images))
	}
	if parsed.Images[0].URL != "https://t.co/UB24oCzQmn" {
		t.Fatalf("image url = %q", parsed.Images[0].URL)
	}
	if len(parsed.Videos) != 0 {
		t.Fatalf("videos = %d want 0", len(parsed.Videos))
	}
}

func TestParseOEmbedHTML_facebook(t *testing.T) {
	t.Parallel()
	html, err := os.ReadFile("testdata/facebook_oembed.html")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseOEmbedHTML(string(html))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(parsed.Text, "Facebook post body text") {
		t.Fatalf("text = %q", parsed.Text)
	}
}

func TestBuildEntry(t *testing.T) {
	t.Parallel()
	post := &Post{
		Platform:     PlatformX,
		CanonicalURL: "https://twitter.com/u/status/1",
		Author: Author{
			DisplayName: "Author",
			Handle:      "author",
			ProfileURL:  "https://x.com/author",
		},
		Text: "Post text here.",
	}
	e, plain := BuildEntry(post, BuildOptions{})
	if e.Type != "social" {
		t.Fatalf("type = %q", e.Type)
	}
	if !strings.Contains(e.Title, "Author") {
		t.Fatalf("title = %q", e.Title)
	}
	if !strings.Contains(e.Body, "Post text here.") {
		t.Fatalf("body = %q", e.Body)
	}
	if e.Extra["platform"] != "x" {
		t.Fatalf("extra platform = %v", e.Extra["platform"])
	}
	if e.Extra["author_name"] != "Author" {
		t.Fatalf("author_name = %v", e.Extra["author_name"])
	}
	author, ok := e.Extra["author"].(map[string]any)
	if !ok {
		t.Fatalf("author extra missing")
	}
	if author["display_name"] != "Author" || author["handle"] != "author" {
		t.Fatalf("author map = %v", author)
	}
	if plain != "Post text here." {
		t.Fatalf("plain = %q", plain)
	}
}

func TestBuildEntry_withThoughts(t *testing.T) {
	t.Parallel()
	post := &Post{
		Platform: PlatformX,
		Text:     "Post text here.",
		Thoughts: "My commentary",
	}
	e, plain := BuildEntry(post, BuildOptions{})
	if !strings.Contains(e.Body, "## Thoughts") || !strings.Contains(e.Body, "My commentary") {
		t.Fatalf("body = %q", e.Body)
	}
	if !strings.Contains(plain, "My commentary") {
		t.Fatalf("plain = %q", plain)
	}
}
