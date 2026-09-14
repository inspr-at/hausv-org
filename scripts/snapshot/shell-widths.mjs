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
// Three properties, all mechanical:
//   1. every route has exactly one visible navigation at every width — never zero
//   2. at any given width, every route agrees on which one
//   3. the pages fit: no sideways scroll, and on tablet widths the primary
//      columns of the two-column pages stay usable (HAUSV-640, below)
import { chromium } from 'playwright';
import { writeFile } from 'node:fs/promises';

const baseURL = process.argv[2];
const outJson = process.argv[3];
if (!baseURL) {
  console.error('usage: shell-widths.mjs <baseURL> [outJson]');
  process.exit(2);
}

// Straddle every breakpoint in play, plus real device widths.
const WIDTHS = [320, 360, 390, 699, 700, 701, 744, 759, 760, 761, 768, 899, 900, 1024, 1049, 1050, 1051, 1100, 1440];

const ROUTES = [
  ['portal', '/app'], ['announcements', '/app/announcements'], ['events', '/app/events'],
  ['issues', '/app/anliegen'], ['issue-board', '/app/anliegen/board'],
  ['documents', '/app/dokumente'], ['ballots', '/app/abstimmungen'],
  ['handovers', '/app/uebergaben'], ['contacts', '/app/kontakte'], ['help', '/app/hilfe'], ['energy-help', '/app/hilfe/energie'], ['parking', '/app/parking'],
  ['onboarding', '/app/zuhause/onboarding'],
  ['settings', '/app/settings'], ['settings-profile', '/app/settings/profile'],
  ['settings-notifications', '/app/settings/notifications'],
  ['settings-building', '/app/settings/building'], ['settings-annual-statement', '/app/settings/annual-statement'], ['settings-users', '/app/settings/users'], ['audit', '/app/audit'],
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
        contextCount: document.querySelectorAll('[data-context-bar]').length,
        // HAUSV-704: every desktop route uses the persisted sidebar token,
        // including the tablet band that formerly switched to 210/250px.
        sidebarGeometry: (() => {
          const el = document.querySelector('.sidebar'); if (!vis(el)) return null;
          const box = el.getBoundingClientRect(), style = getComputedStyle(el);
          return { x: box.x, width: box.width, paddingLeft: style.paddingLeft, paddingRight: style.paddingRight, expectedWidth: parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--sidebar-w')) };
        })(),
        mobile: vis(document.querySelector('[data-context-bar] .context-navigation')),
        // The unified bar is always visible; only its navigation disclosure
        // marks the mobile shell. The account disclosure is a separate control.
        menu: vis(document.querySelector('[data-context-bar] .context-navigation > summary')),
        links: document.querySelectorAll('.nav a').length,
        // Visible is not the same as reachable. Six routes rendered the mobile
        // header AFTER the content, and one behind an empty 100vh grid, so the
        // only navigation on the page sat below the fold — off-screen on load,
        // and this probe called it visible because it had a bounding box.
        headTop: (() => { const h = document.querySelector('[data-context-bar]'); return h ? Math.round(h.getBoundingClientRect().top) : null; })(),
        // The skip link is rendered by every shell but its hiding rule lived in
        // one of them (HAUSV-647): a visible "Zum Inhalt springen" over the
        // sidebar is a defect at every width, so record where it sits.
        skipBottom: (() => { const a = document.querySelector('.skip-link'); return a ? Math.round(a.getBoundingClientRect().bottom) : null; })(),
        // Record the renderer marker so a route cannot silently evade the shell
        // assertions by returning unrelated or obsolete markup.
        templ: [...document.body.attributes].some((a) => a.name.startsWith('data-templ')),
        // The Hausüberblick swaps its whole body at the breakpoint: the desktop
        // grid gives way to .mobile-content. A shell rule that hides
        // .mobile-content unconditionally (HAUSV-637) left the phone page blank
        // below the hero on every role — and this probe green, because it only
        // asked for the navigation. Below the breakpoint the phone body must be
        // visible with real height, and above it it must stay hidden.
        home: (() => {
          const m = document.querySelector('.mobile-content');
          if (!m) return null;
          const d = document.querySelector('.portal-home-desktop');
          return {
            mobileShown: vis(m),
            mobileHeight: Math.round(m.getBoundingClientRect().height),
            desktopShown: vis(d),
            scrollWidth: document.documentElement.scrollWidth,
            innerWidth: window.innerWidth,
          };
        })(),
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
        const h = document.querySelector('[data-context-bar]');
        return {
          sidebar: vis(document.querySelector('.sidebar')),
        contextCount: document.querySelectorAll('[data-context-bar]').length,
        // HAUSV-704: every desktop route uses the persisted sidebar token,
        // including the tablet band that formerly switched to 210/250px.
        sidebarGeometry: (() => {
          const el = document.querySelector('.sidebar'); if (!vis(el)) return null;
          const box = el.getBoundingClientRect(), style = getComputedStyle(el);
          return { x: box.x, width: box.width, paddingLeft: style.paddingLeft, paddingRight: style.paddingRight, expectedWidth: parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--sidebar-w')) };
        })(),
          mobile: vis(document.querySelector('[data-context-bar] .context-navigation')),
          menu: vis(document.querySelector('[data-context-bar] .context-navigation > summary')),
          links: document.querySelectorAll('.nav a').length,
          headTop: h ? Math.round(h.getBoundingClientRect().top) : null,
          templ: [...document.body.attributes].some((a) => a.name.startsWith('data-templ')),
        };
      }, shown.toString());
      results.push({ route: name, width: w, ...state });
    }
  }
}

