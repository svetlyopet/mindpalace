# Development

Contributor documentation for [Mindpalace](README.md). End-user setup and usage stay in the README.

## Prerequisites

- Go version listed in [`go.mod`](go.mod) (1.25 or newer today).
- [`make`](Makefile) for common tasks.
- Optional: [Syft](https://github.com/anchore/syft) on `PATH` when running `make release` (SBOM generation uses the same tool as CI).
- [gitleaks](https://github.com/gitleaks/gitleaks) and [semgrep](https://semgrep.dev/) on `PATH` when using `make setup-hooks` (pre-commit runs both on every commit). Pre-commit also runs `make govulncheck` when staged files include `*.go`, `go.mod`, or `go.sum` (pinned in `go.mod`; needs Go, not a separate binary).

## Local build and test

```bash
make build          # bin/mp
make install        # build and copy mp to /usr/local/bin (PREFIX=/usr for /usr/bin)
make uninstall      # remove the installed mp
make test           # go test ./...
make test-race      # race detector
make test-cover     # coverage summary
make test-e2e       # CLI subprocess E2E (builds mp first)
```

These mirror what CI runs (see below). Run `make help` for all targets.

## Tooling (pinned Go CLIs)

[`tools/tools.go`](tools/tools.go) (`//go:build tools`) and the `tool` block in [`go.mod`](go.mod) pin **gosec** and **govulncheck** versions. They are not linked into `mp`; use `go run …/cmd/gosec` or the Make targets below.

```bash
make gosec          # Go SAST (same family as CI gosec job)
make govulncheck    # dependency vulnerability scan for code paths you use
```

### gosec exclusions

[`.gosec.json`](.gosec.json) configures path-based rule exclusions (`-conf .gosec.json` in `make gosec` and CI). Rationale for each entry:

| Path | Rules | Why |
|------|-------|-----|
| `internal/config/config.go` | G101 | `APIKeyEnv` default is an environment variable **name**, not a secret |
| `internal/server/session.go` | G124 | Local HTTP serve by design; `Secure` is set when the request uses TLS |
| `internal/server/ui.go` | G203 | Markdown body via goldmark without `html.WithUnsafe`; rendered as `template.HTML` |
| `internal/server/ui_safeurl.go` | G203 | Source link fragment; `Href`/`Label` only from `safeHTTPURL`; built with `html/template` |
| `internal/cli/input/editor.go` | G204 | Editor binary from `LookPath`; note path is a temp file |
| `internal/cli/commands/open.go` | G204 | User-invoked `open` / `xdg-open` for vault entry URLs or dirs |
| `internal/cli/commands/serve.go` | G204 | Same for optional `--open` browser URL |
| `internal/cli/commands/add.go` | G304 | Temp path from `os.CreateTemp` |
| `internal/capture/capture.go` | G204, G304 | Fixed `tesseract` binary; paths validated under entry dir |
| `internal/vault/filecrypto.go` | G304 | Vault-scoped paths enforced at call sites |

Product code does not use inline `#nosec`; exclusions live in this file only.

### Semgrep

```bash
make semgrep    # same rulesets as CI semgrep job (requires semgrep on PATH)
```

Path exclusions use [`.semgrepignore`](.semgrepignore) ([Ignoring files, folders, and code](https://semgrep.dev/docs/ignoring-files-folders-code)). That file skips all Semgrep rules on listed paths:

| Path | Why |
|------|-----|
| `internal/server/session.go` | `cookie-missing-secure` false positive on local HTTP; `Secure` is set when `r.TLS != nil` (same rationale as gosec G124 above) |

Template XSS hardening uses validated source links and social author profile links (`safeHTTPURL` + `html/template` anchor builders in [`internal/server/ui_safeurl.go`](internal/server/ui_safeurl.go)) and Go-computed CSS classes in [`internal/server/ui.go`](internal/server/ui.go).

Semgrep also runs locally via the pre-commit hook (`--verbose`); govulncheck runs in that hook when Go sources or `go.mod` / `go.sum` are staged. OWASP ZAP remains CI-only (Docker action). See **Security scanning**.

## Local release snapshot

`make release` runs GoReleaser in snapshot mode (`goreleaser release --snapshot --clean`). Artifacts land in `dist/` only; nothing is published to GitHub. Use it to check cross-compiled binaries, SPDX SBOMs (when Syft is installed), and `mp --version` on built binaries.

This is **not** the same as the tag-driven [`.github/workflows/release.yml`](.github/workflows/release.yml) workflow.
