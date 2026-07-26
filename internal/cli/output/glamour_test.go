package output

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	out, err := RenderMarkdown("## Hello\n\n- item one\n", 80)
	if err != nil {
		t.Fatal(err)
	}
	plain := stripANSI(out)
	if !strings.Contains(plain, "Hello") {
		t.Fatalf("expected heading text in output: %q", plain)
	}
	if !strings.Contains(plain, "item one") {
		t.Fatalf("expected list item in output: %q", plain)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
