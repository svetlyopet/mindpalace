package clictx

import (
	"context"
	"fmt"

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

	Root      string
	Config    *config.Config
	Vault     *vault.Vault
	Index     *index.Index
	Searcher  *search.Searcher
	Capturer  *capture.Capturer
	Lib       *library.Library
}

func (a *App) Open() error {
	root, err := config.ResolveVaultRoot(a.VaultFlag)
	if err != nil {
		return err
	}
	a.Root = root
	v, err := vault.Open(root)
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	ix, err := index.Open(root)
	if err != nil {
		return err
	}
	a.Vault = v
	a.Config = cfg
	if cfg.Vault.Encrypted {
		uc, err := cfg.Vault.UnlockConfig()
		if err != nil {
			_ = ix.Close()
			return err
		}
		vault.ApplyEncryptionConfig(v, true, uc)
	}
	_ = v.PrepareUnlock()
	if v.Unlocked() {
		if err := ix.Refresh(context.Background(), v); err != nil {
			_ = ix.Close()
			return err
		}
	}
	a.Index = ix
	a.Searcher = search.New(ix)
	a.Capturer = capture.New(v, capture.KeywordTagger{}, cfg.Capture)
	a.Lib = library.New(v, ix, a.Searcher, a.Capturer)
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
