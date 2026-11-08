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

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func sourceHTMLCSP(r *http.Request, allowScripts bool) string {
	origin := requestOrigin(r)
	if !allowScripts {
		return "default-src 'none'; script-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-src 'none'; connect-src 'none'; style-src 'unsafe-inline' " + origin + "; img-src " + origin + " data:; font-src " + origin + ";"
	}
	return "default-src 'none'; base-uri 'none'; form-action 'none'; object-src 'none'; " +
		"script-src 'unsafe-inline' https: " + origin + "; " +
		"style-src 'unsafe-inline' https: " + origin + "; " +
		"img-src https: data: blob: " + origin + "; " +
		"font-src https: data: " + origin + "; " +
		"connect-src https: " + origin + "; " +
		"frame-src https:; " +
		"media-src https: blob: " + origin + ";"
}

func embedHTMLCSP() string {
	return "default-src 'none'; base-uri 'none'; form-action 'none'; object-src 'none'; " +
		"script-src https://platform.twitter.com https://connect.facebook.net; " +
		"frame-src https://platform.twitter.com https://syndication.twitter.com https://www.facebook.com https://web.facebook.com; " +
		"style-src 'unsafe-inline' https://platform.twitter.com; " +
		"img-src https: data:; " +
		"connect-src https://platform.twitter.com https://syndication.twitter.com https://connect.facebook.net https://www.facebook.com https://graph.facebook.com;"
}

func writeEntryFile(w http.ResponseWriter, r *http.Request, rel string, data []byte, allowScripts bool) {
	w.Header().Set("X-Content-Type-Options", "nosniff")

	base := filepath.Base(rel)
	ctype := entryFileContentType(base)
	w.Header().Set("Content-Type", ctype)

	switch {
	case base == "source.html":
		w.Header().Set("Content-Security-Policy", sourceHTMLCSP(r, allowScripts))
	case base == "embed.html":
		w.Header().Set("Content-Security-Policy", embedHTMLCSP())
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
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
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
