#!/usr/bin/env node
// HAUSV-705: run through run.sh WORKTREE so every run owns fresh local data.
// HV_CAPTURE=qa-legacy-routes.mjs scripts/snapshot/run.sh WORKTREE <artifacts> [port]
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
const out = process.argv[3];
if (!baseURL || !out) throw new Error('usage: qa-legacy-routes.mjs <baseURL> <artifact-dir>');
const origin = new URL(baseURL).origin;
assert(['localhost', '127.0.0.1', '[::1]'].includes(new URL(origin).hostname), 'Oracle requires the isolated local snapshot server');
const tenantURL = `${origin}/demo`;
mkdirSync(out, { recursive: true });
const scriptsDir = dirname(fileURLToPath(import.meta.url));
const executablePath = [process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  ...(process.env.CI === 'true' ? [] : ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/usr/bin/chromium', '/usr/bin/google-chrome']),
].filter(Boolean).find(existsSync);
const browser = await chromium.launch({ headless: true, args: ['--no-proxy-server'], ...(executablePath ? { executablePath } : {}) });
const rows = [];
const errors = [];
const contexts = [];
let activePage;

async function login(email) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'de-AT', timezoneId: 'Europe/Vienna' });
  contexts.push(context);
  const page = await context.newPage();
  page.on('pageerror', error => errors.push(error.message));
  activePage = page;
  await page.goto(`${tenantURL}/`, { waitUntil: 'networkidle' });
  const details = page.locator('details:has(form[action$="/auth/request"])');
  if (await details.count()) await details.evaluate(node => { node.open = true; });
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  await page.locator('a.dev-link').waitFor({ state: 'visible' });
  const target = new URL(await page.locator('a.dev-link').getAttribute('href'), tenantURL);
  target.protocol = new URL(origin).protocol;
  target.hostname = new URL(origin).hostname;
  target.port = new URL(origin).port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  assert(page.url().includes('/app'), `${email}: login failed`);
  return { context, page };
}

async function post(context, path, data, multipart = false) {
  const response = await context.request.post(new URL(path, origin).href, {
    [multipart ? 'multipart' : 'form']: data, headers: { Origin: origin }, maxRedirects: 0,
  });
  assert.equal(response.status(), 303, `${path}: fixture POST`);
  return new URL(response.headers().location, origin).href;
}

