package commands

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/cli/input"
	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func NewVault(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Create a vault and manage encryption, passwords, and session lock",
		Long:  "If the vault directory does not exist or has no config yet, run mp vault init [path] to create one.",
	}
	clictx.MarkSkipVaultOpen(cmd)
	return cmd
}

func NewVaultEncrypt(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encrypt",
		Short: "Encrypt vault contents at rest",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Open(); err != nil {
				return err
			}
			defer rt.Close()
			if rt.Config.Vault.Encrypted {
				return fmt.Errorf("vault is already encrypted")
			}
			pw, err := input.ReadPasswordConfirm("New vault password: ")
			if err != nil {
				return err
			}
			uc, err := vault.EnableEncryption(rt.Vault, pw)
			if err != nil {
				return err
			}
			config.SetVaultEncryption(rt.Config, uc, base64.StdEncoding.EncodeToString(uc.WrappedKey))
			root := rt.VaultRoot()
			if err := config.Save(root, rt.Config); err != nil {
				return err
			}
			if err := rt.Vault.PersistUnlockSession(); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Vault encrypted. Run mp reindex to rebuild search index.")
			return nil
		},
	}
	clictx.MarkSkipVaultOpen(cmd)
	return cmd
}

func NewVaultDecrypt(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decrypt",
		Short: "Decrypt vault contents at rest and disable encryption",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Open(); err != nil {
				return err
			}
			defer rt.Close()
			if !rt.Config.Vault.Encrypted {
				return fmt.Errorf("vault is not encrypted")
			}
			pw, err := input.ReadPassword("Vault password: ")
			if err != nil {
				return err
			}
			if err := vault.DisableEncryption(rt.Vault, pw); err != nil {
				return err
			}
			config.ClearVaultEncryption(rt.Config)
			root := rt.VaultRoot()
			if err := config.Save(root, rt.Config); err != nil {
				return err
			}
			if err := rt.Vault.ClearUnlockSession(); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Vault decrypted. Run mp reindex to rebuild search index.")
			return nil
		},
	}
	clictx.MarkSkipVaultOpen(cmd)
	return cmd
}

func NewVaultPassword(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "password",
		Short: "Set or change the vault encryption password",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Open(); err != nil {
				return err
			}
			defer rt.Close()
			if !rt.Config.Vault.Encrypted {
				return fmt.Errorf("vault is not encrypted")
			}
			oldPW, err := input.ReadPassword("Current password: ")
			if err != nil {
				return err
			}
			newPW, err := input.ReadPasswordConfirm("New vault password: ")
			if err != nil {
				return err
			}
			uc, err := vault.ChangePassword(rt.Vault, oldPW, newPW)
			if err != nil {
				return err
			}
			config.SetVaultEncryption(rt.Config, uc, base64.StdEncoding.EncodeToString(uc.WrappedKey))
			return config.Save(rt.VaultRoot(), rt.Config)
		},
	}
	clictx.MarkSkipVaultOpen(cmd)
	return cmd
}
