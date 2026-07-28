package server

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"

	webassets "github.com/svetlyopet/mindpalace/web"
)

func (s *Server) apiSessionHTTP(w http.ResponseWriter, r *http.Request) {
	s.setCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if s.lib.Vault.Encrypted() && !s.lib.Vault.Unlocked() {
		http.Error(w, "locked", http.StatusUnauthorized)
		return
	}
	s.setSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiUnlockHTTP(w http.ResponseWriter, r *http.Request) {
	s.setCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req unlockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if !s.lib.Vault.Encrypted() {
		s.setSessionCookie(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := s.lib.Vault.Unlock(req.Password); err != nil {
		http.Error(w, "wrong password", http.StatusUnauthorized)
		return
	}
	_ = s.lib.Vault.PersistUnlockSession()
	_ = s.lib.Index.Refresh(r.Context(), s.lib.Vault)
	s.setSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiLock(w http.ResponseWriter, r *http.Request) {
	s.lib.Vault.Lock()
	_ = s.lib.Vault.ClearUnlockSession()
	s.clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMarkdownPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORS(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if s.lib.Vault.Encrypted() && !s.lib.Vault.Unlocked() {
		http.Error(w, "locked", http.StatusUnauthorized)
		return
	}
	if !s.authorizeAPI(r) {
		s.ensureBrowserSession(w, r)
		if !s.authorizeAPI(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	md := r.FormValue("markdown")
	if md == "" {
		var body struct {
			Markdown string `json:"markdown"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		md = body.Markdown
	}
	html, err := renderMarkdownHTML(md)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (s *Server) handleUnlockPage(w http.ResponseWriter, r *http.Request) {
	if !s.lib.Vault.Encrypted() || s.lib.Vault.Unlocked() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	sub, err := fs.Sub(webassets.FS, "templates")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl, err := template.ParseFS(sub, "unlock.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type unlockReq struct {
	Password string `json:"password"`
}
