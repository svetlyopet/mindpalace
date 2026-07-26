package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const encMagic = "MPENC1"

var (
	ErrWrongPassword = errors.New("wrong password")
	ErrLocked        = errors.New("vault is locked")
)

// UnlockConfig is the vault encryption section from config.yaml.
type UnlockConfig struct {
	Encrypted  bool
	KDF        KDFParams
	WrappedKey []byte
}

// KDFParams holds Argon2id parameters stored in config.
type KDFParams struct {
	Salt        []byte
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	KeyLen      uint32
}

func DefaultKDFParams() KDFParams {
	return KDFParams{
		Salt:        nil,
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		KeyLen:      32,
	}
}

func NewKDFParamsSalt() (KDFParams, error) {
	p := DefaultKDFParams()
	p.Salt = make([]byte, 16)
	if _, err := rand.Read(p.Salt); err != nil {
		return KDFParams{}, err
	}
	return p, nil
}

func DeriveKEK(password string, p KDFParams) []byte {
	return argon2.IDKey([]byte(password), p.Salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLen)
}

func NewDEK() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func WrapDEK(kek, dek []byte) ([]byte, error) {
	return encryptBytes(kek, dek)
}

func UnwrapDEK(kek, wrapped []byte) ([]byte, error) {
	out, err := decryptBytes(kek, wrapped)
	if err != nil {
		return nil, ErrWrongPassword
	}
	if len(out) != 32 {
		return nil, ErrWrongPassword
	}
	return out, nil
}

// Cipher holds the data encryption key for an unlocked vault.
type Cipher struct {
	dek []byte
}

func NewCipher(dek []byte) (*Cipher, error) {
	if len(dek) != 32 {
		return nil, fmt.Errorf("DEK must be 32 bytes")
	}
	dup := make([]byte, 32)
	copy(dup, dek)
	return &Cipher{dek: dup}, nil
}

func (c *Cipher) Encrypt(plain []byte) ([]byte, error) {
	return encryptBytes(c.dek, plain)
}

func (c *Cipher) Decrypt(blob []byte) ([]byte, error) {
	return decryptBytes(c.dek, blob)
}

func encryptBytes(key, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := gcm.Seal(nil, nonce, plain, nil)
	var buf []byte
	buf = append(buf, encMagic...)
	buf = append(buf, nonce...)
	buf = append(buf, out...)
	return buf, nil
}

func decryptBytes(key, blob []byte) ([]byte, error) {
	if len(blob) < len(encMagic)+12 {
		return nil, errors.New("ciphertext too short")
	}
	if string(blob[:len(encMagic)]) != encMagic {
		return nil, errors.New("invalid ciphertext magic")
	}
	blob = blob[len(encMagic):]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("invalid nonce")
	}
	nonce, ct := blob[:ns], blob[ns:]
	return gcm.Open(nil, nonce, ct, nil)
}

func EncodeBlob(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func DecodeBlob(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// VaultFingerprint is a stable id for session cookies.
func VaultFingerprint(root string) string {
	sum := sha256.Sum256([]byte(root))
	return base64.RawURLEncoding.EncodeToString(sum[:8])
}
