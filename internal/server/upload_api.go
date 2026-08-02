package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/svetlyopet/mindpalace/internal/capture"
	"github.com/svetlyopet/mindpalace/internal/dto"
)

const maxCaptureUploadBytes = 10 << 20

func (s *Server) apiCaptureUploadPreview(w http.ResponseWriter, r *http.Request) {
	filename, data, title, err := readCaptureUpload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opts := capture.Options{Title: title, Thoughts: r.FormValue("thoughts")}
	preview, err := s.lib.Capturer.PreviewUpload(r.Context(), filename, data, opts)
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

func (s *Server) apiCaptureUpload(w http.ResponseWriter, r *http.Request) {
	filename, data, title, err := readCaptureUpload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opts := capture.Options{Title: title, Thoughts: r.FormValue("thoughts")}
	if tags, ok, err := parseUploadTags(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else if ok {
		opts.Tags = tags
		opts.TagsExplicit = true
	}
	res, err := s.lib.Capturer.Upload(r.Context(), filename, data, opts)
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

func readCaptureUpload(r *http.Request) (filename string, data []byte, title string, err error) {
	if err := r.ParseMultipartForm(maxCaptureUploadBytes + 1024); err != nil {
		return "", nil, "", err
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		return "", nil, "", err
	}
	defer file.Close()
	data, err = io.ReadAll(io.LimitReader(file, maxCaptureUploadBytes+1))
	if err != nil {
		return "", nil, "", err
	}
	if len(data) > maxCaptureUploadBytes {
		return "", nil, "", errUploadTooLarge
	}
	title = r.FormValue("title")
	return hdr.Filename, data, title, nil
}

func parseUploadTags(r *http.Request) (tags []string, set bool, err error) {
	raw := r.FormValue("tags")
	if raw == "" {
		return nil, false, nil
	}
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil, false, err
	}
	return tags, true, nil
}

var errUploadTooLarge = captureErr("file exceeds size limit (10 MB)")

type captureErr string

func (e captureErr) Error() string { return string(e) }
