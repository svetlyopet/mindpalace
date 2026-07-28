package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/cli/input"
	"github.com/svetlyopet/mindpalace/internal/clictx"
)

func NewVaultUnlock(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock an encrypted vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Open(); err != nil {
				return err
			}
			defer rt.Close()
			if !rt.Config.Vault.Encrypted {
				fmt.Println("Vault is not encrypted.")
				return nil
			}
			if rt.Vault.Unlocked() {
				fmt.Println("Vault is already unlocked.")
				return nil
			}
			pw, err := input.ReadPassword("Vault password: ")
			if err != nil {
				return err
			}
			if err := rt.Vault.Unlock(pw); err != nil {
				return err
			}
			if err := rt.Vault.PersistUnlockSession(); err != nil {
				return err
			}
			fmt.Println("Vault unlocked.")
			return nil
		},
	}
	clictx.MarkSkipVaultOpen(cmd)
	return cmd
}

func NewVaultLock(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Lock the vault and clear the unlock session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Open(); err != nil {
				return err
			}
			defer rt.Close()
			if !rt.Config.Vault.Encrypted {
				fmt.Fprintln(os.Stderr, "Vault is not encrypted; lock does not restrict entry access.")
				return nil
			}
			if err := rt.Vault.ClearUnlockSession(); err != nil {
				return err
			}
			fmt.Println("Vault locked.")
			return nil
		},
	}
	clictx.MarkSkipVaultOpen(cmd)
	return cmd
}
