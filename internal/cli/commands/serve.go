package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/svetlyopet/mindpalace/internal/clictx"
	"github.com/svetlyopet/mindpalace/internal/config"
	"github.com/svetlyopet/mindpalace/internal/server"
	"github.com/svetlyopet/mindpalace/internal/vault"
)

var (
	serveAddr          string
	serveOpen          bool
	serveAllowWildcard bool
)

func NewServe(rt *clictx.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the local web UI and API",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rt.Open(); err != nil {
				return err
			}
			defer rt.Close()

			root := rt.VaultRoot()
			cfg := rt.Config
			if serveAddr != "" {
				cfg.Serve.Addr = serveAddr
			}
			if err := cfg.PrepareServe(serveAllowWildcard); err != nil {
				return err
			}
			token, err := config.EnsureServeToken(root, cfg)
			if err != nil {
				return err
			}
			rt.Config = cfg

			srv := server.New(rt.Lib, cfg, token)
			addr := cfg.Serve.Addr
			if !config.ServeAddrLoopback(addr) {
				fmt.Fprintf(os.Stderr, "warning: serving on %s over HTTP; API token and session cookie are not encrypted in transit — use a trusted network\n", addr)
			}
			if serveOpen {
				openBrowser("http://" + addr)
			}
			fmt.Fprintf(os.Stderr, "Mindpalace listening on http://%s\n", addr)
			fmt.Fprintf(os.Stderr, "API token is in %s (serve.token)\n", vault.ConfigPath(root))

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			return srv.Run(ctx, addr)
		},
	}
	clictx.MarkSkipVaultOpen(cmd)
	cmd.Flags().StringVar(&serveAddr, "addr", "", "listen address (overrides config)")
	cmd.Flags().BoolVar(&serveOpen, "open", false, "open the library in a browser")
	cmd.Flags().BoolVar(&serveAllowWildcard, "allow-wildcard-bind", false, "allow binding to 0.0.0.0")
	return cmd
}

func openBrowser(url string) {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "linux":
		c = exec.Command("xdg-open", url)
	default:
		return
	}
	_ = c.Start()
}