// ── Tablet portrait and the coarse sweep (HAUSV-640) ─────────────────────────
// Between 761 and ~1023px the shell is consistent and the page does not scroll
// sideways, so everything above is green — yet the two-column pages kept a
// ≥290px side column: the Portfolio's Häuser card was 146px wide at 761px, the
// dense Hausüberblick's Anliegen column narrower still, and the inbox hid part
// of its case pane behind overflow:hidden. At every tablet width, per route:
// no sideways scroll, every primary column present, at least MIN_COLUMN wide
// and inside the viewport, and the inbox grid not clipping a pane. A coarse
// sweep from 320 to 1920px repeats the fit checks (sideways scroll, clipped
// inbox pane) on the same routes.
const MIN_COLUMN = 320;
const TABLET_WIDTHS = [761, 768, 800, 810, 820, 834, 900, 1023];
const SWEEP_WIDTHS = Array.from({ length: 41 }, (_, i) => 320 + i * 40);

// The admin's Hausüberblick is the dense working grid only while something is
// open; on the freshly seeded fixture nothing is, and the calm page has no
// side column to squeeze. File one resident issue first, the way the role QA
// does, so the probe measures the layout it was written for — and fail loudly
// below if the page still comes back calm.
{
  const c2 = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'de-AT', timezoneId: 'Europe/Vienna' });
  const p = await c2.newPage();
  await p.goto(`${tenantURL}/`, { waitUntil: 'networkidle' });
  await p.fill('input[name="email"]', 'resident@example.com');
  await p.click('form[action$="/auth/request"] button');
  await p.waitForSelector('a.dev-link', { timeout: 10_000 });
  await p.goto(new URL(await p.getAttribute('a.dev-link', 'href'), `${tenantURL}/`).href, { waitUntil: 'networkidle' });
  await p.goto(`${tenantURL}/app/anliegen`, { waitUntil: 'networkidle' });
  const panel = p.locator('#issue-new');
  if (await panel.evaluate((el) => el.tagName === 'DETAILS' && !el.open)) await panel.locator(':scope > summary').click();
  const form = p.locator('form[data-issue-wizard]');
  await form.locator('textarea[name="body"]').fill('Tablet-Probe: Kellerlicht flackert. Bitte im Haus prüfen.');
  await form.locator('input[name="location_detail"]').fill('Keller, neben dem Fahrradraum');
  await form.locator('[data-issue-step="describe"] [data-issue-next]').click();
  await form.locator('button[type="submit"]').click();
  await p.waitForURL(/\/app\/anliegen\/.+\?created=1$/, { timeout: 10_000 });
  await c2.close();
}
// A context of its own: the energy block above signed the shared context in
// as the cockpit owner, who has no Verwaltung, so the admin has to sign in
// again here.
const fitCtx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'de-AT', timezoneId: 'Europe/Vienna' });
const tp = await fitCtx.newPage();
await tp.goto(`${tenantURL}/`, { waitUntil: 'networkidle' });
await tp.fill('input[name="email"]', 'admin@example.com');
await tp.click('form[action$="/auth/request"] button');
await tp.waitForSelector('a.dev-link', { timeout: 10_000 });
await tp.goto(new URL(await tp.getAttribute('a.dev-link', 'href'), `${tenantURL}/`).href, { waitUntil: 'networkidle' });
// The inbox needs one open item so the case pane exists; a phone note is the
// only intake the fixture can produce without a mailbox.
await tp.goto(`${tenantURL}/app/verwaltung/posteingang`, { waitUntil: 'load' });
{
  const phone = tp.locator('.queue-tools > details').filter({ has: tp.locator('form.phone-panel') });
  await phone.locator(':scope > summary').click();
  const form = tp.locator('form.phone-panel');
  await form.locator('select[name="house"]').selectOption({ index: 1 });
  await form.locator('[name="from_name"]').fill('Tablet Probe');
  await form.locator('[name="subject"]').fill('Tablet-Probe: Anruf wegen Kellerlicht');
  await form.locator('[name="body"]').fill('Bewohnerin meldet flackerndes Licht im Keller.');
  // The header panel scrolls independently, so the real submit stays reachable.
  await form.locator('button[type="submit"]').click();
  await tp.waitForURL(/\/app\/verwaltung\/posteingang/, { timeout: 10_000 });
  await tp.waitForSelector('a.queue-row', { timeout: 10_000 });
}
const caseHref = await tp.getAttribute('a.queue-row', 'href');

