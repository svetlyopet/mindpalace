(function () {
  const listEl = document.getElementById('entry-list');
  const filters = document.getElementById('filters');
  if (!listEl || !filters) return;

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function formatCreated(iso) {
    if (!iso) return '';
    return iso.length >= 10 ? iso.slice(0, 10) : iso;
  }

  function renderEntryList(hits, selectedID) {
    if (!hits || hits.length === 0) {
      return '<p class="msg entry-list-empty">No entries.</p>';
    }
    return hits.map(function (hit) {
      const sel = hit.id === selectedID ? ' is-selected' : '';
      const title = escapeHtml(hit.title || hit.id);
      const id = escapeHtml(hit.id);
      const meta = escapeHtml(formatCreated(hit.created)) + ' · ' + escapeHtml(hit.type || '');
      let tags = '';
      (hit.tags || []).forEach(function (t) {
        const enc = encodeURIComponent(t);
        tags += '<span class="tag"><a href="/?tag=' + enc + '">' + escapeHtml(t) + '</a></span>';
      });
      return (
        '<div class="entry-row' + sel + '">' +
        '<a href="/entry/' + id + '" hx-get="/ui/entry/' + id + '" hx-target="#viewer" hx-push-url="/entry/' + id + '">' + title + '</a>' +
        '<span class="entry-meta">' + meta + '</span>' + tags +
        '</div>'
      );
    }).join('');
  }

  function queryStringFromFilters() {
    const fd = new FormData(filters);
    const params = new URLSearchParams();
    fd.forEach(function (value, key) {
      if (value !== '') {
        params.set(key, value);
      }
    });
    params.delete('selected');
    return params.toString();
  }

  function selectedID() {
    const el = filters.querySelector('[name=selected]');
    return el ? el.value : '';
  }

  let debounceTimer;

  function loadEntryList() {
    const qs = queryStringFromFilters();
    const url = '/api/entries' + (qs ? '?' + qs : '');
    fetch(url, { credentials: 'same-origin' })
      .then(function (r) {
        if (!r.ok) throw new Error('list failed');
        return r.json();
      })
      .then(function (hits) {
        listEl.innerHTML = renderEntryList(hits, selectedID());
      })
      .catch(function () {});
  }

  filters.addEventListener('submit', function (e) {
    e.preventDefault();
    loadEntryList();
  });

  filters.addEventListener('input', function (e) {
    if (e.target.id !== 'global-search' && e.target.getAttribute('form') !== 'filters' && !filters.contains(e.target)) {
      return;
    }
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(loadEntryList, 300);
  });

  filters.addEventListener('change', function () {
    loadEntryList();
  });
})();
