const baseUrlEl = document.getElementById("baseUrl");
const tokenEl = document.getElementById("token");
const defaultFullHtmlEl = document.getElementById("defaultFullHtml");
const statusEl = document.getElementById("status");
const toggleTokenBtn = document.getElementById("toggle-token");

function normalizeBaseUrl(url) {
  return (url || "").trim().replace(/\/$/, "");
}

function setStatus(text, kind) {
  statusEl.textContent = text;
  statusEl.className = "msg" + (kind ? " " + kind : "");
}

function readConfig() {
  return {
    baseUrl: normalizeBaseUrl(baseUrlEl.value),
    token: tokenEl.value.trim(),
    defaultFullHtml: defaultFullHtmlEl.checked,
  };
}

chrome.storage.sync.get(["baseUrl", "token", "defaultFullHtml"], (v) => {
  baseUrlEl.value = v.baseUrl || "http://127.0.0.1:7451";
  tokenEl.value = v.token || "";
  defaultFullHtmlEl.checked = !!v.defaultFullHtml;
});

toggleTokenBtn.addEventListener("click", () => {
  const show = tokenEl.type === "password";
  tokenEl.type = show ? "text" : "password";
  toggleTokenBtn.textContent = show ? "Hide" : "Show";
  toggleTokenBtn.setAttribute("aria-label", show ? "Hide token" : "Show token");
});

document.getElementById("save").addEventListener("click", () => {
  const { baseUrl, token, defaultFullHtml } = readConfig();
  if (!baseUrl) {
    setStatus("Enter a server URL.", "err");
    return;
  }
  if (!token) {
    setStatus("Enter the API token from config.yaml.", "err");
    return;
  }
  chrome.storage.sync.set({ baseUrl, token, defaultFullHtml }, () => {
    setStatus("Saved.", "ok");
  });
});

document.getElementById("test").addEventListener("click", async () => {
  const { baseUrl, token } = readConfig();
  if (!baseUrl || !token) {
    setStatus("Enter URL and token before testing.", "err");
    return;
  }
  setStatus("Testing…", "");
  try {
    const res = await fetch(`${baseUrl}/api/tags`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.status === 401) {
      setStatus("Unauthorized — check serve.token.", "err");
      return;
    }
    if (!res.ok) {
      const body = await res.text();
      setStatus(`Failed (HTTP ${res.status}): ${body.slice(0, 120)}`, "err");
      return;
    }
    setStatus("Connection OK — token accepted.", "ok");
  } catch (err) {
    setStatus(
      "Cannot reach server. Is mp serve running? " + (err && err.message ? err.message : ""),
      "err"
    );
  }
});