// Primary columns per route — the pages' own layout hooks:
//   portfolio          .portfolio-houses  the Häuser card     .portfolio-side  Heute/Zuletzt
//   home               .issues-card       the Anliegen column .side-stack      Termine/Aushang
//   inbox              .queue             the queue pane      .case/.empty-case the case pane
//                      (the queue page opens its first item, so it is usually .case)
//   inbox-case         .queue             the queue pane      .case            the open case
//   verwaltung-settings .settings-card    first settings card
//   annual-statement   .panel.workspace   first workspace section
// A hidden column is allowed — the master-detail inbox shows one pane at a
// time — a missing one, or none visible at all, is not.
const FIT_ROUTES = [
  ['portfolio', '/app/verwaltung', ['.portfolio-houses', '.portfolio-side']],
  ['home', '/app', ['.portal-home-desktop .working-grid>.issues-card', '.portal-home-desktop .working-grid>.side-stack']],
  ['inbox', '/app/verwaltung/posteingang', ['.inbox-grid>.queue', '.inbox-grid>.case,.inbox-grid>.empty-case']],
  ['inbox-case', caseHref, ['.inbox-grid>.queue', '.inbox-grid>.case']],
  ['verwaltung-settings', '/app/verwaltung/einstellungen', ['.verwaltung-settings .settings-card']],
  ['annual-statement', '/app/settings/annual-statement', ['.panel.workspace']],
];

