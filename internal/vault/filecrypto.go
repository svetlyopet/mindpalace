package vault

import (
	"os"
	"path/filepath"

	"github.com/svetlyopet/mindpalace/internal/fsperm"
)

// ReadFileBytes reads a file, decrypting when the vault cipher is set and data is encrypted.
func ReadFileBytes(path string, c *Cipher) ([]byte, error) {
	data, err := readFileRaw(path)
	if err != nil {
		return nil, err
	}
	if len(data) >= len(encMagic) && string(data[:len(encMagic)]) == encMagic {
		if c == nil {
			return nil, ErrLocked
		}
		return c.Decrypt(data)
	}
	return data, nil
}

// WriteFileBytes writes data, encrypting when cipher is non-nil.
func WriteFileBytes(path string, plain []byte, c *Cipher) error {
	out := plain
	if c != nil {
		var err error
		out, err = c.Encrypt(plain)
		if err != nil {
			return err
		}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, fsperm.DirMode); err != nil {
		return err
	}
	return writeFileRaw(path, out)
}

// readFileRaw reads bytes at path. Callers must scope path to the vault or entry tree.
func readFileRaw(path string) ([]byte, error) {
	return os.ReadFile(path) // #nosec G304 -- vault-scoped paths enforced at call sites
}

func writeFileRaw(path string, data []byte) error {
	return os.WriteFile(path, data, fsperm.PrivateFileMode) // #nosec G304 -- vault-scoped paths enforced at call sites
}

// EncryptTree encrypts entry bodies and known asset files under the vault.
func (v *Vault) EncryptTree() error {
	if v.cipher == nil {
		return ErrLocked
	}
	return v.Walk(func(e *Entry) error {
		if err := WriteEntry(e.Dir, e, v.cipher); err != nil {
			return err
		}
		return encryptEntryAssets(e.Dir, v.cipher)
	})
}

func encryptEntryAssets(dir string, c *Cipher) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() {
			if ent.Name() == "assets" {
				sub, _ := filepath.Glob(filepath.Join(dir, "assets", "*"))
				for _, p := range sub {
					if err := encryptFileInPlace(p, c); err != nil {
						return err
					}
				}
			}
			continue
		}
		name := ent.Name()
		if name == "entry.md" {
			continue
		}
		if err := encryptFileInPlace(filepath.Join(dir, name), c); err != nil {
			return err
		}
	}
	return nil
}

func encryptFileInPlace(path string, c *Cipher) error {
	data, err := readFileRaw(path)
	if err != nil {
		return err
	}
	if len(data) >= len(encMagic) && string(data[:len(encMagic)]) == encMagic {
		return nil
	}
	return WriteFileBytes(path, data, c)
}

// DecryptTree decrypts entry bodies and known asset files under the vault.
func (v *Vault) DecryptTree() error {
	if v.cipher == nil {
		return ErrLocked
	}
	c := v.cipher
	return v.Walk(func(e *Entry) error {
		if err := WriteEntry(e.Dir, e, nil); err != nil {
			return err
		}
		return decryptEntryAssets(e.Dir, c)
	})
}

func decryptEntryAssets(dir string, c *Cipher) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() {
			if ent.Name() == "assets" {
				sub, _ := filepath.Glob(filepath.Join(dir, "assets", "*"))
				for _, p := range sub {
					if err := decryptFileInPlace(p, c); err != nil {
						return err
					}
				}
			}
			continue
		}
		name := ent.Name()
		if name == "entry.md" {
			continue
		}
		if err := decryptFileInPlace(filepath.Join(dir, name), c); err != nil {
			return err
		}
	}
	return nil
}

func decryptFileInPlace(path string, c *Cipher) error {
	data, err := readFileRaw(path)
	if err != nil {
		return err
	}
	if len(data) < len(encMagic) || string(data[:len(encMagic)]) != encMagic {
		return nil
	}
	plain, err := c.Decrypt(data)
	if err != nil {
		return err
	}
	return writeFileRaw(path, plain)
}
