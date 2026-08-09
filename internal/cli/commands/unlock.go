package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/apiclient"
	"github.com/svetlyopet/mindpalace/internal/cli/input"
	"github.com/svetlyopet/mindpalace/internal/clictx"
)

func NewVaultUnlock(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock an encrypted vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			if rt.ProbeServe() {
				client := apiclient.New(rt.Config.Serve.Addr, rt.Config.Serve.Token)
				if !rt.Config.Vault.Encrypted {
					fmt.Println("Vault is not encrypted.")
					return nil
				}
				status := apiclient.ProbeSession(rt.Config.Serve.Addr)
				if status == apiclient.ProbeReady {
					fmt.Println("Vault is already unlocked.")
					return nil
				}
				pw, err := input.ReadPassword("Vault password: ")
				if err != nil {
					return err
				}
				if err := client.Unlock(context.Background(), pw); err != nil {
					return err
				}
				// Keep local session in sync for when serve stops.
				if err := rt.OpenVaultOnly(); err == nil {
					_ = rt.Vault.Unlock(pw)
					_ = rt.Vault.PersistUnlockSession()
					rt.Close()
				}
				fmt.Println("Vault unlocked.")
				return nil
			}
			if err := rt.OpenLocal(); err != nil {
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
			if rt.ProbeServe() {
				client := apiclient.New(rt.Config.Serve.Addr, rt.Config.Serve.Token)
				if !rt.Config.Vault.Encrypted {
					fmt.Fprintln(os.Stderr, "Vault is not encrypted; lock does not restrict entry access.")
					return nil
				}
				switch apiclient.ProbeSession(rt.Config.Serve.Addr) {
				case apiclient.ProbeLocked:
					// Already locked on the server; still clear the local session marker.
				case apiclient.ProbeReady:
					if err := client.Lock(context.Background()); err != nil {
						return err
					}
				default:
					return fmt.Errorf("mindpalace server at http://%s: unexpected status", rt.Config.Serve.Addr)
				}
				if err := rt.OpenVaultOnly(); err == nil {
					_ = rt.Vault.ClearUnlockSession()
					rt.Close()
				}
				fmt.Println("Vault locked.")
				return nil
			}
			if err := rt.OpenLocal(); err != nil {
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
