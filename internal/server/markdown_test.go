package server

import (
	"strings"
	"testing"
)

func TestRenderMarkdownHTML_escapesRawHTML(t *testing.T) {
	src := "hello **bold**\n\n<script>alert(1)</script>"
	html, err := renderMarkdownHTML(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Fatalf("expected bold markdown, got %q", html)
	}
	if strings.Contains(html, "<script>") {
		t.Fatalf("expected escaped script tag, got %q", html)
	}
}
