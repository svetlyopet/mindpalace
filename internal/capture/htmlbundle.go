package capture

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/fsperm"
	"golang.org/x/net/html"
)

const (
	maxAssetBytes  = 5 << 20
	maxAssetCount  = 50
	assetsDirName  = "assets"
	sourceHTMLName = "source.html"
)

type assetRef struct {
	node             *html.Node
	attr             string
	raw              string
	allowCrossOrigin bool
}

// bundlePage saves source.html and same-origin assets/ for iteration-2 file serving
// (GET .../files/source.html, .../files/assets/*).
func bundlePage(ctx context.Context, client *http.Client, pageURL *url.URL, rawHTML []byte, entryID, entryDir string) []string {
	var warnings []string
	if client == nil {
		client = http.DefaultClient
	}

	safeHTML, sanitizeWarns := SanitizeBundleHTML(rawHTML)
	warnings = append(warnings, sanitizeWarns...)
	if len(safeHTML) == 0 {
		warnings = append(warnings, "html bundle: using minimal safe document after sanitize failure")
		safeHTML = minimalSafeBundleHTML("Sanitization failed.")
	}
	rawHTML = safeHTML

	doc, err := html.Parse(bytes.NewReader(rawHTML))
	if err != nil {
		warnings = append(warnings, "html bundle: parse failed after sanitize: "+err.Error())
		fallback := minimalSafeBundleHTML("Could not parse saved page HTML.")
		if werr := os.WriteFile(filepath.Join(entryDir, sourceHTMLName), fallback, fsperm.PrivateFileMode); werr != nil {
			warnings = append(warnings, "html bundle: write source.html: "+werr.Error())
		}
		return warnings
	}

	assetsDir := filepath.Join(entryDir, assetsDirName)
	if err := os.MkdirAll(assetsDir, fsperm.DirMode); err != nil {
		warnings = append(warnings, "html bundle: mkdir assets: "+err.Error())
		return warnings
	}

	type ref = assetRef
	var refs []ref
	collectAssetRefs(doc, &refs)

	downloaded := 0
	urlToLocal := map[string]string{}

	for _, r := range refs {
		if downloaded >= maxAssetCount {
			warnings = append(warnings, "html bundle: asset limit reached")
			break
		}
		abs, ok := resolveBundleAsset(pageURL, r.raw, r.allowCrossOrigin)
		if !ok {
			if r.allowCrossOrigin {
				removeNode(r.node)
				warnings = append(warnings, fmt.Sprintf("html bundle: dropped stylesheet with invalid URL %q", r.raw))
			} else if strings.TrimSpace(r.raw) != "" && !strings.HasPrefix(strings.TrimSpace(r.raw), "data:") {
				warnings = append(warnings, fmt.Sprintf("html bundle: skipped cross-origin or invalid URL %q", r.raw))
			}
			continue
		}
		absKey := abs.String()
		localRel, seen := urlToLocal[absKey]
		if !seen {
			data, ext, ct, err := fetchAsset(ctx, client, abs)
			if err != nil {
				if r.allowCrossOrigin {
					removeNode(r.node)
				}
				warnings = append(warnings, fmt.Sprintf("html bundle: fetch %s: %v", absKey, err))
				continue
			}
			if isRejectedScriptAsset(ct, ext, data) {
				if r.allowCrossOrigin {
					removeNode(r.node)
				}
				warnings = append(warnings, fmt.Sprintf("html bundle: rejected script asset %s", absKey))
				continue
			}
			if r.allowCrossOrigin {
				ext = normalizeStylesheetExt(ext, ct, data)
				if ext != ".css" {
					removeNode(r.node)
					warnings = append(warnings, fmt.Sprintf("html bundle: rejected non-css stylesheet %s", absKey))
					continue
				}
			}
			data, err = SanitizeAsset(ext, data)
			if err != nil {
				if r.allowCrossOrigin {
					removeNode(r.node)
				}
				warnings = append(warnings, fmt.Sprintf("html bundle: sanitize asset %s: %v", absKey, err))
				continue
			}
			name := assetFilename(absKey, ext)
			dest := filepath.Join(assetsDir, name)
			if err := os.WriteFile(dest, data, fsperm.PrivateFileMode); err != nil {
				if r.allowCrossOrigin {
					removeNode(r.node)
				}
				warnings = append(warnings, fmt.Sprintf("html bundle: write %s: %v", name, err))
				continue
			}
			localRel = assetsDirName + "/" + name
			urlToLocal[absKey] = localRel
			downloaded++
		}
		setAttr(r.node, r.attr, entryAssetURL(entryID, localRel))
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		warnings = append(warnings, "html bundle: render failed: "+err.Error())
		if werr := os.WriteFile(filepath.Join(entryDir, sourceHTMLName), rawHTML, fsperm.PrivateFileMode); werr != nil {
			warnings = append(warnings, "html bundle: write source.html: "+werr.Error())
		}
		return warnings
	}
	if err := os.WriteFile(filepath.Join(entryDir, sourceHTMLName), buf.Bytes(), fsperm.PrivateFileMode); err != nil {
		warnings = append(warnings, "html bundle: write source.html: "+err.Error())
	}
	return warnings
}

func collectAssetRefs(n *html.Node, refs *[]assetRef) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "img", "source":
			if v := attrVal(n, "src"); v != "" {
				*refs = append(*refs, assetRef{node: n, attr: "src", raw: v})
			}
		case "link":
			rel := strings.ToLower(attrVal(n, "rel"))
			if strings.Contains(rel, "stylesheet") {
				if v := attrVal(n, "href"); v != "" {
					*refs = append(*refs, assetRef{node: n, attr: "href", raw: v, allowCrossOrigin: true})
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectAssetRefs(c, refs)
	}
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func removeNode(n *html.Node) {
	if n != nil && n.Parent != nil {
		n.Parent.RemoveChild(n)
	}
}

func resolveBundleAsset(page *url.URL, raw string, allowCrossOrigin bool) (*url.URL, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "data:") {
		return nil, false
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	abs := page.ResolveReference(ref)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return nil, false
	}
	if !allowCrossOrigin && (abs.Scheme != page.Scheme || abs.Host != page.Host) {
		return nil, false
	}
	return abs, true
}

func normalizeStylesheetExt(ext, ct string, data []byte) string {
	ext = strings.ToLower(ext)
	if ext == ".css" {
		return ext
	}
	ct = strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	if ct == "text/css" {
		return ".css"
	}
	trim := strings.TrimSpace(string(data))
	if len(trim) > 0 && !strings.HasPrefix(trim, "<") {
		return ".css"
	}
	return ext
}

func fetchAsset(ctx context.Context, client *http.Client, u *url.URL) ([]byte, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAssetBytes+1))
	if err != nil {
		return nil, "", ct, err
	}
	if len(data) > maxAssetBytes {
		return nil, "", ct, fmt.Errorf("asset exceeds %d bytes", maxAssetBytes)
	}
	ext := extFromContentType(ct)
	if ext == "" {
		ext = filepath.Ext(u.Path)
	}
	return data, ext, ct, nil
}

func extFromContentType(ct string) string {
	ct = strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	switch ct {
	case "text/css":
		return ".css"
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	default:
		return ""
	}
}

func assetFilename(absURL, ext string) string {
	sum := sha256.Sum256([]byte(absURL))
	name := hex.EncodeToString(sum[:8])
	ext = strings.ToLower(ext)
	if ext == "" {
		ext = ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return name + ext
}
