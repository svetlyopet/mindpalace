package cli

import (
	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/apiclient"
	"github.com/svetlyopet/mindpalace/internal/cli/commands"
	"github.com/svetlyopet/mindpalace/internal/cli/input"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/version"
)

// UsageError marks cobra usage failures for exit code 2.
type UsageError = clictx.UsageError

var active = &clictx.Runtime{App: &clictx.App{}}

func Execute() error {
	return NewRoot().Execute()
}

func NewRoot() *cobra.Command {
	var vaultFlag string
	var jsonOut bool

	root := &cobra.Command{
		Use:           "mp",
		Short:         "Mindpalace — local-first knowledge base",
		Version:       version.Info(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			active.Close()
			active.App = &clictx.App{
				VaultFlag:      vaultFlag,
				JSON:           jsonOut,
				PasswordPrompt: input.ReadPassword,
			}
			if clictx.SkipVaultOpen(cmd) {
				return nil
			}
			if clictx.RefuseWhenServe(cmd) && active.ProbeServe() {
				if cmd.Name() == "reindex" {
					return apiclient.RefuseReindex()
				}
				return apiclient.RefuseEncryptionChange()
			}
			if clictx.VaultOnly(cmd) && active.ProbeServe() {
				if err := active.OpenVaultOnly(); err != nil {
					return err
				}
				return clictx.EnsureVaultUnlocked(active.Vault, active.Config.Vault.Encrypted)
			}
			if err := active.Open(); err != nil {
				return err
			}
			if active.Remote() {
				return nil
			}
			return clictx.EnsureVaultUnlocked(active.Vault, active.Config.Vault.Encrypted)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			active.Close()
		},
	}

	root.PersistentFlags().StringVar(&vaultFlag, "vault", "", "vault directory (default ~/.mindpalace or $MINDPALACE_VAULT)")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON output (read commands)")

	commands.Register(root, active)

	return root
}

// Active returns the runtime for the command being executed.
func Active() *clictx.Runtime {
	return active
}
