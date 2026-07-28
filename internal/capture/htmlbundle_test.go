package capture

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundlePage(t *testing.T) {
	const pageHTML = `<!DOCTYPE html><html><head>
<link rel="stylesheet" href="/style.css">
<script src="/app.js">alert(1)</script>
</head><body>
<img src="/img.png" alt="x">
<img src="https://other.example/x.png">
</body></html>`

	mux := http.NewServeMux()
	mux.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte("body{color:red}"))
	})
	mux.HandleFunc("/img.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47})
	})
	mux.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write([]byte("alert(1)"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(pageHTML))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	u, err := url.Parse(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	entryID := "testentry01"
	warns := bundlePage(context.Background(), srv.Client(), u, []byte(pageHTML), entryID, dir)
	crossOrigin := 0
	scriptRemoved := false
	for _, w := range warns {
		if strings.Contains(w, "cross-origin") {
			crossOrigin++
		}
		if strings.Contains(w, "removed script") {
			scriptRemoved = true
		}
	}
	if crossOrigin != 1 {
		t.Fatalf("expected one cross-origin warning, got %v", warns)
	}
	if !scriptRemoved {
		t.Fatalf("expected script removal warning, got %v", warns)
	}

	srcPath := filepath.Join(dir, "source.html")
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if strings.Contains(strings.ToLower(src), "<script") {
		t.Fatalf("script remained in source.html: %s", src)
	}
	prefix := "/ui/entry/" + entryID + "/file/assets/"
	if !strings.Contains(src, prefix) {
		t.Fatalf("expected absolute UI asset paths in source.html: %s", src)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "assets"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 assets (css+png), got %d", len(entries))
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".js") {
			t.Fatalf("unexpected js asset %s", e.Name())
		}
	}
}

func TestBundlePage_cdnStylesheet(t *testing.T) {
	const pageHTML = `<!DOCTYPE html><html><head>
<link rel="stylesheet" href="https://cdn.example.test/main.css">
</head><body><p>hi</p></body></html>`

	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte("p { font-weight: bold; }"))
	}))
	defer cdn.Close()

	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(strings.Replace(pageHTML, "https://cdn.example.test", cdn.URL, 1)))
	}))
	defer page.Close()

	u, err := url.Parse(page.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	entryID := "cdntest01"
	warns := bundlePage(context.Background(), http.DefaultClient, u, []byte(strings.Replace(pageHTML, "https://cdn.example.test", cdn.URL, 1)), entryID, dir)
	for _, w := range warns {
		if strings.Contains(w, "cross-origin") {
			t.Fatalf("CDN stylesheet should be bundled, not skipped: %v", warns)
		}
	}
	src, err := os.ReadFile(filepath.Join(dir, "source.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	if strings.Contains(s, cdn.URL) {
		t.Fatalf("CDN URL should be rewritten in source.html: %s", s)
	}
	prefix := "/ui/entry/" + entryID + "/file/assets/"
	if !strings.Contains(s, prefix) {
		t.Fatalf("expected local asset path: %s", s)
	}
}

func TestBundlePage_svgSanitized(t *testing.T) {
	const pageHTML = `<!DOCTYPE html><html><body><img src="/icon.svg"></body></html>`
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><circle r="1"/></svg>`)
	mux := http.NewServeMux()
	mux.HandleFunc("/icon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write(svg)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	u, _ := url.Parse(srv.URL + "/")
	dir := t.TempDir()
	_ = bundlePage(context.Background(), srv.Client(), u, []byte(pageHTML), "svgtest01", dir)
	assets, _ := os.ReadDir(filepath.Join(dir, "assets"))
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	data, _ := os.ReadFile(filepath.Join(dir, "assets", assets[0].Name()))
	if strings.Contains(strings.ToLower(string(data)), "<script") {
		t.Fatalf("script in stored svg: %s", data)
	}
}
