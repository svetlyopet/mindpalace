package vault

import (
	"testing"
	"time"
)

func TestPrepareUnlockRespectsCLILockOverEnv(t *testing.T) {
	t.Setenv("MINDPALACE_PASSWORD", "secret")
	v := testVault(t)
	e := testEntry(t, v, "lockenv", "Note")
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	if _, err := EnableEncryption(v, "secret"); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	if err := v.setCLILocked(); err != nil {
		t.Fatal(err)
	}
	if err := v.PrepareUnlock(); err != ErrLocked {
		t.Fatalf("PrepareUnlock = %v, want ErrLocked", err)
	}
	if err := v.Unlock("secret"); err != nil {
		t.Fatal(err)
	}
	if err := v.PrepareUnlock(); err != nil {
		t.Fatalf("after unlock PrepareUnlock = %v", err)
	}
}

func testEntry(t *testing.T, v *Vault, id, title string) *Entry {
	t.Helper()
	return &Entry{
		ID:      id,
		Title:   title,
		Created: eCreated,
		Type:    TypeNote,
		Body:    "body",
	}
}

var eCreated = mustParseTime("2026-07-26T12:00:00Z")

func mustParseTime(s string) (tm time.Time) {
	tm, _ = time.Parse(time.RFC3339, s)
	return tm
}
