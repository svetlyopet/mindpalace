package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/dto"
)

func TestAPIEntriesSearchWithEmptyTypeParam(t *testing.T) {
	s, token := testServer(t)
	req := httptest.NewRequest("GET", "/api/entries?q=Fixture&type=&since=", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.apiListEntries(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
	var hits []dto.SearchHit
	if err := json.Unmarshal(w.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].Title != "Fixture" {
		t.Fatalf("hits = %+v", hits)
	}
}

func TestAPIEntriesSearchWithEmptyTagParam(t *testing.T) {
	s, token := testServer(t)
	req := httptest.NewRequest("GET", "/api/entries?q=Fixture&tag=&selected=&type=&since=", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.apiListEntries(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
	var hits []dto.SearchHit
	if err := json.Unmarshal(w.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].Title != "Fixture" {
		t.Fatalf("hits = %+v", hits)
	}
}
