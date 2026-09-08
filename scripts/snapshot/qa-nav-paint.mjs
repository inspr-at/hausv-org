#!/usr/bin/env node
// HV_CAPTURE=qa-nav-paint.mjs scripts/snapshot/run.sh WORKTREE <out-dir> [port]
// Hold the external shell sheet past DOMContentLoaded and sample real pixels.
// A normal render-blocking sheet also blocks defer scripts/DCL. Mark ONLY that
// link media=print in the intercepted HTML so this probe can expose the first
// paint before CSS arrives; restore media=all after releasing the response.
// No styles, screenshot masks or background substitutions are injected.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { inflateSync } from 'node:zlib';
import { chromium } from 'playwright';

const [baseURL, out = '/private/tmp/hausv-nav-paint-qa'] = process.argv.slice(2);
if (!baseURL) throw new Error('usage: qa-nav-paint.mjs <baseURL> [out-dir]');
mkdirSync(out, { recursive: true });

// Chromium screenshots are non-interlaced 8-bit RGB/RGBA PNGs. Decode their
// actual scanlines instead of inferring the colour from computed CSS.
function pixelAt(png, x, y) {
  assert.equal(png.subarray(1, 4).toString(), 'PNG');
  let width, height, channels;
  const chunks = [];
  for (let offset = 8; offset < png.length;) {
    const size = png.readUInt32BE(offset), kind = png.toString('ascii', offset + 4, offset + 8);
    const data = png.subarray(offset + 8, offset + 8 + size);
    if (kind === 'IHDR') {
      width = data.readUInt32BE(0); height = data.readUInt32BE(4);
      assert.equal(data[8], 8); assert.equal(data[12], 0);
      channels = ({ 2: 3, 6: 4 })[data[9]];
      assert(channels, 'unsupported screenshot PNG colour type');
    } else if (kind === 'IDAT') chunks.push(data);
    offset += size + 12;
  }
  assert(x < width && y < height);
  const bytes = inflateSync(Buffer.concat(chunks)), stride = width * channels;
  let previous = Buffer.alloc(stride), offset = 0;
  for (let row = 0; row <= y; row++) {
    const filter = bytes[offset++], current = Buffer.alloc(stride);
    assert(filter <= 4, 'unsupported PNG filter');
    for (let col = 0; col < stride; col++) {
      const left = col >= channels ? current[col - channels] : 0;
      const above = previous[col], corner = col >= channels ? previous[col - channels] : 0;
      const p = left + above - corner, a = Math.abs(p - left), b = Math.abs(p - above), c = Math.abs(p - corner);
      const predictor = [0, left, above, Math.floor((left + above) / 2), a <= b && a <= c ? left : b <= c ? above : corner][filter];
      current[col] = (bytes[offset++] + predictor) & 255;
    }
    previous = current;
  }
  return [...previous.subarray(x * channels, x * channels + 3)];
}

