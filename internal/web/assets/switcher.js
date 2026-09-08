// HAUSV-719: loaded synchronously in the head after the blocking stylesheet.
// The observer runs between parser tasks, before rendering; wait for the last
// footer link so scrollHeight includes the entire navigation, even in streamed HTML.
(() => {
  const install = () => {
    const sidebar = document.querySelector('aside.sidebar');
    if (!sidebar?.querySelector('.sidebar-release .release-trigger')) return false;
    const tenant = sidebar.querySelector('.nav')?.dataset.tenantSlug || location.pathname.split('/app')[0] || 'default';
    const person = sidebar.querySelector('[data-storage-key]')?.dataset.storageKey || '';
    const key = `hausv:sidebar-scroll:${encodeURIComponent(person)}:${encodeURIComponent(tenant)}`;
    const desktop = matchMedia('(min-width:761px)');
    let timer;

    const restore = () => {
      if (!desktop.matches) return;
      let saved;
      try { saved = sessionStorage.getItem(key); } catch { return; }
      if (saved === null || saved.trim() === '') return;
      const top = Number(saved);
      if (!Number.isFinite(top) || top < 0) return;
      const previous = sidebar.style.scrollBehavior;
      sidebar.style.scrollBehavior = 'auto';
      sidebar.scrollTop = top;
      sidebar.style.scrollBehavior = previous;
      // Deliberately preserve the position even when the active link is offscreen.
    };
    const save = () => {
      clearTimeout(timer); timer = undefined;
      // The hidden mobile aside reports zero: never erase the desktop preference.
      if (!desktop.matches) return;
      try { sessionStorage.setItem(key, String(sidebar.scrollTop)); } catch {}
    };

    restore();
    sidebar.addEventListener('scroll', () => {
      if (timer === undefined) timer = setTimeout(save, 100);
    }, { passive: true });
    sidebar.addEventListener('click', event => {
      if (event.target.closest('.nav a')) save();
    }, { capture: true });
    window.addEventListener('pagehide', save);
    // BFCache already retains the DOM position; only a newly shown desktop
    // sidebar needs restoring after using this document at a phone width.
    desktop.addEventListener('change', restore);
    return true;
  };
  if (!install()) {
    const observer = new MutationObserver(() => { if (install()) observer.disconnect(); });
    observer.observe(document.documentElement, { childList: true, subtree: true });
  }
})();

