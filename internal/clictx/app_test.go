package clictx

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/testenv"
)

func TestSkipVaultOpen(t *testing.T) {
	root := &cobra.Command{Use: "mp"}
	child := &cobra.Command{Use: "init"}
	MarkSkipVaultOpen(child)
	root.AddCommand(child)
	if !SkipVaultOpen(child) {
		t.Fatal("expected skip on init")
	}
	if SkipVaultOpen(root) {
		t.Fatal("root should not skip")
	}
}

func TestAppOpenPlainVault(t *testing.T) {
	_, _, dir := testenv.TempVault(t)
	app := &App{VaultFlag: dir}
	if err := app.Open(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if app.Lib == nil || app.Vault == nil {
		t.Fatal("expected wired app")
	}
}

func TestAppOpenEncryptedLocked(t *testing.T) {
	v, _, dir := testenv.TempEncryptedVault(t, "secret")
	v.Lock()
	app := &App{VaultFlag: dir}
	if err := app.Open(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if app.Vault.Unlocked() {
		t.Fatal("expected locked vault")
	}
	err := EnsureVaultUnlocked(app.Vault, true)
	if err == nil {
		t.Fatal("expected locked error")
	}
}

func TestEnsureVaultUnlockedPlain(t *testing.T) {
	v, _, _ := testenv.TempVault(t)
	if err := EnsureVaultUnlocked(v, false); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureVaultUnlockedWhenUnlocked(t *testing.T) {
	vi := testenv.TempEncryptedVaultIndex(t, "secret", false)
	if !vi.Vault.Unlocked() {
		t.Fatal("expected unlocked after encrypt")
	}
	if err := EnsureVaultUnlocked(vi.Vault, true); err != nil {
		t.Fatal(err)
	}
}

func TestRequireVault(t *testing.T) {
	app := &App{}
	if _, _, err := app.RequireVault(); err == nil {
		t.Fatal("expected error")
	}
	vi := testenv.TempVaultIndex(t, false)
	app.Vault = vi.Vault
	app.Index = vi.Index
	if _, _, err := app.RequireVault(); err != nil {
		t.Fatal(err)
	}
}

func TestAppOpenNotFoundVault(t *testing.T) {
	app := &App{VaultFlag: t.TempDir()}
	err := app.Open()
	if err == nil {
		app.Close()
		t.Fatal("expected open error on empty dir")
	}
}

func TestEnsureVaultUnlockedEncryptedUnlocked(t *testing.T) {
	vi := testenv.TempEncryptedVaultIndex(t, "pw", false)
	if err := EnsureVaultUnlocked(vi.Vault, true); err != nil {
		t.Fatal(err)
	}
}
