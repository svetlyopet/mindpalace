//go:build e2e

package e2e_test

import (
	"strings"
	"testing"

	"github.com/svetlyopet/mindpalace/internal/dto"
)

func TestCLIVaultLifecycle(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	mustRun(t, dir, "add", "note", "-m", "vault body", "--title", "Vault note", "--tags", "vault")

	// encrypt persists an unlock session, so search works until vault lock.
	out, code := runMPPty(t, dir, nil, "secret1\nsecret1\n", "vault", "encrypt")
	if code != 0 || !strings.Contains(out, "Vault encrypted") {
		t.Fatalf("encrypt code=%d out=%q", code, out)
	}
	if _, _, code := runMP(t, dir, nil, "", "--json", "search", "Vault"); code != 0 {
		t.Fatalf("search after encrypt code=%d", code)
	}

	if out, errOut, code := runMP(t, dir, nil, "", "vault", "encrypt"); code == 0 || !strings.Contains(errOut, "already encrypted") {
		t.Fatalf("second encrypt code=%d out=%q err=%q", code, out, errOut)
	}

	lockOut := mustRun(t, dir, "vault", "lock")
	if !strings.Contains(lockOut, "Vault locked") {
		t.Fatalf("lock out = %q", lockOut)
	}
	_, errOut, code := runMP(t, dir, []string{"MINDPALACE_PASSWORD=secret1"}, "", "--json", "search", "Vault")
	if code == 0 || !strings.Contains(errOut, "vault is locked") {
		t.Fatalf("locked search code=%d err=%q", code, errOut)
	}

	out, code = runMPPty(t, dir, nil, "secret1\n", "vault", "unlock")
	if code != 0 || !strings.Contains(out, "Vault unlocked") {
		t.Fatalf("unlock code=%d out=%q", code, out)
	}
	hits := decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "search", "Vault"))
	if len(hits) != 1 {
		t.Fatalf("hits after unlock = %+v", hits)
	}

	out, code = runMPPty(t, dir, nil, "secret1\nsecret2\nsecret2\n", "vault", "password")
	if code != 0 {
		t.Fatalf("password code=%d out=%q", code, out)
	}
	mustRun(t, dir, "vault", "lock")
	out, code = runMPPty(t, dir, nil, "secret1\n", "vault", "unlock")
	if code == 0 || !strings.Contains(out, "wrong password") {
		t.Fatalf("old password unlock code=%d out=%q", code, out)
	}
	out, code = runMPPty(t, dir, nil, "secret2\n", "vault", "unlock")
	if code != 0 || !strings.Contains(out, "Vault unlocked") {
		t.Fatalf("new password unlock code=%d out=%q", code, out)
	}

	out, code = runMPPty(t, dir, nil, "secret2\n", "vault", "decrypt")
	if code != 0 || !strings.Contains(out, "Vault decrypted") {
		t.Fatalf("decrypt code=%d out=%q", code, out)
	}
	hits = decodeJSON[[]dto.SearchHit](t, mustRun(t, dir, "--json", "search", "Vault"))
	if len(hits) != 1 {
		t.Fatalf("hits after decrypt = %+v", hits)
	}
	if _, errOut, code := runMP(t, dir, nil, "", "reindex"); code != 0 {
		t.Fatalf("reindex code=%d err=%q", code, errOut)
	}
}

func TestCLIVaultPlaintextRejectsDecryptAndPassword(t *testing.T) {
	t.Parallel()
	dir := initVault(t)
	_, errOut, code := runMP(t, dir, nil, "", "vault", "decrypt")
	if code == 0 || !strings.Contains(errOut, "not encrypted") {
		t.Fatalf("decrypt code=%d err=%q", code, errOut)
	}
	_, errOut, code = runMP(t, dir, nil, "", "vault", "password")
	if code == 0 || !strings.Contains(errOut, "not encrypted") {
		t.Fatalf("password code=%d err=%q", code, errOut)
	}
}
