package server

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

func (s *Server) startWatcher(ctx context.Context) (func(), error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	root := s.lib.Vault.Root()
	// Entries live in nested date directories. A root-only watch does not see
	// entry.md writes, so edit-while-serve would never refresh the index.
	if err := watchTree(w, root, root); err != nil {
		_ = w.Close()
		return nil, err
	}

	done := make(chan struct{})
	var debounce *time.Timer

	go func() {
		defer w.Close()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Has(fsnotify.Create) {
					fi, statErr := os.Stat(ev.Name)
					if statErr == nil && fi.IsDir() && shouldWatchDir(root, ev.Name) {
						_ = watchTree(w, root, ev.Name)
					}
				}
				if shouldIgnoreWatch(root, ev.Name) {
					continue
				}
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(300*time.Millisecond, func() {
					_ = s.lib.Index.Refresh(context.Background(), s.lib.Vault)
				})
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return func() { close(done) }, nil
}

func watchTree(w *fsnotify.Watcher, vaultRoot, dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if !shouldWatchDir(vaultRoot, path) {
			return filepath.SkipDir
		}
		return w.Add(path)
	})
}

func shouldWatchDir(vaultRoot, path string) bool {
	vaultRoot = filepath.Clean(vaultRoot)
	path = filepath.Clean(path)
	if path == vaultRoot {
		return true
	}
	rel, err := filepath.Rel(vaultRoot, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	top := rel
	if i := strings.Index(rel, string(filepath.Separator)); i >= 0 {
		top = rel[:i]
	}
	if vault.SkipWalkDir(top) {
		return false
	}
	return !vault.IsDerivedPath(vaultRoot, path)
}

func shouldIgnoreWatch(vaultRoot, path string) bool {
	return vault.IsDerivedPath(vaultRoot, path)
}
