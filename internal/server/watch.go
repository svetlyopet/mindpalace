package server

import (
	"context"
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
	if err := w.Add(root); err != nil {
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

func shouldIgnoreWatch(vaultRoot, path string) bool {
	return vault.IsDerivedPath(vaultRoot, path)
}
