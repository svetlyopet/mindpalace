package config

import (
	"encoding/base64"
	"fmt"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

// VaultConfig holds optional encryption settings.
type VaultConfig struct {
	Encrypted  bool     `yaml:"encrypted"`
	KDF        VaultKDF `yaml:"kdf"`
	WrappedKey string   `yaml:"wrapped_key"`
}

type VaultKDF struct {
	Salt        string `yaml:"salt"`
	Memory      uint32 `yaml:"memory"`
	Iterations  uint32 `yaml:"iterations"`
	Parallelism uint8  `yaml:"parallelism"`
}

func (c *Config) validateVault() error {
	if !c.Vault.Encrypted {
		return nil
	}
	if c.Vault.WrappedKey == "" {
		return fmt.Errorf("vault.encrypted is true but vault.wrapped_key is missing")
	}
	if c.Vault.KDF.Salt == "" {
		return fmt.Errorf("vault.encrypted is true but vault.kdf.salt is missing")
	}
	return nil
}

// UnlockConfig converts to vault.UnlockConfig.
func (c VaultConfig) UnlockConfig() (vault.UnlockConfig, error) {
	if !c.Encrypted {
		return vault.UnlockConfig{}, nil
	}
	salt, err := base64.StdEncoding.DecodeString(c.KDF.Salt)
	if err != nil {
		return vault.UnlockConfig{}, fmt.Errorf("vault.kdf.salt: %w", err)
	}
	wrapped, err := base64.StdEncoding.DecodeString(c.WrappedKey)
	if err != nil {
		return vault.UnlockConfig{}, fmt.Errorf("vault.wrapped_key: %w", err)
	}
	mem := c.KDF.Memory
	if mem == 0 {
		mem = 64 * 1024
	}
	it := c.KDF.Iterations
	if it == 0 {
		it = 3
	}
	par := c.KDF.Parallelism
	if par == 0 {
		par = 4
	}
	return vault.UnlockConfig{
		Encrypted: true,
		KDF: vault.KDFParams{
			Salt:        salt,
			Memory:      mem,
			Iterations:  it,
			Parallelism: par,
			KeyLen:      32,
		},
		WrappedKey: wrapped,
	}, nil
}

// SetVaultEncryption writes encryption fields into config.
func SetVaultEncryption(cfg *Config, uc vault.UnlockConfig, wrappedB64 string) {
	cfg.Vault.Encrypted = true
	cfg.Vault.WrappedKey = wrappedB64
	cfg.Vault.KDF.Salt = base64.StdEncoding.EncodeToString(uc.KDF.Salt)
	cfg.Vault.KDF.Memory = uc.KDF.Memory
	cfg.Vault.KDF.Iterations = uc.KDF.Iterations
	cfg.Vault.KDF.Parallelism = uc.KDF.Parallelism
}

// ClearVaultEncryption removes encryption metadata from config.
func ClearVaultEncryption(cfg *Config) {
	cfg.Vault = VaultConfig{}
}
