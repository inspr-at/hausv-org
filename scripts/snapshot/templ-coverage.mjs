#!/usr/bin/env node
// Do all routes in the maintained authenticated coverage list render templ?
//   node templ-coverage.mjs <baseURL> [outJson]
import { chromium } from 'playwright';
import { writeFile } from 'node:fs/promises';
const baseURL = process.argv[2];
const ROUTES = [
  '/app', '/app/announcements', '/app/events', '/app/anliegen', '/app/anliegen/board',
  '/app/dokumente', '/app/abstimmungen', '/app/uebergaben', '/app/kontakte', '/app/hilfe',
  '/app/energie', '/app/parking', '/app/zuhause/onboarding',
  '/app/settings', '/app/settings/profile', '/app/settings/notifications',
  '/app/settings/building', '/app/settings/users', '/app/settings/home', '/app/audit',
];
const tenant = (process.env.DEFAULT_TENANT || 'demo').replace(/^\/+|\/+$/g, '');
const t = `${baseURL}/${tenant}`;
const ENERGY = { tenant: 'cockpit', email: 'cockpit-owner@example.com',
  routes: ['/app/energie', '/app/settings/home', '/app/zuhause/onboarding'] };
const TITLE_VIEWPORTS = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
];

async function compactTitleGeometry(page) {
  const landing = await page.$('[data-portal-section-landing]');
  if (!landing || await landing.getAttribute('data-portal-hero') !== 'false') return [];

  const measurements = [];
  for (const viewport of TITLE_VIEWPORTS) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    measurements.push(await page.evaluate((viewportName) => {
      const title = document.querySelector('[data-portal-section-header] .portal-section-title h1');
      if (!title) return { viewport: viewportName, error: 'shared compact H1 missing' };

      const original = title.innerHTML;
      title.textContent = 'gypq';
      const baselineMarker = document.createElement('span');
      baselineMarker.setAttribute('aria-hidden', 'true');
      baselineMarker.style.cssText = 'display:inline-block;width:0;height:0;padding:0;margin:0;vertical-align:baseline';
      title.append(baselineMarker);

      const style = getComputedStyle(title);
      const canvas = document.createElement('canvas');
      const context = canvas.getContext('2d');
      context.font = `${style.fontStyle} ${style.fontVariant} ${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
      const glyphs = context.measureText('gypq');
      const rect = title.getBoundingClientRect();
      const baseline = baselineMarker.getBoundingClientRect().bottom;
      const clipBottom = rect.bottom - parseFloat(style.borderBottomWidth || '0');
      const inkBottom = baseline + glyphs.actualBoundingBoxDescent;
      const clearance = clipBottom - inkBottom;
      const measurement = {
        viewport: viewportName,
        clearance: Number(clearance.toFixed(3)),
        lineHeight: style.lineHeight,
        paddingBottom: style.paddingBottom,
      };

      title.innerHTML = original;
      return measurement;
    }, viewport.name));
  }
  return measurements;
}

const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : {}) });
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'de-AT' });
const page = await ctx.newPage();
await page.goto(`${t}/`, { waitUntil: 'networkidle' });
await page.fill('input[name="email"]', 'admin@example.com');
await page.click('form[action$="/auth/request"] button');
await page.waitForSelector('a.dev-link', { timeout: 10000 });
await page.goto(new URL(await page.getAttribute('a.dev-link', 'href'), `${t}/`).href, { waitUntil: 'networkidle' });
const rows = [];
for (const r of ROUTES) {
  const res = await page.goto(`${t}${r}`, { waitUntil: 'domcontentloaded' });
  const info = await page.evaluate(() => ({
    templ: [...document.body.attributes].some((a) => a.name.startsWith('data-templ')),
    marker: [...document.body.attributes].map((a) => a.name).filter((n) => n.startsWith('data-templ')).join(','),
    authed: document.body.hasAttribute('data-authenticated-app'),
    shell: !!document.querySelector('.shell') || !!document.querySelector('.app-shell'),
  }));
  // Record where we ACTUALLY landed. /app/settings/home redirects to onboarding
  // when the seeded profile is incomplete, so without this the row silently
  // reports the onboarding page's renderer under the settings route's name.
  const landed = new URL(page.url()).pathname.replace(`/${tenant}`, '');
  const titleGeometry = landed === r ? await compactTitleGeometry(page) : [];
  rows.push({ route: r, landed, redirected: landed !== r, status: res ? res.status() : 0, titleGeometry, ...info });
}
// The demo profile never completes onboarding, so these three only render as
// themselves for the cockpit tenant. Measure them where they exist.
{
  const t2 = `${baseURL}/${ENERGY.tenant}`;
  const p2 = await ctx.newPage();
  await p2.goto(`${t2}/`, { waitUntil: 'networkidle' });
  await p2.fill('input[name="email"]', ENERGY.email);
  await p2.click('form[action$="/auth/request"] button');
  await p2.waitForSelector('a.dev-link', { timeout: 10000 });
  await p2.goto(new URL(await p2.getAttribute('a.dev-link', 'href'), `${t2}/`).href, { waitUntil: 'networkidle' });
  for (const r of ENERGY.routes) {
    const res = await p2.goto(`${t2}${r}`, { waitUntil: 'domcontentloaded' });
    const info = await p2.evaluate(() => ({
      templ: [...document.body.attributes].some((a) => a.name.startsWith('data-templ')),
      marker: [...document.body.attributes].map((a) => a.name).filter((n) => n.startsWith('data-templ')).join(','),
      authed: document.body.hasAttribute('data-authenticated-app'),
      shell: !!document.querySelector('.shell'),
    }));
    const landed = new URL(p2.url()).pathname.replace(`/${ENERGY.tenant}`, '');
    const i = rows.findIndex((x) => x.route === r);
    const titleGeometry = landed === r ? await compactTitleGeometry(p2) : [];
    const row = { route: r, landed, redirected: landed !== r, status: res ? res.status() : 0, tenant: ENERGY.tenant, titleGeometry, ...info };
    if (i >= 0) rows[i] = row; else rows.push(row);
  }
}

await browser.close();
const wrongRenderer = rows.filter((r) => r.status === 200 && !r.templ);
const clippedTitles = rows.flatMap((r) => r.titleGeometry
  .filter((measurement) => measurement.error || measurement.clearance < 0)
  .map((measurement) => ({ route: r.route, ...measurement })));
const failed = rows.filter((r) => r.status !== 200 || !r.templ || r.redirected);
console.log(`${'route'.padEnd(34)} ${'status'.padStart(6)}  renderer`);
for (const r of rows) console.log(`${r.route.padEnd(30)} ${String(r.status).padStart(4)}  ${(r.templ ? 'templ' : 'WRONG').padEnd(7)} ${r.redirected ? '→ ' + r.landed + ' (NOT MEASURED)' : r.marker}`);
const red = rows.filter((r) => r.redirected);
console.log(`\n${rows.filter((r) => r.templ && !r.redirected).length} templ · ${wrongRenderer.length} wrong renderer · ${red.length} redirected away and therefore unmeasured`);
if (red.length) console.log(`unmeasured: ${red.map((r) => r.route + ' → ' + r.landed).join(', ')}`);
if (wrongRenderer.length) console.log(`wrong renderer: ${wrongRenderer.map((r) => r.route).join(', ')}`);
if (clippedTitles.length) {
  console.log(`clipped compact titles: ${clippedTitles.map((r) => `${r.route} (${r.viewport}: ${r.error || `${r.clearance}px clearance`})`).join(', ')}`);
}
if (process.argv[3]) await writeFile(process.argv[3], JSON.stringify(rows, null, 2));
if (failed.length || clippedTitles.length) process.exit(1);