// HAUSV-697/701. Store only opaque context IDs per person; fetch all labels and
// permissions afresh. Storage denial/private mode degrades to the GET form.
document.addEventListener('DOMContentLoaded', () => {
  const read = key => { try { const value = JSON.parse(localStorage.getItem(key) || '[]'); return Array.isArray(value) ? value.filter(v => typeof v === 'string').slice(0, 6) : []; } catch { return []; } };
  const write = (key, value) => { if (!key) return; try { localStorage.setItem(key, JSON.stringify(value)); } catch {} };
  const remember = (key, entry) => { if (!key || !entry) return; const previous=read(key); if(previous[0]!==entry) write(key, [entry, ...previous.filter(v => v !== entry)].slice(0, 6)); };
  const positioners = [];
  let positionFrame;
  const positionOpenPanels = () => {
    cancelAnimationFrame(positionFrame);
    positionFrame = requestAnimationFrame(() => positioners.forEach(position => position()));
  };
  window.addEventListener('resize', positionOpenPanels);
  window.visualViewport?.addEventListener('resize', positionOpenPanels);
  window.visualViewport?.addEventListener('scroll', positionOpenPanels);
  document.addEventListener('scroll', event => {
    if (!event.target.closest?.('.switcher-panel')) positionOpenPanels();
  }, true);
  document.querySelectorAll('[data-switcher]').forEach(picker => {
    const summary = picker.querySelector(':scope > summary');
    const panel = picker.querySelector('.switcher-panel');
    const input = picker.querySelector('[data-switcher-search]');
    const results = picker.querySelector('[data-switcher-results]');
    const initial = picker.querySelector('[data-switcher-initial]');
    const status = picker.querySelector('[data-switcher-status]');
    const recent = picker.querySelector('[data-switcher-recent]');
    const recentSection = picker.querySelector('[data-switcher-recent-section]');
    const storage = picker.dataset.storageKey;
    const endpoint = picker.dataset.searchUrl;
    if (!input || !endpoint || !summary || !panel) return;
    // Top-layer painting escapes the sticky sidebar and the drawer's scroll
    // clip without moving the panel away from its native details/form owner.
    const topLayer = typeof panel.showPopover === 'function';
    if (topLayer) panel.setAttribute('popover', 'manual');
    const positionPanel = () => {
      if (!picker.open) return;
      const card = summary.getBoundingClientRect();
      if (!card.width || !card.height) { picker.open = false; return; }
      const viewport = window.visualViewport;
      const viewportTop = viewport?.offsetTop || 0;
      const viewportLeft = viewport?.offsetLeft || 0;
      const viewportWidth = viewport?.width || document.documentElement.clientWidth;
      const viewportBottom = viewportTop + (viewport?.height || innerHeight);
      const barHeight = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--context-bar-h'));
      const minTop = viewportTop + barHeight + 8;
      const width = Math.min(440, viewportWidth - 24);
      const besideCard = !!picker.closest('.sidebar');
      // Keep at least the heading/current/search area available when a card is
      // low in a short viewport; the rest scrolls inside the panel.
      const maxTop = Math.max(minTop, viewportBottom - 8 - Math.min(panel.scrollHeight, 320));
      const top = Math.max(minTop, Math.min(besideCard ? card.top : card.bottom + 8, maxTop));
      const left = Math.max(viewportLeft + 12, Math.min(besideCard ? card.right + 8 : card.left, viewportLeft + viewportWidth - width - 12));
      panel.style.setProperty('--switcher-top', `${top}px`);
      panel.style.setProperty('--switcher-left', `${left}px`);
      panel.style.setProperty('--switcher-width', `${width}px`);
      panel.style.setProperty('--switcher-max-height', `${Math.max(0, viewportBottom - top - 8)}px`);
    };
    positioners.push(positionPanel);
    new ResizeObserver(positionOpenPanels).observe(summary);
    const drawer = picker.closest('details.menu');
    drawer?.addEventListener('toggle', () => { if (!drawer.open) picker.open = false; });
    const current = picker.querySelector('[data-switcher-current]')?.dataset.switcherCurrent;
    remember(storage, current);
    let timer, request, generation = 0, active = -1, entries = [], options = [];
    const setActive = index => {
      active = index;
      options.forEach((option, i) => option.setAttribute('aria-selected', String(i === index)));
      if (options[index]) { input.setAttribute('aria-activedescendant', options[index].id); options[index].scrollIntoView({block: 'nearest'}); }
      else input.removeAttribute('aria-activedescendant');
    };
    const choose = entry => {
      if (entry.current) return;
      remember(storage, entry.key);
      const form = document.createElement('form');
      form.method = 'post';
      const target = new URL(endpoint, location.href);
      target.pathname = target.pathname.replace(/\/liegenschaften\/suche$/, '/context');
      target.search = ''; form.action = target.href;
      for (const [name, value] of Object.entries({tenant: entry.tenant, role: entry.role})) {
        const field = document.createElement('input'); field.type = 'hidden'; field.name = name; field.value = value; form.append(field);
      }
      document.body.append(form); form.requestSubmit();
    };
    const row = (entry, option, index) => {
      const element = document.createElement(option ? 'div' : 'button');
      element.className = 'switcher-option';
      if (option) { element.id = `${picker.id}-option-${index}`; element.setAttribute('role', 'option'); element.setAttribute('aria-selected', 'false'); element.setAttribute('aria-disabled', String(entry.current)); }
      else { element.type = 'button'; element.disabled = entry.current; }
      const copy = document.createElement('span'); copy.className = 'switcher-row-copy';
      const name = document.createElement('strong'); name.textContent = entry.name; name.title = entry.name;
      const address = document.createElement('small'); address.textContent = entry.address;
      const role = document.createElement('small'); role.textContent = `${entry.group} · ${entry.role}`;
      copy.append(name, address, role);
      const badge = document.createElement('span'); badge.className = 'switcher-row-badge'; badge.textContent = `${entry.open} offen`;
      const mark = document.createElement('span'); mark.className = 'switcher-row-mark nav-icon' + (entry.current ? ' is-current' : '');
      if (entry.current) { mark.title = 'aktuell'; mark.setAttribute('aria-label', 'aktuell'); } else mark.setAttribute('aria-hidden', 'true');
      const icon = picker.querySelector(entry.current ? '[data-switcher-check-icon]' : '[data-switcher-forward-icon]');
      if (icon) mark.append(icon.content.cloneNode(true));
      element.append(copy, badge, mark); element.addEventListener('click', () => choose(entry));
      return element;
    };
    async function loadRecent() {
      const keys = read(storage); if (!keys.length) return;
      const url = new URL(endpoint, location.href); url.searchParams.set('format', 'json'); keys.forEach(key => url.searchParams.append('recent', key));
      try {
        const response = await fetch(url, {headers:{Accept:'application/json'}}); if (!response.ok) return;
        const data = await response.json(); recent.replaceChildren(...data.entries.slice(0,6).map((entry,i) => row(entry,false,i))); recentSection.hidden = !data.entries.length || !!input.value.trim();
      } catch { /* Search and server links remain available. */ }
    }
    async function search() {
      const ownGeneration = ++generation;
      request?.abort(); request = new AbortController();
      entries = []; options = []; results.replaceChildren(); setActive(-1);
      if (!input.value.trim()) { initial.hidden=false; status.textContent=''; input.setAttribute('aria-expanded','false'); await loadRecent(); return; }
      initial.hidden=true; recentSection.hidden=true; status.textContent='Suche läuft …';
      const url = new URL(endpoint, location.href); url.searchParams.set('q',input.value.trim()); url.searchParams.set('format','json');
      try {
        const response = await fetch(url,{signal:request.signal,headers:{Accept:'application/json'}});
        if (!response.ok) throw new Error('search');
        const data = await response.json(); if (ownGeneration !== generation) return;
        entries = data.entries.slice(0,8); options=entries.map((entry,i) => row(entry,true,i)); results.replaceChildren(...options);
        input.setAttribute('aria-expanded',String(options.length>0));
        status.textContent=entries.length ? `${entries.length} Treffer${entries.length===8 ? ' · Weitere über Suchen' : ''}` : 'Keine Liegenschaft gefunden.';
      } catch (error) { if (error.name!=='AbortError' && ownGeneration===generation) { status.textContent='Suche nicht erreichbar. Mit „Suchen“ die Ergebnisseite öffnen.'; input.setAttribute('aria-expanded','false'); } }
    }
    input.addEventListener('input',() => { clearTimeout(timer); ++generation; request?.abort(); entries=[]; options=[]; results.replaceChildren(); setActive(-1); input.setAttribute('aria-expanded','false'); timer=setTimeout(search,220); });
    input.addEventListener('keydown',event => {
      if (event.key==='ArrowDown' || event.key==='ArrowUp') {
        if (!options.length) return; event.preventDefault(); const step=event.key==='ArrowDown'?1:-1;
        setActive(active<0?(step>0?0:options.length-1):(active+step+options.length)%options.length);
      } else if (event.key==='Enter' && active>=0) { event.preventDefault(); choose(entries[active]); }
    });
    picker.addEventListener('keydown',event => { if (event.key==='Escape') { event.preventDefault(); event.stopPropagation(); picker.open=false; picker.querySelector(':scope > summary').focus(); } });
    picker.addEventListener('toggle',() => {
      if (!picker.open) { if (topLayer && panel.matches(':popover-open')) panel.hidePopover(); clearTimeout(timer); ++generation; request?.abort(); input.setAttribute('aria-expanded','false'); return; }
      document.querySelectorAll('[data-switcher][open]').forEach(other => { if (other!==picker) other.open=false; });
      positionPanel();
      if (topLayer && !panel.matches(':popover-open')) panel.showPopover();
      positionPanel();
      input.focus({preventScroll:true}); panel.scrollTop = 0;
      if (input.value.trim()) search(); else loadRecent();
    });
    picker.addEventListener('submit',event => { const key=event.target.dataset.switcherEntry; if (key) remember(storage,key); });
  });
  document.addEventListener('click',event => {
    document.querySelectorAll('[data-switcher][open], [data-context-account][open]').forEach(detail => { if (!detail.contains(event.target)) detail.open=false; });
  });
  document.querySelectorAll('[data-context-account]').forEach(account => account.addEventListener('keydown',event => { if(event.key==='Escape'){account.open=false;account.querySelector('summary').focus();} }));

  const widthKey='hausv:sidebar-width';
  const handles=[...document.querySelectorAll('[data-sidebar-resize]')];
  const setWidth = value => {
    if (!Number.isFinite(value)) return;
    const width=Math.round(Math.max(240,Math.min(420,value)));
    document.documentElement.style.setProperty('--sidebar-w',`${width}px`);
    handles.forEach(handle => handle.setAttribute('aria-valuenow',String(width)));
    try { localStorage.setItem(widthKey,String(width)); } catch {}
    document.cookie=`hausv-sidebar-w=${width}; Path=/; Max-Age=31536000; SameSite=Lax${location.protocol==='https:'?'; Secure':''}`;
    positionOpenPanels();
  };
  // A cookie-backed document already has its final width. Only migrate the
  // legacy localStorage preference once; never replay stale storage over HTML.
  const cookieWidth = Number(document.cookie.match(/(?:^|;\s*)hausv-sidebar-w=(\d+)(?:;|$)/)?.[1]);
  const initialWidth = parseFloat(document.documentElement.style.getPropertyValue('--sidebar-w')) || 280;
  handles.forEach(handle => handle.setAttribute('aria-valuenow', String(initialWidth)));
  if (!(cookieWidth >= 240 && cookieWidth <= 420)) {
    try { const saved=Number(localStorage.getItem(widthKey)); if (saved>=240 && saved<=420) setWidth(saved); } catch {}
  }
  handles.forEach(handle => {
    let dragging=false;
    // Track the drag on the window: the handle is 10 px wide, so the pointer
    // leaves it immediately, and pointer capture is not available everywhere.
    handle.addEventListener('pointerdown',event => { if (event.button!==0) return; event.preventDefault(); dragging=true; try { handle.setPointerCapture(event.pointerId); } catch {} });
    window.addEventListener('pointermove',event => { if(dragging)setWidth(event.clientX); });
    window.addEventListener('pointerup',() => { dragging=false; });
    window.addEventListener('pointercancel',() => { dragging=false; });
    handle.addEventListener('dblclick',() => setWidth(280));
    handle.addEventListener('keydown',event => { if (event.key==='ArrowLeft'||event.key==='ArrowRight') { event.preventDefault(); setWidth(Number(handle.getAttribute('aria-valuenow'))+(event.key==='ArrowRight'?10:-10)); } else if (event.key==='Home') { event.preventDefault(); setWidth(280); } });
  });
}, { once: true });
