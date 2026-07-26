package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/search"
)

func TestSearchParityLibraryAndAPI(t *testing.T) {
	s, token := testServer(t)
	ctx := context.Background()

	q := search.Query{Text: "hello", Limit: 20}
	libHits, err := s.lib.Search(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	libJSON, err := json.Marshal(dto.SearchHitsFrom(libHits))
	if err != nil {
		t.Fatal(err)
	}
	var libDecoded []dto.SearchHit
	if err := json.Unmarshal(libJSON, &libDecoded); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/entries?q=hello", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var apiDecoded []dto.SearchHit
	if err := json.NewDecoder(resp.Body).Decode(&apiDecoded); err != nil {
		t.Fatal(err)
	}
	apiJSON, err := json.Marshal(apiDecoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(libJSON, apiJSON) {
		t.Fatalf("API JSON differs from library:\nlib %s\napi %s", libJSON, apiJSON)
	}
}

func TestDeleteParityLibraryAndAPI(t *testing.T) {
	s, token := testServer(t)
	ctx := context.Background()

	res, err := s.lib.DeleteEntry(ctx, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	libDTO := dto.DeleteFromLibrary(res.ID, res.Title, res.Reindex)

	s2, token2 := testServer(t)
	ts := httptest.NewServer(s2.Handler())
	defer ts.Close()
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/entries/abc123", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var apiDTO dto.DeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiDTO); err != nil {
		t.Fatal(err)
	}
	if libDTO.ID != apiDTO.ID || libDTO.Title != apiDTO.Title {
		t.Fatalf("delete DTO mismatch: lib %+v api %+v", libDTO, apiDTO)
	}
	_ = token
}
