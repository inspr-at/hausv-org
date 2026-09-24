#!/usr/bin/env node
// HAUSV-714/716/717. Disposable local fixtures; five lanes and an async detail panel.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
const out = process.argv[3];
if (!baseURL) throw new Error('usage: qa-issue-board.mjs <baseURL> [artifactDir]');
const origin = new URL(baseURL).origin;
assert(['localhost', '127.0.0.1'].includes(new URL(baseURL).hostname), 'Use the local disposable snapshot server');
const tenant = (process.env.DEFAULT_TENANT || 'demo').replace(/^\/+|\/+$/g, '');
const tenantURL = `${baseURL}/${tenant}`;
const boardURL = `${tenantURL}/app/anliegen/board`;
if (out) mkdirSync(out, { recursive: true });
const executablePath = [process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  ...(process.env.CI === 'true' ? [] : ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Chromium.app/Contents/MacOS/Chromium', '/usr/bin/chromium', '/usr/bin/chromium-browser']),
].filter(Boolean).find(existsSync);
const browser = await chromium.launch({ headless: true, ...(executablePath ? { executablePath } : {}) });
const errors = [];
const measurements = [];
let page;
const lane = status => page.locator(`[data-board-status=${JSON.stringify(status)}]`);
const cardByID = id => page.locator(`[id=${JSON.stringify(id)}][data-board-card]`);

async function assertStatus(id, status, reload = false) {
  if (reload) await page.reload({ waitUntil: 'networkidle' });
  await page.waitForFunction(({ id, status }) => {
    const card = document.getElementById(id);
    return card?.dataset.issueStatus === status && card.closest('[data-board-status]')?.dataset.boardStatus === status &&
      !card.hasAttribute('aria-busy') && !card.classList.contains('is-pending');
  }, { id, status });
  await assertCounts();
}

async function assertCounts() {
  const counts = await page.locator('[data-board-status]').evaluateAll(columns => columns.map(column => ({
    status: column.dataset.boardStatus,
    actual: column.querySelectorAll('[data-board-card]').length,
    count: Number(column.querySelector('[data-board-count]').textContent),
    emptyHidden: column.querySelector('.board-column-empty').hidden,
  })));
  for (const c of counts) {
    assert.equal(c.count, c.actual, `${c.status}: count after move`);
    assert.equal(c.emptyHidden, c.actual > 0, `${c.status}: empty state after move`);
  }
}

const panel = () => page.locator('[data-board-panel]');
async function openPanel(id, keyboard = false) {
  const card = cardByID(id);
  if (keyboard) { await card.focus(); await page.keyboard.press('Enter'); }
  else await card.locator('h3').click();
  await panel().locator('[data-board-move]').waitFor();
  assert.equal(await card.getAttribute('aria-current'), 'true');
}
async function keyboardMove(id, storedStatus) {
  await openPanel(id, true);
  const select = panel().locator('[data-board-move] select[name="status"]');
  await select.focus();
  await select.selectOption(storedStatus);
  await page.keyboard.press('Tab');
  assert(await panel().locator('[data-board-move] button[type="submit"]').evaluate(el => el === document.activeElement), 'Tab reaches Verschieben');
  await page.keyboard.press('Enter');
  await assertStatus(id, storedStatus);
  await panel().locator('[data-board-move]').waitFor();
  await page.keyboard.press('Escape');
  assert(await cardByID(id).evaluate(el => el === document.activeElement), 'Esc returns focus to the card');
}
async function dragTo(id, status) {
  await page.evaluate(([cardID, status]) => {
    const card = document.getElementById(cardID);
    const target = document.querySelector(`[data-board-status="${status}"] .board-column-head`);
    const dt = new DataTransfer();
    const fire = (node, type) => node.dispatchEvent(new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer: dt }));
    fire(card, 'dragstart'); fire(target, 'dragenter'); fire(target, 'dragover'); fire(target, 'drop'); fire(card, 'dragend');
  }, [id, status]);
}
async function transitionDialog(id, status, heading) {
  const dialog = panel().locator('[data-board-transition]');
  await dialog.locator('h3').waitFor();
  assert.equal(await dialog.locator('h3').textContent(), heading);
  assert.equal(await cardByID(id).evaluate(el => el.closest('[data-board-status]').dataset.boardStatus), status);
  assert(await cardByID(id).evaluate(el => el.classList.contains('is-pending')), 'card waits visibly in target lane');
  await assertCounts();
  return dialog;
}

