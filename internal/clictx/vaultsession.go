package clictx

import (
	"fmt"

	"github.com/svetlyopet/mindpalace/internal/vault"
)

func EnsureVaultUnlocked(v *vault.Vault, encrypted bool) error {
	if !encrypted {
		return nil
	}
	if v.Unlocked() {
		return nil
	}
	if err := v.PrepareUnlock(); err != nil {
		return fmt.Errorf("vault is locked (run mp vault unlock or set MINDPALACE_PASSWORD)")
	}
	return nil
}
