package server

import "testing"

func TestSafeHTTPURL(t *testing.T) {
	tests := []struct {
		raw    string
		wantOK bool
		want   string
	}{
		{"https://example.com/path", true, "https://example.com/path"},
		{"http://example.com", true, "http://example.com"},
		{"javascript:alert(1)", false, ""},
		{"data:text/html,test", false, ""},
		{"", false, ""},
		{"not a url", false, ""},
		{"/relative/path", false, ""},
	}
	for _, tt := range tests {
		got, ok := safeHTTPURL(tt.raw)
		if ok != tt.wantOK {
			t.Errorf("safeHTTPURL(%q) ok = %v, want %v", tt.raw, ok, tt.wantOK)
			continue
		}
		if ok && got != tt.want {
			t.Errorf("safeHTTPURL(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestNavLinkClass(t *testing.T) {
	if navLinkClass("library", "library") != "nav-link is-active" {
		t.Fatal("expected active library class")
	}
	if navLinkClass("library", "tags") != "nav-link" {
		t.Fatal("expected inactive tags class")
	}
}

func TestAnchorLinkHTML(t *testing.T) {
	if got := anchorLinkHTML("", "https://example.com", "label", ""); got != "" {
		t.Fatalf("expected empty for blank class, got %q", got)
	}
	if got := anchorLinkHTML("entry-author-name", "", "label", ""); got != "" {
		t.Fatalf("expected empty for blank href, got %q", got)
	}
	if got := anchorLinkHTML("entry-author-name", "https://example.com", "", ""); got != "" {
		t.Fatalf("expected empty for blank label, got %q", got)
	}
	got := string(anchorLinkHTML("entry-author-name", "https://example.com", "Display", "noopener noreferrer"))
	want := `<a class="entry-author-name" href="https://example.com" target="_blank" rel="noopener noreferrer">Display</a>`
	if got != want {
		t.Fatalf("anchorLinkHTML = %q, want %q", got, want)
	}
}

func TestSourceLinkHTML(t *testing.T) {
	got := string(sourceLinkHTML("https://example.com", "https://example.com"))
	want := `<a class="source-link" href="https://example.com" target="_blank" rel="noopener">https://example.com</a>`
	if got != want {
		t.Fatalf("sourceLinkHTML = %q, want %q", got, want)
	}
}
