package vault

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	sessionDirName  = ".mp"
	sessionFileName = "session"
	cliLockFileName = "locked"
	sessionTTL      = 8 * time.Hour
)

func (v *Vault) Encrypted() bool {
	return v.encCfg != nil && v.encCfg.Encrypted
}

func (v *Vault) Unlocked() bool {
	return !v.Encrypted() || v.cipher != nil
}

func (v *Vault) Cipher() *Cipher {
	return v.cipher
}

func (v *Vault) SetEncryption(cfg UnlockConfig) {
	if !cfg.Encrypted {
		v.encCfg = nil
		return
	}
	c := cfg
	v.encCfg = &c
}

func (v *Vault) EncryptionConfig() *UnlockConfig {
	return v.encCfg
}

func (v *Vault) Unlock(password string) error {
	if v.encCfg == nil || !v.encCfg.Encrypted {
		v.cipher = nil
		return nil
	}
	kek := DeriveKEK(password, v.encCfg.KDF)
	dek, err := UnwrapDEK(kek, v.encCfg.WrappedKey)
	if err != nil {
		return err
	}
	c, err := NewCipher(dek)
	if err != nil {
		return err
	}
	v.cipher = c
	_ = v.clearCLILocked()
	return nil
}

func (v *Vault) Lock() {
	v.cipher = nil
}

func (v *Vault) TryUnlockFromEnv() error {
	if !v.Encrypted() {
		return nil
	}
	pw := os.Getenv("MINDPALACE_PASSWORD")
	if pw == "" {
		return ErrLocked
	}
	return v.Unlock(pw)
}

// RestoreFromSession loads the DEK from a local session file (mode 0600).
func (v *Vault) RestoreFromSession() error {
	if !v.Encrypted() {
		return nil
	}
	if v.cipher != nil {
		return nil
	}
	path := v.sessionPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return ErrLocked
	}
	lines := splitLines(string(data))
	if len(lines) < 3 {
		return ErrLocked
	}
	exp, err := strconv.ParseInt(lines[0], 10, 64)
	if err != nil {
		return ErrLocked
	}
	if time.Now().Unix() > exp {
		_ = os.Remove(path)
		return ErrLocked
	}
	mac, err := base64.StdEncoding.DecodeString(lines[1])
	if err != nil {
		return ErrLocked
	}
	if !hmac.Equal(mac, v.sessionMAC(exp)) {
		return ErrLocked
	}
	dek, err := DecodeBlob(lines[2])
	if err != nil || len(dek) != 32 {
		return ErrLocked
	}
	c, err := NewCipher(dek)
	if err != nil {
		return err
	}
	v.cipher = c
	return nil
}

func (v *Vault) PersistUnlockSession() error {
	if !v.Encrypted() || v.cipher == nil {
		return nil
	}
	if err := v.clearCLILocked(); err != nil {
		return err
	}
	dir := filepath.Join(v.root, sessionDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	exp := time.Now().Add(sessionTTL).Unix()
	mac := v.sessionMAC(exp)
	payload := fmt.Sprintf("%d\n%s\n%s\n",
		exp,
		base64.StdEncoding.EncodeToString(mac),
		EncodeBlob(v.cipher.dek),
	)
	return os.WriteFile(v.sessionPath(), []byte(payload), 0o600)
}

func (v *Vault) ClearUnlockSession() error {
	v.Lock()
	path := v.sessionPath()
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if v.Encrypted() {
		return v.setCLILocked()
	}
	return nil
}

func (v *Vault) cliLocked() bool {
	_, err := os.Stat(v.cliLockPath())
	return err == nil
}

func (v *Vault) setCLILocked() error {
	dir := filepath.Join(v.root, sessionDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(v.cliLockPath(), []byte("1\n"), 0o600)
}

func (v *Vault) clearCLILocked() error {
	err := os.Remove(v.cliLockPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (v *Vault) cliLockPath() string {
	return filepath.Join(v.root, sessionDirName, cliLockFileName)
}

func (v *Vault) sessionPath() string {
	return filepath.Join(v.root, sessionDirName, sessionFileName)
}

func (v *Vault) sessionMAC(exp int64) []byte {
	key := sha256.Sum256([]byte("mp-session:" + v.root))
	m := hmac.New(sha256.New, key[:])
	_, _ = m.Write([]byte(strconv.FormatInt(exp, 10)))
	return m.Sum(nil)
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// PrepareUnlock tries env, then session file, unless the CLI lock marker is set.
func (v *Vault) PrepareUnlock() error {
	if !v.Encrypted() {
		return nil
	}
	if v.cliLocked() {
		return ErrLocked
	}
	if err := v.TryUnlockFromEnv(); err == nil {
		return nil
	}
	return v.RestoreFromSession()
}
