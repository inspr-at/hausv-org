#!/usr/bin/env node
// At every width, does every route still have navigation — and the same kind?
//
//   node shell-widths.mjs <baseURL> [outJson]
//
// The mobile shell used to hide the sidebar at 700px on six routes and 760px on
// five, so between those widths some pages showed the sidebar and others the
// hamburger. Screenshots at one width cannot see that, and screenshots at eleven
// widths are eleven things to eyeball. Assert it instead.
//
// Two properties, both mechanical:
//   1. every route has exactly one visible navigation at every width — never zero
//   2. at any given width, every route agrees on which one
import { chromium } from 'playwright';
import { writeFile } from 'node:fs/promises';

const baseURL = process.argv[2];
const outJson = process.argv[3];
if (!baseURL) {
  console.error('usage: shell-widths.mjs <baseURL> [outJson]');
  process.exit(2);
}

// Straddle every breakpoint in play, plus real device widths.
const WIDTHS = [360, 390, 699, 700, 701, 744, 759, 760, 761, 768, 899, 900, 1024, 1049, 1050, 1051, 1100, 1440];

const ROUTES = [
  ['portal', '/app'], ['announcements', '/app/announcements'], ['events', '/app/events'],
  ['issues', '/app/anliegen'], ['issue-board', '/app/anliegen/board'],
  ['documents', '/app/dokumente'], ['ballots', '/app/abstimmungen'],
  ['handovers', '/app/uebergaben'], ['contacts', '/app/kontakte'], ['help', '/app/hilfe'], ['parking', '/app/parking'],
  ['onboarding', '/app/zuhause/onboarding'],
  ['settings', '/app/settings'], ['settings-profile', '/app/settings/profile'],
  ['settings-notifications', '/app/settings/notifications'],
  ['settings-building', '/app/settings/building'], ['settings-users', '/app/settings/users'], ['audit', '/app/audit'],
];

const tenant = (process.env.DEFAULT_TENANT || 'demo').replace(/^\/+|\/+$/g, '');
const tenantURL = `${baseURL}/${tenant}`;

const browser = await chromium.launch({
  headless: true,
  ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH
    ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : {}),
});
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'de-AT', timezoneId: 'Europe/Vienna' });
const page = await ctx.newPage();

await page.goto(`${tenantURL}/`, { waitUntil: 'networkidle' });
await page.fill('input[name="email"]', 'admin@example.com');
await page.click('form[action$="/auth/request"] button');
await page.waitForSelector('a.dev-link', { timeout: 10_000 });
await page.goto(new URL(await page.getAttribute('a.dev-link', 'href'), `${tenantURL}/`).href, { waitUntil: 'networkidle' });

const shown = (el) => {
  if (!el) return false;
  const s = getComputedStyle(el);
  return s.display !== 'none' && s.visibility !== 'hidden' && el.getClientRects().length > 0;
};

const ENERGY = { tenant: 'cockpit', email: 'cockpit-owner@example.com', routes: [
  ['energy', '/app/energie'], ['settings-home', '/app/settings/home'],
] };

