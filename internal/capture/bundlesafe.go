package capture

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	dangerousURLScheme = regexp.MustCompile(`(?i)^(?:javascript|vbscript):`)
	cssJSURLRe         = regexp.MustCompile(`(?i)url\s*\(\s*['"]?\s*javascript:`)
)

// SanitizeBundleHTML strips script execution vectors while preserving page markup.
func SanitizeBundleHTML(raw []byte) ([]byte, []string) {
	var warnings []string
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, append(warnings, "html bundle: sanitize parse failed: "+err.Error())
	}

	warnings = append(warnings, stripScriptVectors(doc)...)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, append(warnings, "html bundle: sanitize render failed: "+err.Error())
	}
	if buf.Len() == 0 {
		return minimalSafeBundleHTML("Page content could not be sanitized safely."), warnings
	}
	return buf.Bytes(), warnings
}

func minimalSafeBundleHTML(message string) []byte {
	msg := html.EscapeString(message)
	return []byte("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Saved page</title></head><body><p>" + msg + "</p></body></html>")
}

func stripScriptVectors(doc *html.Node) []string {
	var warnings []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			if c.Type == html.ElementNode {
				tag := strings.ToLower(c.Data)
				switch tag {
				case "script":
					src := attrVal(c, "src")
					if src != "" {
						warnings = append(warnings, fmt.Sprintf("html bundle: removed script referencing %q", src))
					} else {
						warnings = append(warnings, "html bundle: removed script element")
					}
					n.RemoveChild(c)
					c = next
					continue
				case "link":
					if isScriptLoadingLink(c) {
						warnings = append(warnings, fmt.Sprintf("html bundle: removed script-loading link rel=%q", attrVal(c, "rel")))
						n.RemoveChild(c)
						c = next
						continue
					}
				}
				stripScriptExecutionAttrs(c, &warnings)
			}
			walk(c)
			c = next
		}
	}
	walk(doc)
	return warnings
}

func isScriptLoadingLink(n *html.Node) bool {
	rel := strings.ToLower(attrVal(n, "rel"))
	if strings.Contains(rel, "modulepreload") {
		return true
	}
	if strings.Contains(rel, "preload") {
		as := strings.ToLower(attrVal(n, "as"))
		return as == "script"
	}
	return false
}

func stripScriptExecutionAttrs(n *html.Node, warnings *[]string) {
	var kept []html.Attribute
	for _, a := range n.Attr {
		key := strings.ToLower(a.Key)
		if strings.HasPrefix(key, "on") {
			*warnings = append(*warnings, fmt.Sprintf("html bundle: stripped event handler %s", key))
			continue
		}
		switch key {
		case "href", "src", "xlink:href", "formaction":
			if dangerousURLScheme.MatchString(strings.TrimSpace(a.Val)) {
				*warnings = append(*warnings, fmt.Sprintf("html bundle: stripped dangerous URL in %s", key))
				continue
			}
		}
		kept = append(kept, a)
	}
	n.Attr = kept
}

func entryAssetURL(entryID, assetsRel string) string {
	return "/ui/entry/" + entryID + "/file/" + assetsRel
}

func isRejectedScriptAsset(ct, ext string, data []byte) bool {
	ct = strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	switch ct {
	case "application/javascript", "text/javascript", "application/ecmascript",
		"text/ecmascript", "application/x-javascript", "application/wasm":
		return true
	}
	ext = strings.ToLower(ext)
	switch ext {
	case ".js", ".mjs", ".wasm":
		return true
	}
	trim := strings.TrimSpace(string(data))
	if len(trim) > 0 && (strings.HasPrefix(trim, "#!") || strings.HasPrefix(trim, "(function")) {
		if ct == "" || strings.Contains(ct, "text") || strings.Contains(ct, "javascript") {
			return true
		}
	}
	return false
}

// SanitizeAsset blocks script content in CSS/SVG assets; other types pass through unchanged.
func SanitizeAsset(ext string, data []byte) ([]byte, error) {
	ext = strings.ToLower(ext)
	if ext == ".css" || ext == "css" {
		return sanitizeCSS(data)
	}
	if ext == ".svg" || ext == "svg" {
		return sanitizeSVG(data)
	}
	return data, nil
}

func sanitizeCSS(data []byte) ([]byte, error) {
	s := string(data)
	if cssJSURLRe.MatchString(s) {
		return nil, fmt.Errorf("css contains javascript URL")
	}
	return data, nil
}

func sanitizeSVG(data []byte) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("svg parse: %w", err)
	}
	_ = stripScriptVectors(doc)
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
