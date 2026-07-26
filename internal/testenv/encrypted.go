package testenv

import (
	"encoding/base64"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

// TempEncryptedVaultIndex encrypts a temp vault and persists encryption config.
func TempEncryptedVaultIndex(t *testing.T, password string, withFixture bool) VaultIndex {
	t.Helper()
	vi := TempVaultIndex(t, withFixture)
	encryptVault(t, vi.Vault, vi.Config, vi.Dir, password)
	return vi
}

// EncryptVaultAt encrypts an existing vault directory (for E2E setup).
func EncryptVaultAt(t *testing.T, dir, password string) {
	t.Helper()
	v, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	encryptVault(t, v, cfg, dir, password)
}

func encryptVault(t *testing.T, v *vault.Vault, cfg *config.Config, dir, password string) {
	t.Helper()
	uc, err := vault.EnableEncryption(v, password)
	if err != nil {
		t.Fatal(err)
	}
	config.SetVaultEncryption(cfg, uc, base64.StdEncoding.EncodeToString(uc.WrappedKey))
	if err := config.Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
}
