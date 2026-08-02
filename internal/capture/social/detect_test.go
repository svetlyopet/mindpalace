package social

import "testing"

func TestMatchX(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in       string
		wantPlat Platform
		wantURL  string
		ok       bool
	}{
		{"https://x.com/Interior/status/463440424141459456", PlatformX, "https://twitter.com/Interior/status/463440424141459456", true},
		{"https://twitter.com/Interior/status/463440424141459456?s=20", PlatformX, "https://twitter.com/Interior/status/463440424141459456", true},
		{"https://x.com/Interior", "", "", false},
		{"https://example.com/status/123", "", "", false},
	}
	for _, tc := range cases {
		plat, canon, ok := Match(tc.in)
		if ok != tc.ok {
			t.Fatalf("Match(%q) ok=%v want %v", tc.in, ok, tc.ok)
		}
		if !ok {
			continue
		}
		if plat != tc.wantPlat {
			t.Fatalf("Match(%q) plat=%q want %q", tc.in, plat, tc.wantPlat)
		}
		if canon != tc.wantURL {
			t.Fatalf("Match(%q) canon=%q want %q", tc.in, canon, tc.wantURL)
		}
	}
}

func TestMatchFacebook(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in string
		ok bool
	}{
		{"https://www.facebook.com/page/posts/123456", true},
		{"https://m.facebook.com/story.php?story_fbid=999&id=100", true},
		{"https://www.facebook.com/photo.php?fbid=123", true},
		{"https://www.facebook.com/watch?v=12345", true},
		{"https://fb.watch/abc123/", true},
		{"https://www.facebook.com/page", false},
	}
	for _, tc := range cases {
		_, _, ok := Match(tc.in)
		if ok != tc.ok {
			t.Fatalf("Match(%q) ok=%v want %v", tc.in, ok, tc.ok)
		}
	}
}
