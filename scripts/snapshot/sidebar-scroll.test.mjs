import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { runInNewContext } from 'node:vm';

const source = readFileSync(new URL('../../internal/web/assets/switcher.js', import.meta.url), 'utf8');

function documentSession(storage, { tenant = 'demo', person = 'person-a', desktop = true, pathname = '/demo/app', denyStorage = false, parsing = false } = {}) {
  const listeners = {}, pending = new Map(), writes = [], assignments = [];
  let top = 0, nextTimer = 1, parsed = !parsing, mutation, disconnected = false;
  const sidebar = {
    dataset: { tenantSlug: tenant }, style: { scrollBehavior: 'smooth' },
    get scrollTop() { return top; },
    set scrollTop(value) { top = value; assignments.push({ top: value, behavior: this.style.scrollBehavior }); },
    querySelector: selector => selector === '.nav' ? { dataset: { tenantSlug: tenant } }
      : selector === '.sidebar-release .release-trigger' ? (parsed ? {} : null)
      : { dataset: { storageKey: person } },
    addEventListener: (event, handler) => { listeners[event] = handler; },
  };
  const media = { matches: desktop, addEventListener: (_, handler) => { listeners.breakpoint = handler; } };
  runInNewContext(source, {
    document: { querySelector: () => sidebar, documentElement: {}, addEventListener() {} }, location: { pathname },
    MutationObserver: class {
      constructor(callback) { mutation = callback; }
      observe() {}
      disconnect() { disconnected = true; }
    },
    window: { addEventListener: (event, handler) => { listeners[event] = handler; } },
    sessionStorage: {
      getItem(key) { if (denyStorage) throw new Error('storage denied'); return storage.get(key) ?? null; },
      setItem(key, value) { if (denyStorage) throw new Error('storage denied'); storage.set(key, value); writes.push({ key, value }); },
    },
    matchMedia: () => media,
    setTimeout(handler) { const id = nextTimer++; pending.set(id, handler); return id; },
    clearTimeout(id) { pending.delete(id); },
  });
  return { sidebar, media, pending, writes, assignments,
    parseFooter() { parsed = true; mutation(); assert(disconnected, 'observer disconnects after the complete sidebar'); },
    parseChunk() { mutation(); },
    scroll(value) { top = value; listeners.scroll(); },
    click() { listeners.click({ target: { closest: () => true } }); },
    leave: () => listeners.pagehide(),
    resize(value) { media.matches = value; listeners.breakpoint(); },
    flush() { for (const handler of [...pending.values()]) handler(); },
  };
}

test('scroll saves are throttled and restore synchronously without smooth scrolling', () => {
  const storage = new Map(), first = documentSession(storage);
  first.scroll(100); first.scroll(200); first.scroll(300);
  assert.equal(first.pending.size, 1);
  assert.equal(first.writes.length, 0);
  first.flush();
  assert.equal(first.writes.length, 1);
  const next = documentSession(storage);
  assert.equal(next.sidebar.scrollTop, 300);
  assert.deepEqual(next.assignments, [{ top: 300, behavior: 'auto' }]);
  assert.equal(next.sidebar.style.scrollBehavior, 'smooth');
});

test('streamed markup restores only once the complete sidebar has been parsed', () => {
  const storage = new Map(), first = documentSession(storage);
  first.scroll(300); first.leave();
  const next = documentSession(storage, { parsing: true });
  next.parseChunk();
  assert.equal(next.assignments.length, 0, 'incomplete scrollHeight must not clamp the saved position');
  next.parseFooter();
  assert.deepEqual(next.assignments, [{ top: 300, behavior: 'auto' }]);
});

test('navigation and pagehide flush immediately, scoped to person and tenant', () => {
  const storage = new Map(), first = documentSession(storage);
  first.scroll(300); first.click();
  assert.equal(first.pending.size, 0);
  assert.equal(documentSession(storage).sidebar.scrollTop, 300);
  assert.equal(documentSession(storage, { tenant: 'other' }).sidebar.scrollTop, 0);
  assert.equal(documentSession(storage, { person: 'person-b' }).sidebar.scrollTop, 0);
  first.scroll(450); first.leave();
  assert.equal(documentSession(storage).sidebar.scrollTop, 450);
  assert.equal(first.pending.size, 0);
});

test('a hidden mobile aside never overwrites the desktop preference', () => {
  const storage = new Map(), first = documentSession(storage);
  first.scroll(300); first.leave();
  const mobile = documentSession(storage, { desktop: false });
  mobile.scroll(0); mobile.flush(); mobile.leave();
  assert.equal(mobile.writes.length, 0);
  mobile.resize(true);
  assert.equal(mobile.sidebar.scrollTop, 300);
});

test('tenant prefix fallback, invalid values and denied storage degrade safely', () => {
  const storage = new Map(), first = documentSession(storage, { tenant: '' });
  first.scroll(300); first.leave();
  assert.equal(documentSession(storage, { tenant: '' }).sidebar.scrollTop, 300);
  assert.equal(documentSession(storage, { tenant: '', pathname: '/other/app' }).sidebar.scrollTop, 0);
  const key = first.writes[0].key;
  for (const value of ['', ' ', '-1', 'NaN', 'Infinity', '{}']) {
    storage.set(key, value);
    assert.deepEqual(documentSession(storage, { tenant: '' }).assignments, []);
  }
  const denied = documentSession(storage, { denyStorage: true });
  denied.scroll(300); denied.flush(); denied.click(); denied.leave();
  assert.deepEqual(denied.assignments, []);
});
