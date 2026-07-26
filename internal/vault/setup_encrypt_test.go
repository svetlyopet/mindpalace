package vault

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnableDisableEncryptionRoundTrip(t *testing.T) {
	v := testVault(t)
	e := &Entry{
		ID:      "enc001",
		Title:   "Secret note",
		Created: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC),
		Type:    TypeNote,
		Body:    "plaintext body",
	}
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	assetPath := filepath.Join(e.Dir, "assets", "pic.bin")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, []byte("asset-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(e.Dir, "sidecar.txt")
	if err := os.WriteFile(sidecar, []byte("sidecar"), 0o644); err != nil {
		t.Fatal(err)
	}

	uc, err := EnableEncryption(v, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if !v.Encrypted() || uc.WrappedKey == nil {
		t.Fatal("expected encrypted vault")
	}
	entryMD, err := os.ReadFile(filepath.Join(e.Dir, "entry.md"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(entryMD)
	if strings.Contains(raw, "mp_body_enc") || strings.Contains(raw, "mp_enc") {
		t.Fatalf("entry.md must not contain mp_* frontmatter keys:\n%s", raw)
	}
	if !strings.Contains(raw, "title: Secret note") {
		t.Fatal("expected plaintext title in frontmatter")
	}
	idx := strings.Index(raw, "---\n")
	if idx < 0 {
		t.Fatal("missing frontmatter")
	}
	rest := entryMD[strings.LastIndex(raw, "---\n")+4:]
	if !isEncryptedBlob(rest) {
		t.Fatal("expected MPENC1 encrypted body below frontmatter")
	}

	if err := DisableEncryption(v, "secret"); err != nil {
		t.Fatal(err)
	}
	if v.Encrypted() {
		t.Fatal("expected vault encryption cleared")
	}
	got, err := ReadEntry(e.Dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got.Body) != "plaintext body" {
		t.Fatalf("body = %q, want plaintext", got.Body)
	}
	plainAsset, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(plainAsset) != "asset-bytes" {
		t.Fatalf("asset = %q", plainAsset)
	}
	plainSidecar, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatal(err)
	}
	if string(plainSidecar) != "sidecar" {
		t.Fatalf("sidecar = %q", plainSidecar)
	}
}

func TestDisableEncryptionWrongPassword(t *testing.T) {
	v := testVault(t)
	e := &Entry{
		ID:      "enc002",
		Title:   "Note",
		Created: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC),
		Type:    TypeNote,
		Body:    "body",
	}
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	if _, err := EnableEncryption(v, "secret"); err != nil {
		t.Fatal(err)
	}
	err := DisableEncryption(v, "wrong")
	if !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("DisableEncryption(wrong) = %v, want ErrWrongPassword", err)
	}
	if !v.Encrypted() {
		t.Fatal("vault should remain encrypted after wrong password")
	}
}

func TestReadEncryptedEntryBodyLocked(t *testing.T) {
	v := testVault(t)
	e := &Entry{
		ID:      "enc003",
		Title:   "Locked read",
		Created: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC),
		Type:    TypeNote,
		Body:    "secret",
	}
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	if _, err := EnableEncryption(v, "secret"); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	_, err := ReadEntry(e.Dir, nil)
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("ReadEntry locked = %v, want ErrLocked", err)
	}
}

func TestLegacyFrontmatterEncryptionReadAndRewrite(t *testing.T) {
	v := testVault(t)
	e := &Entry{
		ID:      "leg001",
		Title:   "Legacy",
		Created: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC),
		Type:    TypeNote,
		Body:    "legacy body",
	}
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	dek, err := NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewCipher(dek)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := c.Encrypt([]byte("legacy body"))
	if err != nil {
		t.Fatal(err)
	}
	legacyMD := "---\nid: leg001\ntitle: Legacy\ncreated: 2026-07-25T12:00:00Z\ntype: note\nmp_enc: true\nmp_body_enc: " + EncodeBlob(enc) + "\n---\n"
	if err := os.WriteFile(filepath.Join(e.Dir, "entry.md"), []byte(legacyMD), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadEntry(e.Dir, c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got.Body) != "legacy body" {
		t.Fatalf("body = %q", got.Body)
	}
	if err := WriteEntry(e.Dir, got, c); err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(filepath.Join(e.Dir, "entry.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(onDisk), "mp_body_enc") {
		t.Fatal("rewrite should remove legacy mp_body_enc")
	}
	after, err := ReadEntry(e.Dir, c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(after.Body) != "legacy body" {
		t.Fatalf("after rewrite body = %q", after.Body)
	}
}