const results = [];
for (const [name, route] of ROUTES) {
  await page.goto(`${tenantURL}${route}`, { waitUntil: 'domcontentloaded' });
  for (const w of WIDTHS) {
    await page.setViewportSize({ width: w, height: 900 });
    const state = await page.evaluate((visibleSrc) => {
      const vis = eval(`(${visibleSrc})`);
      return {
        sidebar: vis(document.querySelector('.sidebar')),
        mobile: vis(document.querySelector('.mobile-head')),
        // The hamburger must be operable, not merely present — and matched by
        // structure, not by class. Keying this on `.menu` reported eight false
        // failures on the very code it was written to judge, because those pages
        // spell it `.mobile-menu`. A probe that only passes the layout it was
        // built for measures nothing.
        menu: vis(document.querySelector('.mobile-head details > summary')),
        links: document.querySelectorAll('.nav a').length,
        // Visible is not the same as reachable. Six routes rendered the mobile
        // header AFTER the content, and one behind an empty 100vh grid, so the
        // only navigation on the page sat below the fold — off-screen on load,
        // and this probe called it visible because it had a bounding box.
        headTop: (() => { const h = document.querySelector('.mobile-head'); return h ? Math.round(h.getBoundingClientRect().top) : null; })(),
        // This probe describes the TEMPL shell: sidebar below the breakpoint is
        // hidden and a mobile header takes over. The legacy shell does something
        // else entirely — it turns the sidebar into a sticky bar — so a legacy
        // route reads as "desktop" at every width. Recording which renderer served
        // the page keeps that from being reported as a defect it is not.
        templ: [...document.body.attributes].some((a) => a.name.startsWith('data-templ')),
      };
    }, shown.toString());
    results.push({ route: name, width: w, ...state });
  }
}
// Reachable only for the cockpit tenant; see capture.mjs for why.
{
  const t2 = `${baseURL}/${ENERGY.tenant}`;
  const p2 = await ctx.newPage();
  await p2.goto(`${t2}/`, { waitUntil: 'networkidle' });
  await p2.fill('input[name="email"]', ENERGY.email);
  await p2.click('form[action$="/auth/request"] button');
  await p2.waitForSelector('a.dev-link', { timeout: 10000 });
  await p2.goto(new URL(await p2.getAttribute('a.dev-link', 'href'), `${t2}/`).href, { waitUntil: 'networkidle' });
  for (const [name, route] of ENERGY.routes) {
    await p2.goto(`${t2}${route}`, { waitUntil: 'domcontentloaded' });
    for (const w of WIDTHS) {
      await p2.setViewportSize({ width: w, height: 900 });
      const state = await p2.evaluate((visibleSrc) => {
        const vis = eval(`(${visibleSrc})`);
        const h = document.querySelector('.mobile-head');
        return {
          sidebar: vis(document.querySelector('.sidebar')),
          mobile: vis(h),
          menu: vis(document.querySelector('.mobile-head details > summary')),
          links: document.querySelectorAll('.nav a').length,
          headTop: h ? Math.round(h.getBoundingClientRect().top) : null,
          templ: [...document.body.attributes].some((a) => a.name.startsWith('data-templ')),
        };
      }, shown.toString());
      results.push({ route: name, width: w, ...state });
    }
  }
}

await browser.close();

let failures = 0;
console.log(`${'width'.padStart(6)}  ${'shell'.padEnd(8)} routes`);
for (const w of WIDTHS) {
  const all = results.filter((r) => r.width === w);
  const at = all.filter((r) => r.templ);
  const legacy = all.filter((r) => !r.templ);
  const none = at.filter((r) => !r.sidebar && !r.mobile);
  const both = at.filter((r) => r.sidebar && r.mobile);
  const desktop = at.filter((r) => r.sidebar && !r.mobile).map((r) => r.route);
  const mob = at.filter((r) => r.mobile && !r.sidebar).map((r) => r.route);
  const noLinks = at.filter((r) => r.links === 0);
  const noMenu = at.filter((r) => r.mobile && !r.menu);
  const buried = at.filter((r) => r.mobile && r.headTop !== null && r.headTop > 0);

  let kind = desktop.length && mob.length ? 'SPLIT' : (mob.length ? 'mobile' : 'desktop');
  const skipped = legacy.length ? `  (${legacy.length} on the legacy renderer, not covered: ${[...new Set(legacy.map((r) => r.route))].join(', ')})` : '';
  console.log(`${String(w).padStart(6)}  ${kind.padEnd(8)} ${desktop.length} desktop / ${mob.length} mobile${skipped}`);

  for (const [label, list] of [['no navigation at all', none], ['both shells at once', both],
                               ['no nav links', noLinks], ['mobile head without a usable menu', noMenu],
                               ['mobile header pushed below the top', buried.map((r) => ({ ...r, route: `${r.route} (y=${r.headTop})` }))]]) {
    if (list.length) { failures += list.length; console.log(`         ${label}: ${list.map((r) => r.route).join(', ')}`); }
  }
  if (desktop.length && mob.length) {
    failures++;
    console.log(`         routes disagree — desktop: ${desktop.join(', ')}`);
    console.log(`                           mobile: ${mob.join(', ')}`);
  }
}

if (outJson) await writeFile(outJson, JSON.stringify(results, null, 2));
const legacyRoutes = [...new Set(results.filter((r) => !r.templ).map((r) => r.route))];
console.log(`\n${results.filter((r) => r.templ).length} route/width combinations checked`);
if (legacyRoutes.length) {
  console.log(`not covered — still on the legacy renderer: ${legacyRoutes.join(', ')}`);
}
if (failures) { console.log(`FAIL — ${failures} problem(s)`); process.exit(1); }
console.log('PASS — every route has exactly one navigation at every width, it sits at the top, and they all agree');
