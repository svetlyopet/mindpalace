package vault

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/svetlyopet/mindpalace/internal/fsperm"
)

var ErrNotFound = errors.New("entry not found")

type Type string

const (
	TypeArticle    Type = "article"
	TypeNote       Type = "note"
	TypeSocial     Type = "social"
	TypeScreenshot Type = "screenshot"
	TypeSnippet    Type = "snippet"
)

func (t Type) Valid() bool {
	switch t {
	case TypeArticle, TypeNote, TypeSocial, TypeScreenshot, TypeSnippet:
		return true
	default:
		return false
	}
}

type Entry struct {
	ID      string
	Title   string
	Created time.Time
	Type    Type
	Source  string
	Tags    []string
	Extra   map[string]any
	Body    string
	Dir     string
}

type Vault struct {
	root   string
	encCfg *UnlockConfig
	cipher *Cipher
}

func Init(root string) (*Vault, error) {
	root = filepath.Clean(root)
	if err := os.MkdirAll(root, fsperm.DirMode); err != nil {
		return nil, fmt.Errorf("create vault root: %w", err)
	}
	return &Vault{root: root}, nil
}

func Open(root string) (*Vault, error) {
	root = filepath.Clean(root)
	cfgPath := ConfigPath(root)
	if _, err := os.Stat(cfgPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not a vault: missing %s (run mp vault init)", configFileName)
		}
		return nil, err
	}
	return &Vault{root: root}, nil
}

func (v *Vault) Root() string {
	return v.root
}

func NewID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return strings.ToLower(enc)[:6]
}

func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 40 {
		out = strings.Trim(out[:40], "-")
	}
	return out
}

func (v *Vault) Create(e *Entry) error {
	if e == nil {
		return errors.New("entry is nil")
	}
	if e.ID == "" {
		e.ID = NewID()
	}
	if e.Created.IsZero() {
		e.Created = time.Now()
	}
	if e.Type == "" {
		e.Type = TypeNote
	}
	if !e.Type.Valid() {
		return fmt.Errorf("invalid entry type %q", e.Type)
	}
	if strings.TrimSpace(e.Title) == "" {
		return fmt.Errorf("entry title is required")
	}

	slug := Slugify(e.Title)
	if slug == "" {
		slug = string(e.Type)
	}

	created := e.Created.Local()
	dirName := fmt.Sprintf("%s-%s", e.ID, slug)
	rel := filepath.Join(
		fmt.Sprintf("%04d", created.Year()),
		fmt.Sprintf("%02d", int(created.Month())),
		fmt.Sprintf("%02d", created.Day()),
		dirName,
	)

	for attempt := 0; attempt < 5; attempt++ {
		abs := filepath.Join(v.root, rel)
		if _, err := os.Stat(abs); errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(abs, fsperm.DirMode); err != nil {
				return err
			}
			e.Dir = abs
			return WriteEntry(abs, e, v.cipher)
		}
		e.ID = NewID()
		dirName = fmt.Sprintf("%s-%s", e.ID, slug)
		rel = filepath.Join(
			fmt.Sprintf("%04d", created.Year()),
			fmt.Sprintf("%02d", int(created.Month())),
			fmt.Sprintf("%02d", created.Day()),
			dirName,
		)
	}
	return fmt.Errorf("could not allocate unique entry directory")
}

func (v *Vault) Get(id string) (*Entry, error) {
	dir, err := v.findDir(id)
	if err != nil {
		return nil, err
	}
	return ReadEntry(dir, v.cipher)
}

func (v *Vault) Update(e *Entry) error {
	if e == nil || e.Dir == "" {
		return errors.New("entry dir required for update")
	}
	return WriteEntry(e.Dir, e, v.cipher)
}

func (v *Vault) Delete(id string) error {
	dir, err := v.findDir(id)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(v.root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to delete path outside vault: %s", dir)
	}
	if IsDerivedPath(v.root, dir) {
		return fmt.Errorf("refusing to delete derived vault path: %s", dir)
	}
	return os.RemoveAll(dir)
}

func (v *Vault) Walk(fn func(*Entry) error) error {
	return filepath.WalkDir(v.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != v.root && SkipWalkDir(filepath.Base(path)) {
			return fs.SkipDir
		}
		entryPath := filepath.Join(path, "entry.md")
		if _, err := os.Stat(entryPath); err != nil {
			return nil
		}
		e, err := ReadEntry(path, v.cipher)
		if err != nil {
			return err
		}
		return fn(e)
	})
}

func (v *Vault) WalkDirs(fn func(dir string, mtime time.Time) error) error {
	return filepath.WalkDir(v.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != v.root && SkipWalkDir(filepath.Base(path)) {
			return fs.SkipDir
		}
		entryPath := filepath.Join(path, "entry.md")
		st, err := os.Stat(entryPath)
		if err != nil {
			return nil
		}
		return fn(path, st.ModTime())
	})
}

func (v *Vault) findDir(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", ErrNotFound
	}
	var found string
	_ = v.WalkDirs(func(dir string, _ time.Time) error {
		if found != "" {
			return nil
		}
		base := filepath.Base(dir)
		if strings.HasPrefix(base, id+"-") || base == id {
			found = dir
		}
		return nil
	})
	if found == "" {
		return "", ErrNotFound
	}
	return found, nil
}
