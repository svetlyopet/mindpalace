package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Go error handling", "go-error-handling"},
		{"  Hello!!!  ", "hello"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := Slugify(tc.in); got != tc.want {
			t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestInitOpenFlatLayout(t *testing.T) {
	dir := t.TempDir()
	if _, err := Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("editor: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, configFileName)
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected %s: %v", cfgPath, err)
	}
	v, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v.Root() != dir {
		t.Fatalf("root = %q", v.Root())
	}
}

func TestOpenLegacyLayout(t *testing.T) {
	dir := t.TempDir()
	legacyDir := filepath.Join(dir, ".mindpalace")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "config.yaml"), []byte("editor: vim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err != nil {
		t.Fatal(err)
	}
	if got := ConfigPath(dir); got != filepath.Join(legacyDir, "config.yaml") {
		t.Fatalf("ConfigPath = %q", got)
	}
}

func TestWalkFindsEntriesUnderDotMindpalaceRoot(t *testing.T) {
	dir := t.TempDir()
	// Simulate default vault path whose basename is ".mindpalace".
	root := filepath.Join(dir, ".mindpalace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte("editor: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entryDir := filepath.Join(root, "2026", "07", "25", "abc123-note")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	md := "---\nid: abc123\ntitle: Note\ncreated: 2026-07-25T12:00:00Z\ntype: note\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(entryDir, "entry.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := v.Walk(func(e *Entry) error {
		n++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("Walk entries = %d, want 1", n)
	}
}

func TestSkipWalkDir(t *testing.T) {
	if !SkipWalkDir("index") || !SkipWalkDir(".mindpalace") {
		t.Fatal("expected skip for index and .mindpalace")
	}
	if SkipWalkDir("2026") {
		t.Fatal("should not skip date dirs")
	}
}

func testVault(t *testing.T) *Vault {
	t.Helper()
	dir := t.TempDir()
	if _, err := Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("editor: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCreateRequiresTitle(t *testing.T) {
	v := testVault(t)
	e := &Entry{
		ID:   "notitle",
		Type: TypeNote,
		Body: "body",
	}
	if err := v.Create(e); err == nil {
		t.Fatal("expected error creating entry without title")
	}
}

func TestDeleteRemovesEntryDir(t *testing.T) {
	v := testVault(t)
	e := &Entry{
		ID:    "del001",
		Title: "To delete",
		Type:  TypeNote,
		Body:  "body",
	}
	if err := v.Create(e); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.Dir, "sidecar.txt"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := v.Delete(e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get(e.ID); err != ErrNotFound {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
	if _, err := os.Stat(e.Dir); !os.IsNotExist(err) {
		t.Fatalf("entry dir still exists: %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	v := testVault(t)
	if err := v.Delete("missing"); err != ErrNotFound {
		t.Fatalf("Delete(missing) = %v, want ErrNotFound", err)
	}
}
