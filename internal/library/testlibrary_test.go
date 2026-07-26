package library

import (
	"testing"

	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/testenv"
)

func testLibrary(t *testing.T) *Library {
	t.Helper()
	vi := testenv.TempVaultIndex(t, true)
	sr := search.New(vi.Index)
	cap := testenv.NewCapturer(vi.Vault, vi.Config)
	return New(vi.Vault, vi.Index, sr, cap)
}
