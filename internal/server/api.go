package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/svetlyopet/mindpalace/internal/capture"
	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/library"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func (s *Server) apiListEntries(w http.ResponseWriter, r *http.Request) {
	q, err := queryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	results, err := s.lib.Search(r.Context(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, dto.SearchHitsFrom(results))
}

func (s *Server) apiGetEntryByID(w http.ResponseWriter, r *http.Request, id string) {
	e, err := s.lib.GetEntry(id)
	if err != nil {
		if errors.Is(err, vault.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, dto.EntryFromVault(e))
}

func (s *Server) apiTags(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, dto.TagCountsFrom(s.lib.ListTags()))
}

type tagUpdateReq struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

func (s *Server) apiUpdateTags(w http.ResponseWriter, r *http.Request, id string) {
	var req tagUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	e, err := s.lib.UpdateTags(r.Context(), id, req.Add, req.Remove)
	if err != nil {
		if errors.Is(err, vault.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, dto.EntryFromVault(e))
}

func (s *Server) apiDeleteEntry(w http.ResponseWriter, r *http.Request, id string) {
	res, err := s.lib.DeleteEntry(r.Context(), id)
	if err != nil {
		if errors.Is(err, vault.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, dto.DeleteFromLibrary(res.ID, res.Title, res.Reindex))
}

type captureReq struct {
	Kind  string    `json:"kind"`
	Text  string    `json:"text"`
	URL   string    `json:"url"`
	HTML  string    `json:"html"`
	Title string    `json:"title"`
	Tags  *[]string `json:"tags"`
	Type  string    `json:"type"`
	Full  bool      `json:"full"`
}

func captureOptionsFromReq(req captureReq) capture.Options {
	tagsExplicit := req.Tags != nil
	var tags []string
	if req.Tags != nil {
		tags = *req.Tags
	}
	typ := vault.Type(req.Type)
	return library.CaptureOptionsFromFields(req.Title, tags, tagsExplicit, typ, req.Full)
}

func (s *Server) apiCapturePreview(w http.ResponseWriter, r *http.Request) {
	var req captureReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opts := captureOptionsFromReq(req)
	if req.Type != "" && !opts.Type.Valid() {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	var preview *capture.Preview
	var err error
	switch req.Kind {
	case "note":
		preview, err = s.lib.Capturer.PreviewNote(r.Context(), req.Text, opts)
	case "url":
		preview, err = s.lib.Capturer.PreviewURL(r.Context(), req.URL, opts)
	case "html":
		preview, err = s.lib.Capturer.PreviewHTML(r.Context(), req.URL, []byte(req.HTML), opts)
	default:
		http.Error(w, "invalid kind", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{
		"title":          preview.Title,
		"type":           preview.Type,
		"suggested_tags": preview.SuggestedTags,
	})
}

func (s *Server) apiCapture(w http.ResponseWriter, r *http.Request) {
	var req captureReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opts := captureOptionsFromReq(req)
	if req.Type != "" {
		if !opts.Type.Valid() {
			http.Error(w, "invalid type", http.StatusBadRequest)
			return
		}
	}
	var res *capture.Result
	var err error
	switch req.Kind {
	case "note":
		res, err = s.lib.Capturer.Note(r.Context(), req.Text, opts)
	case "url":
		res, err = s.lib.Capturer.URL(r.Context(), req.URL, opts)
	case "html":
		res, err = s.lib.Capturer.URLFromHTML(r.Context(), req.URL, []byte(req.HTML), opts)
	default:
		http.Error(w, "invalid kind", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.lib.CommitCapture(r.Context(), res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, dto.NewCaptureResponse(res.Entry, res.Warnings, res.SuggestedTags))
}

func queryFromRequest(r *http.Request) (search.Query, error) {
	return search.QueryFromURL(*r.URL, time.Now())
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) apiServeFile(w http.ResponseWriter, r *http.Request, id, rel string) {
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
