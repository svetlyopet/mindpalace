package clictx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/config"
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

func TestAppOpenRemoteWhenServeUp(t *testing.T) {
	_, cfg, dir := testenv.TempVault(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/session" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	cfg.Serve.Addr = strings.TrimPrefix(srv.URL, "http://")
	cfg.Serve.Token = "test-token"
	if err := config.Save(dir, cfg); err != nil {
		t.Fatal(err)
	}

	app := &App{VaultFlag: dir}
	if err := app.Open(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if !app.Remote() {
		t.Fatal("expected remote mode when serve is up")
	}
	if app.Index != nil || app.Lib != nil {
		t.Fatal("remote mode must not open Bleve")
	}
}

func TestAppOpenLocalWhenServeDown(t *testing.T) {
	_, cfg, dir := testenv.TempVault(t)
	cfg.Serve.Addr = "127.0.0.1:1"
	cfg.Serve.Token = "test-token"
	if err := config.Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	app := &App{VaultFlag: dir}
	if err := app.Open(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if app.Remote() {
		t.Fatal("expected local mode when serve is down")
	}
	if app.Lib == nil {
		t.Fatal("expected local library")
	}
}

func TestAppOpenLocalIgnoresServe(t *testing.T) {
	_, cfg, dir := testenv.TempVault(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	cfg.Serve.Addr = strings.TrimPrefix(srv.URL, "http://")
	cfg.Serve.Token = "test-token"
	if err := config.Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	app := &App{VaultFlag: dir}
	if err := app.OpenLocal(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if app.Remote() {
		t.Fatal("OpenLocal must not attach API")
	}
	if app.Lib == nil {
		t.Fatal("expected local library")
	}
}

func TestAppOpenIgnoresNonMindpalaceListener(t *testing.T) {
	_, cfg, dir := testenv.TempVault(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "ok", http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	cfg.Serve.Addr = strings.TrimPrefix(srv.URL, "http://")
	cfg.Serve.Token = "test-token"
	if err := config.Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	app := &App{VaultFlag: dir}
	if err := app.Open(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if app.Remote() {
		t.Fatal("non-mp response should use local mode")
	}
}

func TestMarkVaultOnlyAndRefuseWhenServe(t *testing.T) {
	root := &cobra.Command{Use: "mp"}
	edit := &cobra.Command{Use: "edit"}
	reindex := &cobra.Command{Use: "reindex"}
	MarkVaultOnly(edit)
	MarkRefuseWhenServe(reindex)
	root.AddCommand(edit, reindex)
	if !VaultOnly(edit) {
		t.Fatal("VaultOnly")
	}
	if !RefuseWhenServe(reindex) {
		t.Fatal("RefuseWhenServe")
	}
}