try {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'de-AT', timezoneId: 'Europe/Vienna', hasTouch: true });
  page = await context.newPage();
  page.on('pageerror', error => errors.push(error.message));
  await page.goto(`${tenantURL}/`, { waitUntil: 'networkidle' });
  await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(items => items.forEach(item => { item.open = true; }));
  await page.locator('input[name="email"]').fill('admin@example.com');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor();
  const login = new URL(await devLink.getAttribute('href'), `${tenantURL}/`);
  login.protocol = new URL(baseURL).protocol;
  login.port = new URL(baseURL).port;
  await page.goto(login.href, { waitUntil: 'networkidle' });

  const created = await context.request.post(`${tenantURL}/app/anliegen`, {
    headers: { Origin: origin }, maxRedirects: 0,
    multipart: { title: 'QA HAUSV-714 Tür prüfen', body: 'Die Tür beim Keller schließt nicht richtig.', category: 'Reparatur', location_type: 'common', location_detail: 'Keller' },
  });
  assert.equal(created.status(), 303, 'Create the oracle issue');
  const location = new URL(created.headers().location, baseURL);
  assert.equal(location.searchParams.get('created'), '1', 'Fixture creation must succeed');
  const issueID = decodeURIComponent(location.pathname.split('/').at(-1));
  const id = `issue-${issueID}`;
  const second = await context.request.post(`${tenantURL}/app/anliegen`, {
    headers: { Origin: origin }, maxRedirects: 0,
    multipart: { title: 'QA Board Tastaturnavigation', body: 'Zweites Anliegen für J/K in Leserichtung.', category: 'Frage', location_type: 'common', location_detail: 'Stiegenhaus' },
  });
  assert.equal(new URL(second.headers().location, baseURL).searchParams.get('created'), '1', 'Second keyboard fixture must exist');
  await page.goto(boardURL, { waitUntil: 'networkidle' });
  await assertStatus(id, 'Neu');
  // Expanded filters are part of the width/tap-target contract, too.
  await page.locator('.board-tools').evaluate(el => { el.open = true; });

  for (const width of [1280, 1440, 1920, 320, 390, 760, 768, 1024]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.evaluate(() => document.fonts.ready);
    const m = await page.evaluate(() => {
      const visible = el => el && el.getClientRects().length > 0 && getComputedStyle(el).visibility !== 'hidden';
      const box = el => { const b = el.getBoundingClientRect(); return { left: b.left, right: b.right, top: b.top, width: b.width, height: b.height }; };
      const grid = document.querySelector('.board-columns');
      const content = document.querySelector('.portal-section-content');
      const main = document.querySelector('.issue-board-main');
      const columns = [...grid.querySelectorAll('[data-board-status]')].map(el => ({ status: el.dataset.boardStatus, ...box(el) }));
      const firstRow = columns.filter(c => Math.abs(c.top - columns[0].top) < 1);
      const gap = parseFloat(getComputedStyle(grid).columnGap);
      const headerButtons = [...main.querySelectorAll('.portal-section-header-action .button')];
      const targets = [...main.querySelectorAll('a,button,select,input:not([type="hidden"]),summary')].filter(visible);
      const tiny = targets.filter(el => { const b = el.getBoundingClientRect(); return b.width < 43.5 || b.height < 43.5; }).map(el => el.outerHTML.slice(0, 160));
      const offscreen = [...main.querySelectorAll('*')].filter(visible).filter(el => {
        if (el.classList.contains('sr-only') || el.closest('.board-columns')) return false;
        const b = el.getBoundingClientRect();
        return b.left < -1 || b.right > innerWidth + 1;
      }).map(el => el.tagName + '.' + el.className);
      const navs = [document.querySelector('.sidebar'), document.querySelector('[data-context-bar]')].filter(visible);
      const mobile = document.querySelector('[data-context-bar]');
      return { width: innerWidth, documentWidth: document.documentElement.scrollWidth, main: box(main), content: box(content),
        contentPadding: parseFloat(getComputedStyle(content).paddingLeft), header: box(main.querySelector('.portal-section-header')),
        toolbar: box(main.querySelector('.board-toolbar')), grid: box(grid), gridClient: grid.clientWidth, gridScroll: grid.scrollWidth,
        columns, firstRowSum: firstRow.reduce((sum, c) => sum + c.width, 0) + (firstRow.length - 1) * gap,
        contextHeight: main.querySelector('.portal-section-context').getBoundingClientRect().height,
        navigationCount: navs.length, mobileHeight: visible(mobile) ? mobile.getBoundingClientRect().height : null,
        tiny, offscreen, filledHeaderActions: headerButtons.filter(el => el.classList.contains('primary') || !['transparent', 'rgba(0, 0, 0, 0)'].includes(getComputedStyle(el).backgroundColor)).length,
      };
    });
    measurements.push(m);
    assert.equal(m.columns.length, 5, `${width}px: five main lanes remain present`);
    assert(m.documentWidth <= width + 1, `${width}px: no page overflow`);
    assert(m.gridScroll <= m.gridClient + 1, `${width}px: lanes fit without a sideways strip`);
    const widths = m.columns.map(c => c.width);
    assert(Math.max(...widths) - Math.min(...widths) <= 2, `${width}px: equal column widths`);
    if (width < 900) {
      for (let i = 1; i < m.columns.length; i++) {
        assert(m.columns[i].top >= m.columns[i - 1].top + m.columns[i - 1].height - 1, `${width}px: one status column per row`);
      }
    }
    if (width >= 900) assert(Math.abs(m.firstRowSum - (m.grid.width - 4)) <= 2, `${width}px: five lanes plus gaps fill content`);
    assert(Math.abs(m.grid.width - (m.content.width - 2 * m.contentPadding)) <= 2, `${width}px: grid uses full content`);
    assert(Math.abs(m.toolbar.width - m.grid.width) <= 2, `${width}px: filter toolbar uses full width`);
    assert(Math.abs(m.header.width - m.main.width) <= 2 && Math.abs(m.content.width - m.main.width) <= 2,
      `${width}px: header/content use viewport minus navigation`);
    if (width >= 1280) assert.equal(m.contentPadding, 32, `${width}px: portfolio padding`);
    // Desktop shows the sidebar and the context bar together. Phones keep only the mobile bar.
    const phone = width <= 760;
    assert.equal(m.navigationCount, phone ? 1 : 2, `${width}px: shell navigation`);
    if (phone) assert(m.mobileHeight !== null && m.mobileHeight >= 74, `${width}px: mobile header ${m.mobileHeight}px`);
    assert(Math.abs(m.contextHeight - 40) <= 1, `${width}px: context bar 40px`);
    assert.equal(m.filledHeaderActions, 0, `${width}px: ghost header actions`);
    assert.deepEqual(m.tiny, [], `${width}px: targets >= 44px`);
    assert.deepEqual(m.offscreen, [], `${width}px: no offscreen content`);
    if (out && [1280, 1440, 1920, 390].includes(width)) await page.screenshot({ path: join(out, `board-${width}.png`), fullPage: true });
  }

  await page.locator('.board-tools').evaluate(el => { el.open = false; });
  await page.setViewportSize({ width: 1920, height: 1100 });
  await page.evaluate(() => window.scrollTo(0, 0));
  // Click, Esc, J/K and arrows choose cards in DOM/reading order.
  await openPanel(id);
  assert.match(await panel().locator('h2').textContent(), /QA HAUSV-714/);
  await page.keyboard.press('Escape');
  assert(await panel().isHidden());
  assert(await cardByID(id).evaluate(el => el === document.activeElement));
  const ordered = await page.locator('[data-board-card]').evaluateAll(items => items.filter(el => !el.hidden).map(el => el.id));
  assert(ordered.length >= 2, 'J/K requires at least two cards');
  {
    await openPanel(ordered[0], true);
    await page.keyboard.press('j');
    await page.waitForFunction(id => document.getElementById(id)?.getAttribute('aria-current') === 'true', ordered[1]);
    await panel().locator('[data-board-move]').waitFor();
    await page.keyboard.press('k');
    await page.waitForFunction(id => document.getElementById(id)?.getAttribute('aria-current') === 'true', ordered[0]);
    await panel().locator('[data-board-move]').waitFor();
    await page.keyboard.press('ArrowDown');
    await page.waitForFunction(id => document.getElementById(id)?.getAttribute('aria-current') === 'true', ordered[1]);
    await panel().locator('[data-board-move]').waitFor();
    await page.keyboard.press('ArrowUp');
    await page.waitForFunction(id => document.getElementById(id)?.getAttribute('aria-current') === 'true', ordered[0]);
    await panel().locator('[data-board-move]').waitFor();
  }
  await page.keyboard.press('Escape');
  const originalOrder = await lane('Neu').locator('[data-board-card]').evaluateAll(items => items.map(el => el.id));
  await dragTo(id, 'Angenommen');
  let dialog = await transitionDialog(id, 'Angenommen', 'Wer übernimmt?');
  assert.equal(await dialog.locator('select[name="assignee_email"]').inputValue(), 'admin@example.com', 'Ich übernehme is preselected');
  await dialog.locator('[data-board-cancel]').click();
  assert.deepEqual(await lane('Neu').locator('[data-board-card]').evaluateAll(items => items.map(el => el.id)), originalOrder, 'Cancel restores exact order');
  await assertStatus(id, 'Neu');
  await dragTo(id, 'Angenommen');
  dialog = await transitionDialog(id, 'Angenommen', 'Wer übernimmt?');
  // Delay the real POST to prove the pending state and atomic assignment/status.
  let release;
  const gate = new Promise(resolve => { release = resolve; });
  const delayedPost = async route => { await gate; await route.continue(); };
  await page.route('**/app/anliegen/workflow', delayedPost);
  try {
    await dialog.locator('button[type="submit"]').click();
    await page.waitForFunction(id => document.getElementById(id)?.getAttribute('aria-busy') === 'true', id);
  } finally { release(); }
  await assertStatus(id, 'Angenommen');
  await page.unroute('**/app/anliegen/workflow', delayedPost);
  await page.waitForFunction(() => document.querySelector('[data-board-feedback]').textContent.includes('Verschoben nach Angenommen · Zuständig:'));
  assert.match(await cardByID(id).locator('.issue-assignee-avatar').textContent(), /[A-ZÄÖÜ]/);
  assert.equal(await cardByID(id).getAttribute('data-assignee'), 'admin@example.com');
  assert.equal(await panel().locator('[data-board-panel-status]').textContent(), 'Angenommen');
  assert.match(await panel().locator('.board-history').textContent(), /Neu → Angenommen/);
  await assertStatus(id, 'Angenommen', true);

  await dragTo(id, 'Termin vereinbart');
  dialog = await transitionDialog(id, 'Termin vereinbart', 'Wann?');
  await dialog.locator('[name="service_start"]').fill('2026-09-10T10:00');
  await dialog.locator('[name="service_end"]').fill('2026-09-10T09:00');
  await dialog.locator('button[type="submit"]').click();
  await page.waitForFunction(() => document.querySelector('[data-board-feedback][data-error="true"]')?.textContent.includes('Ende'));
  await assertStatus(id, 'Angenommen');
  assert.match(await panel().locator('[data-board-panel-feedback]').textContent(), /Ende/);
  await dragTo(id, 'Termin vereinbart');
  dialog = await transitionDialog(id, 'Termin vereinbart', 'Wann?');
  await dialog.locator('[name="service_start"]').fill('2026-09-10T10:00');
  await dialog.locator('[name="service_end"]').fill('2026-09-10T11:00');
  await dialog.locator('button[type="submit"]').click();
  await assertStatus(id, 'Termin vereinbart');
  await assertStatus(id, 'Termin vereinbart', true);
  await dragTo(id, 'Neu');
  await assertStatus(id, 'Neu');
  assert(await panel().locator('[data-board-transition]').isHidden(), 'Backwards move to Neu needs no dialog');
  await assertStatus(id, 'Neu', true);
  await keyboardMove(id, 'In Bearbeitung');
  await assertStatus(id, 'In Bearbeitung', true);

  const abortPost = route => route.abort('failed');
  await page.route('**/app/anliegen/workflow', abortPost);
  await dragTo(id, 'Neu');
  await page.waitForFunction(() => document.querySelector('[data-board-feedback][data-error="true"]')?.textContent.includes('zurückgesetzt'));
  await assertStatus(id, 'In Bearbeitung');
  assert.equal(await panel().locator('[data-board-move] select[name="status"]').inputValue(), 'In Bearbeitung');
  assert.match(await panel().locator('[data-board-panel-feedback]').textContent(), /zurückgesetzt/);
  await page.unroute('**/app/anliegen/workflow', abortPost);
  await assertStatus(id, 'In Bearbeitung', true);

  // Panel fits desktop, and becomes a viewport sheet on mobile. Page never scrolls sideways.
  await openPanel(id);
  for (const width of [1280, 1024, 768, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    const m = await page.evaluate(() => {
      const panel = document.querySelector('[data-board-panel]');
      const b = panel.getBoundingClientRect();
      const targets = [...panel.querySelectorAll('a,button,input:not([type="hidden"]),select')].filter(el => el.getClientRects().length);
      return { width: innerWidth, documentWidth: document.documentElement.scrollWidth, panel: { width:b.width, height:b.height, left:b.left },
        tiny: targets.filter(el => { const b=el.getBoundingClientRect(); return b.width<43.5 || b.height<43.5; }).map(el=>el.outerHTML.slice(0,160)) };
    });
    assert(m.documentWidth <= width + 1, `${width}px: open panel has no page overflow`);
    assert(Math.abs(m.panel.width - (width < 760 ? width : 360)) < 2, `${width}px: panel width`);
    if (width < 760) assert(Math.abs(m.panel.height - 1000) < 2, 'mobile panel fills viewport');
    assert.deepEqual(m.tiny, [], `${width}px: panel targets >=44px`);
  }
  await page.keyboard.press('Escape');
  await page.setViewportSize({ width: 1920, height: 1100 });

  // Actual browser touch events exercise pointer capture/cancellation and the
  // fallback without constructing synthetic DragEvents or bypassing handlers.
  await keyboardMove(id, 'Neu');
  await page.evaluate(() => window.scrollTo(0, 0));
  const touch = await context.newCDPSession(page);
  const handle = await cardByID(id).locator('[data-board-drag]').boundingBox();
  const destination = await lane('Angenommen').locator('.board-column-head').boundingBox();
  assert(handle && destination, 'Touch source and destination exist');
  const from = { x: handle.x + handle.width / 2, y: handle.y + handle.height / 2 };
  const to = { x: destination.x + destination.width / 2, y: destination.y + destination.height / 2 };
  await touch.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [from] });
  await touch.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: from.x + 20, y: from.y + 20 }] });
  await page.locator('.board-touch-preview').waitFor();
  await touch.send('Input.dispatchTouchEvent', { type: 'touchCancel', touchPoints: [] });
  await page.locator('.board-touch-preview').waitFor({ state: 'detached' });
  await assertStatus(id, 'Neu');
  await touch.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [from] });
  for (let step = 1; step <= 8; step++) {
    await touch.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: from.x + (to.x - from.x) * step / 8, y: from.y + (to.y - from.y) * step / 8 }] });
  }
  await touch.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  await assertStatus(id, 'Angenommen');
  await assertStatus(id, 'Angenommen', true);
  await touch.detach();

  const noJS = await browser.newContext({ javaScriptEnabled: false, storageState: await context.storageState(), viewport: { width: 1440, height: 1000 } });
  const plain = await noJS.newPage();
  await plain.goto(boardURL);
  const plainCard = plain.locator(`[id=${JSON.stringify(id)}]`);
  await plainCard.locator('.board-card-fallback').click();
  await plain.locator('[data-board-move] select[name="status"]').selectOption('In Bearbeitung');
  await Promise.all([
    plain.waitForURL(url => url.searchParams.get('issue') === 'updated'),
    plain.locator('[data-board-move] button[type="submit"]').click(),
  ]);
  assert.equal(await plainCard.getAttribute('data-issue-status'), 'In Bearbeitung', 'No-JS form persists the move');
  await noJS.close();
  await assertStatus(id, 'In Bearbeitung', true);
  assert.deepEqual(errors, [], 'No uncaught browser errors');
  console.log('PASS — compact board/chrome, panel + keyboard/focus, pending dialogs + cancellation, atomic assignment/appointment, network rollback, touch, no-JS');
} catch (error) {
  if (out && page) await page.screenshot({ path: join(out, 'failure.png'), fullPage: true }).catch(() => {});
  throw error;
} finally {
  if (out) writeFileSync(join(out, 'measurements.json'), JSON.stringify({ measurements, errors }, null, 2));
  await browser.close();
}
