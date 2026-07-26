package vault

import "fmt"

// EnableEncryption generates a DEK, wraps it with password, and encrypts all entries.
func EnableEncryption(v *Vault, password string) (UnlockConfig, error) {
	if v.Encrypted() {
		return UnlockConfig{}, fmt.Errorf("vault is already encrypted")
	}
	dek, err := NewDEK()
	if err != nil {
		return UnlockConfig{}, err
	}
	kdf, err := NewKDFParamsSalt()
	if err != nil {
		return UnlockConfig{}, err
	}
	kek := DeriveKEK(password, kdf)
	wrapped, err := WrapDEK(kek, dek)
	if err != nil {
		return UnlockConfig{}, err
	}
	c, err := NewCipher(dek)
	if err != nil {
		return UnlockConfig{}, err
	}
	v.cipher = c
	if err := v.EncryptTree(); err != nil {
		return UnlockConfig{}, err
	}
	uc := UnlockConfig{
		Encrypted:  true,
		KDF:        kdf,
		WrappedKey: wrapped,
	}
	v.SetEncryption(uc)
	return uc, nil
}

// ChangePassword re-wraps the DEK with a new password (vault must be unlocked).
func ChangePassword(v *Vault, oldPW, newPW string) (UnlockConfig, error) {
	if !v.Encrypted() {
		return UnlockConfig{}, fmt.Errorf("vault is not encrypted")
	}
	if err := v.Unlock(oldPW); err != nil {
		return UnlockConfig{}, err
	}
	dek := make([]byte, 32)
	copy(dek, v.cipher.dek)
	kdf, err := NewKDFParamsSalt()
	if err != nil {
		return UnlockConfig{}, err
	}
	kek := DeriveKEK(newPW, kdf)
	wrapped, err := WrapDEK(kek, dek)
	if err != nil {
		return UnlockConfig{}, err
	}
	uc := UnlockConfig{
		Encrypted:  true,
		KDF:        kdf,
		WrappedKey: wrapped,
	}
	v.SetEncryption(uc)
	return uc, nil
}

// DisableEncryption unlocks with password, decrypts all entries at rest, and clears encryption state.
func DisableEncryption(v *Vault, password string) error {
	if !v.Encrypted() {
		return fmt.Errorf("vault is not encrypted")
	}
	if err := v.Unlock(password); err != nil {
		return err
	}
	if err := v.DecryptTree(); err != nil {
		return err
	}
	v.cipher = nil
	v.SetEncryption(UnlockConfig{})
	return nil
}

// ApplyEncryptionConfig loads encryption settings from stored config fields.
func ApplyEncryptionConfig(v *Vault, encrypted bool, uc UnlockConfig) {
	if !encrypted {
		v.SetEncryption(UnlockConfig{})
		return
	}
	v.SetEncryption(uc)
}