const fit = [];
const FIT_WIDTHS = [...new Set([...TABLET_WIDTHS, ...SWEEP_WIDTHS])].sort((a, b) => a - b);
for (const [name, route, selectors] of FIT_ROUTES) {
  await tp.goto(`${tenantURL}${route}`, { waitUntil: 'load' });
  for (const w of FIT_WIDTHS) {
    await tp.setViewportSize({ width: w, height: 900 });
    const state = await tp.evaluate(([sels]) => {
      const vis = (el) => !!el && getComputedStyle(el).display !== 'none' && el.getClientRects().length > 0;
      const grid = document.querySelector('.inbox-grid');
      return {
        innerWidth: window.innerWidth,
        scrollWidth: document.documentElement.scrollWidth,
        dense: !!document.querySelector('.portal-home-desktop .working-grid'),
        // overflow:hidden swallows the overflow of a too-wide grid; scrollWidth still sees it.
        clipped: grid && getComputedStyle(grid).display === 'grid' ? Math.max(0, grid.scrollWidth - grid.clientWidth) : 0,
        columns: sels.map((sel) => {
          const el = document.querySelector(sel);
          if (!el) return { sel, state: 'missing' };
          if (!vis(el)) return { sel, state: 'hidden' };
          const b = el.getBoundingClientRect();
          return { sel, state: 'shown', width: Math.round(b.width), right: Math.round(b.right) };
        }),
      };
    }, [selectors]);
    fit.push({ route: name, width: w, tablet: TABLET_WIDTHS.includes(w), ...state });
  }
}

// HAUSV-721: the case column's sticky action bar must end inside the viewport on
// common laptop heights (14" MacBook Pro = 1512×982). The columns size themselves
// to the space below the page head, so the bar is never cut off.
const ACTION_HEIGHTS = [700, 800, 900, 982];
const ACTION_WIDTHS = [1101, 1280, 1440, 1512];
const actionBars = [];
await tp.goto(`${tenantURL}/app/verwaltung/posteingang`, { waitUntil: 'load' });
// The action bar exists only on a case that carries a suggestion; the fixture seeds several.
const suggestedHref = await tp.evaluate(() => [...document.querySelectorAll('a.queue-row')].find((a) => /Vorschlag liegt vor/.test(a.textContent))?.getAttribute('href') || null);
await tp.goto(`${tenantURL}${suggestedHref || caseHref}`, { waitUntil: 'load' });
for (const w of ACTION_WIDTHS) {
  for (const h of ACTION_HEIGHTS) {
    await tp.setViewportSize({ width: w, height: h });
    await tp.waitForTimeout(60);
    const state = await tp.evaluate(() => {
      // The CI fixture has no mailbox, so its only case is a phone note without a
      // suggestion and therefore without the action bar. The panes must still end
      // inside the viewport (that is what cut the bar off), and when the bar or
      // any primary action is present it must be visible and clickable.
      const queue = document.querySelector('.inbox-grid>.queue');
      const pane = document.querySelector('.inbox-grid>.case');
      if (!queue || !pane) return { missing: true };
      const bar = pane.querySelector('.action-bar');
      const primary = pane.querySelector('.action-bar .case-button.primary, .case-button.primary');
      const q = queue.getBoundingClientRect(), c = pane.getBoundingClientRect();
      const out = { queueBottom: Math.round(q.bottom), caseBottom: Math.round(c.bottom), caseHeight: Math.round(c.height), innerHeight: window.innerHeight, hasBar: !!bar };
      if (bar) { const b = bar.getBoundingClientRect(); out.barTop = Math.round(b.top); out.barBottom = Math.round(b.bottom); }
      if (primary) { const p = primary.getBoundingClientRect(); const hit = document.elementFromPoint(p.left + p.width / 2, p.top + p.height / 2); out.primaryBottom = Math.round(p.bottom); out.primaryHit = !!hit && primary.contains(hit); }
      return out;
    });
    actionBars.push({ width: w, height: h, ...state });
  }
}

await browser.close();

