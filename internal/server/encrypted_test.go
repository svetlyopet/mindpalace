package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/library"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/testenv"
)

func testEncryptedServer(t *testing.T, password string) (*Server, string) {
	t.Helper()
	vi := testenv.TempEncryptedVaultIndex(t, password, true)
	sr := search.New(vi.Index)
	cap := testenv.NewCapturer(vi.Vault, vi.Config)
	lib := library.New(vi.Vault, vi.Index, sr, cap)
	token := testenv.ServeToken(t, vi.Dir, vi.Config)
	return New(lib, vi.Config, token), token
}

func TestAPIUnlockEncryptedVault(t *testing.T) {
	const pw = "test-secret"
	s, _ := testEncryptedServer(t, pw)
	s.lib.Vault.Lock()
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/session")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("session status = %d", resp.StatusCode)
	}

	body, _ := json.Marshal(map[string]string{"password": pw})
	resp, err = http.Post(ts.URL+"/api/unlock", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("unlock status = %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/api/session")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("session after unlock = %d", resp.StatusCode)
	}
}

func TestAPIEntryFileEncrypted(t *testing.T) {
	const pw = "file-secret"
	s, token := testEncryptedServer(t, pw)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/entries/abc123/files/entry.md", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
