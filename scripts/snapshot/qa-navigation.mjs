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
const measurements = [], footerMeasurements = [], scrollMeasurements = [], failures = [];
const expectedBlocks = ['map', 'organisation-identity', 'organisation', 'house-navigation', 'release'];
// The expanded geometry matrix must not exhaust the fixture's login limit.
const sessions = new Map();

async function login(context, email) {
  const page = await context.newPage();
  if (sessions.has(email)) {
    await context.addCookies(sessions.get(email));
    await page.goto(`${baseURL}/app`, { waitUntil: 'networkidle' });
    assert(new URL(page.url()).pathname.endsWith('/app'), 'cached login did not open the portal');
    return page;
  }
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const link = page.locator('a.dev-link'); await link.waitFor({ state: 'visible' });
  const target = new URL(await link.getAttribute('href'), baseURL), origin = new URL(baseURL);
  target.protocol = origin.protocol; target.port = origin.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  assert(new URL(page.url()).pathname.endsWith('/app'), 'login did not open the portal');
  sessions.set(email, await context.cookies());
  return page;
}
async function navigation(page, width) {
  const menu = page.locator('[data-context-bar] details.menu');
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
      organisation: nav.dataset.twoLevel === 'true',
      mapTopGap: surface.querySelector('.side-map-hero').getBoundingClientRect().top - rect.top + surface.scrollTop,
      navTopGap: nav.getBoundingClientRect().top - surface.querySelector('.side-map-hero').getBoundingClientRect().bottom - parseFloat(getComputedStyle(surface.querySelector('.side-map-hero')).marginBottom),
      houseLabelGap: parseFloat(getComputedStyle(nav.querySelector('.nav-house-label')).marginTop) + parseFloat(getComputedStyle(nav.querySelector('.nav-house-label')).paddingTop),
      blocks: blocks.map(el => ({ name: el.dataset.navigationBlock, visible: visible(el), className: el.dataset.navigationBlock === "map" ? "side-map-hero" : el.className })),
      links: links.map(el => ({ href: el.getAttribute('href'), label: el.querySelector('.nav-label')?.textContent.trim() })),
      active: links.filter(el => el.getAttribute('aria-current') === 'page').length,
      taps: [...links, document.querySelector('[data-context-bar] .house-header-card'), surface.querySelector('.release-trigger')].map(el => ({ label: el?.textContent.trim(), height: el?.getBoundingClientRect().height })),
      navigationCount: [...document.querySelectorAll('nav[aria-label="Bereiche"]')].filter(visible).length,
      barCount: bar.length, barHeight: bar[0]?.getBoundingClientRect().height, activeView: !!bar[0]?.querySelector('.context-view'),
      headerCount: document.querySelectorAll('[data-portal-section-header]').length,
      h1Count: document.querySelectorAll('main h1').length, headerVisible: visible(header),
      identity: !!header?.querySelector('.portal-section-identity'),
      // HAUSV-765: a header may carry exactly ONE filled main action (.button.primary);
      // every other header action stays an outline/ghost button. All are >=40px.
      filledActions: ghost.filter(el => el.classList.contains('primary')).map(el => el.textContent.trim()),
      badActions: ghost.filter(el => { const s = getComputedStyle(el); const filled = el.classList.contains('primary'); return (!filled && s.backgroundColor !== 'rgba(0, 0, 0, 0)') || parseFloat(s.borderTopWidth) < 1 || el.getBoundingClientRect().height < 40; }).map(el => el.textContent.trim()),
      calm: (() => {
        const calm = document.querySelector('.calm-column');
        if (!visible(calm)) return null;
        const content = calm.closest('.portal-section-content'), s = getComputedStyle(content);
        return { width: calm.getBoundingClientRect().width, available: content.clientWidth - parseFloat(s.paddingLeft) - parseFloat(s.paddingRight), columns: getComputedStyle(calm.querySelector('.disclosures')).gridTemplateColumns.split(' ').length, locationsWrap: [...calm.querySelectorAll('.issue-location')].every(el => getComputedStyle(el).whiteSpace === 'normal') };
      })(),
      overview: document.querySelector('[data-context-bar] .house-header-copy strong')?.textContent.trim(),
      portfolioMap: !!surface.querySelector('.side-map-portfolio'),
      layoutWidth:bar[0]?.getBoundingClientRect().width, left, overflow: document.documentElement.scrollWidth > innerWidth + 1,
    };
  });
  measurements.push({ label, viewport: width, sidebarWidth, ...result });
  if (width > 760 && result.calm) {
    assert(Math.abs(result.calm.width - result.calm.available) < 1, `${label}: summary fills available content width`);
    assert.equal(result.calm.columns, width >= 1100 && result.calm.width > 700 ? 2 : 1, `${label}: responsive summary columns`);
    assert(result.calm.locationsWrap, `${label}: long issue locations can wrap`);
  }
  assert.equal(result.navigationCount, 1, `${label}: exactly one navigation`);
  assert.equal(result.barCount, 1, `${label}: exactly one context bar`);
  // HAUSV-765: the second context row exists only while a role or support view is active.
  assert.equal(Math.round(result.barHeight), width <= 760 ? (result.activeView ? 148 : 108) : width <= 1100 && result.activeView ? 116 : 72, `${label}: context height`);
  assert.deepEqual(result.blocks.map(b => b.name), result.organisation ? expectedBlocks : ['map','house-navigation','release'], `${label}: block order`);
  assert(Math.abs(result.mapTopGap) < 1, `${label}: map starts flush at navigation surface edge`);
  // Without organisation context, navigation follows the map's normal bottom margin.
  if (!result.organisation && width > 760) {
    assert(Math.abs(result.navTopGap) < 1, `${label}: no empty organisation header gap`);
    assert.equal(result.houseLabelGap, 0, `${label}: house label starts at usual surface padding`);
  }
  assert(result.blocks.every(b => b.visible), `${label}: all blocks visible`);
  // Mein Zuhause (Energie-Einrichtung, HAUSV-691) carries its own approved page head
  // instead of the shared PortalSectionHeader; every other route has exactly one.
  assert.equal(result.headerCount, label.startsWith('Mein Zuhause') ? 0 : 1, `${label}: one page header`);
  assert.equal(result.h1Count, 1, `${label}: one H1`);
  if (!label.startsWith('Mein Zuhause')) assert(result.headerVisible && result.identity, `${label}: visible header with identity`);
  assert.equal(result.active, 1, `${label}: one active menu entry`);
  assert.deepEqual(result.badActions, [], `${label}: header actions must be ghost buttons ≥40px (except one filled primary)`);
  assert(result.filledActions.length <= 1, `${label}: at most one filled primary in the header, got ${result.filledActions.join(', ')}`);
  assert(result.taps.every(t => t.height >= 40), `${label}: tap targets ${JSON.stringify(result.taps)}`);
  assert.deepEqual(result.left, [], `${label}: no element left of viewport`);
  assert(!result.overflow, `${label}: no sideways overflow`);
  assert.equal(result.box.paddingLeft, '18px', `${label}: shared padding`);
  assert.equal(result.box.paddingRight, '18px', `${label}: shared padding`);
  assert(Math.abs(result.box.x) <= 1, `${label}: sidebar starts at x=0`);
  assert(Math.abs(result.box.width - (width <= 760 ? result.layoutWidth : sidebarWidth)) <= 1, `${label}: shared width`);
  if (new URL(page.url()).pathname.includes('/app/verwaltung')) {
    assert.match(result.overview, /^Alle Liegenschaften/, `${label}: overview card`);
    assert(result.portfolioMap, `${label}: neutral portfolio map`);
  } else assert(!result.portfolioMap, `${label}: house map`);
  return result;
}

