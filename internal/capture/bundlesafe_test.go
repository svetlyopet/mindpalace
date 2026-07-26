package capture

import (
	"strings"
	"testing"
)

func TestSanitizeBundleHTML_removesScriptAndHandlers(t *testing.T) {
	raw := `<!DOCTYPE html><html><head><script>alert(1)</script>
<script src="/app.js"></script></head><body>
<img src="/x.png" onerror="alert(2)">
<a href="javascript:alert(3)">x</a>
<iframe src="/frame"></iframe>
<link rel="modulepreload" href="/chunk.js">
</body></html>`
	out, warns := SanitizeBundleHTML([]byte(raw))
	if len(out) == 0 {
		t.Fatal("expected sanitized output")
	}
	s := string(out)
	if strings.Contains(strings.ToLower(s), "<script") {
		t.Fatalf("script tag remained: %s", s)
	}
	if strings.Contains(s, "onerror") {
		t.Fatalf("onerror remained: %s", s)
	}
	if strings.Contains(s, "javascript:") {
		t.Fatalf("javascript URL remained: %s", s)
	}
	if !strings.Contains(s, "iframe") {
		t.Fatalf("iframe should be preserved: %s", s)
	}
	if strings.Contains(s, "modulepreload") {
		t.Fatalf("script-loading link should be removed: %s", s)
	}
	if len(warns) == 0 {
		t.Fatal("expected sanitize warnings")
	}
}

func TestSanitizeAsset_cssAndSVG(t *testing.T) {
	css := []byte("body { color: red; }\n@import url('http://cdn.example/x.css');")
	out, err := SanitizeAsset(".css", css)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "@import") {
		t.Fatalf("expected @import preserved: %s", out)
	}

	badSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><foreignObject><body>x</body></foreignObject></svg>`)
	out, err = SanitizeAsset(".svg", badSVG)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(strings.ToLower(s), "<script") {
		t.Fatalf("script in svg: %s", s)
	}
	if !strings.Contains(s, "foreignObject") {
		t.Fatalf("foreignObject should be preserved: %s", s)
	}
}

func TestIsRejectedScriptAsset(t *testing.T) {
	if !isRejectedScriptAsset("application/javascript", ".bin", []byte("alert(1)")) {
		t.Fatal("expected JS content-type rejected")
	}
	if !isRejectedScriptAsset("", ".js", []byte("x=1")) {
		t.Fatal("expected .js rejected")
	}
	if isRejectedScriptAsset("text/css", ".css", []byte("body{}")) {
		t.Fatal("css should pass")
	}
}
