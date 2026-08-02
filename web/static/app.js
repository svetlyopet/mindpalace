(function () {
  const tags = window.MPCaptureTags;
  if (!tags) return;

  fetch('/api/session', { credentials: 'same-origin' }).catch(function () {});

  const modal = document.getElementById('note-modal');
  const panel = document.getElementById('note-panel');
  const toggleBtn = document.getElementById('note-panel-toggle');
  const backdrop = document.getElementById('note-modal-backdrop');
  const ta = document.getElementById('note-md');
  const preview = document.getElementById('note-preview');
  const tagsInput = document.getElementById('note-tags');
  const noteTitleInput = document.getElementById('note-title');
  const tagSuggestionsEl = document.getElementById('note-tag-suggestions');
  const noteStatusEl = document.getElementById('note-status');

  const urlModal = document.getElementById('url-modal');
  const socialModal = document.getElementById('social-modal');
  const fileModal = document.getElementById('file-modal');
  const sizeKey = 'mp.noteModalSize';
  const minW = 320;
  const minH = 224;
  const maxW = function () { return Math.min(672, window.innerWidth * 0.92); };
  const maxH = function () { return Math.min(448, window.innerHeight * 0.85); };

  async function parseCaptureResponse(r, statusEl, okMessage) {
    if (!r.ok) {
      statusEl.textContent = await r.text();
      return { ok: false };
    }
    let data;
    try {
      data = await r.json();
    } catch (e) {
      statusEl.textContent = 'Invalid response from server.';
      return { ok: false };
    }
    statusEl.textContent = okMessage;
    const entryId = data.entry && data.entry.id;
    return { ok: true, entryId: entryId || '' };
  }

  function libraryURL(opts) {
    opts = opts || {};
    const filters = document.getElementById('filters');
    if (!filters) return '/';
    const fd = new FormData(filters);
    if (opts.clearSelected) {
      fd.set('selected', '');
    }
    const params = new URLSearchParams();
    fd.forEach(function (value, key) {
      if (value !== '') {
        params.set(key, value);
      }
    });
    const qs = params.toString();
    return qs ? '/?' + qs : '/';
  }

  function navigateAfterLibraryChange(path) {
    location.assign(path);
  }

  function onCaptureSuccess(entryId, closeModal) {
    closeModal();
    if (entryId) {
      navigateAfterLibraryChange('/entry/' + encodeURIComponent(entryId));
      return;
    }
    navigateAfterLibraryChange(libraryURL());
  }

  const noteTagState = tags.createTagPromptState();
  const urlTagState = tags.createTagPromptState();
  const socialTagState = tags.createTagPromptState();
  const fileTagState = tags.createTagPromptState();

  function resetNoteTagPrompt() {
    tags.resetTagPromptState(noteTagState, tagSuggestionsEl);
  }

  function applySavedSize() {
    try {
      const saved = JSON.parse(localStorage.getItem(sizeKey) || '');
      if (saved && saved.w && saved.h) {
        panel.style.width = Math.min(maxW(), Math.max(minW, saved.w)) + 'px';
        panel.style.height = Math.min(maxH(), Math.max(minH, saved.h)) + 'px';
      }
    } catch (e) {}
  }

  function saveSize() {
    localStorage.setItem(sizeKey, JSON.stringify({
      w: panel.offsetWidth,
      h: panel.offsetHeight
    }));
  }

  applySavedSize();

  function openNoteModal() {
    closeUrlModal();
    closeSocialModal();
    closeFileModal();
    modal.hidden = false;
    toggleBtn.setAttribute('aria-expanded', 'true');
    ta.focus();
  }

  function closeNoteModal() {
    modal.hidden = true;
    toggleBtn.setAttribute('aria-expanded', 'false');
    resetNoteTagPrompt();
    noteStatusEl.textContent = '';
  }

  toggleBtn.addEventListener('click', function () {
    if (modal.hidden) openNoteModal();
    else closeNoteModal();
  });
  backdrop.addEventListener('click', closeNoteModal);

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') {
      if (!modal.hidden) { e.preventDefault(); closeNoteModal(); return; }
      if (!urlModal.hidden) { e.preventDefault(); closeUrlModal(); return; }
      if (!socialModal.hidden) { e.preventDefault(); closeSocialModal(); return; }
      if (!fileModal.hidden) { e.preventDefault(); closeFileModal(); return; }
    }
  });

  (function initResize() {
    panel.querySelectorAll('.resize-handle').forEach(function (handle) {
      handle.addEventListener('mousedown', function (e) {
        e.preventDefault();
        const startX = e.clientX;
        const startY = e.clientY;
        const startW = panel.offsetWidth;
        const startH = panel.offsetHeight;
        const east = handle.classList.contains('resize-e') || handle.classList.contains('resize-se');
        const south = handle.classList.contains('resize-s') || handle.classList.contains('resize-se');
        document.body.classList.add('note-resizing');
        function onMove(ev) {
          if (east) {
            panel.style.width = Math.min(maxW(), Math.max(minW, startW + (ev.clientX - startX))) + 'px';
          }
          if (south) {
            panel.style.height = Math.min(maxH(), Math.max(minH, startH + (ev.clientY - startY))) + 'px';
          }
        }
        function onUp() {
          document.body.classList.remove('note-resizing');
          document.removeEventListener('mousemove', onMove);
          document.removeEventListener('mouseup', onUp);
          saveSize();
        }
        document.addEventListener('mousemove', onMove);
        document.addEventListener('mouseup', onUp);
      });
    });
  })();

  let previewTimer;
  function schedulePreview() {
    clearTimeout(previewTimer);
    previewTimer = setTimeout(function () {
      fetch('/ui/markdown-preview', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ markdown: ta.value })
      }).then(function (r) { return r.text(); }).then(function (html) {
        preview.innerHTML = html;
      }).catch(function () {});
    }, 300);
  }
  ta.addEventListener('input', function () {
    schedulePreview();
    if (ta.value.trim() !== noteTagState.key) {
      resetNoteTagPrompt();
    }
  });

  document.querySelectorAll('[data-md]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      const kind = btn.getAttribute('data-md');
      const start = ta.selectionStart;
      const end = ta.selectionEnd;
      const sel = ta.value.slice(start, end);
      let insert = sel;
      switch (kind) {
        case 'h': insert = sel ? '## ' + sel : '## Heading\n'; break;
        case 'b': insert = sel ? '**' + sel + '**' : '**bold**'; break;
        case 'i': insert = sel ? '*' + sel + '*' : '*italic*'; break;
        case 'link': insert = sel ? '[' + sel + '](url)' : '[text](url)'; break;
        case 'list': insert = sel ? '- ' + sel : '- item\n'; break;
      }
      ta.setRangeText(insert, start, end, 'end');
      ta.focus();
      schedulePreview();
    });
  });

  document.getElementById('note-save').addEventListener('click', function () {
    const title = noteTitleInput.value.trim();
    const text = ta.value.trim();
    if (!title) {
      noteStatusEl.textContent = 'Title is required.';
      noteTitleInput.focus();
      return;
    }
    if (!text) return;
    tags.runTagPromptSave(noteTagState, {
      tagsInput: tagsInput,
      suggestionsEl: tagSuggestionsEl,
      statusEl: noteStatusEl,
      getPromptKey: function () { return title + '|' + text; },
      fetchPreview: function () {
        return fetch('/api/capture/preview', {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ kind: 'note', text: text, title: title })
        });
      },
      capture: function (tagList) {
        return fetch('/api/capture', {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ kind: 'note', text: text, title: title, tags: tagList })
        }).then(function (r) {
          return parseCaptureResponse(r, noteStatusEl, 'Saved.');
        });
      },
      onSuccess: function (entryId) {
        noteTitleInput.value = '';
        ta.value = '';
        tagsInput.value = '';
        preview.innerHTML = '';
        resetNoteTagPrompt();
        onCaptureSuccess(entryId, closeNoteModal);
      }
    });
  });

  const urlToggle = document.getElementById('url-panel-toggle');
  const urlBackdrop = document.getElementById('url-modal-backdrop');
  const urlInput = document.getElementById('url-input');
  const urlTitle = document.getElementById('url-title');
  const urlFull = document.getElementById('url-full');
  const urlTags = document.getElementById('url-tags');
  const urlThoughts = document.getElementById('url-thoughts');
  const urlSuggestions = document.getElementById('url-tag-suggestions');
  const urlStatus = document.getElementById('url-status');

  function resetUrlTagPrompt() {
    tags.resetTagPromptState(urlTagState, urlSuggestions);
  }

  function closeUrlModal() {
    urlModal.hidden = true;
    urlToggle.setAttribute('aria-expanded', 'false');
    resetUrlTagPrompt();
    urlStatus.textContent = '';
  }

  function openUrlModal() {
    closeNoteModal();
    closeSocialModal();
    closeFileModal();
    urlModal.hidden = false;
    urlToggle.setAttribute('aria-expanded', 'true');
    urlTitle.focus();
  }

  urlToggle.addEventListener('click', function () {
    if (urlModal.hidden) openUrlModal();
    else closeUrlModal();
  });
  urlBackdrop.addEventListener('click', closeUrlModal);

  urlInput.addEventListener('input', function () {
    if (urlInput.value.trim() !== urlTagState.key) resetUrlTagPrompt();
  });

  document.getElementById('url-save').addEventListener('click', function () {
    const link = urlInput.value.trim();
    if (!link) return;
    let title = urlTitle.value.trim();
    const full = urlFull.checked;

    function runSave() {
      title = urlTitle.value.trim();
      if (!title) {
        urlStatus.textContent = 'Title is required.';
        urlTitle.focus();
        return;
      }
      const promptKey = link + '|' + title + '|' + full;
      tags.runTagPromptSave(urlTagState, {
        tagsInput: urlTags,
        suggestionsEl: urlSuggestions,
        statusEl: urlStatus,
        previewLoadingText: 'Fetching page…',
        loadingText: 'Saving link…',
        getPromptKey: function () { return promptKey; },
        fetchPreview: function () {
          return fetch('/api/capture/preview', {
            method: 'POST',
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ kind: 'url', url: link, full: full, title: title, thoughts: urlThoughts.value.trim() })
          });
        },
        capture: function (tagList) {
          return fetch('/api/capture', {
            method: 'POST',
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ kind: 'url', url: link, full: full, title: title, tags: tagList, thoughts: urlThoughts.value.trim() })
          }).then(function (r) {
            return parseCaptureResponse(r, urlStatus, 'Saved.');
          });
        },
        onSuccess: function (entryId) {
          urlInput.value = '';
          urlTitle.value = '';
          urlFull.checked = false;
          urlTags.value = '';
          urlThoughts.value = '';
          resetUrlTagPrompt();
          onCaptureSuccess(entryId, closeUrlModal);
        }
      });
    }

    if (!title) {
      urlStatus.textContent = 'Fetching page…';
      fetch('/api/capture/preview', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ kind: 'url', url: link, full: full })
      }).then(function (r) {
        if (!r.ok) {
          return r.text().then(function (t) { urlStatus.textContent = t; });
        }
        return r.json();
      }).then(function (data) {
        if (!data) return;
        if (data.title) urlTitle.value = data.title;
        urlStatus.textContent = 'Confirm title and save again.';
        urlTitle.focus();
      }).catch(function () {
        urlStatus.textContent = 'Could not load page preview.';
      });
      return;
    }
    runSave();
  });

  const socialToggle = document.getElementById('social-panel-toggle');
  const socialBackdrop = document.getElementById('social-modal-backdrop');
  const socialInput = document.getElementById('social-input');
  const socialTitle = document.getElementById('social-title');
  const socialTags = document.getElementById('social-tags');
  const socialThoughts = document.getElementById('social-thoughts');
  const socialSuggestions = document.getElementById('social-tag-suggestions');
  const socialStatus = document.getElementById('social-status');

  function resetSocialTagPrompt() {
    tags.resetTagPromptState(socialTagState, socialSuggestions);
  }

  function closeSocialModal() {
    socialModal.hidden = true;
    socialToggle.setAttribute('aria-expanded', 'false');
    resetSocialTagPrompt();
    socialStatus.textContent = '';
  }

  function openSocialModal() {
    closeNoteModal();
    closeUrlModal();
    closeFileModal();
    socialModal.hidden = false;
    socialToggle.setAttribute('aria-expanded', 'true');
    socialTitle.focus();
  }

  socialToggle.addEventListener('click', function () {
    if (socialModal.hidden) openSocialModal();
    else closeSocialModal();
  });
  socialBackdrop.addEventListener('click', closeSocialModal);

  socialInput.addEventListener('input', function () {
    if (socialInput.value.trim() !== socialTagState.key) resetSocialTagPrompt();
  });

  document.getElementById('social-save').addEventListener('click', function () {
    const link = socialInput.value.trim();
    if (!link) return;
    let title = socialTitle.value.trim();

    function runSave() {
      title = socialTitle.value.trim();
      if (!title) {
        socialStatus.textContent = 'Title is required.';
        socialTitle.focus();
        return;
      }
      const promptKey = link + '|' + title;
      tags.runTagPromptSave(socialTagState, {
        tagsInput: socialTags,
        suggestionsEl: socialSuggestions,
        statusEl: socialStatus,
        previewLoadingText: 'Fetching post…',
        loadingText: 'Saving social post…',
        getPromptKey: function () { return promptKey; },
        fetchPreview: function () {
          return fetch('/api/capture/preview', {
            method: 'POST',
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ kind: 'social', url: link, title: title, thoughts: socialThoughts.value.trim() })
          });
        },
        capture: function (tagList) {
          return fetch('/api/capture', {
            method: 'POST',
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ kind: 'social', url: link, title: title, tags: tagList, thoughts: socialThoughts.value.trim() })
          }).then(function (r) {
            return parseCaptureResponse(r, socialStatus, 'Saved.');
          });
        },
        onSuccess: function (entryId) {
          socialInput.value = '';
          socialTitle.value = '';
          socialTags.value = '';
          socialThoughts.value = '';
          resetSocialTagPrompt();
          onCaptureSuccess(entryId, closeSocialModal);
        }
      });
    }

    if (!title) {
      socialStatus.textContent = 'Fetching post…';
      fetch('/api/capture/preview', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ kind: 'social', url: link })
      }).then(function (r) {
        if (!r.ok) {
          return r.text().then(function (t) { socialStatus.textContent = t; });
        }
        return r.json();
      }).then(function (data) {
        if (!data) return;
        if (data.title) socialTitle.value = data.title;
        socialStatus.textContent = 'Confirm title and save again.';
        socialTitle.focus();
      }).catch(function () {
        socialStatus.textContent = 'Could not load post preview.';
      });
      return;
    }
    runSave();
  });

  const fileToggle = document.getElementById('file-panel-toggle');
  const fileBackdrop = document.getElementById('file-modal-backdrop');
  const fileInput = document.getElementById('file-input');
  const fileTitle = document.getElementById('file-title');
  const fileTags = document.getElementById('file-tags');
  const fileThoughts = document.getElementById('file-thoughts');
  const fileSuggestions = document.getElementById('file-tag-suggestions');
  const fileStatus = document.getElementById('file-status');
  const fileNameHint = document.getElementById('file-name-hint');

  function resetFileTagPrompt() {
    tags.resetTagPromptState(fileTagState, fileSuggestions);
  }

  function closeFileModal() {
    fileModal.hidden = true;
    fileToggle.setAttribute('aria-expanded', 'false');
    resetFileTagPrompt();
    fileStatus.textContent = '';
  }

  function openFileModal() {
    closeNoteModal();
    closeUrlModal();
    closeSocialModal();
    fileModal.hidden = false;
    fileToggle.setAttribute('aria-expanded', 'true');
    fileTitle.focus();
  }

  fileToggle.addEventListener('click', function () {
    if (fileModal.hidden) openFileModal();
    else closeFileModal();
  });
  fileBackdrop.addEventListener('click', closeFileModal);

  fileInput.addEventListener('change', function () {
    const f = fileInput.files && fileInput.files[0];
    fileNameHint.textContent = f ? f.name : '';
    resetFileTagPrompt();
  });

  document.getElementById('file-save').addEventListener('click', function () {
    const f = fileInput.files && fileInput.files[0];
    if (!f) {
      fileStatus.textContent = 'Choose a file.';
      return;
    }

    function runSave() {
      const title = fileTitle.value.trim();
      if (!title) {
        fileStatus.textContent = 'Title is required.';
        fileTitle.focus();
        return;
      }
      const promptKey = f.name + '|' + f.size + '|' + title;

      function uploadFormData(tagList) {
        const fd = new FormData();
        fd.append('file', f);
        fd.append('title', title);
        fd.append('tags', JSON.stringify(tagList));
        fd.append('thoughts', fileThoughts.value.trim());
        return fetch('/api/capture/upload', {
          method: 'POST',
          credentials: 'same-origin',
          body: fd
        });
      }

      tags.runTagPromptSave(fileTagState, {
        tagsInput: fileTags,
        suggestionsEl: fileSuggestions,
        statusEl: fileStatus,
        previewLoadingText: 'Analyzing file…',
        loadingText: 'Importing…',
        getPromptKey: function () { return promptKey; },
        fetchPreview: function () {
          const fd = new FormData();
          fd.append('file', f);
          fd.append('title', title);
          return fetch('/api/capture/upload/preview', {
            method: 'POST',
            credentials: 'same-origin',
            body: fd
          });
        },
        capture: function (tagList) {
          return uploadFormData(tagList).then(function (r) {
            return parseCaptureResponse(r, fileStatus, 'Imported.');
          });
        },
        onSuccess: function (entryId) {
          fileInput.value = '';
          fileTitle.value = '';
          fileTags.value = '';
          fileThoughts.value = '';
          fileNameHint.textContent = '';
          resetFileTagPrompt();
          onCaptureSuccess(entryId, closeFileModal);
        }
      });
    }

    if (!fileTitle.value.trim()) {
      fileStatus.textContent = 'Analyzing file…';
      const fd = new FormData();
      fd.append('file', f);
      fetch('/api/capture/upload/preview', {
        method: 'POST',
        credentials: 'same-origin',
        body: fd
      }).then(function (r) {
        if (!r.ok) {
          return r.text().then(function (t) { fileStatus.textContent = t; });
        }
        return r.json();
      }).then(function (data) {
        if (!data) return;
        if (data.title) fileTitle.value = data.title;
        fileStatus.textContent = 'Confirm title and save again.';
        fileTitle.focus();
      }).catch(function () {
        fileStatus.textContent = 'Could not analyze file.';
      });
      return;
    }
    runSave();
  });

  document.getElementById('sidebar-toggle').addEventListener('click', function () {
    document.body.classList.toggle('sidebar-open');
  });

  const deleteModal = document.getElementById('delete-modal');
  const deleteBackdrop = document.getElementById('delete-modal-backdrop');
  const deleteTitleEl = document.getElementById('delete-confirm-title');
  const deleteStatusEl = document.getElementById('delete-confirm-status');
  let pendingDeleteId = '';

  function closeDeleteModal() {
    deleteModal.hidden = true;
    pendingDeleteId = '';
    deleteStatusEl.textContent = '';
  }

  function openDeleteModal(id, title) {
    pendingDeleteId = id;
    deleteTitleEl.textContent = title;
    deleteStatusEl.textContent = '';
    deleteModal.hidden = false;
    document.getElementById('delete-cancel').focus();
  }

  document.body.addEventListener('click', function (e) {
    const btn = e.target.closest('.btn-delete-entry');
    if (!btn) return;
    openDeleteModal(btn.getAttribute('data-entry-id'), btn.getAttribute('data-entry-title') || btn.getAttribute('data-entry-id'));
  });

  document.getElementById('delete-cancel').addEventListener('click', closeDeleteModal);
  deleteBackdrop.addEventListener('click', closeDeleteModal);
  document.getElementById('delete-confirm').addEventListener('click', function () {
    if (!pendingDeleteId) return;
    const id = pendingDeleteId;
    deleteStatusEl.textContent = 'Deleting…';
    fetch('/api/entries/' + encodeURIComponent(id), {
      method: 'DELETE',
      credentials: 'same-origin'
    }).then(async function (r) {
      if (!r.ok) {
        deleteStatusEl.textContent = await r.text();
        return;
      }
      closeDeleteModal();
      navigateAfterLibraryChange(libraryURL({ clearSelected: true }));
    }).catch(function () {
      deleteStatusEl.textContent = 'Delete failed.';
    });
  });

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && !deleteModal.hidden) {
      e.preventDefault();
      closeDeleteModal();
      return;
    }
    if (e.key !== '/' || e.ctrlKey || e.metaKey || e.altKey) return;
    if (!modal.hidden || !urlModal.hidden || !fileModal.hidden) return;
    const t = e.target;
    if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable)) return;
    e.preventDefault();
    document.getElementById('global-search').focus();
  });

  function initEntryTabs(root) {
    root.querySelectorAll('.entry-tabs').forEach(function (tabRoot) {
      tabRoot.querySelectorAll('[data-tab]').forEach(function (btn) {
        if (btn.dataset.tabBound) return;
        btn.dataset.tabBound = '1';
        btn.addEventListener('click', function () {
          const name = btn.getAttribute('data-tab');
          tabRoot.querySelectorAll('[data-tab]').forEach(function (b) {
            b.classList.toggle('is-active', b === btn);
          });
          tabRoot.querySelectorAll('[data-panel]').forEach(function (p) {
            p.hidden = p.getAttribute('data-panel') !== name;
          });
        });
      });
    });
  }
  initEntryTabs(document);
  document.body.addEventListener('htmx:afterSwap', function (e) {
    if (e.detail.target && e.detail.target.id === 'viewer') {
      initEntryTabs(e.detail.target);
    }
  });
})();
