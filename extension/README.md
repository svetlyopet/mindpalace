# Mindpalace Capture (Chrome extension)

Save the current browser tab to your local Mindpalace vault via `mp serve`.

## Prerequisites

1. A Mindpalace vault initialized (`mp vault init` or equivalent).
2. `mp serve` running (default listen address `http://127.0.0.1:7451`).
3. Vault **unlocked** — use the web UI at the server URL or `mp vault unlock` if the vault is encrypted.

## Install (Load unpacked)

1. Open Chrome and go to `chrome://extensions`.
2. Enable **Developer mode** (top right).
3. Click **Load unpacked** and select this directory (`extension/` in the Mindpalace repo).
4. Pin **Mindpalace Capture** from the extensions menu if you want quick access.

## Configure

1. Right-click the extension icon → **Options**, or open Options from `chrome://extensions`.
2. **Server base URL** — usually `http://127.0.0.1:7451` (must match `serve.addr` in vault `config.yaml`).
3. **API token** — copy `serve.token` from the same `config.yaml`. When you run `mp serve`, the config path is printed on stderr.
4. Click **Test connection** — should report “Connection OK”.
5. Optionally enable **Default: save full HTML bundle** if you usually want the sanitized page HTML and fetched CSS stored with each capture.
6. Click **Save**.

## Use

1. Open a normal **http** or **https** page (not `chrome://`, the Web Store, or other restricted URLs).
2. Click the Mindpalace toolbar icon.
3. Edit the title or tags if you want. Use **Save full HTML bundle** to include the full page archive for this save (initial state follows the Options default).
4. Click **Save**. Full-bundle saves can take longer while the server fetches linked stylesheets.
5. Optionally click **Open Mindpalace** in the popup to view the library.

## Troubleshooting

| Symptom | What to check |
|--------|----------------|
| Notification: not configured | Set URL and token in Options. |
| Unauthorized / HTTP 401 | Token must match `serve.token` exactly. |
| Mindpalace unreachable | Is `mp serve` running? Is the URL correct (including port)? |
| Cannot capture this page | Extension cannot read Chrome internal pages or the Web Store. Use a normal website tab. |
| Preview or save fails with vault errors | Unlock the vault in the browser or via CLI. |

The extension talks to localhost only (`host_permissions` for `127.0.0.1` and `localhost`). It uses `Authorization: Bearer <token>`; cookie-based web UI sessions are not used.

## Files

- `manifest.json` — MV3 manifest
- `background.js` — capture flow (page HTML → preview API → popup)
- `capture.html` / `capture.js` — tag entry and save
- `options.html` / `options.js` — server URL and token
- `icons/` — toolbar icons
