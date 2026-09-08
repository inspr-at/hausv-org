// Progressive enhancement of the existing workflow forms. The server remains
// authoritative for permissions, transitions and appointment prerequisites.
(() => {
  const board = document.querySelector('[data-issue-board]');
  if (!board || !window.fetch) return;
  board.classList.add('is-enhanced');
  const feedback = board.querySelector('[data-board-feedback]');
  const columns = () => [...board.querySelectorAll('[data-board-status]')];
  let dragged = null;
  let target = null;
  let pending = false;
  let pointer = null;
  let preview = null;
  let frame = 0;
  let suppressClick = false;

  function announce(message, error = false) {
    feedback.dataset.error = String(error);
    feedback.textContent = message;
  }

  function recount() {
    columns().forEach(column => {
      const count = column.querySelectorAll('[data-board-card]').length;
      column.querySelector('[data-board-count]').textContent = String(count);
      column.querySelector('.board-column-empty').hidden = count !== 0;
    });
  }

  function highlight(column) {
    if (target === column) return;
    target?.classList.remove('is-drop-target');
    target = column;
    target?.classList.add('is-drop-target');
  }

  function finishDrag() {
    dragged?.classList.remove('is-dragging');
    dragged = null;
    highlight(null);
    cancelAnimationFrame(frame);
    if (pointer?.handle.hasPointerCapture(pointer.id)) {
      pointer.handle.releasePointerCapture(pointer.id);
    }
    pointer = null;
    preview?.remove();
    preview = null;
  }

  const failureMessage = code => ({
    termin: 'Bitte zuerst einen Termin unter „Bearbeiten“ vereinbaren.',
    invalid: 'Dieser Statuswechsel ist nicht möglich. Bitte die Angaben unter „Bearbeiten“ prüfen.',
    missing: 'Das Anliegen ist nicht mehr verfügbar. Bitte das Board neu laden.',
  }[code] || 'Verschieben nicht bestätigt. Bitte das Board neu laden und den Status prüfen.');

  async function move(card, destination, keyboard = false) {
    if (pending || !card || !destination) return;
    const source = card.closest('[data-board-status]');
    if (source === destination) return;
    const form = card.querySelector('[data-board-move]');
    const status = destination.dataset.boardStatus;
    const label = destination.querySelector('h2').textContent;
    const originalNext = card.nextSibling;
    const oldStatus = card.dataset.issueStatus;
    const pill = card.querySelector('.issue-meta .pill');
    const oldLabel = pill.textContent;
    const body = new URLSearchParams(new FormData(form));
    body.set('status', status);
    pending = true;
    let currentCard = card;
    card.querySelector('.board-move-menu').open = false;
    card.setAttribute('aria-busy', 'true');
    card.draggable = false;
    destination.querySelector('.issue-list').prepend(card);
    card.dataset.issueStatus = status;
    pill.textContent = label;
    recount();
    announce(`Verschieben nach ${label} …`);
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 20000);
    try {
      // Use the rendered action: it already contains any tenant path prefix.
      // A validation failure also redirects to an HTML page with HTTP 200.
      const response = await fetch(form.action, {
        method: 'POST', credentials: 'same-origin', body,
        headers: { Accept: 'text/html' }, signal: controller.signal,
      });
      const resultURL = new URL(response.url);
      if (response.status === 403) {
        throw new Error('Dieser Statuswechsel ist für Sie nicht erlaubt.');
      }
      if (!response.ok || resultURL.searchParams.get('issue') !== 'updated') {
        throw new Error(failureMessage(resultURL.searchParams.get('issue')));
      }
      const documentResult = new DOMParser().parseFromString(await response.text(), 'text/html');
      const saved = documentResult.getElementById(card.id);
      if (!saved?.matches('[data-board-card]') || saved.dataset.issueStatus !== status) {
        throw new Error(failureMessage(''));
      }
      // Reuse server-rendered status, next step and form values after each move.
      // Event delegation also covers this replacement, including a second move.
      currentCard = document.importNode(saved, true);
      card.replaceWith(currentCard);
      const summary = documentResult.querySelector('.board-summary');
      if (summary) board.querySelector('.board-summary')?.replaceWith(document.importNode(summary, true));
      announce(`Verschoben nach ${label}`);
    } catch (error) {
      source.querySelector('.issue-list').insertBefore(card, originalNext);
      card.dataset.issueStatus = oldStatus;
      pill.textContent = oldLabel;
      form.elements.status.value = oldStatus;
      const message = error.name === 'AbortError' || error instanceof TypeError ? failureMessage('') : error.message;
      announce(`${message || failureMessage('')} Die Karte wurde zurückgesetzt.`, true);
    } finally {
      clearTimeout(timeout);
      currentCard.removeAttribute('aria-busy');
      currentCard.draggable = true;
      pending = false;
      recount();
      if (keyboard) currentCard.querySelector('.board-move-menu > summary').focus();
    }
  }

  board.addEventListener('submit', event => {
    const form = event.target.closest('[data-board-move]');
    if (!form) return;
    event.preventDefault();
    // app.js locks normal POSTs. This form owns its asynchronous busy state.
    event.stopPropagation();
    void move(form.closest('[data-board-card]'),
      columns().find(column => column.dataset.boardStatus === form.elements.status.value), true);
  });

  board.addEventListener('dragstart', event => {
    const card = event.target.closest('[data-board-card]');
    if (pending || pointer || !card || event.target.closest('a,select,input,summary,.board-move-menu')) {
      event.preventDefault();
      return;
    }
    dragged = card;
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', card.id);
    card.classList.add('is-dragging');
  });
  board.addEventListener('dragover', event => {
    if (!dragged || pointer) return;
    const column = event.target.closest('[data-board-status]');
    event.preventDefault();
    event.dataTransfer.dropEffect = column ? 'move' : 'none';
    highlight(column);
  });
  board.addEventListener('dragleave', event => {
    if (!board.contains(event.relatedTarget)) highlight(null);
  });
  board.addEventListener('drop', event => {
    if (!dragged || pointer) return;
    event.preventDefault();
    const card = dragged;
    const destination = event.target.closest('[data-board-status]');
    finishDrag();
    void move(card, destination);
  });
  board.addEventListener('dragend', finishDrag);

  function pointAtColumn() {
    if (!pointer || !dragged) return;
    highlight(document.elementFromPoint(pointer.x, pointer.y)?.closest('[data-board-status]') || null);
    const width = Math.min(260, window.innerWidth - 16);
    preview.style.width = `${width}px`;
    preview.style.left = `${Math.max(8, Math.min(pointer.x + 12, window.innerWidth - width - 8))}px`;
    preview.style.top = `${Math.max(8, Math.min(pointer.y + 16, window.innerHeight - preview.offsetHeight - 8))}px`;
  }

  function scrollDuringTouchDrag() {
    if (!pointer || !dragged) return;
    // Wrapped lanes can be below the fold. Holding at an edge reaches them.
    const delta = pointer.y < 110 ? -12 : pointer.y > window.innerHeight - 70 ? 12 : 0;
    if (delta) window.scrollBy(0, delta);
    const grid = board.querySelector('.board-columns');
    if (grid.scrollWidth > grid.clientWidth) {
      const bounds = grid.getBoundingClientRect();
      grid.scrollLeft += pointer.x < bounds.left + 40 ? -12 : pointer.x > bounds.right - 40 ? 12 : 0;
    }
    pointAtColumn();
    frame = requestAnimationFrame(scrollDuringTouchDrag);
  }

  board.addEventListener('pointerdown', event => {
    if (event.pointerType === 'mouse' || !event.isPrimary || pending || pointer) return;
    const handle = event.target.closest('[data-board-drag]');
    if (!handle) return;
    pointer = { id: event.pointerId, handle, x: event.clientX, y: event.clientY,
      startX: event.clientX, startY: event.clientY };
    suppressClick = false;
    handle.setPointerCapture(event.pointerId);
  });
  board.addEventListener('pointermove', event => {
    if (!pointer || event.pointerId !== pointer.id) return;
    pointer.x = event.clientX;
    pointer.y = event.clientY;
    if (!dragged && Math.hypot(pointer.x - pointer.startX, pointer.y - pointer.startY) >= 8) {
      dragged = pointer.handle.closest('[data-board-card]');
      dragged.classList.add('is-dragging');
      preview = document.createElement('div');
      preview.className = 'board-touch-preview';
      preview.setAttribute('aria-hidden', 'true');
      preview.textContent = dragged.querySelector('h3').textContent;
      document.body.append(preview);
      suppressClick = true;
      frame = requestAnimationFrame(scrollDuringTouchDrag);
    }
    if (dragged) event.preventDefault();
    pointAtColumn();
  });
  board.addEventListener('pointerup', event => {
    if (!pointer || event.pointerId !== pointer.id) return;
    pointer.x = event.clientX;
    pointer.y = event.clientY;
    pointAtColumn();
    const card = dragged;
    const destination = target;
    finishDrag();
    void move(card, destination);
  });
  board.addEventListener('pointercancel', finishDrag);
  board.addEventListener('lostpointercapture', () => { if (pointer) finishDrag(); });
  board.addEventListener('click', event => {
    const handle = event.target.closest('[data-board-drag]');
    if (!handle) return;
    event.preventDefault();
    if (suppressClick) { suppressClick = false; return; }
    const menu = handle.closest('[data-board-card]').querySelector('.board-move-menu');
    menu.open = true;
    menu.querySelector('select').focus();
  });
  document.addEventListener('keydown', event => {
    if (event.key !== 'Escape') return;
    finishDrag();
    const menu = event.target.closest('.board-move-menu');
    if (menu) { menu.open = false; menu.querySelector('summary').focus(); }
  });
  window.addEventListener('blur', finishDrag);
})();
