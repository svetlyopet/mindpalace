package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/svetlyopet/mindpalace/internal/dto"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestProbeSession(t *testing.T) {
	t.Parallel()
	ready := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/session" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ready.Close()
	if got := ProbeSession(ready.URL); got != ProbeReady {
		t.Fatalf("ProbeSession ready = %v, want ProbeReady", got)
	}

	locked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "locked", http.StatusUnauthorized)
	}))
	defer locked.Close()
	if got := ProbeSession(locked.URL); got != ProbeLocked {
		t.Fatalf("ProbeSession locked = %v, want ProbeLocked", got)
	}

	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusOK)
	}))
	defer other.Close()
	if got := ProbeSession(other.URL); got != ProbeOther {
		t.Fatalf("ProbeSession other = %v, want ProbeOther", got)
	}

	if got := ProbeSession("127.0.0.1:1"); got != ProbeDown {
		t.Fatalf("ProbeSession down = %v, want ProbeDown", got)
	}
}

func TestClientListEntriesAndAuth(t *testing.T) {
	t.Parallel()
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/entries" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("q") != "go" {
			t.Errorf("q=%q", r.URL.Query().Get("q"))
		}
		_ = json.NewEncoder(w).Encode([]dto.SearchHit{{ID: "abc", Title: "Go"}})
	}))
	defer srv.Close()

	c := New(srv.Listener.Addr().String(), "tok")
	c.BaseURL = srv.URL
	hits, err := c.ListEntries(context.Background(), search.Query{Text: "go", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "abc" {
		t.Fatalf("hits=%v", hits)
	}
	if sawAuth != "Bearer tok" {
		t.Fatalf("Authorization=%q", sawAuth)
	}
}

func TestClientGetEntryNotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := New(srv.URL, "t")
	_, err := c.GetEntry(context.Background(), "missing")
	if !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestClientUnlockLocked(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "wrong password", http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := New(srv.URL, "t")
	err := c.Unlock(context.Background(), "bad")
	if !errors.Is(err, vault.ErrWrongPassword) {
		t.Fatalf("err=%v, want ErrWrongPassword", err)
	}
}

func TestClientCapture(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/capture" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(dto.CaptureResponse{
			Entry: dto.Entry{ID: "n1", Title: "Note", Created: time.Now().Format(time.RFC3339), Type: "note"},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, "t")
	tags := []string{"a"}
	res, err := c.Capture(context.Background(), CaptureReq{Kind: "note", Text: "hi", Title: "Note", Tags: &tags})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry.ID != "n1" {
		t.Fatalf("entry=%v", res.Entry)
	}
}

func TestRefuseErrors(t *testing.T) {
	t.Parallel()
	if !errors.Is(RefuseReindex(), ErrServeRunning) {
		t.Fatal("RefuseReindex")
	}
	if !errors.Is(RefuseEncryptionChange(), ErrServeRunning) {
		t.Fatal("RefuseEncryptionChange")
	}
}