let failures = 0;
console.log(`${'width'.padStart(6)}  ${'shell'.padEnd(8)} routes`);
for (const w of WIDTHS) {
  const all = results.filter((r) => r.width === w);
  const wrongRenderer = all.filter((r) => !r.templ);
  const none = all.filter((r) => (!r.sidebar && !r.mobile) || r.contextCount !== 1);
  const both = all.filter((r) => r.sidebar && r.mobile);
  const desktop = all.filter((r) => r.sidebar && !r.mobile).map((r) => r.route);
  const mob = all.filter((r) => r.mobile && !r.sidebar).map((r) => r.route);
  const noLinks = all.filter((r) => r.links === 0);
  const wrongGeometry = all.filter(r => r.sidebarGeometry && (Math.abs(r.sidebarGeometry.x) > 1 || Math.abs(r.sidebarGeometry.width - r.sidebarGeometry.expectedWidth) > 1 || r.sidebarGeometry.paddingLeft !== '18px' || r.sidebarGeometry.paddingRight !== '18px'));
  if (wrongGeometry.length) { failures += wrongGeometry.length; console.log(`         inconsistent sidebar width/padding: ${wrongGeometry.map(r => r.route).join(', ')}`); }
  const noMenu = all.filter((r) => r.mobile && !r.menu);
  const buried = all.filter((r) => r.mobile && r.headTop !== null && r.headTop > 0);
  const skipShown = all.filter((r) => r.skipBottom !== null && r.skipBottom > 0);

  let kind = desktop.length && mob.length ? 'SPLIT' : (mob.length ? 'mobile' : 'desktop');
  console.log(`${String(w).padStart(6)}  ${kind.padEnd(8)} ${desktop.length} desktop / ${mob.length} mobile`);

  for (const [label, list] of [['no navigation at all', none], ['both shells at once', both],
                               ['no nav links', noLinks], ['mobile head without a usable menu', noMenu],
                               ['mobile header pushed below the top', buried.map((r) => ({ ...r, route: `${r.route} (y=${r.headTop})` }))],
                               ['skip link visible without focus', skipShown.map((r) => ({ ...r, route: `${r.route} (bottom=${r.skipBottom})` }))]]) {
    if (list.length) { failures += list.length; console.log(`         ${label}: ${list.map((r) => r.route).join(', ')}`); }
  }
  if (desktop.length && mob.length) {
    failures++;
    console.log(`         routes disagree — desktop: ${desktop.join(', ')}`);
    console.log(`                           mobile: ${mob.join(', ')}`);
  }
  if (wrongRenderer.length) {
    failures += wrongRenderer.length;
    console.log(`         wrong renderer: ${wrongRenderer.map((r) => r.route).join(', ')}`);
  }

  // The home body must follow the shell: phone body below the breakpoint, the
  // desktop grid above it, never both, never neither — and no sideways scroll.
  const homes = all.filter((r) => r.home);
  const phoneShell = mob.length > 0 && desktop.length === 0;
  const blankHome = homes.filter((r) => phoneShell
    ? !(r.home.mobileShown && r.home.mobileHeight >= 200 && !r.home.desktopShown)
    : !(r.home.desktopShown && !r.home.mobileShown));
  const sideways = homes.filter((r) => r.home.scrollWidth > r.home.innerWidth);
  if (blankHome.length) {
    failures += blankHome.length;
    console.log(`         home body does not match the ${phoneShell ? 'phone' : 'desktop'} shell: ${blankHome.map((r) => `${r.route} (mobile ${r.home.mobileShown ? r.home.mobileHeight + 'px' : 'hidden'}, desktop ${r.home.desktopShown ? 'shown' : 'hidden'})`).join(', ')}`);
  }
  if (sideways.length) {
    failures += sideways.length;
    console.log(`         home scrolls sideways: ${sideways.map((r) => `${r.route} (${r.home.scrollWidth} > ${r.home.innerWidth})`).join(', ')}`);
  }
}

