package library

import (
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/testenv"
)

func TestReadEntryFileEncrypted(t *testing.T) {
	vi := testenv.TempEncryptedVaultIndex(t, "secret", true)
	sr := search.New(vi.Index)
	cap := testenv.NewCapturer(vi.Vault, vi.Config)
	lib := New(vi.Vault, vi.Index, sr, cap)

	got, err := lib.ReadEntryFile("abc123", "entry.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got.Data), "hello") && !strings.Contains(string(got.Data), "Fixture") {
		t.Fatalf("unexpected entry.md content len=%d", len(got.Data))
	}

	vi.Vault.Lock()
	_, err = lib.ReadEntryFile("abc123", "entry.md")
	if err == nil {
		t.Fatal("expected error reading encrypted entry while locked")
	}
}
