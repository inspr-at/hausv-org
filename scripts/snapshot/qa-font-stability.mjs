#!/usr/bin/env node
// HAUSV-696: run against the disposable, locally seeded run.sh app.
// HV_CAPTURE=qa-font-stability.mjs scripts/snapshot/run.sh WORKTREE /tmp/font-qa 8099
// No routing interception, cache disabling, or fonts.load before measurement.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
const out = process.argv[3] || process.env.HV_QA_ARTIFACT_DIR || '/tmp/hausv-font-qa';
if (!baseURL || !['localhost', '127.0.0.1', 'hausv.test'].includes(new URL(baseURL).hostname)) {
  throw new Error('usage: qa-font-stability.mjs <local run.sh baseURL> [artifact directory]');
}
mkdirSync(out, { recursive: true });
const executablePath = [
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  ...(process.env.CI === 'true' ? [] : [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Chromium.app/Contents/MacOS/Chromium',
    '/usr/bin/chromium', '/usr/bin/chromium-browser', '/usr/bin/google-chrome',
  ]),
].filter(Boolean).find(existsSync);
const browser = await chromium.launch({
  headless: process.env.CI === 'true' || process.env.HV_QA_HEADLESS !== 'false',
  ...(executablePath ? { executablePath } : {}),
  args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'],
});
const report = { widths: [], failures: [] };
let activePage;

// Install before each document. rAF samples run before paint, including while
// a swap font is loading. Checking only after fonts.ready would miss the bug.
function observeFirstPaint() {
  const state = { frames: [], paints: [], violations: [] };
  window.__fontStability = state;
  document.addEventListener('securitypolicyviolation', (event) => {
    state.violations.push({ directive: event.effectiveDirective, blockedURI: event.blockedURI });
  });
  new PerformanceObserver((list) => {
    for (const entry of list.getEntries()) state.paints.push(entry.toJSON());
  }).observe({ type: 'paint', buffered: true });
  const visible = (selector) => [...document.querySelectorAll(selector)].find((element) => {
    const rect = element.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0 && getComputedStyle(element).visibility !== 'hidden';
  });
  const measure = (element) => {
    if (!element) return null;
    const rect = element.getBoundingClientRect();
    return { x: rect.x, y: rect.y, width: rect.width, height: rect.height,
      text: element.textContent.trim(), family: getComputedStyle(element).fontFamily };
  };
  state.sample = () => ({
    at: performance.now(),
    check: document.fonts.check('600 16px "Source Serif 4"'),
    // fonts.check alone returns true when the face has not been declared yet.
    loadedFace: [...document.fonts].some((face) => face.family.replaceAll('"', '') === 'Source Serif 4' && face.weight === '600' && face.status === 'loaded'),
    logo: measure(visible('.sidebar .brand, .sidebar .organisation-identity strong, .mobile-head .mobile-identity strong')),
    heading: measure(visible('.case-title-row h2')),
  });
  function frame() {
    const sample = state.sample();
    if (sample.logo && sample.heading) state.frames.push(sample);
    if (state.frames.length < 240) requestAnimationFrame(frame);
  }
  requestAnimationFrame(frame);
}

function closeBox(actual, expected, label) {
  for (const axis of ['x', 'y', 'width', 'height']) {
    assert.ok(Math.abs(actual[axis] - expected[axis]) <= 0.5,
      `${label}: ${axis} changed from ${expected[axis]} to ${actual[axis]}`);
  }
}

