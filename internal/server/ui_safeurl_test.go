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
