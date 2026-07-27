# Mindpalace

Mindpalace is a local-first personal knowledge base. Everything you capture becomes plain files on your machine—searchable from the terminal or a local web UI. One binary (`mp`) does the work; your data never depends on a cloud account.

## Requirements

- **From source:** Go 1.25 or newer, then `make build` to produce `bin/mp`. See [DEVELOPMENT.md](DEVELOPMENT.md) for contributors.
- **Optional:** a web browser for the library UI; Chrome if you use the capture extension.

## Quick start

```bash
make build
./bin/mp vault init
./bin/mp add note -m "Hello" --title "First note" --tags demo
./bin/mp search demo
./bin/mp serve --open
```

The web UI opens at `http://127.0.0.1:7451` by default. To use a specific vault directory:

```bash
./bin/mp vault init /path/to/vault
export MINDPALACE_VAULT=/path/to/vault   # or pass --vault on every command
```

## Your vault

**Default location:** `~/.mindpalace`

**Overrides:** `--vault /path` on any command, or the `MINDPALACE_VAULT` environment variable.

Entries live under date-based folders. Each entry is a directory with `entry.md` as the anchor (YAML frontmatter plus markdown body). Optional files include captured HTML, images, and assets. You can open any entry in a normal text editor or back up the whole vault folder with standard tools.

Example layout:

```
~/.mindpalace/
  config.yaml
  index/              # search index (safe to delete)
  meta.json           # metadata cache (derived)
  2026/
    07/
      26/
        a1b2c3-first-note/
          entry.md
```

If search feels stale or you removed `index/`, run `mp reindex` to rebuild derived data from the files on disk.

## Everyday usage

### Capture

```bash
mp add note -m "Body text" --title "Title" --tags tag1 --tags tag2
mp add url https://example.com/article --title "Article title" --tags web
mp add file ./document.pdf --title "Document" --tags docs
```

For notes, omit `-m` and pipe text on stdin only with `--title` and `--tags`, or run `mp add note` on a TTY to use editors for body, tags, and title. Entry types include `article`, `note`, `social`, `screenshot`, and `snippet` (see `--type` on `mp add`).

### Find entries

```bash
mp search golang errors
mp list
mp list --tag work --since 2w
mp search --tag work --type note --limit 50
```

Filters on `search` and `list`:

- `--tag` (repeatable, AND)
- `--type` (repeatable)
- `--since` (e.g. `2w`, `3d`, `6mo`, or `2026-07-01`)
- `--domain` (source hostname)
- `--limit` (default 20)

### Inspect and open

```bash
mp show <id>
mp open <id>
mp --json show <id>    # machine-readable output for scripts
```

### Tags

```bash
mp tag <id> +new-tag -old-tag
mp tags
```

### Maintain

```bash
mp delete <id> -y      # skip confirmation (needed in scripts)
mp reindex
mp edit <id>           # uses editor from config.yaml or $EDITOR
```

### Web UI and API

```bash
mp serve
mp serve --open
mp serve --addr 127.0.0.1:8080
```

On startup, the server prints the listen URL and reminds you that the API token lives in the vault `config.yaml` under `serve.token`. The extension and HTTP API use that token; the browser UI uses a separate session cookie after unlock.

For full flags: `mp help` and `mp <command> --help`.

## Encrypted vault

```bash
mp vault encrypt
mp vault unlock
mp vault lock
mp vault password
mp vault decrypt
```

After encrypting or decrypting, run `mp reindex` so search matches on-disk content.

For scripts and automation when the vault is encrypted, set `MINDPALACE_PASSWORD` before commands that need to read entries. After `mp vault lock`, the CLI lock marker prevents unlock from env or saved session until you run `mp vault unlock` interactively.

## Browser capture

Save the current Chrome tab into your vault while `mp serve` is running. Install and configure the unpacked extension from [extension/README.md](extension/README.md) (server URL, API token from `config.yaml`, vault unlocked if encrypted).

## Configuration

Settings live in `config.yaml` at the vault root (or legacy `~/.mindpalace/.mindpalace/config.yaml` on older vaults).

Common keys:

- **editor** — program used by `mp edit` and interactive note capture (falls back to `$EDITOR` or `vim`).
- **serve.addr** — listen address (default `127.0.0.1:7451`).
- **serve.token** — bearer token for the API and Chrome extension.
- **capture.auto_tag** — suggest tags on capture when enabled.
- **capture.full_html** — default for saving full page HTML on URL capture.

LLM-related settings exist for future features; the default backend is `none`.

## Build from source

```bash
make build    # produces bin/mp
make serve    # build and run mp serve
make test     # run unit tests
```

The Chrome extension is not built by Make; load `extension/` as an unpacked extension in Chrome (see linked README above).
