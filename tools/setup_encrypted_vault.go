//go:build ignore

package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: setup-encrypted-vault <dir>")
		os.Exit(2)
	}
	root := os.Args[1]
	if _, err := vault.Init(root); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("editor: \"\"\n"), 0o644); err != nil {
		panic(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		panic(err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		panic(err)
	}
	e := &vault.Entry{
		ID:      "lktest",
		Title:   "Lock test",
		Created: time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC),
		Type:    vault.TypeNote,
		Body:    "secret body",
	}
	if err := v.Create(e); err != nil {
		panic(err)
	}
	uc, err := vault.EnableEncryption(v, "secret")
	if err != nil {
		panic(err)
	}
	config.SetVaultEncryption(cfg, uc, base64.StdEncoding.EncodeToString(uc.WrappedKey))
	if err := config.Save(root, cfg); err != nil {
		panic(err)
	}
	fmt.Println(root)
}