async function capture(page, name, url) {
  activePage = page;
  for (const width of [1440, 1024, 390]) {
    await page.setViewportSize({ width, height: 900 });
    // Playwright yields no response for a same-document navigation; reload to get one.
    const response = (await page.goto(url, { waitUntil: 'networkidle' })) || (await page.reload({ waitUntil: 'networkidle' }));
    assert.equal(response.status(), 200, `${name} ${width}: GET`);
    assert.equal(new URL(page.url()).pathname, new URL(url, origin).pathname, `${name}: unexpected redirect`);
    await page.locator('[data-portal-section-header]').waitFor({ state: 'visible' });
    await page.evaluate(() => document.fonts.ready);
    const state = await page.evaluate(() => {
      const shown = node => {
        const box = node?.getBoundingClientRect();
        const css = node && getComputedStyle(node);
        return box && box.width > 0 && box.height > 0 && css.display !== 'none' && css.visibility !== 'hidden';
      };
      const box = node => { const b = node?.getBoundingClientRect(); return b && { x: b.x, y: b.y, width: b.width, height: b.height }; };
      const bar = [...document.querySelectorAll('[data-context-bar]')].filter(shown);
      const sidebar = document.querySelector('aside.sidebar');
      const mobile = document.querySelector('[data-context-bar]');
      const controls = [...document.querySelectorAll('a, button, input, select, textarea, summary')].filter(shown);
      // Match the existing chrome oracle: inspect visible controls and content,
      // excluding the intentionally clipped OSM tile images inside the map.
      // A closed energy-mode popover is parked off-canvas by the shared strip; it is not route content.
      const left = [...document.querySelectorAll('main *')].filter(shown).filter(node => !node.closest('.energy-mode-control:not([open])')).filter(node => node.getBoundingClientRect().left < -1);
      const actions = [...document.querySelectorAll('[data-portal-section-header] :is(a,button,summary)')].filter(shown);
      // The context bar's compact account/logout controls are covered by qa-switcher.mjs;
      // this oracle measures the ported route content and hidden (closed details) controls are not tap targets.
      const tap = controls.filter(shown).filter(node => !node.closest('[data-context-bar]') && node.matches('button,summary,.button') && node.getBoundingClientRect().height < 40);
      return {
        templ: document.body.hasAttribute('data-templ-legacy'),
        bars: bar.map(box), contextHeight: parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--context-bar-h')),
        sidebar: shown(sidebar) ? box(sidebar) : null, sidebarCount: document.querySelectorAll('aside.sidebar').length,
        mobile: !!shown(mobile), navigationCount: Number(!!shown(sidebar)) + Number(!!shown(mobile)),
        headings: document.querySelectorAll('main h1').length,
        scrollWidth: document.documentElement.scrollWidth, width: innerWidth,
        offscreen: controls.filter(node => { const b = node.getBoundingClientRect(); return b.left < -1 || b.right > innerWidth + 1; }).map(node => node.tagName + '.' + node.className),
        left: left.map(node => node.tagName + '.' + node.className),
        smallTargets: tap.map(node => node.tagName + '.' + node.className + ' ' + Math.round(node.getBoundingClientRect().height) + 'px'),
        nonghost: actions.filter(node => !node.classList.contains('ghost')).map(node => node.textContent.trim()),
      };
    });
    rows.push({ name, width, ...state });
    await page.screenshot({ path: join(out, `${name}-${width}.png`), fullPage: true });
    assert(state.templ, `${name}: templ renderer missing`);
    assert.equal(state.headings, 1, `${name}: exactly one title`);
    assert.equal(state.bars.length, 1, `${name} ${width}: one visible context bar`);
    assert(Math.abs(state.bars[0].y) <= 1, `${name}: context bar at viewport top`);
    const expectedHeight = width === 390 ? 74 : 40;
    assert.equal(state.contextHeight, expectedHeight, `${name}: --context-bar-h`);
    assert(Math.abs(state.bars[0].height - expectedHeight) <= 1, `${name}: context bar height`);
    assert.equal(state.sidebarCount, 1, `${name}: shared sidebar DOM`);
    assert.equal(state.navigationCount, 1, `${name}: exactly one navigation`);
    if (width > 760) assert(state.sidebar?.width > 0, `${name}: sidebar box`);
    assert(state.scrollWidth <= state.width + 1, `${name} ${width}: horizontal overflow`);
    assert.deepEqual(state.offscreen, [], `${name} ${width}: offscreen controls`);
    assert.deepEqual(state.left, [], `${name} ${width}: content left of -1px`);
    assert.deepEqual(state.smallTargets, [], `${name} ${width}: tap targets below 40px`);
    assert.deepEqual(state.nonghost, [], `${name}: header actions must be ghost buttons`);
    if (width === 390) {
      const menu = page.locator('[data-context-bar] > details.menu');
      await menu.locator(':scope > summary').click();
      assert(await menu.locator('.menu-panel nav[aria-label="Bereiche"]').isVisible(), `${name}: mobile navigation opens`);
      const panelBox = await menu.locator('.menu-panel').boundingBox();
      assert(panelBox && panelBox.x >= -1 && panelBox.x + panelBox.width <= width + 1, `${name}: mobile navigation box`);
      await menu.locator(':scope > summary').click();
    }
  }
}

