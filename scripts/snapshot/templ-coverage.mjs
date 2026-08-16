#!/usr/bin/env node
// Which authenticated routes actually render templ when the switch is on?
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
  rows.push({ route: r, landed, redirected: landed !== r, status: res ? res.status() : 0, ...info });
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
    const row = { route: r, landed, redirected: landed !== r, status: res ? res.status() : 0, tenant: ENERGY.tenant, ...info };
    if (i >= 0) rows[i] = row; else rows.push(row);
  }
}

await browser.close();
const legacy = rows.filter((r) => r.status === 200 && !r.templ);
console.log(`${'route'.padEnd(34)} ${'status'.padStart(6)}  renderer`);
for (const r of rows) console.log(`${r.route.padEnd(30)} ${String(r.status).padStart(4)}  ${(r.templ ? 'templ' : 'LEGACY').padEnd(7)} ${r.redirected ? '→ ' + r.landed + ' (NOT MEASURED)' : r.marker}`);
const red = rows.filter((r) => r.redirected);
console.log(`\n${rows.filter((r) => r.templ && !r.redirected).length} templ · ${legacy.length} legacy · ${red.length} redirected away and therefore unmeasured`);
if (red.length) console.log(`unmeasured: ${red.map((r) => r.route + ' → ' + r.landed).join(', ')}`);
if (legacy.length) console.log(`still legacy: ${legacy.map((r) => r.route).join(', ')}`);
if (process.argv[3]) await writeFile(process.argv[3], JSON.stringify(rows, null, 2));
