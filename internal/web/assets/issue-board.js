// The server owns permissions and workflow validation. Cards and panel contents
// are replaced only with tenant-scoped, server-rendered HTML after a saved POST.
(() => {
  const board = document.querySelector('[data-issue-board]');
  if (!board || !window.fetch) return;
  board.classList.add('is-enhanced');
  const panel = board.querySelector('[data-board-panel]');
  const feedback = board.querySelector('[data-board-feedback]');
  const boardURL = document.querySelector('.board-view-switch [aria-current="page"]').href;
  const columns = () => [...board.querySelectorAll('[data-board-status]')];
  const cards = () => [...board.querySelectorAll('[data-board-card]')].filter(card => !card.hidden);
  const laneFor = status => columns().find(column => column.dataset.boardStatus ===
    (['Abgelehnt', 'Duplikat'].includes(status) ? 'Erledigt' : status));
  let selected = null;
  let panelRequest = null;
  let transition = null;
  let dragged = null;
  let target = null;
  let pending = false;
  let saving = false;
  let pointer = null;
  let preview = null;
  let frame = 0;
  let suppressClick = false;

  function announce(message, error = false) {
    for (const element of [feedback, panel.querySelector('[data-board-panel-feedback]')]) {
      if (!element) continue;
      element.dataset.error = String(error);
      element.textContent = message;
    }
  }

  function recount() {
    columns().forEach(column => {
      const count = [...column.querySelectorAll('[data-board-card]')].filter(card => !card.hidden).length;
      column.querySelector('[data-board-count]').textContent = String(count);
      column.querySelector('.board-column-empty').hidden = count !== 0;
    });
  }

  function selectCard(card) {
    selected?.classList.remove('is-selected');
    selected?.removeAttribute('aria-current');
    selected = card;
    selected?.classList.add('is-selected');
    selected?.setAttribute('aria-current', 'true');
  }

  const failureMessage = code => ({
    zustaendig: 'Bitte auswählen, wer das Anliegen übernimmt.',
    termin: 'Bitte Datum und Uhrzeit für den Termin angeben.',
    invalid: 'Bitte die Angaben prüfen. Das Ende darf nicht vor dem Start liegen.',
    missing: 'Das Anliegen ist nicht mehr verfügbar. Bitte das Board neu laden.',
  }[code] || 'Speichern nicht bestätigt. Bitte das Board neu laden und den Status prüfen.');

  async function openPanel(card, focus = true) {
    panelRequest?.abort();
    const request = new AbortController();
    panelRequest = request;
    selectCard(card);
    panel.hidden = false;
    panel.setAttribute('aria-busy', 'true');
    // Prevent actions on the previously selected issue while loading.
    panel.replaceChildren();
    const close = document.createElement('button');
    close.type = 'button'; close.className = 'board-panel-close';
    close.dataset.boardClose = ''; close.textContent = '×';
    close.setAttribute('aria-label', 'Anliegen schließen');
    const loading = document.createElement('p'); loading.textContent = 'Anliegen wird geladen …';
    panel.append(close, loading);
    if (focus) panel.focus({ preventScroll: true });
    const timeout = setTimeout(() => request.abort(), 20000);
    try {
      const response = await fetch(`${boardURL}/${encodeURIComponent(card.dataset.issueId)}/panel`, {
        credentials: 'same-origin', headers: { Accept: 'text/html' }, signal: request.signal,
      });
      const html = await response.text();
      const doc = new DOMParser().parseFromString(html, 'text/html');
      if (!response.ok || !doc.querySelector('[data-board-move]')) {
        throw new Error(response.status === 404 ? failureMessage('missing') : 'Das Anliegen konnte nicht geladen werden.');
      }
      if (panelRequest !== request) return false;
      doc.querySelectorAll('noscript').forEach(node => node.remove());
      panel.replaceChildren(...[...doc.body.childNodes].map(node => document.importNode(node, true)));
      if (focus) panel.querySelector('[data-board-close]').focus({ preventScroll: true });
      return true;
    } catch (error) {
      if (panelRequest !== request) return false;
      loading.textContent = error.name === 'AbortError' ? 'Das Laden dauert zu lange. Bitte erneut versuchen.' : error.message;
      announce(loading.textContent, true);
      return false;
    } finally {
      clearTimeout(timeout);
      if (panelRequest === request) panel.removeAttribute('aria-busy');
    }
  }

  function rollback() {
    if (!transition) return;
    const { card, source, next } = transition;
    source.insertBefore(card, next?.parentNode === source ? next : null);
    card.classList.remove('is-pending');
    card.removeAttribute('aria-busy');
    card.draggable = true;
    transition = null;
    pending = false;
    panel.querySelector('[data-board-transition]')?.replaceChildren();
    const dialog = panel.querySelector('[data-board-transition]');
    if (dialog) dialog.hidden = true;
    const statusSelect = panel.querySelector('[data-board-move] [name="status"]');
    if (statusSelect) statusSelect.value = card.dataset.issueStatus;
    panel.querySelectorAll('[data-board-assign], [data-board-move]').forEach(form => { form.inert = false; });
    recount();
  }

  function closePanel() {
    if (!saving) rollback();
    panelRequest?.abort();
    panelRequest = null;
    panel.hidden = true;
    const card = selected;
    selectCard(null);
    card?.focus({ preventScroll: true });
  }

  function beginMove(card, status) {
    const destination = laneFor(status);
    if (!destination) return false;
    transition = { card, source: card.parentNode, next: card.nextSibling, status };
    destination.querySelector('.issue-list').prepend(card);
    card.classList.add('is-pending');
    pending = true;
    recount();
    return true;
  }

  async function move(card, destination, keyboard = false, requestedStatus = '') {
    if (pending || !card || !destination) return;
    const status = requestedStatus || destination.dataset.boardStatus;
    if (card.dataset.issueStatus === status) return;
    if (!beginMove(card, status)) return;
    if (!await openPanel(card, keyboard)) { rollback(); return; }
    const needsAssignee = ['Angenommen', 'In Bearbeitung'].includes(status) && !card.dataset.assignee;
    const needsAppointment = status === 'Termin vereinbart' && card.dataset.hasAppointment !== 'true';
    const form = panel.querySelector('[data-board-move]');
    form.elements.status.value = status;
    if (needsAssignee || needsAppointment) {
      const dialog = panel.querySelector('[data-board-transition]');
      dialog.hidden = false;
      const heading = document.createElement('h3'); heading.textContent = needsAssignee ? 'Wer übernimmt?' : 'Wann?';
      const transitionForm = form.cloneNode(true);
      transitionForm.removeAttribute('data-board-move');
      transitionForm.dataset.boardTransitionForm = '';
      transitionForm.querySelector('label').remove();
      transitionForm.querySelector('select[name="status"]').remove();
      const statusInput = document.createElement('input');
      statusInput.type = 'hidden'; statusInput.name = 'status'; statusInput.value = status;
      transitionForm.append(statusInput);
      const submit = transitionForm.querySelector('button');
      if (needsAssignee) transitionForm.querySelector('[name="assignee_email"]').remove();
      const fields = panel.querySelector(needsAssignee ? '[data-board-assignee-options]' : '[data-board-appointment-fields]');
      transitionForm.insertBefore(fields.content.cloneNode(true), submit);
      const actions = document.createElement('div'); actions.className = 'board-panel-actions';
      const cancel = document.createElement('button'); cancel.type = 'button'; cancel.dataset.boardCancel = ''; cancel.textContent = 'Abbrechen';
      actions.append(submit, cancel); transitionForm.append(actions);
      dialog.append(heading, transitionForm);
      panel.querySelectorAll('[data-board-assign], [data-board-move]').forEach(item => { item.inert = true; });
      announce(`${heading.textContent} Verschieben nach ${status} vorbereiten.`);
      transitionForm.querySelector('select,input:not([type="hidden"])').focus();
      return;
    }
    await save(form, card, status, true);
  }

  async function save(form, card, status, moving) {
    if (saving) return;
    saving = true; pending = true;
    const body = new URLSearchParams(new FormData(form));
    form.querySelectorAll('button,select,input').forEach(control => { control.disabled = true; });
    card.setAttribute('aria-busy', 'true'); card.draggable = false;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 20000);
    let currentCard = card;
    try {
      const response = await fetch(form.action, {
        method: 'POST', credentials: 'same-origin', body,
        headers: { Accept: 'text/html' }, signal: controller.signal,
      });
      const resultURL = new URL(response.url);
      if (response.status === 403) throw new Error('Diese Änderung ist für Sie nicht erlaubt.');
      if (!response.ok || resultURL.searchParams.get('issue') !== 'updated') {
        throw new Error(failureMessage(resultURL.searchParams.get('issue')));
      }
      const doc = new DOMParser().parseFromString(await response.text(), 'text/html');
      const saved = doc.getElementById(card.id);
      if (!saved?.matches('[data-board-card]') || saved.dataset.issueStatus !== status) throw new Error(failureMessage(''));
      const restoreCardFocus = document.activeElement === card;
      currentCard = document.importNode(saved, true);
      card.replaceWith(currentCard);
      if (restoreCardFocus) currentCard.focus({ preventScroll: true });
      transition = null;
      for (const selector of ['.board-summary', '.board-quick']) {
        const updated = doc.querySelector(selector);
        if (!updated) continue;
        // Keep the current filter selection while refreshing the global counts.
        if (selector === '.board-quick') {
          board.querySelectorAll('.board-quick a').forEach((link, i) => {
            const count = updated.querySelectorAll('a')[i]?.querySelector('span');
            if (count) link.querySelector('span').textContent = count.textContent;
          });
        } else board.querySelector(selector)?.replaceWith(document.importNode(updated, true));
      }
      const loaded = panel.hidden || await openPanel(currentCard, true);
      const assignee = currentCard.querySelector('.issue-assignee-avatar').getAttribute('aria-label');
      announce(`${moving ? `Verschoben nach ${status}` : 'Zuständigkeit gespeichert'} · ${assignee}${loaded ? '' : ' · Details bitte erneut öffnen.'}`);
    } catch (error) {
      rollback();
      announce(`${error.name === 'AbortError' || error instanceof TypeError ? failureMessage('') : error.message}${moving ? ' Die Karte wurde zurückgesetzt.' : ''}`, true);
    } finally {
      clearTimeout(timeout);
      currentCard.removeAttribute('aria-busy'); currentCard.draggable = true;
      form.querySelectorAll('button,select,input').forEach(control => { control.disabled = false; });
      saving = false; pending = false;
      recount();
    }
  }

  board.addEventListener('submit', event => {
    const form = event.target.closest('[data-board-move],[data-board-assign],[data-board-transition-form]');
    if (!form) return;
    event.preventDefault(); event.stopPropagation();
    if (saving || !selected) return;
    if (form.matches('[data-board-transition-form]')) {
      void save(form, selected, transition.status, true);
    } else if (form.matches('[data-board-assign]')) {
      if (!pending) void save(form, selected, selected.dataset.issueStatus, false);
    } else {
      void move(selected, laneFor(form.elements.status.value), true, form.elements.status.value);
    }
  });

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
    const previous = pointer;
    pointer = null;
    if (previous?.handle.hasPointerCapture(previous.id)) previous.handle.releasePointerCapture(previous.id);
    preview?.remove(); preview = null;
  }

  board.addEventListener('dragstart', event => {
    const card = event.target.closest('[data-board-card]');
    if (pending || pointer || !card || event.target.closest('a,select,input')) { event.preventDefault(); return; }
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
  board.addEventListener('dragleave', event => { if (!board.contains(event.relatedTarget)) highlight(null); });
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
    if (event.target.closest('[data-board-close]')) { event.preventDefault(); closePanel(); return; }
    if (event.target.closest('[data-board-cancel]')) {
      const card = selected; rollback(); announce('Verschieben abgebrochen.'); card?.focus(); return;
    }
    if (event.target.closest('[data-board-drag]')) { event.preventDefault(); suppressClick = false; return; }
    const card = event.target.closest('[data-board-card]');
    if (!card || pending) return;
    if (suppressClick) { suppressClick = false; return; }
    void openPanel(card);
  });
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape') { finishDrag(); closePanel(); return; }
    const editing = event.target.closest('input,select,textarea,[contenteditable="true"]');
    if (editing || pending) return;
    const card = event.target.closest('[data-board-card]');
    if (card && ['Enter', ' '].includes(event.key)) { event.preventDefault(); void openPanel(card); return; }
    if (!board.contains(event.target)) return;
    const direction = { j: 1, k: -1, ArrowDown: 1, ArrowUp: -1 }[event.key];
    if (!direction) return;
    event.preventDefault();
    const list = cards();
    const index = list.indexOf(card || selected);
    const next = list[Math.max(0, Math.min(list.length - 1, index < 0 ? 0 : index + direction))];
    if (next) void openPanel(next);
  });
  // A mobile sheet keeps Tab inside the panel; Escape returns to its card.
  panel.addEventListener('keydown', event => {
    if (event.key !== 'Tab' || !matchMedia('(max-width:759px)').matches) return;
    const targets = [...panel.querySelectorAll('a,button,input:not([type="hidden"]),select,textarea')]
      .filter(el => !el.disabled && !el.closest('[inert]') && el.getClientRects().length);
    const first = targets[0], last = targets.at(-1);
    if (event.shiftKey && (document.activeElement === first || document.activeElement === panel)) { event.preventDefault(); last?.focus(); }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
  });
  document.querySelector('[data-board-search]').addEventListener('input', event => {
    if (pending) return;
    const query = event.target.value.trim().toLocaleLowerCase('de');
    for (const card of board.querySelectorAll('[data-board-card]')) {
      card.hidden = !card.textContent.toLocaleLowerCase('de').includes(query);
    }
    if (selected?.hidden) closePanel();
    recount();
  });
  window.addEventListener('blur', finishDrag);
})();