try {
  const admin = await login('admin@example.com');
  const resident = await login('resident@example.com');
  const owner = await login('owner@example.com');
  // Claim a Home profile through the existing UI; no private fixture schema.
  await owner.page.goto(`${tenantURL}/app/zuhause/onboarding`, { waitUntil: 'networkidle' });
  await owner.page.getByRole('button', { name: 'Verstanden, weiter' }).click();
  await owner.page.waitForURL(/step=2/);
  await owner.page.locator('[name="household_name"]').fill('QA Legacy Zuhause');
  await owner.page.locator('[name="home_type"]').selectOption('house');
  await owner.page.getByRole('button', { name: 'Weiter zu den Verbrauchern' }).click();
  await owner.page.waitForURL(/step=3/);

  const issueURL = await post(resident.context, '/demo/app/anliegen', {
    title: 'QA Legacy Kellerlicht', body: 'Das Licht im Keller bleibt dunkel.', category: 'Reparatur', location_type: 'gemeinschaft', location_detail: 'Stiegenhaus',
  }, true);
  assert(/\/app\/anliegen\/[^/?]+/.test(new URL(issueURL).pathname), 'resident issue detail redirect');
  const issueID = new URL(issueURL).pathname.split('/').at(-1);
  await post(admin.context, '/demo/app/anliegen/comment', { id: issueID, body: 'Welches Licht ist betroffen?', message_type: 'question' });

  await admin.page.goto(`${tenantURL}/app/parking`, { waitUntil: 'networkidle' });
  const monthHref = await admin.page.locator('a[href*="/app/parking/month/"]').first().getAttribute('href');
  assert(monthHref, 'run.sh must seed the populated parking fixture');
  const monthURL = new URL(monthHref, origin).href;

  const routes = [
    [resident.page, 'issue-detail', issueURL],
    [admin.page, 'parking-settings', `${tenantURL}/app/parking/settings`],
    [admin.page, 'parking-charging', `${tenantURL}/app/parking/settings?section=charging`],
    [admin.page, 'parking-telegram', `${tenantURL}/app/parking/settings?section=telegram`],
    [admin.page, 'parking-month', monthURL],
    [admin.page, 'parking-access', `${tenantURL}/app/settings/parking-access`],
    [admin.page, 'modules', `${tenantURL}/app/settings/modules`],
    [admin.page, 'data-export', `${tenantURL}/app/settings/data-export`],
    [owner.page, 'energy-data', `${tenantURL}/app/settings/energy-data`],
    [admin.page, 'invoice-import', `${tenantURL}/app/dokumente/rechnungen/import`],
    [admin.page, 'payment-import', `${tenantURL}/app/settings/payments/import`],
  ];
  for (const route of routes) await capture(...route);

  await admin.page.goto(`${tenantURL}/app/settings/building?section=units`, { waitUntil: 'networkidle' });
  await admin.page.getByRole('link', { name: 'Einheit hinzufügen', exact: true }).click();
  const add = admin.page.locator('#unit-add');
  await add.locator('[name="label"]').fill('QA Legacy Top');
  await add.getByRole('button', { name: 'Einheit anlegen' }).click();
  await admin.page.waitForURL(/unit=saved/);
  const camt = readFileSync(join(scriptsDir, '../../internal/integrations/testdata/camt053-2019.xml'));
  const paymentURL = await post(admin.context, '/demo/app/settings/payments/import/preview', {
    period: '2026-07', camt_file: { name: 'bank.xml', mimeType: 'application/xml', buffer: camt },
  }, true);
  assert(new URL(paymentURL).searchParams.has('preview'), 'payment preview token missing');
  await capture(admin.page, 'payment-preview', paymentURL);

  const invoice = readFileSync(join(scriptsDir, '../../internal/integrations/testdata/ebinterface-6p0.xml'));
  const previewURL = await post(admin.context, '/demo/app/dokumente/rechnungen/import/preview', { invoice_file: { name: 'rechnung.xml', mimeType: 'application/xml', buffer: invoice } }, true);
  assert(new URL(previewURL).searchParams.has('preview'), 'invoice preview token missing');
  await capture(admin.page, 'invoice-preview', previewURL);
  const exportURL = await post(admin.context, '/demo/app/settings/data-export/preview', { source: 'parking-months' });
  assert(new URL(exportURL).searchParams.has('preview'), 'data export preview token missing');
  await capture(admin.page, 'data-export-preview', exportURL);
  assert.deepEqual(errors, [], 'browser runtime errors');
  console.log(`PASS — ${rows.length} route/width checks; screenshots: ${out}`);
} catch (error) {
  if (activePage) await activePage.screenshot({ path: join(out, 'failure.png'), fullPage: true }).catch(() => {});
  throw error;
} finally {
  writeFileSync(join(out, 'legacy-routes.json'), JSON.stringify({ rows, errors }, null, 2));
  for (const context of contexts) await context.close();
  await browser.close();
}
