package testenv

import (
	"testing"
	"time"

	"github.com/svetlyopet/mindpalace/internal/capture"
	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/index"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

// DefaultFixtureEntry is the standard test entry used across package tests.
func DefaultFixtureEntry() *vault.Entry {
	return &vault.Entry{
		ID:      "abc123",
		Title:   "Fixture",
		Created: time.Now().UTC(),
		Type:    vault.TypeNote,
		Tags:    []string{"alpha", "beta"},
		Body:    "hello world",
	}
}

// TempVault initializes a vault with default config in a temp directory.
func TempVault(t *testing.T) (*vault.Vault, *config.Config, string) {
	t.Helper()
	dir := t.TempDir()
	v, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteDefault(dir); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return v, cfg, dir
}

// TempIndex opens a Bleve index for dir and registers cleanup.
func TempIndex(t *testing.T, dir string) *index.Index {
	t.Helper()
	ix, err := index.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	return ix
}

// WriteEntry creates an entry in the vault and indexes it.
func WriteEntry(t *testing.T, v *vault.Vault, ix *index.Index, e *vault.Entry) {
	t.Helper()
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	if ix != nil {
		if err := ix.Put(e); err != nil {
			t.Fatal(err)
		}
	}
}

// VaultIndex holds an initialized vault and search index.
type VaultIndex struct {
	Vault  *vault.Vault
	Index  *index.Index
	Config *config.Config
	Dir    string
}

// TempVaultIndex returns a vault, config, and open index; optionally indexes DefaultFixtureEntry.
func TempVaultIndex(t *testing.T, withFixture bool) VaultIndex {
	t.Helper()
	v, cfg, dir := TempVault(t)
	ix := TempIndex(t, dir)
	if withFixture {
		WriteEntry(t, v, ix, DefaultFixtureEntry())
	}
	return VaultIndex{Vault: v, Index: ix, Config: cfg, Dir: dir}
}

// NewCapturer builds a capturer for test vault/config pairs.
func NewCapturer(v *vault.Vault, cfg *config.Config) *capture.Capturer {
	return capture.New(v, capture.KeywordTagger{}, cfg.Capture)
}

// ServeToken ensures a serve API token exists for the vault dir.
func ServeToken(t *testing.T, dir string, cfg *config.Config) string {
	t.Helper()
	token, err := config.EnsureServeToken(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
