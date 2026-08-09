package clictx

import "github.com/spf13/cobra"

const (
	AnnSkipVaultOpen   = "mindpalace.skip_vault_open"
	AnnVaultOnly       = "mindpalace.vault_only"
	AnnRefuseWhenServe = "mindpalace.refuse_when_serve"
)

func MarkSkipVaultOpen(cmd *cobra.Command) {
	setAnn(cmd, AnnSkipVaultOpen)
}

func SkipVaultOpen(cmd *cobra.Command) bool {
	return hasAnn(cmd, AnnSkipVaultOpen)
}

func MarkVaultOnly(cmd *cobra.Command) {
	setAnn(cmd, AnnVaultOnly)
}

func VaultOnly(cmd *cobra.Command) bool {
	return hasAnn(cmd, AnnVaultOnly)
}

func MarkRefuseWhenServe(cmd *cobra.Command) {
	setAnn(cmd, AnnRefuseWhenServe)
}

func RefuseWhenServe(cmd *cobra.Command) bool {
	return hasAnn(cmd, AnnRefuseWhenServe)
}

func setAnn(cmd *cobra.Command, key string) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[key] = "1"
}

func hasAnn(cmd *cobra.Command, key string) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations != nil && c.Annotations[key] == "1" {
			return true
		}
	}
	return false
}
