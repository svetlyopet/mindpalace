package testenv

import (
	"encoding/base64"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

// TempEncryptedVault encrypts a new temp vault without opening the search index.
func TempEncryptedVault(t *testing.T, password string) (*vault.Vault, *config.Config, string) {
	t.Helper()
	v, cfg, dir := TempVault(t)
	uc, err := vault.EnableEncryption(v, password)
	if err != nil {
		t.Fatal(err)
	}
	config.SetVaultEncryption(cfg, uc, base64.StdEncoding.EncodeToString(uc.WrappedKey))
	if err := config.Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	return v, cfg, dir
}
