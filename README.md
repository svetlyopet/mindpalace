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
mp add social https://x.com/user/status/123 --title "Post title" --tags social
mp add file ./document.pdf --title "Document" --tags docs
```

For notes, omit `-m` and pipe text on stdin only with `--title` and `--tags`, or run `mp add note` on a TTY to use editors for body, tags, and title. Entry types include `article`, `note`, `social`, `screenshot`, and `snippet` (see `--type` on `mp add`).

**Social posts:** use `mp add social` (or the web UI **Add social** button, or the extension **Save as social post** context menu) for public X or Facebook post URLs. **Add link** / `mp add url` works for any URL and may auto-detect social hosts with a Readability fallback.

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
mp tag <id> --add new-tag --remove old-tag
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

While `mp serve` is running, other CLI commands (`search`, `list`, `show`, `add`, `tag`, `tags`, `delete`, `vault unlock` / `lock`) talk to that local API automatically so they do not fight the search index lock. `edit` and `open` still use vault files on disk (the server refreshes the index). Stop the server before `mp reindex` or vault encryption changes (`encrypt` / `decrypt` / `password`).

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

- **Toolbar icon** — save the current page (optional full HTML bundle).
- **Right-click page → Save as social post to Mindpalace** — oEmbed capture for public X or Facebook posts on the open tab.
- **Web UI Add social** — same oEmbed path when `mp serve` is running.

Public **X** and **Facebook** post URLs use oEmbed when `capture.social_oembed` is enabled (default): post text is stored as the entry body, images are saved under the entry's `assets/` folder, and videos are stored as a poster image (when available) plus a link to the original. **Add link** / the toolbar save may auto-detect social URLs with a Readability fallback; **Add social** / the context menu requires a supported post URL and uses oEmbed only.

Social entries also store author metadata in entry frontmatter:

- **X:** display name, `@handle`, profile URL, and a downloaded profile photo (when available)
- **Facebook:** display name, account link, and profile photo (when available)

Example frontmatter:

```yaml
type: social
platform: x
post_id: "2083588839524700409"
author_name: mRr3b00t
author_url: https://x.com/UK_Daniel_Card
author:
  display_name: mRr3b00t
  handle: UK_Daniel_Card
  profile_url: https://x.com/UK_Daniel_Card
  avatar: assets/author-a1b2c3d4.jpg
```

The web UI entry viewer shows the author card when this metadata is present. API responses include an `extra` field with the same data.

### Thoughts (optional commentary)

When capturing links, social posts, or imported files (not notes), you can add personal commentary with `--thoughts` on the CLI, the **Thoughts** field in the web UI capture modals, or the `thoughts` field in API/extension capture requests. Non-empty thoughts are appended to the entry body as a `## Thoughts` markdown section and included in search indexing.

```bash
mp add url "https://example.com" --title "Article" --tags read --thoughts "Read before the meeting"
mp add social "https://x.com/u/status/1" --title "Post" --tags social --thoughts "Key quote inside"
mp add file ./notes.txt --title "Snippet" --tags ref --thoughts "Related to project X"
```

Notes use the entry body for your writing; `--thoughts` is ignored for `mp add note`.

## Configuration

Settings live in `config.yaml` at the vault root (or legacy `~/.mindpalace/.mindpalace/config.yaml` on older vaults).

Common keys:

- **editor** — program used by `mp edit` and interactive note capture (falls back to `$EDITOR` or `vim`).
- **serve.addr** — listen address (default `127.0.0.1:7451`).
- **serve.token** — bearer token for the API and Chrome extension.
- **capture.auto_tag** — suggest tags on capture when enabled.
- **capture.full_html** — default for saving full page HTML on URL capture.
- **capture.social_oembed** — when enabled (default), public X and Facebook post URLs are captured via oEmbed with post text and archived images; videos are stored as a poster image plus link to the original.

LLM-related settings exist for future features; the default backend is `none`.

## Build from source

```bash
make build    # produces bin/mp
make install  # build and copy mp to /usr/local/bin (use sudo if needed)
make serve    # build and run mp serve
make test     # run unit tests
```

`make install PREFIX=/usr` installs to `/usr/bin` instead. `make uninstall` removes the installed binary.

The Chrome extension is not built by Make; load `extension/` as an unpacked extension in Chrome (see linked README above).
