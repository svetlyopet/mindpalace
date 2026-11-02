package vault

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadEncryptedBodyTrailingBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		extra   []byte
		wantErr bool
	}{
		{name: "trailing newline", extra: []byte("\n")},
		{name: "tampered byte", extra: []byte("X"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dek, err := NewDEK()
			if err != nil {
				t.Fatal(err)
			}
			c, err := NewCipher(dek)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			const body = "secret body\n"
			e := &Entry{
				ID:      "nl0001",
				Title:   "Trailing",
				Created: time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC),
				Type:    TypeNote,
				Body:    body,
			}
			if err := WriteEntry(dir, e, c); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "entry.md")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(raw, tc.extra...), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := ReadEntry(dir, c)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected decrypt error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Body != body {
				t.Fatalf("body = %q, want %q", got.Body, body)
			}
		})
	}
}