try {
  const loginContext = await browser.newContext();
  const login = await loginContext.newPage();
  activePage = login;
  await login.goto(baseURL, { waitUntil: 'networkidle' });
  await login.locator('details:has(form[action$="/auth/request"])').evaluateAll((elements) => elements.forEach((element) => { element.open = true; }));
  await login.locator('input[name="email"]').fill('admin@example.com');
  await login.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = login.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible' });
  const target = new URL(await devLink.getAttribute('href'), baseURL);
  target.protocol = new URL(baseURL).protocol;
  target.port = new URL(baseURL).port;
  await login.goto(target.href, { waitUntil: 'networkidle' });
  await login.goto(`${baseURL}/demo/app/verwaltung/posteingang`, { waitUntil: 'networkidle' });
  const form = login.locator('form[action$="/verwaltung/telefonnotiz"]');
  assert.equal(await form.count(), 1, 'run.sh must expose the local phone-note fixture form');
  const action = new URL(await form.getAttribute('action'), login.url()).href;
  const house = await form.locator('select[name="house"] option[value="demo"]').getAttribute('value');
  assert.equal(house, 'demo');
  // Three deterministic cases, differing in title length. This affects only
  // run.sh's throwaway DB; the script refuses nonlocal origins above.
  const query = 'QA Schriftstabilität';
  for (const subject of [
    `${query} – Lampe`,
    `${query} – Beleuchtung im gemeinsamen Stiegenhaus prüfen und Termin abstimmen`,
    `${query} – Rückfrage zur Garage`,
  ]) {
    const response = await loginContext.request.post(action, {
      headers: { Origin: new URL(baseURL).origin },
      form: { house, from_name: 'QA Font', subject, body: 'Lokale Testnotiz zur Schriftstabilität.' },
      maxRedirects: 0,
    });
    assert.equal(response.status(), 303, 'phone-note fixture must be accepted');
  }
  const queueURL = `${baseURL}/demo/app/verwaltung/posteingang?source=phone&house=demo`;
  await login.goto(queueURL, { waitUntil: 'networkidle' });
  const fixtureRows = login.locator('.queue-row').filter({ hasText: query });
  assert.equal(await fixtureRows.count(), 3, 'three synthetic cases must be present in the queue');
  // On phones the queue and case are separate screens. Start at the first case
  // on every width so the measured heading is visible, then use real J/K links.
  const firstCaseURL = new URL(await fixtureRows.first().getAttribute('href'), login.url()).href;
  const storageState = await loginContext.storageState();
  await loginContext.close();

  // Straddle the shared 760px breakpoint; retain narrow-phone coverage.
  for (const width of [1440, 1024, 761, 760, 390, 320]) {
    const context = await browser.newContext({ viewport: { width, height: 1000 }, storageState });
    await context.addInitScript(observeFirstPaint);
    const page = await context.newPage();
    activePage = page;
    const cdp = await context.newCDPSession(page);
    await cdp.send('Network.enable');
    let responses = [];
    const fontRequests = new Set();
    const cacheRequests = new Set();
    cdp.on('Network.requestWillBeSent', ({ requestId, request }) => {
      if (new URL(request.url).pathname.endsWith('/source-serif-4-semibold.woff2')) fontRequests.add(requestId);
    });
    cdp.on('Network.requestServedFromCache', ({ requestId }) => cacheRequests.add(requestId));
    cdp.on('Network.responseReceived', ({ requestId, response }) => {
      if (!fontRequests.has(requestId)) return;
      const headers = Object.fromEntries(Object.entries(response.headers).map(([key, value]) => [key.toLowerCase(), value]));
      responses.push({ requestId, url: response.url, status: response.status,
        diskCache: !!response.fromDiskCache, cacheControl: headers['cache-control'], etag: headers.etag });
    });
    const runs = [];
    report.widths.push({ width, runs });
    let baseline;
    for (const [index, key] of [null, 'j', 'k', 'j', 'k'].entries()) {
      const before = index ? await page.evaluate(() => window.__fontStability.sample()) : null;
      const previousTimeOrigin = index ? await page.evaluate(() => performance.timeOrigin) : null;
      responses = [];
      if (key) {
        const nav = page.locator(`[data-nav="${key === 'j' ? 'next' : 'prev'}"]`);
        assert.equal(await nav.count(), 1, `missing ${key.toUpperCase()} target at ${width}px`);
        const nextURL = new URL(await nav.getAttribute('href'), page.url()).href;
        // Focus on a form control intentionally suppresses J/K; blur it first.
        await page.evaluate(() => document.activeElement?.blur());
        await Promise.all([
          page.waitForURL(nextURL, { waitUntil: 'load' }),
          page.keyboard.press(key),
        ]);
        assert.notEqual(await page.evaluate(() => performance.timeOrigin), previousTimeOrigin, 'J/K must exercise a full document navigation');
      } else {
        const response = await page.goto(firstCaseURL, { waitUntil: 'load' });
        assert.equal(response.status(), 200);
        assert.equal(await page.locator('.queue-row').filter({ hasText: query }).count(), 3, 'fixture cases must remain in the queue');
      }
      await page.waitForFunction(() => window.__fontStability?.paints.some((paint) => paint.name === 'first-contentful-paint'));
      // Only now wait for settling; the pre-paint observations are retained.
      const measurement = await page.evaluate(async () => {
        await document.fonts.ready;
        await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
        const state = window.__fontStability;
        const firstPaint = state.paints.find((paint) => paint.name === 'first-contentful-paint');
        const atPaint = state.frames.filter((frame) => frame.at <= firstPaint.startTime).at(-1);
        return { firstPaint, atPaint, firstFrame: state.frames[0], settled: state.sample(),
          violations: state.violations,
          preload: document.querySelector('link[rel="preload"][as="font"]')?.href,
          fonts: performance.getEntriesByType('resource').filter((entry) => new URL(entry.name).pathname.endsWith('/source-serif-4-semibold.woff2')).map((entry) => entry.toJSON()),
          viewport: { width: innerWidth, scrollWidth: document.documentElement.scrollWidth } };
      });
      const network = responses.map((response) => ({ ...response, cacheHit: response.diskCache || cacheRequests.has(response.requestId) }));
      const run = { step: index, key, before, ...measurement, network };
      runs.push(run);
      assert.ok(measurement.settled.check && measurement.settled.loadedFace, 'Source Serif 4 must load');
      assert.ok(measurement.firstFrame && measurement.atPaint, 'pre-paint logo and case observations are required');
      assert.ok(measurement.settled.logo.family.includes('Source Serif 4') && measurement.settled.heading.family.includes('Source Serif 4'), 'both measured elements must use Source Serif 4');
      assert.equal(measurement.violations.length, 0, `CSP violation: ${JSON.stringify(measurement.violations)}`);
      assert.ok(measurement.viewport.scrollWidth <= width + 1, 'shared shell must not overflow horizontally');
      assert.equal(measurement.fonts.length, 1, 'preload and font-face must share one request');
      assert.equal(measurement.fonts[0].name, measurement.preload);
      assert.ok(network.length > 0 && network.every((response) => response.cacheControl === 'public, max-age=31536000, immutable' && response.etag), 'font must carry immutable caching and an ETag');
      if (index > 0) {
        assert.ok(measurement.atPaint.check && measurement.atPaint.loadedFace, `warm visit ${index} at ${width}px painted without the font`);
        // The first animation frame can run before the cached face has resolved; what
        // the person sees is the first contentful paint, asserted above.
        run.firstFrameUsedFont = !!(measurement.firstFrame.check && measurement.firstFrame.loadedFace);
        assert.ok(measurement.fonts[0].transferSize === 0 || network.some((response) => response.cacheHit), 'warm font must hit cache');
        closeBox(measurement.settled.logo, baseline.logo, 'logo across J/K');
        closeBox(measurement.atPaint.logo, measurement.settled.logo, 'logo first paint vs settled');
        closeBox(measurement.atPaint.heading, measurement.settled.heading, 'case heading first paint vs settled');
        if (key === 'k') closeBox(measurement.settled.heading, baseline.heading, 'return to original case');
      } else baseline = measurement.settled;
      await page.screenshot({ path: join(out, `font-${width}-${index}-${key || 'cold'}.png`), fullPage: true });
    }
    await context.close();
  }
  console.log('Font stability OK: cold visit + J/K/J/K at 1440, 1024, 761, 760, 390 and 320px.');
} catch (error) {
  report.failures.push(error.message);
  if (activePage && !activePage.isClosed()) await activePage.screenshot({ path: join(out, 'failure.png'), fullPage: true }).catch(() => {});
  throw error;
} finally {
  writeFileSync(join(out, 'font-stability.json'), `${JSON.stringify(report, null, 2)}\n`);
  await browser.close();
}
