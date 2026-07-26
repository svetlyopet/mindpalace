package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

func TestEnsureServeToken(t *testing.T) {
	dir := t.TempDir()
	cfg := Default()
	tok, err := EnsureServeToken(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" {
		t.Fatal("expected token")
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Serve.Token != tok {
		t.Fatalf("token not persisted: got %q", loaded.Serve.Token)
	}
}

func TestValidateServeAddr(t *testing.T) {
	if err := ValidateServeAddr("127.0.0.1:7451", false); err != nil {
		t.Fatal(err)
	}
	if err := ValidateServeAddr("0.0.0.0:7451", false); err == nil {
		t.Fatal("expected error for wildcard bind")
	}
	if err := ValidateServeAddr("0.0.0.0:7451", true); err != nil {
		t.Fatal(err)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := Default()
	cfg.Editor = "vim"
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	path := vault.ConfigPath(dir)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Editor != "vim" {
		t.Fatalf("editor = %q", loaded.Editor)
	}
}

func TestEditorCommand(t *testing.T) {
	t.Setenv("EDITOR", "")
	cfg := Default()
	ed, err := cfg.EditorCommand()
	if err != nil {
		if _, lookErr := exec.LookPath("vim"); lookErr != nil {
			t.Skip("vim not on PATH")
		}
		t.Fatal(err)
	}
	if ed != "vim" {
		t.Fatalf("EditorCommand() = %q, want vim", ed)
	}

	cfg.Editor = "vim"
	ed, err = cfg.EditorCommand()
	if err != nil {
		t.Fatal(err)
	}
	if ed != "vim" {
		t.Fatalf("explicit editor = %q", ed)
	}
}

func TestResolveVaultRoot_default(t *testing.T) {
	t.Setenv(envVault, "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, defaultVaultDir)
	got, err := ResolveVaultRoot("")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ResolveVaultRoot(\"\") = %q, want %q", got, want)
	}
}

func TestClearVaultEncryption(t *testing.T) {
	cfg := Default()
	uc, err := vault.NewKDFParamsSalt()
	if err != nil {
		t.Fatal(err)
	}
	SetVaultEncryption(cfg, vault.UnlockConfig{
		Encrypted:  true,
		KDF:        uc,
		WrappedKey: []byte("wrapped"),
	}, "wrapped-b64")
	ClearVaultEncryption(cfg)
	if cfg.Vault.Encrypted || cfg.Vault.WrappedKey != "" || cfg.Vault.KDF.Salt != "" {
		t.Fatalf("vault config not cleared: %+v", cfg.Vault)
	}
	dir := t.TempDir()
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Vault.Encrypted {
		t.Fatal("expected encrypted false after load")
	}
}
