#!/usr/bin/env node
// HAUSV-706. Run on disposable snapshot data, never against production:
// HV_CAPTURE=qa-issue-board.mjs scripts/snapshot/run.sh WORKTREE /private/tmp/hausv-706-board 8106
// Seven 260px lanes wrap into rows at laptop widths. The width sum therefore
// includes the gaps of a complete row; all seven lanes remain present/visible.
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
      !card.hasAttribute('aria-busy');
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

async function keyboardMove(id, storedStatus) {
  const card = cardByID(id);
  const summary = card.locator('.board-move-menu > summary');
  await summary.focus();
  await page.keyboard.press('Enter');
  await page.keyboard.press('Tab');
  const select = card.locator('select[name="status"]');
  assert(await select.evaluate(el => el === document.activeElement), 'Tab must reach the destination select');
  const options = await select.locator('option').evaluateAll(items => items.map(item => item.value));
  const index = options.indexOf(storedStatus);
  assert(index >= 0, `Missing destination ${storedStatus}`);
  // Headless Chromium does not commit ArrowDown on a closed <select>; choose the
  // option the way assistive tech does (change event) and continue by keyboard.
  await select.selectOption(storedStatus);
  await page.keyboard.press('Tab');
  assert(await card.locator('button[type="submit"]').evaluate(el => el === document.activeElement), 'Tab must reach Verschieben');
  await page.keyboard.press('Enter');
  await assertStatus(id, storedStatus);
  assert(await cardByID(id).locator('.board-move-menu > summary').evaluate(el => el === document.activeElement),
    'Focus must return to the moved card menu');
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
    multipart: { title: 'QA HAUSV-706 Tür prüfen', body: 'Die Tür beim Keller schließt nicht richtig.', category: 'Reparatur', location_type: 'common', location_detail: 'Keller' },
  });
  assert.equal(created.status(), 303, 'Create the oracle issue');
  const location = new URL(created.headers().location, baseURL);
  assert.equal(location.searchParams.get('created'), '1', 'Fixture creation must succeed');
  const issueID = decodeURIComponent(location.pathname.split('/').at(-1));
  const id = `issue-${issueID}`;
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
      const tiny = targets.filter(el => { const b = el.getBoundingClientRect(); return b.width < 39.5 || b.height < 39.5; }).map(el => el.outerHTML.slice(0, 160));
      const offscreen = [...main.querySelectorAll('*')].filter(visible).filter(el => {
        if (el.classList.contains('sr-only')) return false;
        const b = el.getBoundingClientRect();
        return b.left < -1 || b.right > innerWidth + 1;
      }).map(el => el.tagName + '.' + el.className);
      const navs = [document.querySelector('.sidebar'), document.querySelector('.mobile-head')].filter(visible);
      const mobile = document.querySelector('.mobile-head');
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
    assert.equal(m.columns.length, 7, `${width}px: all seven lanes remain present`);
    assert(m.documentWidth <= width + 1 && m.gridScroll <= m.gridClient + 1, `${width}px: no horizontal overflow`);
    const widths = m.columns.map(c => c.width);
    assert(Math.max(...widths) - Math.min(...widths) <= 2, `${width}px: equal column widths`);
    assert(Math.min(...widths) >= 259.5, `${width}px: columns at least 260px`);
    assert(Math.abs(m.firstRowSum - m.grid.width) <= 2, `${width}px: complete row plus gaps fills content`);
    assert(Math.abs(m.grid.width - (m.content.width - 2 * m.contentPadding)) <= 2, `${width}px: grid uses full content`);
    assert(Math.abs(m.toolbar.width - m.grid.width) <= 2, `${width}px: filter toolbar uses full width`);
    assert(Math.abs(m.header.width - m.main.width) <= 2 && Math.abs(m.content.width - m.main.width) <= 2,
      `${width}px: header/content use viewport minus navigation`);
    if (width >= 1280) assert.equal(m.contentPadding, 32, `${width}px: portfolio padding`);
    assert.equal(m.navigationCount, 1, `${width}px: exactly one navigation`);
    if (m.mobileHeight !== null) assert(Math.abs(m.mobileHeight - 74) <= 1, `${width}px: mobile header 74px`);
    assert(Math.abs(m.contextHeight - 40) <= 1, `${width}px: context bar 40px`);
    assert.equal(m.filledHeaderActions, 0, `${width}px: ghost header actions`);
    assert.deepEqual(m.tiny, [], `${width}px: targets >= 40px`);
    assert.deepEqual(m.offscreen, [], `${width}px: no offscreen content`);
    if (out && [1280, 1440, 1920, 390].includes(width)) await page.screenshot({ path: join(out, `board-${width}.png`), fullPage: true });
  }

  await page.locator('.board-tools').evaluate(el => { el.open = false; });
  await page.setViewportSize({ width: 1920, height: 1100 });
  await page.evaluate(() => window.scrollTo(0, 0));
  // Delay the real POST so the assertion proves optimistic movement, followed
  // by the real server response and a reload proving persistence.
  let release;
  const gate = new Promise(resolve => { release = resolve; });
  const delayedPost = async route => { await gate; await route.continue(); };
  await page.route('**/app/anliegen/workflow', delayedPost);
  try {
    // Playwright's dragTo does not drive the native HTML5 drag session in
    // headless Chromium; dispatch the same DragEvents the browser would fire.
    await page.evaluate(([cardID, status]) => {
      const card = document.getElementById(cardID);
      const target = document.querySelector(`[data-board-status="${status}"] .board-column-head`) || document.querySelector(`[data-board-status="${status}"]`);
      const dt = new DataTransfer();
      const fire = (node, type) => node.dispatchEvent(new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer: dt }));
      fire(card, 'dragstart'); fire(target, 'dragenter'); fire(target, 'dragover'); fire(target, 'drop'); fire(card, 'dragend');
    }, [id, 'Angenommen']);
    await page.waitForFunction(id => document.getElementById(id)?.getAttribute('aria-busy') === 'true' &&
      document.getElementById(id)?.closest('[data-board-status]')?.dataset.boardStatus === 'Angenommen', id);
  } finally { release(); }
  await assertStatus(id, 'Angenommen');
  assert.equal(await page.locator('[data-board-feedback]').textContent(), 'Verschoben nach Angenommen');
  await page.unroute('**/app/anliegen/workflow', delayedPost);
  await assertStatus(id, 'Angenommen', true);

  // The keyboard path is exercised on the way back to Neu (the lifecycle refuses
  // skipping „Termin vereinbart“ from Angenommen; that refusal is asserted below).
  await keyboardMove(id, 'Neu');
  await assertStatus(id, 'Neu', true);
  const dragTo = async status => page.evaluate(([cardID, target]) => {
    const card = document.getElementById(cardID);
    const lane = document.querySelector(`[data-board-status="${target}"] .board-column-head`) || document.querySelector(`[data-board-status="${target}"]`);
    const dt = new DataTransfer();
    const fire = (node, type) => node.dispatchEvent(new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer: dt }));
    fire(card, 'dragstart'); fire(lane, 'dragenter'); fire(lane, 'dragover'); fire(lane, 'drop'); fire(card, 'dragend');
  }, [id, status]);
  await dragTo('Angenommen');
  await assertStatus(id, 'Angenommen');
  await assertStatus(id, 'Angenommen', true);
  await dragTo('Termin vereinbart');
  await page.waitForFunction(() => document.querySelector('[data-board-feedback][data-error="true"]')?.textContent.includes('Termin'));
  await assertStatus(id, 'Angenommen');
  await assertStatus(id, 'Angenommen', true);

  // A failed network request must also restore lane, count, selected status and focus.
  const abortPost = route => route.abort('failed');
  await page.route('**/app/anliegen/workflow', abortPost);
  await dragTo('Neu');
  await page.waitForFunction(() => document.querySelector('[data-board-feedback][data-error="true"]')?.textContent.includes('zurückgesetzt'));
  await assertStatus(id, 'Angenommen');
  assert.equal(await cardByID(id).locator('select[name="status"]').inputValue(), 'Angenommen');
  await page.unroute('**/app/anliegen/workflow', abortPost);
  await assertStatus(id, 'Angenommen', true);

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
  await plainCard.locator('.board-move-menu > summary').click();
  await plainCard.locator('select[name="status"]').selectOption('In Bearbeitung');
  await Promise.all([
    plain.waitForURL(url => url.searchParams.get('issue') === 'updated'),
    plainCard.locator('button[type="submit"]').click(),
  ]);
  assert.equal(await plainCard.getAttribute('data-issue-status'), 'In Bearbeitung', 'No-JS form persists the move');
  await noJS.close();
  await assertStatus(id, 'In Bearbeitung', true);
  assert.deepEqual(errors, [], 'No uncaught browser errors');
  console.log('PASS — board widths/chrome, optimistic dragTo + persisted status, keyboard + focus, rejection + network rollback, touch + cancellation, no-JS');
} catch (error) {
  if (out && page) await page.screenshot({ path: join(out, 'failure.png'), fullPage: true }).catch(() => {});
  throw error;
} finally {
  if (out) writeFileSync(join(out, 'measurements.json'), JSON.stringify({ measurements, errors }, null, 2));
  await browser.close();
}