// Measure actual scroll-end geometry and hit testing, including the mobile
// drawer. Preserve the caller's scroll position for the route/spacing oracles.
async function measureFooter(page, width, sidebarWidth, label) {
  const surface = await navigation(page, width);
  const result = await surface.evaluate(element => {
    const previous = element.scrollTop;
    const behavior = element.style.scrollBehavior;
    element.style.scrollBehavior = 'auto';
    const links = [...element.querySelectorAll('.nav-organisation > a, .nav-house-items > a')];
    const last = links.at(-1);
    const footer = element.querySelector('.sidebar-release');
    const nav = element.querySelector('.nav');
    // Sticky bottom can cover the last VISIBLE row well before true scroll end.
    // Keep that screenshot regression in addition to the scroll-end oracle.
    const coveredRows = [];
    const maxScroll = element.scrollHeight - element.clientHeight;
    for (const top of [0, Math.min(300, maxScroll), maxScroll / 2, maxScroll]) {
      element.scrollTop = top;
      const release = footer.getBoundingClientRect(), surface = element.getBoundingClientRect();
      for (const link of links) {
        const row = link.getBoundingClientRect();
        if (Math.min(row.bottom, release.bottom, surface.bottom, innerHeight) > Math.max(row.top, release.top, surface.top)
          && Math.min(row.right, release.right) > Math.max(row.left, release.left)) {
          coveredRows.push({ top: element.scrollTop, label: link.textContent.trim() });
        }
      }
    }
    element.scrollTop = element.scrollHeight;
    const row = last.getBoundingClientRect(), release = footer.getBoundingClientRect(), box = element.getBoundingClientRect();
    const x = row.x + row.width / 2, y = row.y + row.height / 2;
    const hit = document.elementFromPoint(x, y);
    const result = { height: innerHeight, width: box.width, label: last.textContent.trim(),
      scrollTop: element.scrollTop, maxScroll, coveredRows,
      row: row.toJSON(), footer: release.toJSON(), surface: box.toJSON(),
      navScrollTop: nav.scrollTop, navOverflow: getComputedStyle(nav).overflowY,
      hit: hit?.tagName + '.' + hit?.className, hitBelongsToRow: last === hit || last.contains(hit),
      overlap: Math.max(0, Math.min(row.right, release.right) - Math.max(row.left, release.left)) * Math.max(0, Math.min(row.bottom, release.bottom) - Math.max(row.top, release.top)) };
    element.scrollTop = previous;
    element.style.scrollBehavior = behavior;
    return result;
  });
  footerMeasurements.push({ scenario: label, viewport: width, sidebarWidth, ...result });
  assert(Math.abs(result.scrollTop - result.maxScroll) <= 1, `${label}: surface must be at scroll end`);
  assert.deepEqual(result.coveredRows, [], `${label}: footer covers menu rows during scrolling`);
  assert.equal(result.overlap, 0, `${label}: footer must not overlap last navigation row`);
  assert(result.footer.top >= result.row.bottom - 0.5, `${label}: release belongs below the last menu item`);
  assert(result.row.top >= result.surface.top && result.row.bottom <= Math.min(result.surface.bottom, result.height), `${label}: entire last row must be visible`);
  assert(result.footer.bottom <= Math.min(result.surface.bottom, result.height) + 1, `${label}: footer reachable at scroll end`);
  assert(result.hitBelongsToRow, `${label}: last-row midpoint is covered by ${result.hit}`);
}