const executablePath = [process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH, ...(process.env.CI === 'true' ? [] : ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/Applications/Chromium.app/Contents/MacOS/Chromium', '/usr/bin/chromium'])].filter(Boolean).find(existsSync);
const browser = await chromium.launch({ headless: true, ...(executablePath ? { executablePath } : {}), args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'] });
const checks = [], failures = [];
let releaseSheet;
try {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1, locale: 'de-AT' });
  const page = await context.newPage();
  page.on('pageerror', error => failures.push(error.message));
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
  await page.locator('input[name="email"]').fill('admin@example.com');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const link = page.locator('a.dev-link'); await link.waitFor({ state: 'visible' });
  const target = new URL(await link.getAttribute('href'), baseURL), origin = new URL(baseURL);
  target.protocol = origin.protocol; target.port = origin.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  await page.goto(`${baseURL}/app/verwaltung`, { waitUntil: 'networkidle' });

  // Persist through the real keyboard resize interaction, then navigate using
  // the actual sidebar links. This exercises JS -> cookie -> server -> CSS.
  const handle = page.locator('[data-sidebar-resize]');
  await handle.focus(); await page.keyboard.press('Home');
  for (let i = 0; i < 8; i++) await page.keyboard.press('ArrowRight');
  assert.equal((await context.cookies()).find(cookie => cookie.name === 'hausv-sidebar-w')?.value, '360');
  // A stale localStorage value must never override the cookie-backed document.
  await page.evaluate(() => localStorage.setItem('hausv:sidebar-width', '280'));

  await page.route('**/*', async route => {
    if (!route.request().isNavigationRequest() || route.request().resourceType() !== 'document') return route.fallback();
    const response = await route.fetch();
    const html = await response.text();
    const body = html.replace(/(<link rel="stylesheet" href="[^"]*\/assets\/portal-shell\.css[^"\s]*")\s*\/?\s*>/, '$1 media="print">');
    assert.notEqual(body, html, 'navigation must include the external shell stylesheet');
    await route.fulfill({ response, body });
  });
  let sheetHeld = false, sheetApplied = false, sheetGate;
  await page.route('**/assets/portal-shell.css?*', async route => {
    sheetHeld = true;
    await sheetGate;
    await route.continue();
  });
  page.on('response', response => { if (response.url().includes('/assets/portal-shell.css?')) sheetApplied = true; });

  const routes = ['/app/verwaltung/posteingang', '/app/verwaltung/rechte', '/app/verwaltung/textbausteine', '/app/verwaltung/einstellungen', '/app/verwaltung'];
  for (const width of [1440, 1024, 768]) {
    await page.setViewportSize({ width, height: 900 });
    for (const path of routes) {
      sheetHeld = false; sheetApplied = false;
      sheetGate = new Promise(resolve => { releaseSheet = resolve; });
      const before = await page.locator('aside.sidebar').boundingBox();
      const item = page.locator(`aside.sidebar .nav a[href$="${path}"]`).first();
      await Promise.all([
        page.waitForURL(url => url.pathname.endsWith(path), { waitUntil: 'domcontentloaded' }),
        item.click({ noWaitAfter: true }),
      ]);
      await page.waitForFunction(() => document.readyState !== 'loading');
      const name = `${width}-${path.split('/').at(-1)}`;
      const png = await page.screenshot({ path: join(out, `${name}-first-paint.png`), type: 'png', fullPage: false });
      assert(sheetHeld && !sheetApplied, `${name}: sample must precede the stylesheet response`);
      // x=20 intersects the gold active Rechte link at y=300. Sample the
      // unoccupied outer gutter and verify its hit target, so a legitimate
      // active item can neither fail nor satisfy the background assertion.
      const samples = [{ x: 2, y: 60 }, { x: 2, y: 300 }];
      for (const point of samples) {
        assert(await page.evaluate(({ x, y }) => document.elementFromPoint(x, y) === document.querySelector('aside.sidebar'), point), `${name}: paint sample must hit bare sidebar at (${point.x},${point.y})`);
        point.rgb = pixelAt(png, point.x, point.y);
        assert(Math.max(...point.rgb) < 100, `${name}: sidebar first paint is light at (${point.x},${point.y}): ${point.rgb}`);
      }
      const early = await page.locator('aside.sidebar').boundingBox();
      const htmlWidth = await page.locator('html').evaluate(element => element.style.getPropertyValue('--sidebar-w'));
      assert.equal(htmlWidth, '360px', `${name}: document carries the saved width`);
      assert.equal(Math.round(early.width), 360, `${name}: first-paint width`);
      assert(Math.abs(before.width - early.width) <= 1, `${name}: width changes across navigation`);
      releaseSheet();
      await page.waitForLoadState('networkidle');
      await page.locator('link[href*="/assets/portal-shell.css"]').evaluate(element => { element.media = 'all'; });
      await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
      const final = await page.locator('aside.sidebar').boundingBox();
      assert(Math.abs(final.width - early.width) <= 1 && Math.abs(final.x - early.x) <= 1, `${name}: stylesheet causes a sidebar jump`);
      const bar = await page.locator('[data-context-bar]:visible').boundingBox();
      assert.equal(Math.round(bar.height), 40, `${name}: context height`);
      assert.equal(await page.locator('nav[aria-label="Bereiche"]:visible').count(), 1, `${name}: one navigation`);
      const active = page.locator('aside.sidebar .nav a.active').first();
      const activeColour = await active.evaluate(element => getComputedStyle(element).backgroundColor);
      await active.hover();
      assert.equal(await active.evaluate(element => getComputedStyle(element).backgroundColor), activeColour, `${name}: active item stays gold on hover`);
      const inactive = page.locator('aside.sidebar .nav a:not(.active):not(.side-map)').first();
      const restingBox = await inactive.boundingBox();
      const restingColour = await inactive.evaluate(element => getComputedStyle(element).backgroundColor);
      await inactive.hover();
      await page.waitForFunction(() => {
        const element = document.querySelector('aside.sidebar .nav a:not(.active):not(.side-map):hover');
        return element && getComputedStyle(element).backgroundColor === 'rgba(255, 255, 255, 0.08)';
      });
      assert.notEqual(await inactive.evaluate(element => getComputedStyle(element).backgroundColor), restingColour, `${name}: inactive item brightens`);
      assert.deepEqual(await inactive.boundingBox(), restingBox, `${name}: hover must not change layout`);
      await inactive.focus();
      await page.keyboard.press('Tab'); await page.keyboard.press('Shift+Tab');
      assert(await inactive.evaluate(element => getComputedStyle(element).outlineStyle !== 'none' && parseFloat(getComputedStyle(element).outlineWidth) >= 2), `${name}: visible keyboard focus`);
      checks.push({ width, path, samples, earlyWidth: early.width, finalWidth: final.width });
    }
  }
  assert.deepEqual(failures, [], 'browser errors');
} catch (error) {
  failures.push(error.stack || String(error)); process.exitCode = 1;
} finally {
  releaseSheet?.();
  await browser.close();
  writeFileSync(join(out, 'qa-nav-paint.json'), JSON.stringify({ checks, failures }, null, 2));
}
console.log(`${checks.length} Paint-Prüfungen; ${failures.length} Fehler`);
if (failures.length) console.error(failures.join('\n'));
