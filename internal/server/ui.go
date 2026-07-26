package server

import (
	"bytes"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/svetlyopet/mindpalace/internal/library"
	webassets "github.com/svetlyopet/mindpalace/web"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

var mdRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

var entryTypes = []string{"article", "note", "social", "screenshot", "snippet"}

func (s *Server) parseTemplates(extra ...string) (*template.Template, error) {
	sub, err := fs.Sub(webassets.FS, "templates")
	if err != nil {
		return nil, err
	}
	files := append([]string{"layout.html"}, extra...)
	return template.ParseFS(sub, files...)
}

func (s *Server) shellTemplateFiles(page ...string) []string {
	base := []string{
		"partials/sidebar.html",
		"partials/entries.html",
		"partials/viewer-empty.html",
	}
	return append(base, page...)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(webassets.FS, "static")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.StripPrefix("/static/", http.FileServer(http.FS(sub))).ServeHTTP(w, r)
}

func (s *Server) handleUIEntryFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rel := r.PathValue("path")
	f, err := s.lib.ReadEntryFile(id, rel)
	if err != nil {
		if errors.Is(err, vault.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, library.ErrForbiddenPath) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if errors.Is(err, vault.ErrLocked) {
			http.Error(w, "locked", http.StatusForbidden)
			return
		}
		http.NotFound(w, r)
		return
	}
	writeEntryFile(w, r, f.Rel, f.Data)
}

type entryViewerData struct {
	Entry           *vault.Entry
	BodyHTML        template.HTML
	ScreenshotURL   string
	HasSourceHTML   bool
	ShowNoteTab     bool
	DefaultTabPage  bool
}

type shellPageData struct {
	Token      string
	Query      string
	Tag        string
	TypeFilter string
	Since      string
	Types      []string
	Results    any
	SelectedID string
	ActiveNav  string
	TagCounts  []search.TagCount
	Viewer     entryViewerData
}

func (s *Server) buildShellData(r *http.Request, selectedID, activeNav string) (shellPageData, error) {
	q, err := queryFromRequest(r)
	if err != nil {
		return shellPageData{}, err
	}
	results, err := s.lib.Search(r.Context(), q)
	if err != nil {
		return shellPageData{}, err
	}
	return shellPageData{
		Token:      s.token,
		Query:      r.URL.Query().Get("q"),
		Tag:        r.URL.Query().Get("tag"),
		TypeFilter: r.URL.Query().Get("type"),
		Since:      r.URL.Query().Get("since"),
		Types:      entryTypes,
		Results:    results,
		SelectedID: selectedID,
		ActiveNav:  activeNav,
	}, nil
}

func (s *Server) renderLayout(w http.ResponseWriter, r *http.Request, pageFiles []string, data shellPageData) {
	s.ensureBrowserSession(w, r)
	tmpl, err := s.parseTemplates(s.shellTemplateFiles(pageFiles...)...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := s.buildShellData(r, "", "library")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.renderLayout(w, r, []string{"library.html"}, data)
}

func (s *Server) handleTagsPage(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildShellData(r, "", "tags")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data.TagCounts = s.lib.ListTags()
	s.renderLayout(w, r, []string{"tags.html"}, data)
}

func (s *Server) buildEntryViewerData(e *vault.Entry) (entryViewerData, error) {
	bodyHTML, err := renderMarkdownHTML(e.Body)
	if err != nil {
		return entryViewerData{}, err
	}
	hasSource := fileExists(filepath.Join(e.Dir, "source.html"))
	showNote := strings.TrimSpace(e.Body) != ""
	data := entryViewerData{
		Entry:          e,
		BodyHTML:       template.HTML(bodyHTML),
		HasSourceHTML:  hasSource,
		ShowNoteTab:    showNote,
		DefaultTabPage: hasSource,
	}
	for _, name := range []string{"screenshot.png", "screenshot.jpg", "screenshot.gif", "screenshot.webp"} {
		if fileExists(filepath.Join(e.Dir, name)) {
			data.ScreenshotURL = "/ui/entry/" + e.ID + "/file/" + name
			break
		}
	}
	return data, nil
}

func (s *Server) handleUIEntryViewer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	e, err := s.lib.GetEntry(id)
	if err != nil {
		if errors.Is(err, vault.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	viewer, err := s.buildEntryViewerData(e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl, err := s.parseTemplates("partials/entry-viewer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "partials/entry-viewer.html", viewer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func (s *Server) handleEntryPage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	e, err := s.lib.GetEntry(id)
	if err != nil {
		if errors.Is(err, vault.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := s.buildShellData(r, id, "library")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	viewer, err := s.buildEntryViewerData(e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data.Viewer = viewer
	s.renderLayout(w, r, []string{"entry.html", "partials/entry-viewer.html"}, data)
}

func renderMarkdownHTML(src string) (string, error) {
	var buf bytes.Buffer
	if err := mdRenderer.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
