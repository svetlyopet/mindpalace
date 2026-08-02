function parseTags(raw) {
  const seen = {};
  const out = [];
  raw.split(/[,\s]+/).forEach((part) => {
    part = part.trim();
    if (!part || seen[part]) return;
    seen[part] = true;
    out.push(part);
  });
  return out;
}

function appendTag(input, tag) {
  const parts = parseTags(input.value);
  if (parts.includes(tag)) return;
  input.value = parts.concat(tag).join(", ");
}

async function loadDefaultFullHtml() {
  const { defaultFullHtml } = await chrome.storage.sync.get("defaultFullHtml");
  return !!defaultFullHtml;
}

async function loadDraft() {
  const { captureDraft } = await chrome.storage.session.get("captureDraft");
  if (!captureDraft) {
    document.getElementById("status").textContent = "Nothing to save.";
    return null;
  }
  const mode = captureDraft.mode || "page";
  document.getElementById("capture-heading").textContent =
    mode === "social" ? "Save social post" : "Save page";
  const fullLabel = document.getElementById("capture-full-wrap");
  if (fullLabel) {
    fullLabel.hidden = mode === "social";
  }
  if (mode === "page") {
    document.getElementById("capture-full").checked = await loadDefaultFullHtml();
  }
  const titleEl = document.getElementById("capture-title");
  titleEl.value = captureDraft.title || captureDraft.url || "";
  const suggestionsEl = document.getElementById("suggestions");
  const suggested = captureDraft.suggested_tags || [];
  if (suggested.length) {
    const label = document.createElement("span");
    label.textContent = "Suggested:";
    suggestionsEl.appendChild(label);
    const input = document.getElementById("tags");
    suggested.forEach((tag) => {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "chip";
      btn.textContent = tag;
      btn.addEventListener("click", () => appendTag(input, tag));
      suggestionsEl.appendChild(btn);
    });
    suggestionsEl.hidden = false;
  }
  return captureDraft;
}

async function saveDraft(draft) {
  const status = document.getElementById("status");
  const title = document.getElementById("capture-title").value.trim();
  if (!title) {
    status.textContent = "Title is required.";
    document.getElementById("capture-title").focus();
    return;
  }
  const tags = parseTags(document.getElementById("tags").value);
  const thoughts = document.getElementById("capture-thoughts").value.trim();
  const mode = draft.mode || "page";
  const full = mode === "social" ? false : document.getElementById("capture-full").checked;
  status.textContent =
    mode === "social"
      ? "Saving social post…"
      : full
        ? "Saving (full bundle may take a moment)…"
        : "Saving…";
  const base = draft.baseUrl.replace(/\/$/, "");
  let body;
  if (mode === "social") {
    body = {
      kind: "social",
      url: draft.url,
      title,
      tags,
      thoughts,
    };
  } else {
    body = {
      kind: "html",
      url: draft.url,
      html: draft.html,
      title,
      tags,
      thoughts,
      full,
    };
  }
  let res;
  try {
    res = await fetch(`${base}/api/capture`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${draft.token}`,
      },
      body: JSON.stringify(body),
    });
  } catch (err) {
    status.textContent = "Network error: " + (err && err.message ? err.message : "failed");
    return;
  }
  if (!res.ok) {
    status.textContent = await res.text();
    return;
  }
  await chrome.storage.session.remove("captureDraft");
  if (mode === "page") {
    document.getElementById("capture-full").checked = await loadDefaultFullHtml();
  }
  const afterSave = document.getElementById("after-save");
  const openLink = document.getElementById("open-library");
  if (openLink && afterSave) {
    openLink.href = base + "/";
    afterSave.hidden = false;
    status.textContent = "Saved.";
    status.style.color = "#0d6b0d";
    document.getElementById("save").disabled = true;
    return;
  }
  window.close();
}

document.getElementById("cancel").addEventListener("click", () => {
  chrome.storage.session.remove("captureDraft");
  window.close();
});

loadDraft().then((draft) => {
  if (!draft) return;
  document.getElementById("save").addEventListener("click", () => saveDraft(draft));
  document.getElementById("capture-title").focus();
});
