# Development

Contributor documentation for [Mindpalace](README.md). End-user setup and usage stay in the README.

## Prerequisites

- Go version listed in [`go.mod`](go.mod) (1.25 or newer today).
- [`make`](Makefile) for common tasks.
- Optional: [Syft](https://github.com/anchore/syft) on `PATH` when running `make release` (SBOM generation uses the same tool as CI).

## Local build and test

```bash
make build          # bin/mp
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

Semgrep and OWASP ZAP remain CI-only (Python/Docker actions); see **Security scanning**.

## Local release snapshot

`make release` runs GoReleaser in snapshot mode (`goreleaser release --snapshot --clean`). Artifacts land in `dist/` only; nothing is published to GitHub. Use it to check cross-compiled binaries, SPDX SBOMs (when Syft is installed), and `mp --version` on built binaries.

This is **not** the same as the tag-driven [`.github/workflows/release.yml`](.github/workflows/release.yml) workflow.

## CI pipeline

Workflow: [`.github/workflows/ci.yml`](.github/workflows/ci.yml) (`name: ci`).

| Job | What it does |
| --- | --- |
| **unit** | `go vet ./...`, `go test -race -count=1 ./...` |
| **coverage** | `go test -coverprofile=coverage.out`, summary, uploads `coverage` artifact |
| **e2e** | `make test-e2e` |

Triggers: pushes to `main` / `master` and all pull requests.

If you rename or split this workflow, re-select required status checks under branch protection (GitHub keys checks off workflow/job names).

## Security scanning (post-CI)

These workflows run only after [`ci`](.github/workflows/ci.yml) completes successfully (`workflow_run`). They do not run on tag/release workflows.

| Workflow | Tools | Results |
| --- | --- | --- |
| [`.github/workflows/sast.yml`](.github/workflows/sast.yml) | [gosec](https://github.com/securego/gosec), [Semgrep](https://semgrep.dev/) (`p/golang`, `p/security-audit`), [govulncheck](https://go.dev/security/vuln/) | gosec/Semgrep SARIF → **Security → Code scanning**; failures fail the workflow |
| [`.github/workflows/dast.yml`](.github/workflows/dast.yml) | [OWASP ZAP](https://www.zaproxy.org/) baseline against `mp serve` on loopback | Actions job log and ZAP report artifacts from the action |

Both check out the same commit CI tested (`workflow_run.head_sha`). `sast` and `dast` must exist on the default branch before they trigger on pull requests.

## Release automation (overview)

Tag pushes matching semver (`v*.*.*`) trigger [`.github/workflows/release.yml`](.github/workflows/release.yml). Configuration lives in [`.goreleaser.yaml`](.goreleaser.yaml).

- Cross-platform `mp` binaries, checksums, changelog, and SPDX SBOMs per archive (Syft).
- GitHub Artifact SBOM attestations after publish.
- Version strings are injected at link time via [`internal/version`](internal/version/version.go).
- The release job uses GitHub Environment **`release`**. Create that environment in the repository settings and configure **required reviewers** there so publishes wait for approval. This document does not cover tagging, approving runs, or publishing steps.
