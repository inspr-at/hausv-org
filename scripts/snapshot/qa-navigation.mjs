#!/usr/bin/env node
// HV_CAPTURE=qa-navigation.mjs scripts/snapshot/run.sh WORKTREE <out-dir> [port]
// HAUSV-704: click every actual menu entry, preserving the selected house.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const [baseURL, out = '/private/tmp/hausv-navigation-qa'] = process.argv.slice(2);
if (!baseURL) throw new Error('usage: qa-navigation.mjs <baseURL> [out-dir]');
mkdirSync(out, { recursive: true });
const executablePath = [process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH, ...(process.env.CI === 'true' ? [] : ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/Applications/Chromium.app/Contents/MacOS/Chromium', '/usr/bin/chromium'])].filter(Boolean).find(existsSync);
const browser = await chromium.launch({ headless: true, ...(executablePath ? { executablePath } : {}), args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'] });
const measurements = [], failures = [];
const expectedBlocks = ['organisation-identity', 'organisation', 'house-card', 'map', 'house-navigation', 'release'];

async function login(context) {
  const page = await context.newPage();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
  await page.locator('input[name="email"]').fill('admin@example.com');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const link = page.locator('a.dev-link'); await link.waitFor({ state: 'visible' });
  const target = new URL(await link.getAttribute('href'), baseURL), origin = new URL(baseURL);
  target.protocol = origin.protocol; target.port = origin.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  assert(new URL(page.url()).pathname.endsWith('/app'), 'login did not open the portal');
  return page;
}
async function navigation(page, width) {
  const menu = page.locator('.mobile-head details.menu');
  if (width <= 760 && await menu.getAttribute('open') === null) await menu.locator(':scope > summary').click();
  return page.locator('[data-navigation-surface]:visible');
}
async function measure(page, width, sidebarWidth, label) {
  await navigation(page, width);
  const result = await page.evaluate(() => {
    const visible = el => !!el?.getClientRects().length && getComputedStyle(el).visibility !== 'hidden' && getComputedStyle(el).display !== 'none';
    const surface = [...document.querySelectorAll('[data-navigation-surface]')].find(visible);
    const rect = surface.getBoundingClientRect(), style = getComputedStyle(surface);
    const blocks = [...surface.querySelectorAll('[data-navigation-block]')];
    const nav = surface.querySelector('nav[aria-label="Bereiche"]');
    const links = [...nav.querySelectorAll('.nav-organisation > a, .nav-house-items > a')];
    const bar = [...document.querySelectorAll('[data-context-bar]')].filter(visible);
    const header = document.querySelector('[data-portal-section-header]');
    const ghost = [...(header?.querySelectorAll('.button') || [])].filter(visible);
    const left = [...document.querySelectorAll('body *')].filter(el => visible(el) && !el.closest('.sr-only,.skip-link,.side-map-tiles') && el.getBoundingClientRect().left < -1).map(el => `${el.tagName}.${el.className}`);
    return {
      box: { x: rect.x, width: rect.width, paddingLeft: style.paddingLeft, paddingRight: style.paddingRight },
      blocks: blocks.map(el => ({ name: el.dataset.navigationBlock, visible: visible(el), className: el.dataset.navigationBlock === "map" ? "side-map-hero" : el.className })),
      links: links.map(el => ({ href: el.getAttribute('href'), label: el.querySelector('.nav-label')?.textContent.trim() })),
      active: links.filter(el => el.getAttribute('aria-current') === 'page').length,
      taps: [...links, surface.querySelector('.house-header-card'), surface.querySelector('.release-trigger')].map(el => ({ label: el?.textContent.trim(), height: el?.getBoundingClientRect().height })),
      navigationCount: [...document.querySelectorAll('nav[aria-label="Bereiche"]')].filter(visible).length,
      barCount: bar.length, barHeight: bar[0]?.getBoundingClientRect().height,
      headerCount: document.querySelectorAll('[data-portal-section-header]').length,
      h1Count: document.querySelectorAll('main h1').length, headerVisible: visible(header),
      identity: !!header?.querySelector('.portal-section-identity'),
      badActions: ghost.filter(el => { const s = getComputedStyle(el); return s.backgroundColor !== 'rgba(0, 0, 0, 0)' || parseFloat(s.borderTopWidth) < 1 || el.getBoundingClientRect().height < 40; }).map(el => el.textContent.trim()),
      overview: surface.querySelector('.house-header-copy strong')?.textContent.trim(),
      portfolioMap: !!surface.querySelector('.side-map-portfolio'),
      left, overflow: document.documentElement.scrollWidth > innerWidth + 1,
    };
  });
  measurements.push({ label, viewport: width, sidebarWidth, ...result });
  assert.equal(result.navigationCount, 1, `${label}: exactly one navigation`);
  assert.equal(result.barCount, 1, `${label}: exactly one context bar`);
  assert.equal(Math.round(result.barHeight), width <= 760 ? 74 : 40, `${label}: context height`);
  assert.deepEqual(result.blocks.map(b => b.name), expectedBlocks, `${label}: block order`);
  assert(result.blocks.every(b => b.visible), `${label}: all blocks visible`);
  // Mein Zuhause (Energie-Einrichtung, HAUSV-691) carries its own approved page head
  // instead of the shared PortalSectionHeader; every other route has exactly one.
  assert.equal(result.headerCount, label.startsWith('Mein Zuhause') ? 0 : 1, `${label}: one page header`);
  assert.equal(result.h1Count, 1, `${label}: one H1`);
  if (!label.startsWith('Mein Zuhause')) assert(result.headerVisible && result.identity, `${label}: visible header with identity`);
  assert.equal(result.active, 1, `${label}: one active menu entry`);
  assert.deepEqual(result.badActions, [], `${label}: header actions must be ghost buttons ≥40px`);
  assert(result.taps.every(t => t.height >= 40), `${label}: tap targets ${JSON.stringify(result.taps)}`);
  assert.deepEqual(result.left, [], `${label}: no element left of viewport`);
  assert(!result.overflow, `${label}: no sideways overflow`);
  assert.equal(result.box.paddingLeft, '18px', `${label}: shared padding`);
  assert.equal(result.box.paddingRight, '18px', `${label}: shared padding`);
  assert(Math.abs(result.box.x) <= 1, `${label}: sidebar starts at x=0`);
  assert(Math.abs(result.box.width - (width <= 760 ? width : sidebarWidth)) <= 1, `${label}: shared width`);
  if (new URL(page.url()).pathname.includes('/app/verwaltung')) {
    assert.match(result.overview, /^Alle Liegenschaften · \d+$/, `${label}: overview card`);
    assert(result.portfolioMap, `${label}: neutral portfolio map`);
  } else assert(!result.portfolioMap, `${label}: house map`);
  return result;
}
try {
  for (const [width, sidebarWidth] of [[1440, 280], [1024, 280], [390, 280], [1440, 240], [1440, 420]]) {
    const context = await browser.newContext({ viewport: { width, height: 900 }, locale: 'de-AT', timezoneId: 'Europe/Vienna' });
    try {
      const page = await login(context);
      // Set the width through the accessible keyboard control, so persistence is exercised.
      if (width > 760) {
        const handle = page.locator('[data-sidebar-resize]');
        await handle.dblclick();
        const key = sidebarWidth < 280 ? 'ArrowLeft' : 'ArrowRight';
        for (let i = 0; i < Math.abs(sidebarWidth - 280) / 10; i++) await handle.press(key);
      }
      const baseline = await measure(page, width, sidebarWidth, 'Hausüberblick');
      assert(baseline.links.length >= 15, 'admin fixture must expose the full navigation');
      for (const [index, item] of baseline.links.entries()) {
        const surface = await navigation(page, width);
        const link = surface.locator('.nav-organisation > a, .nav-house-items > a').nth(index);
        assert.equal(await link.getAttribute('href'), item.href, `${item.label}: menu target changed`);
        await link.click(); await page.waitForLoadState('networkidle');
        const result = await measure(page, width, sidebarWidth, item.label);
        assert.deepEqual(result.box, baseline.box, `${item.label}: geometry changed between routes`);
        assert.deepEqual(result.blocks, baseline.blocks, `${item.label}: block visibility/classes changed`);
        assert.deepEqual(result.links, baseline.links, `${item.label}: navigation links changed`);
      }
    } catch (error) { failures.push(`${width}/${sidebarWidth}: ${error.stack}`); }
    finally { await context.close(); }
  }
} finally {
  await browser.close();
  writeFileSync(join(out, 'navigation.json'), JSON.stringify({ measurements, failures }, null, 2));
}
for (const failure of failures) console.error(failure);
assert.equal(failures.length, 0, `${failures.length} navigation scenarios failed`);
console.log(`PASS: ${measurements.length} route/width measurements; shared shell, blocks, widths and headers`);