async function measureScrollPersistence(context, page) {
  await page.setViewportSize({ width: 1440, height: 700 });
  await context.addInitScript(() => {
    window.__sidebarScrollFrames = [];
    // Also watch the parser's early frames: a DCL-only probe can miss a jump.
    const frame = () => {
      const sidebar = document.querySelector('aside.sidebar');
      if (sidebar?.clientHeight) window.__sidebarScrollFrames.push({ top: sidebar.scrollTop, at: performance.now() });
      if (window.__sidebarScrollFrames.length < 60) requestAnimationFrame(frame);
    };
    requestAnimationFrame(frame);
    document.addEventListener('DOMContentLoaded', () => {
      requestAnimationFrame(() => { window.__sidebarScrollAtDCLFrame = document.querySelector('aside.sidebar')?.scrollTop; });
    }, { once: true });
  });
  const sidebar = page.locator('aside.sidebar');
  const target = await sidebar.evaluate(element => {
    element.scrollTop = 300;
    const rect = element.getBoundingClientRect();
    const links = [...element.querySelectorAll('.nav-organisation > a, .nav-house-items > a')];
    const visible = links.filter(link => {
      const row = link.getBoundingClientRect();
      return row.top >= rect.top && row.bottom <= Math.min(rect.bottom, innerHeight) && !link.classList.contains('active');
    });
    const last = visible.at(-1);
    if (!last) throw new Error('no fully visible link at scrollTop 300');
    const row = last.getBoundingClientRect();
    return { href: last.href, top: element.scrollTop, x: row.x + row.width / 2, y: row.y + row.height / 2 };
  });
  assert.equal(target.top, 300, 'admin fixture must support scrollTop 300');
  // A real click at the visible midpoint avoids Playwright auto-scrolling the
  // target, which would change the preference we intend to verify.
  await Promise.all([
    page.waitForURL(target.href, { waitUntil: 'domcontentloaded' }),
    page.mouse.click(target.x, target.y),
  ]);
  await page.waitForFunction(() => window.__sidebarScrollAtDCLFrame !== undefined);
  await page.waitForLoadState('networkidle');
  const result = await page.evaluate(() => ({ firstDCLFrame: window.__sidebarScrollAtDCLFrame,
    frames: window.__sidebarScrollFrames, settled: document.querySelector('aside.sidebar').scrollTop }));
  scrollMeasurements.push({ href: target.href, expected: 300, ...result });
  assert(Math.abs(result.firstDCLFrame - 300) <= 2, 'scrollTop must already be 300 in the first rAF after DCL');
  assert(result.frames.length > 0 && result.frames.every(frame => Math.abs(frame.top - 300) <= 2), 'every visible sidebar frame must retain scrollTop 300');
  assert(Math.abs(result.settled - 300) <= 2, 'sidebar must stay at scrollTop 300 after load');
}

