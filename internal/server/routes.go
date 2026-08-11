package server

import (
	"net/http"
)

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /static/", s.handleStatic)
	mux.HandleFunc("GET /unlock", s.handleUnlockPage)
	mux.HandleFunc("POST /ui/markdown-preview", s.handleMarkdownPreview)
	mux.HandleFunc("POST /api/unlock", s.apiUnlockHTTP)
	mux.HandleFunc("GET /api/session", s.apiSessionHTTP)

	mux.HandleFunc("GET /ui/entry/{id}", s.wrapUI(s.handleUIEntryViewer))
	mux.HandleFunc("GET /ui/entry/{id}/file/{path...}", s.wrapUI(s.handleUIEntryFile))
	mux.HandleFunc("GET /", s.wrapUI(s.handleLibrary))
	mux.HandleFunc("GET /tags", s.wrapUI(s.handleTagsPage))
	mux.HandleFunc("GET /entry/{id}", s.wrapUI(s.handleEntryPage))

	s.handleAPIFunc(mux, "POST /api/lock", s.apiLock)
	s.handleAPIFunc(mux, "GET /api/tags", s.apiTags)
	s.handleAPIFunc(mux, "GET /api/entries", s.apiListEntries)
	s.handleAPIFunc(mux, "POST /api/capture", s.apiCapture)
	s.handleAPIFunc(mux, "POST /api/capture/preview", s.apiCapturePreview)
	s.handleAPIFunc(mux, "POST /api/capture/upload", s.apiCaptureUpload)
	s.handleAPIFunc(mux, "POST /api/capture/upload/preview", s.apiCaptureUploadPreview)
	s.handleAPIFunc(mux, "GET /api/entries/{id}", s.apiGetEntry)
	s.handleAPIFunc(mux, "GET /api/entries/{id}/files/{path...}", s.apiServeFileRoute)
	s.handleAPIFunc(mux, "POST /api/entries/{id}/tags", s.apiUpdateTagsRoute)
	s.handleAPIFunc(mux, "DELETE /api/entries/{id}", s.apiDeleteEntryRoute)

	// Catch unknown UI paths. Split patterns avoid conflicting with GET / —
	// a lone GET /{path...} also matches the root in Go 1.22+ ServeMux.
	mux.HandleFunc("GET /{path}", s.wrapUI(s.redirectToHome))
	mux.HandleFunc("GET /{path}/{rest...}", s.wrapUI(s.redirectToHome))
}

func (s *Server) wrapUI(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.requireUnlockedUI(w, r) {
			return
		}
		fn(w, r)
	}
}

func (s *Server) handleAPIFunc(mux *http.ServeMux, pattern string, fn http.HandlerFunc) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			s.setCORS(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !s.authorizeAPI(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.setCORS(w, r)
		fn(w, r)
	})
}

func (s *Server) apiServeFileRoute(w http.ResponseWriter, r *http.Request) {
	s.apiServeFile(w, r, r.PathValue("id"), r.PathValue("path"))
}

func (s *Server) apiUpdateTagsRoute(w http.ResponseWriter, r *http.Request) {
	s.apiUpdateTags(w, r, r.PathValue("id"))
}

func (s *Server) apiGetEntry(w http.ResponseWriter, r *http.Request) {
	s.apiGetEntryByID(w, r, r.PathValue("id"))
}

func (s *Server) apiDeleteEntryRoute(w http.ResponseWriter, r *http.Request) {
	s.apiDeleteEntry(w, r, r.PathValue("id"))
}
