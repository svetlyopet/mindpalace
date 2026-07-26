package server

import (
	"bytes"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const assetCSSCSP = "default-src 'none'; style-src 'unsafe-inline'"

func sourceHTMLCSP(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	origin := scheme + "://" + r.Host
	return "default-src 'none'; script-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-src 'none'; connect-src 'none'; style-src 'unsafe-inline' " + origin + "; img-src " + origin + " data:; font-src " + origin + ";"
}

func writeEntryFile(w http.ResponseWriter, r *http.Request, rel string, data []byte) {
	w.Header().Set("X-Content-Type-Options", "nosniff")

	base := filepath.Base(rel)
	ctype := entryFileContentType(base)
	w.Header().Set("Content-Type", ctype)

	switch {
	case base == "source.html":
		w.Header().Set("Content-Security-Policy", sourceHTMLCSP(r))
	case strings.HasPrefix(filepath.ToSlash(rel), "assets/") && strings.HasSuffix(strings.ToLower(base), ".css"):
		w.Header().Set("Content-Security-Policy", assetCSSCSP)
	}

	http.ServeContent(w, r, base, time.Time{}, bytes.NewReader(data))
}

func entryFileContentType(base string) string {
	ext := strings.ToLower(filepath.Ext(base))
	switch ext {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".txt", ".md":
		return "text/plain; charset=utf-8"
	case ".pdf":
		return "application/pdf"
	default:
		if byExt := mime.TypeByExtension(ext); byExt != "" {
			return byExt
		}
		return "application/octet-stream"
	}
}
