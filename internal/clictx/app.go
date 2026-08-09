package clictx

import (
	"context"
	"fmt"
	"os"

	"github.com/svetlyopet/mindpalace/internal/apiclient"
	"github.com/svetlyopet/mindpalace/internal/capture"
	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/index"
	"github.com/svetlyopet/mindpalace/internal/library"
	"github.com/svetlyopet/mindpalace/internal/search"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

// App holds wired dependencies for CLI commands.
type App struct {
	VaultFlag string
	JSON      bool

	Root     string
	Config   *config.Config
	Vault    *vault.Vault
	Index    *index.Index
	Searcher *search.Searcher
	Capturer *capture.Capturer
	Lib      *library.Library
	API      *apiclient.Client

	// PasswordPrompt reads a vault password interactively. Optional; falls back to MINDPALACE_PASSWORD only.
	PasswordPrompt func(prompt string) (string, error)
}

func (a *App) Remote() bool {
	return a != nil && a.API != nil
}

// Open resolves the vault and either attaches to a running mp serve API or opens Bleve locally.
func (a *App) Open() error {
	return a.open(openOpts{})
}

// OpenLocal always opens the vault and Bleve index (used by mp serve).
func (a *App) OpenLocal() error {
	return a.open(openOpts{forceLocal: true})
}

// OpenVaultOnly opens vault + config without Bleve (edit/open while serve is up).
func (a *App) OpenVaultOnly() error {
	return a.open(openOpts{vaultOnly: true})
}

type openOpts struct {
	forceLocal bool
	vaultOnly  bool
}

func (a *App) open(opts openOpts) error {
	root, err := config.ResolveVaultRoot(a.VaultFlag)
	if err != nil {
		return err
	}
	a.Root = root
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	a.Config = cfg

	if !opts.forceLocal {
		if client, status, ok := a.tryRemote(cfg); ok {
			a.API = client
			if opts.vaultOnly {
				return a.openLocal(true)
			}
			return a.ensureRemoteUnlocked(status)
		}
	}

	return a.openLocal(opts.vaultOnly)
}

func (a *App) tryRemote(cfg *config.Config) (*apiclient.Client, apiclient.ProbeStatus, bool) {
	if cfg.Serve.Addr == "" || cfg.Serve.Token == "" {
		return nil, apiclient.ProbeDown, false
	}
	status := apiclient.ProbeSession(cfg.Serve.Addr)
	switch status {
	case apiclient.ProbeReady, apiclient.ProbeLocked:
		return apiclient.New(cfg.Serve.Addr, cfg.Serve.Token), status, true
	default:
		return nil, status, false
	}
}

// ProbeServe reports whether mp serve appears to be running for this vault.
func (a *App) ProbeServe() bool {
	root, err := config.ResolveVaultRoot(a.VaultFlag)
	if err != nil {
		return false
	}
	cfg, err := config.Load(root)
	if err != nil {
		return false
	}
	a.Root = root
	a.Config = cfg
	if cfg.Serve.Addr == "" || cfg.Serve.Token == "" {
		return false
	}
	switch apiclient.ProbeSession(cfg.Serve.Addr) {
	case apiclient.ProbeReady, apiclient.ProbeLocked:
		return true
	default:
		return false
	}
}

func (a *App) openLocal(vaultOnly bool) error {
	v, err := vault.Open(a.Root)
	if err != nil {
		return err
	}
	if a.Config.Vault.Encrypted {
		uc, err := a.Config.Vault.UnlockConfig()
		if err != nil {
			return err
		}
		vault.ApplyEncryptionConfig(v, true, uc)
	}
	_ = v.PrepareUnlock()
	a.Vault = v

	if vaultOnly {
		return nil
	}

	ix, err := index.Open(a.Root)
	if err != nil {
		return err
	}
	if v.Unlocked() {
		if err := ix.Refresh(context.Background(), v); err != nil {
			_ = ix.Close()
			return err
		}
	}
	a.Index = ix
	a.Searcher = search.New(ix)
	a.Capturer = capture.New(v, capture.KeywordTagger{}, a.Config.Capture)
	a.Lib = library.New(v, ix, a.Searcher, a.Capturer)
	return nil
}

func (a *App) ensureRemoteUnlocked(status apiclient.ProbeStatus) error {
	if status != apiclient.ProbeLocked {
		return nil
	}
	if a.Config != nil && !a.Config.Vault.Encrypted {
		return nil
	}
	pw := os.Getenv("MINDPALACE_PASSWORD")
	if pw == "" && a.PasswordPrompt != nil {
		var err error
		pw, err = a.PasswordPrompt("Vault password: ")
		if err != nil {
			return err
		}
	}
	if pw == "" {
		return fmt.Errorf("vault is locked (run mp vault unlock or set MINDPALACE_PASSWORD)")
	}
	if err := a.API.Unlock(context.Background(), pw); err != nil {
		return err
	}
	return nil
}

func (a *App) Close() {
	if a == nil {
		return
	}
	if a.Index != nil {
		_ = a.Index.Close()
		a.Index = nil
	}
	a.API = nil
}

func (a *App) IndexEntry(e *vault.Entry) error {
	if a.Lib == nil {
		return nil
	}
	return a.Lib.IndexEntry(e)
}

func (a *App) VaultRoot() string {
	if a.Root != "" {
		return a.Root
	}
	root, err := config.ResolveVaultRoot(a.VaultFlag)
	if err != nil {
		return ""
	}
	return root
}

func (a *App) RequireVault() (*vault.Vault, *index.Index, error) {
	if a.Vault == nil || a.Index == nil {
		return nil, nil, fmt.Errorf("vault not open")
	}
	return a.Vault, a.Index, nil
}

// Runtime is the active command context passed into command constructors.
type Runtime struct {
	*App
}
