(function (global) {
  function parseTags(raw) {
    const seen = {};
    const out = [];
    raw.split(/[,\s]+/).forEach(function (part) {
      part = part.trim();
      if (!part || seen[part]) return;
      seen[part] = true;
      out.push(part);
    });
    return out;
  }

  function renderTagSuggestions(suggestionsEl, tagsInput, suggested) {
    suggestionsEl.innerHTML = '';
    if (!suggested || !suggested.length) {
      suggestionsEl.hidden = true;
      return;
    }
    const label = document.createElement('span');
    label.className = 'tag-suggestions-label';
    label.textContent = 'Suggested:';
    suggestionsEl.appendChild(label);
    suggested.forEach(function (tag) {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'tag-chip';
      btn.textContent = tag;
      btn.addEventListener('click', function () {
        const parts = parseTags(tagsInput.value);
        if (parts.indexOf(tag) !== -1) return;
        tagsInput.value = parts.concat(tag).join(', ');
      });
      suggestionsEl.appendChild(btn);
    });
    suggestionsEl.hidden = false;
  }

  function createTagPromptState() {
    return { ready: false, key: '' };
  }

  function resetTagPromptState(state, suggestionsEl) {
    state.ready = false;
    state.key = '';
    suggestionsEl.hidden = true;
    suggestionsEl.innerHTML = '';
  }

  function runTagPromptSave(state, opts) {
    const tags = parseTags(opts.tagsInput.value);
    if (tags.length > 0 || state.ready) {
      opts.statusEl.textContent = opts.loadingText || 'Saving…';
      return opts.capture(tags).then(function (result) {
        if (result && result.ok && opts.onSuccess) {
          opts.onSuccess(result.entryId);
        }
      });
    }
    opts.statusEl.textContent = opts.previewLoadingText || 'Loading suggestions…';
    return opts.fetchPreview().then(async function (r) {
      if (!r.ok) {
        opts.statusEl.textContent = await r.text();
        return;
      }
      return r.json();
    }).then(function (data) {
      if (!data) return;
      state.ready = true;
      state.key = opts.getPromptKey();
      renderTagSuggestions(opts.suggestionsEl, opts.tagsInput, data.suggested_tags || []);
      opts.statusEl.textContent = 'Add tags, or save again to continue without tags.';
      opts.tagsInput.focus();
    }).catch(function () {
      opts.statusEl.textContent = 'Could not load tag suggestions.';
    });
  }

  global.MPCaptureTags = {
    parseTags: parseTags,
    renderTagSuggestions: renderTagSuggestions,
    createTagPromptState: createTagPromptState,
    resetTagPromptState: resetTagPromptState,
    runTagPromptSave: runTagPromptSave
  };
})(window);