try {
  const scenarios = [[1440, 280], [1024, 280], [390, 280], [1440, 240], [1440, 420]].map(([width, sidebarWidth]) => ({ width, sidebarWidth, email: 'admin@example.com' }));
  for (const email of ['owner@example.com', 'resident@example.com']) {
    for (const width of [1440, 1024, 390]) scenarios.push({ width, sidebarWidth: 280, email });
  }
  for (const { width, sidebarWidth, email } of scenarios) {
    const context = await browser.newContext({ viewport: { width, height: 900 }, locale: 'de-AT', timezoneId: 'Europe/Vienna' });
    try {
      const page = await login(context, email);
      // Set the width through the accessible keyboard control, so persistence is exercised.
      if (width > 760) {
        const handle = page.locator('[data-sidebar-resize]');
        await handle.dblclick();
        const key = sidebarWidth < 280 ? 'ArrowLeft' : 'ArrowRight';
        for (let i = 0; i < Math.abs(sidebarWidth - 280) / 10; i++) await handle.press(key);
      }
      const baseline = await measure(page, width, sidebarWidth, 'Hausüberblick');
      await measureFooter(page, width, sidebarWidth, `${email}/Hausüberblick`);
      assert.equal(baseline.organisation, email === 'admin@example.com', `${email}: correct organisation membership`);
      if (email === 'admin@example.com') assert(baseline.links.length >= 15, 'admin fixture must expose the full navigation');
      else assert(baseline.links.some(link => link.label === 'Hausüberblick') && baseline.links.some(link => link.label === 'Hilfe'), `${email}: resident navigation is present`);
      for (const [index, item] of baseline.links.entries()) {
        const surface = await navigation(page, width);
        const link = surface.locator('.nav-organisation > a, .nav-house-items > a').nth(index);
        assert.equal(await link.getAttribute('href'), item.href, `${item.label}: menu target changed`);
        await link.click(); await page.waitForLoadState('networkidle');
        const result = await measure(page, width, sidebarWidth, item.label);
        await measureFooter(page, width, sidebarWidth, `${email}/${item.label}`);
        assert.deepEqual(result.box, baseline.box, `${item.label}: geometry changed between routes`);
        assert.deepEqual(result.blocks, baseline.blocks, `${item.label}: block visibility/classes changed`);
        assert.deepEqual(result.links, baseline.links, `${item.label}: navigation links changed`);
      }
      if (email === 'admin@example.com' && width === 1440 && sidebarWidth === 280) await measureScrollPersistence(context, page);
    } catch (error) { failures.push(`${email}/${width}/${sidebarWidth}: ${error.stack}`); }
    finally { await context.close(); }
  }
  // Full footer matrix: all roles, narrow/default/wide sidebars and the mobile
  // panel at short, medium and tall viewport heights. Route checks above stay.
  for (const email of ['admin@example.com', 'owner@example.com', 'resident@example.com']) {
    const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'de-AT', timezoneId: 'Europe/Vienna' });
    try {
      const page = await login(context, email);
      for (const height of [700, 900, 1100, 1400]) {
        for (const sidebarWidth of [240, 280, 420]) {
          await page.setViewportSize({ width: 1440, height });
          const handle = page.locator('[data-sidebar-resize]');
          await handle.press('Home');
          const key = sidebarWidth < 280 ? 'ArrowLeft' : 'ArrowRight';
          for (let i = 0; i < Math.abs(sidebarWidth - 280) / 10; i++) await handle.press(key);
          await measureFooter(page, 1440, sidebarWidth, `${email}/${sidebarWidth}/${height}`);
        }
        for (const width of [320, 390, 420]) {
          await page.setViewportSize({ width, height });
          await measureFooter(page, width, width, `${email}/mobile/${width}/${height}`);
        }
      }
    } catch (error) { failures.push(`footer matrix/${email}: ${error.stack}`); }
    finally { await context.close(); }
  }
} finally {
  await browser.close();
  writeFileSync(join(out, 'navigation.json'), JSON.stringify({ measurements, footerMeasurements, scrollMeasurements, failures }, null, 2));
}
for (const failure of failures) console.error(failure);
assert.equal(failures.length, 0, `${failures.length} navigation scenarios failed`);
console.log(`PASS: ${measurements.length} route/width measurements; ${footerMeasurements.length} footer checks; ${scrollMeasurements.length} first-frame scroll checks; shared shell, blocks, widths and headers`);
