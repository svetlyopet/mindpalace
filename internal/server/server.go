package server

import (
	"context"
	"net/http"
	"time"

	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/library"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

// Server is the local HTTP API and UI.
type Server struct {
	lib *library.Library
	cfg *config.Config

	token   string
	vaultFP string

	httpServer  *http.Server
	watcherStop func()
}

func New(lib *library.Library, cfg *config.Config, token string) *Server {
	return &Server{
		lib:     lib,
		cfg:     cfg,
		token:   token,
		vaultFP: vault.VaultFingerprint(lib.Vault.Root()),
	}
}

func (s *Server) Run(ctx context.Context, addr string) error {
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	stopWatch, err := s.startWatcher(ctx)
	if err != nil {
		return err
	}
	s.watcherStop = stopWatch

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
		if s.watcherStop != nil {
			s.watcherStop()
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

// Handler returns the HTTP handler (for tests).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Method-specific patterns never dispatch OPTIONS, so preflight is handled here.
		if r.Method == http.MethodOptions {
			s.setCORS(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
