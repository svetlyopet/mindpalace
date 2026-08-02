function normalizeBaseUrl(url) {
  return (url || "").trim().replace(/\/$/, "");
}

function truncate(text, max) {
  const s = String(text || "").replace(/\s+/g, " ").trim();
  if (s.length <= max) return s;
  return s.slice(0, max - 1) + "…";
}

function notify(title, message) {
  chrome.notifications.create({
    type: "basic",
    iconUrl: "icons/icon128.png",
    title,
    message: truncate(message, 240),
  });
}

function openOptions() {
  chrome.runtime.openOptionsPage();
}

async function loadConfig() {
  const { baseUrl, token } = await chrome.storage.sync.get(["baseUrl", "token"]);
  const normalized = normalizeBaseUrl(baseUrl);
  if (!normalized || !token) {
    notify(
      "Mindpalace not configured",
      "Set the server URL and API token in extension options (from vault config.yaml serve.token)."
    );
    openOptions();
    return null;
  }
  return { baseUrl: normalized, token };
}

function restrictedTabMessage(url) {
  if (!url) return "No active page to capture.";
  if (url.startsWith("chrome://") || url.startsWith("chrome-extension://")) {
    return "Cannot capture Chrome internal pages. Open a normal website first.";
  }
  if (url.startsWith("https://chrome.google.com/webstore")) {
    return "Cannot capture the Chrome Web Store.";
  }
  if (url.startsWith("edge://") || url.startsWith("about:")) {
    return "Cannot capture this browser page.";
  }
  return "Cannot read this page. Try a regular http(s) tab.";
}

function isRestrictedTabUrl(tabUrl) {
  return (
    !tabUrl ||
    tabUrl.startsWith("chrome://") ||
    tabUrl.startsWith("chrome-extension://") ||
    tabUrl.startsWith("https://chrome.google.com/webstore") ||
    tabUrl.startsWith("edge://") ||
    tabUrl.startsWith("about:")
  );
}

function validateTab(tab) {
  if (!tab || !tab.id) {
    notify("Capture failed", "No active tab.");
    return false;
  }
  const tabUrl = tab.url || "";
  if (isRestrictedTabUrl(tabUrl)) {
    notify("Cannot capture this page", restrictedTabMessage(tabUrl));
    return false;
  }
  return true;
}

async function capturePageHtml(tabId) {
  const [{ result: html }] = await chrome.scripting.executeScript({
    target: { tabId },
    func: () => document.documentElement.outerHTML,
  });
  return html;
}

async function openCapturePopup(draft) {
  await chrome.storage.session.set({ captureDraft: draft });
  await chrome.windows.create({
    url: chrome.runtime.getURL("capture.html"),
    type: "popup",
    width: 420,
    height: 360,
  });
}

async function fetchPreview(baseUrl, token, body) {
  return fetch(`${baseUrl}/api/capture/preview`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  });
}

function handlePreviewError(previewRes, body) {
  if (previewRes.status === 401) {
    notify("Unauthorized", "Check serve.token in extension options.");
    openOptions();
    return;
  }
  notify("Preview failed", `HTTP ${previewRes.status}: ${truncate(body, 160)}`);
}

async function beginPageCapture(tab) {
  const config = await loadConfig();
  if (!config || !validateTab(tab)) return;

  const tabUrl = tab.url || "";
  let html;
  try {
    html = await capturePageHtml(tab.id);
  } catch (err) {
    notify("Capture failed", restrictedTabMessage(tabUrl) + " " + truncate(err && err.message, 80));
    return;
  }

  let previewRes;
  try {
    previewRes = await fetchPreview(config.baseUrl, config.token, {
      kind: "html",
      url: tabUrl,
      html,
    });
  } catch (err) {
    notify("Mindpalace unreachable", "Is mp serve running? " + truncate(err && err.message, 120));
    return;
  }

  if (!previewRes.ok) {
    handlePreviewError(previewRes, await previewRes.text());
    return;
  }

  const preview = await previewRes.json();
  await openCapturePopup({
    mode: "page",
    baseUrl: config.baseUrl,
    token: config.token,
    url: tabUrl,
    html,
    title: preview.title,
    suggested_tags: preview.suggested_tags || [],
  });
}

async function beginSocialCapture(tab) {
  const config = await loadConfig();
  if (!config || !validateTab(tab)) return;

  const tabUrl = tab.url || "";
  let previewRes;
  try {
    previewRes = await fetchPreview(config.baseUrl, config.token, {
      kind: "social",
      url: tabUrl,
    });
  } catch (err) {
    notify("Mindpalace unreachable", "Is mp serve running? " + truncate(err && err.message, 120));
    return;
  }

  if (!previewRes.ok) {
    handlePreviewError(previewRes, await previewRes.text());
    return;
  }

  const preview = await previewRes.json();
  await openCapturePopup({
    mode: "social",
    baseUrl: config.baseUrl,
    token: config.token,
    url: tabUrl,
    title: preview.title,
    suggested_tags: preview.suggested_tags || [],
  });
}

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: "save-social",
    title: "Save as social post to Mindpalace",
    contexts: ["page"],
  });
});

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId !== "save-social") return;
  try {
    await beginSocialCapture(tab);
  } catch (err) {
    notify("Capture failed", truncate(err && err.message, 200));
  }
});

chrome.action.onClicked.addListener(async (tab) => {
  try {
    await beginPageCapture(tab);
  } catch (err) {
    notify("Capture failed", truncate(err && err.message, 200));
  }
});
