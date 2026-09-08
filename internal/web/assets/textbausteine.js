// Same-origin enhancement for the list and the textbaustein editor.
(function () {
  const title = document.querySelector('[data-textbaustein-title]');
  const key = document.querySelector('[data-textbaustein-key]');
  const body = document.querySelector('[data-textbaustein-body]');
  if (title && key && !key.value) {
    title.addEventListener('input', function () {
      key.value = title.value.toLowerCase().replace(/ä/g, 'ae').replace(/ö/g, 'oe')
        .replace(/ü/g, 'ue').replace(/ß/g, 'ss').replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
    });
  }
  document.querySelectorAll('[data-placeholder]').forEach(function (button) {
    button.addEventListener('click', function () {
      if (!body) return;
      body.setRangeText(button.dataset.placeholder || '', body.selectionStart || 0, body.selectionEnd || 0, 'end');
      body.focus();
    });
  });
})();

// Keep the complete server-rendered list usable when JavaScript is unavailable.
(function () {
  const toolbar = document.querySelector('[data-textbausteine-search]');
  if (!toolbar) return;
  const input = toolbar.querySelector('input');
  const count = toolbar.querySelector('[data-textbausteine-count]');
  const empty = document.querySelector('[data-textbausteine-no-results]');
  const groups = Array.from(document.querySelectorAll('.textbausteine-group')).map(group => ({
    element: group,
    category: group.querySelector('h2 [data-search-text]'),
    rows: Array.from(group.querySelectorAll('[data-textbaustein-row]')).map(element => ({
      element,
      title: element.querySelector('.textbausteine-title > [data-search-text]'),
      body: element.querySelector('.textbausteine-body [data-search-text]'),
      details: element.querySelector('.textbausteine-body'),
      wasOpen: false,
    })),
  }));
  const originals = new WeakMap();
  groups.forEach(group => {
    originals.set(group.category, group.category.textContent);
    group.rows.forEach(row => [row.title, row.body].forEach(node => originals.set(node, node.textContent)));
  });
  const total = groups.reduce((sum, group) => sum + group.rows.length, 0);
  let searching = false;
  function highlight(node, pattern) {
    const text = originals.get(node);
    node.textContent = '';
    let end = 0;
    if (pattern) {
      for (const match of text.matchAll(pattern)) {
        node.append(document.createTextNode(text.slice(end, match.index)));
        const mark = document.createElement('mark');
        mark.textContent = match[0];
        node.append(mark);
        end = match.index + match[0].length;
      }
    }
    node.append(document.createTextNode(text.slice(end)));
  }
  function filter() {
    const query = input.value.trim();
    const pattern = query ? new RegExp(query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'giu') : null;
    const matches = node => !pattern || new RegExp(pattern.source, 'iu').test(originals.get(node));
    let visible = 0;
    groups.forEach(group => {
      let groupVisible = 0;
      const categoryMatch = matches(group.category);
      highlight(group.category, pattern);
      group.rows.forEach(row => {
        if (query && !searching) row.wasOpen = row.details.open;
        const bodyMatch = matches(row.body);
        const match = categoryMatch || matches(row.title) || bodyMatch;
        row.element.hidden = !match;
        highlight(row.title, pattern);
        highlight(row.body, pattern);
        if (query) row.details.open = match && bodyMatch;
        else if (searching) row.details.open = row.wasOpen;
        if (match) groupVisible++;
      });
      group.element.hidden = groupVisible === 0;
      visible += groupVisible;
    });
    searching = Boolean(query);
    count.textContent = visible + ' von ' + total;
    empty.hidden = visible !== 0;
  }
  input.addEventListener('input', filter);
  toolbar.hidden = false;
  filter();
})();
