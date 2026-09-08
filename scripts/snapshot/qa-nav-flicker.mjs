#!/usr/bin/env node
// Coordinator-only browser run against the disposable snapshot app:
// node scripts/snapshot/qa-nav-flicker.mjs http://localhost:8099 [out-dir]
// CDP frames are decoded on a separate page: no screenshots, masks, injected
// portal styles, cache overrides or forced document background colours.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const [baseURL, outputArgument] = process.argv.slice(2);
const out = process.env.HV_QA_ARTIFACT_DIR || outputArgument || '/private/tmp/hausv-nav-flicker-qa';
if (!baseURL || !['localhost', '127.0.0.1', 'hausv.test'].includes(new URL(baseURL).hostname)) {
  throw new Error('usage: qa-nav-flicker.mjs <local fixture baseURL> [out-dir]');
}
mkdirSync(out, { recursive: true });
const executablePath = [process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH, ...(process.env.CI === 'true' ? [] : ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/Applications/Chromium.app/Contents/MacOS/Chromium', '/usr/bin/chromium'])].filter(Boolean).find(existsSync);
const browser = await chromium.launch({ headless: process.env.HV_QA_HEADLESS !== 'false', ...(executablePath ? { executablePath } : {}), args: ['--host-resolver-rules=MAP hausv.test 127.0.0.1, MAP *.hausv.test 127.0.0.1', '--no-proxy-server'] });
const report = { viewport: { width: 1440, height: 900 }, thresholds: { sidebarAbove: 0.60, contentBelow: 0.30 }, navigations: [], frames: [], flashes: [], failures: [] };
const frames = [], acknowledgements = new Set();
let cdp, decoder, phase = 'Hausüberblick', casting = false;

// A canvas in the decoder context reads the actual JPEG pixels. The 280 CSS px
// sidebar becomes 140 JPEG px at maxWidth=720; the context bar is excluded from
// BOTH samples so it cannot make the cream content appear dark.
async function brightness(frame) {
  return decoder.evaluate(async ({ data, viewport }) => {
    const img = new Image();
    img.src = `data:image/jpeg;base64,${data}`;
    await img.decode();
    const canvas = document.createElement('canvas');
    canvas.width = img.naturalWidth; canvas.height = img.naturalHeight;
    const context = canvas.getContext('2d', { willReadFrequently: true });
    context.drawImage(img, 0, 0);
    const { data: pixels } = context.getImageData(0, 0, canvas.width, canvas.height);
    const boundary = Math.round(280 * canvas.width / viewport.width);
    const top = Math.ceil(40 * canvas.height / viewport.height);
    let sidebar = 0, content = 0, sidebarCount = 0, contentCount = 0;
    for (let y = top; y < canvas.height; y++) {
      for (let x = 0; x < canvas.width; x++) {
        const i = (y * canvas.width + x) * 4;
        const luma = (0.2126 * pixels[i] + 0.7152 * pixels[i + 1] + 0.0722 * pixels[i + 2]) / 255;
        if (x < boundary) { sidebar += luma; sidebarCount++; }
        else { content += luma; contentCount++; }
      }
    }
    return { width: canvas.width, height: canvas.height, sidebar: sidebar / sidebarCount, content: content / contentCount };
  }, { data: frame.data, viewport: report.viewport });
}

try {
  const decoderContext = await browser.newContext();
  decoder = await decoderContext.newPage();
  const context = await browser.newContext({ viewport: report.viewport, deviceScaleFactor: 1, locale: 'de-AT', timezoneId: 'Europe/Vienna' });
  const page = await context.newPage();
  page.on('pageerror', error => report.failures.push(error.message));
  await page.goto(baseURL, { waitUntil: 'networkidle' });
  await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
  await page.locator('input[name="email"]').fill('admin@example.com');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const link = page.locator('a.dev-link'); await link.waitFor({ state: 'visible' });
  const target = new URL(await link.getAttribute('href'), baseURL), origin = new URL(baseURL);
  target.protocol = origin.protocol; target.port = origin.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  assert(new URL(page.url()).pathname.endsWith('/app'), 'Vera must start on Hausüberblick');
  await page.locator('[data-sidebar-resize]').press('Home');
  assert.equal(Math.round((await page.locator('aside.sidebar').boundingBox()).width), 280);
  await page.bringToFront();
  cdp = await context.newCDPSession(page);
  await cdp.send('Page.enable');
  cdp.on('Page.screencastFrame', event => {
    const frame = { data: event.data, timestamp: event.metadata.timestamp, receivedAt: Date.now(), phase, metadata: event.metadata };
    frames.push(frame);
    // ACK immediately, even while a navigation is destroying its JS context.
    const ack = cdp.send('Page.screencastFrameAck', { sessionId: event.sessionId }).catch(error => report.failures.push(`frame ACK: ${error.message}`));
    acknowledgements.add(ack);
    void ack.finally(() => acknowledgements.delete(ack));
  });
  await cdp.send('Page.startScreencast', { format: 'jpeg', quality: 50, maxWidth: 720, everyNthFrame: 1 });
  casting = true;
  await page.waitForTimeout(300);
  // Warm and cold asset-cache passes; the second round also exercises restored
  // sidebar scroll positions. All targets are real links in Vera's navigation.
  for (let pass = 1; pass <= 2; pass++) {
    for (const label of ['Aushang', 'Termine', 'Kontakte', 'Dokumente', 'Anliegen', 'Abstimmungen', 'Portfolio', 'Posteingang', 'Hausüberblick']) {
      phase = `${pass}:${label}`;
      const item = page.locator('aside.sidebar .nav a').filter({ has: page.locator('.nav-label', { hasText: new RegExp(`^${label}$`) }) });
      assert.equal(await item.count(), 1, `${phase}: exactly one navigation target`);
      const href = new URL(await item.getAttribute('href'), page.url()).href;
      const startFrame = frames.length, startedAt = Date.now();
      await Promise.all([page.waitForURL(href, { waitUntil: 'load' }), item.click()]);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(200);
      report.navigations.push({ phase, href, startedAt, endedAt: Date.now(), startFrame, endFrame: frames.length });
    }
  }
} catch (error) {
  report.failures.push(error.stack || String(error));
} finally {
  if (casting) await cdp.send('Page.stopScreencast').catch(error => report.failures.push(error.message));
  await Promise.all(acknowledgements);
  try {
    for (const [index, frame] of frames.entries()) {
      const timestamp = frame.timestamp ?? frame.receivedAt / 1000;
      const file = `${String(index).padStart(5, '0')}-${timestamp.toFixed(6)}.jpg`;
      writeFileSync(join(out, file), Buffer.from(frame.data, 'base64'));
      const sample = await brightness(frame);
      report.frames.push({ index, timestamp, receivedAt: frame.receivedAt, phase: frame.phase, file, metadata: frame.metadata, ...sample,
        abnormal: sample.sidebar > report.thresholds.sidebarAbove || sample.content < report.thresholds.contentBelow });
    }
    // Group consecutive abnormal frames, not just isolated one-frame flashes.
    for (let i = 0; i < report.frames.length; i++) {
      if (!report.frames[i].abnormal) continue;
      const start = i;
      while (i + 1 < report.frames.length && report.frames[i + 1].abnormal) i++;
      report.flashes.push({ start, end: i, betweenNormalFrames: start > 0 && i + 1 < report.frames.length });
    }
    assert(report.frames.length > 0, 'CDP must deliver frames');
    for (const navigation of report.navigations) {
      assert(navigation.endFrame > navigation.startFrame, `${navigation.phase}: no frames received`);
      const samples = report.frames.slice(navigation.startFrame, navigation.endFrame);
      assert(samples.some(frame => !frame.abnormal), `${navigation.phase}: no normal frame received`);
    }
    assert.equal(report.navigations.length, 18, 'both complete navigation passes must be captured');
    assert.equal(report.flashes.length, 0, 'light sidebar or dark content frame detected (see timestamped JPEGs)');
  } catch (error) { report.failures.push(error.stack || String(error)); }
  await browser.close();
  writeFileSync(join(out, 'qa-nav-flicker.json'), JSON.stringify(report, null, 2));
}
console.log(`${report.frames.length} Screencast-Frames; ${report.navigations.length} Seitenwechsel; ${report.flashes.length} Helligkeitswechsel; ${report.failures.length} Fehler — ${out}`);
if (report.failures.length) { console.error(report.failures.join('\n')); process.exitCode = 1; }
