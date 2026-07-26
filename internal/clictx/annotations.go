package clictx

import "github.com/spf13/cobra"

const AnnSkipVaultOpen = "mindpalace.skip_vault_open"

func MarkSkipVaultOpen(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[AnnSkipVaultOpen] = "1"
}

func SkipVaultOpen(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations != nil && c.Annotations[AnnSkipVaultOpen] == "1" {
			return true
		}
	}
	return false
}