// The fit contract. Every width: no sideways scroll, no clipped inbox pane.
// Tablet widths additionally: the dense grid is on the page, every primary
// column is present, the visible ones are at least MIN_COLUMN wide and end
// inside the viewport, and at least one of them is visible.
const fitProblems = (r) => {
  const problems = [];
  if (r.scrollWidth > r.innerWidth) problems.push(`scrolls sideways (${r.scrollWidth} > ${r.innerWidth})`);
  if (r.clipped > 1) problems.push(`inbox grid clips ${r.clipped}px of its panes`);
  if (!r.tablet) return problems;
  if (r.route === 'home' && !r.dense) problems.push('Hausüberblick is calm, not the dense grid — nothing to measure');
  const shown = r.columns.filter((c) => c.state === 'shown');
  for (const c of r.columns) if (c.state === 'missing') problems.push(`${c.sel} is missing`);
  if (!shown.length && r.columns.some((c) => c.state === 'hidden')) problems.push('no primary column is visible');
  for (const c of shown) {
    if (c.width < MIN_COLUMN) problems.push(`${c.sel} is ${c.width}px wide (< ${MIN_COLUMN}px)`);
    if (c.right > r.innerWidth + 1) problems.push(`${c.sel} runs past the viewport (right edge ${c.right} > ${r.innerWidth})`);
  }
  return problems;
};

console.log(`\ntablet portrait — primary columns ≥ ${MIN_COLUMN}px, no sideways scroll, no clipped inbox pane`);
for (const w of TABLET_WIDTHS) {
  const rows = fit.filter((r) => r.width === w);
  console.log(`${String(w).padStart(6)}  ${rows.map((r) => `${r.route}[${r.columns.map((c) => (c.state === 'shown' ? c.width : c.state)).join('/')}]`).join(' ')}`);
  for (const r of rows) {
    for (const problem of fitProblems(r)) { failures++; console.log(`         ${r.route}: ${problem}`); }
  }
}
{
  const sweep = fit.filter((r) => !r.tablet);
  const bad = sweep.flatMap((r) => fitProblems(r).map((problem) => `${r.width}px ${r.route}: ${problem}`));
  failures += bad.length;
  console.log(`\nsweep ${SWEEP_WIDTHS[0]}–${SWEEP_WIDTHS[SWEEP_WIDTHS.length - 1]}px in steps of 40 (${SWEEP_WIDTHS.length} widths × ${FIT_ROUTES.length} routes): ${bad.length ? bad.length + ' problem(s)' : 'fits'}`);
  for (const line of bad) console.log(`         ${line}`);
}

{
  const bad = actionBars.flatMap((r) => {
    const problems = [];
    if (r.missing) problems.push('queue or case pane missing');
    else {
      if (r.caseBottom > r.innerHeight) problems.push(`case pane ends at ${r.caseBottom}px, viewport is ${r.innerHeight}px`);
      if (r.queueBottom > r.innerHeight) problems.push(`queue ends at ${r.queueBottom}px, viewport is ${r.innerHeight}px`);
      if (r.hasBar && r.barBottom > r.innerHeight) problems.push(`action bar ends at ${r.barBottom}px, viewport is ${r.innerHeight}px`);
      if (r.hasBar && r.barTop < 0) problems.push(`action bar starts above the viewport (${r.barTop}px)`);
      if (r.primaryHit === false) problems.push(`primary button is covered or ends at ${r.primaryBottom}px`);
    }
    return problems.map((problem) => `${r.width}×${r.height}: ${problem}`);
  });
  failures += bad.length;
  console.log(`\ninbox action bar at ${ACTION_WIDTHS.length} widths × ${ACTION_HEIGHTS.length} heights: ${bad.length ? bad.length + ' problem(s)' : 'always inside the viewport'}`);
  for (const line of bad) console.log(`         ${line}`);
}

if (outJson) await writeFile(outJson, JSON.stringify({ shell: results, fit }, null, 2));
console.log(`\n${results.length} route/width combinations checked, ${fit.length} fit measurements`);
if (failures) { console.log(`FAIL — ${failures} problem(s)`); process.exit(1); }
console.log('PASS — every route has exactly one navigation at every width, it sits at the top, they all agree, and the pages fit');
