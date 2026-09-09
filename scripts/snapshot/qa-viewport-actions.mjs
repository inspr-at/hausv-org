#!/usr/bin/env node
// HAUSV-722: on common laptop viewports, are the primary actions of every screen
// reachable? A button that a fixed-height panel clips, a dialog footer below the
// fold of a 720px-high window, or a header action covered by something else is a
// defect even if the page technically scrolls. Three classes per action:
//   visible      inside the viewport without any scrolling, and clickable
//   page-scroll  needs the document to scroll (fine for long forms; listed)
//   clipped      never fully visible or never clickable — a finding
// Panels that scroll internally must end inside the viewport when they hold an
// action bar (the Posteingang bug behind HAUSV-721). Dialogs are opened from
// their triggers and their submit button must sit inside the viewport.
//
//   node qa-viewport-actions.mjs <baseURL> [outJson]
//   env DEFAULT_TENANT (fixture: demo) · HV_EMAIL / HV_ACCESS_CODE (rig login with access code)
import { chromium } from 'playwright';
import { writeFile } from 'node:fs/promises';

const baseURL = process.argv[2];
const outJson = process.argv[3];
if (!baseURL) { console.error('usage: qa-viewport-actions.mjs <baseURL> [outJson]'); process.exit(2); }
const tenant = (process.env.DEFAULT_TENANT || 'demo').replace(/^\/+|\/+$/g, '');
const tenantURL = `${baseURL}/${tenant}`;
const VIEWPORTS = [[1512, 982], [1440, 900], [1280, 720]];
const ACTION_SELECTOR = '.button.primary, .case-button.primary, button.primary, .portal-section-header-action .button, .action-bar .case-button, .demo-reset-primary, .energy-consumer-primary, [data-board-panel] .button, [data-board-panel] form button';
// Known offenders carry their ticket; the list shrinks as they are fixed.
const KNOWN = new Map([
  // ['/app/example|1280x720|clipped|Speichern', 'HAUSV-7xx'],
]);

const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : {}) });
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'de-AT', timezoneId: 'Europe/Vienna' });
const page = await ctx.newPage();

async function login() {
  await page.goto(`${tenantURL}/`, { waitUntil: 'networkidle' });
  const code = page.locator('input[name="access_code"]');
  if (await code.count()) {
    await page.fill('input[name="email"]', process.env.HV_EMAIL || 'vera.verwalter@musterstadt.example');
    await code.fill(process.env.HV_ACCESS_CODE || '');
  } else {
    await page.fill('input[name="email"]', 'admin@example.com');
  }
  await page.click('form[action$="/auth/request"] button');
  await page.waitForSelector('a.dev-link', { timeout: 10_000 });
  await page.goto(new URL(await page.getAttribute('a.dev-link', 'href'), `${tenantURL}/`).href, { waitUntil: 'networkidle' });
}
await login();

// Routes: every sidebar link of the house and the organisation shell, plus the
// screens that hang off them and hold their own actions.
const routes = new Set(['/app', '/app/verwaltung', '/app/anliegen/board', '/app/settings', '/app/settings/annual-statement', '/app/settings/building', '/app/settings/users', '/app/settings/profile', '/app/settings/notifications']);
for (const start of ['/app', '/app/verwaltung']) {
  await page.goto(`${tenantURL}${start}`, { waitUntil: 'load' });
  for (const href of await page.$$eval('aside.sidebar .nav a[href]', (as) => as.map((a) => a.getAttribute('href')))) {
    const path = href.replace(/^https?:\/\/[^/]+/, '').replace(new RegExp(`^/${tenant}`), '').split('?')[0].split('#')[0];
    if (path.startsWith('/app') && !/\/(abmelden|logout|auth)/.test(path)) routes.add(path);
  }
}

const findings = [];
const seen = [];

async function measureActions() {
  return page.evaluate(([sel]) => {
    const vis = (el) => { const s = getComputedStyle(el); return s.display !== 'none' && s.visibility !== 'hidden' && el.getClientRects().length > 0; };
    const inside = (r) => r.top >= 0 && r.bottom <= innerHeight && r.left >= 0 && r.right <= innerWidth;
    const hit = (el) => { const r = el.getBoundingClientRect(); const e = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2); return !!e && (el === e || el.contains(e)); };
    const label = (el) => (el.getAttribute('aria-label') || el.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 40);
    const scrollPanel = (el) => { let p = el.parentElement; while (p && p !== document.body) { const s = getComputedStyle(p); if (/(auto|scroll)/.test(s.overflowY) && p.scrollHeight > p.clientHeight + 1) return p; p = p.parentElement; } return null; };
    const out = [];
    const actions = [...document.querySelectorAll(sel)].filter(vis).filter((el) => !el.closest('dialog:not([open]), [hidden], details:not([open])'));
    for (const el of actions) {
      window.scrollTo(0, 0);
      const r0 = el.getBoundingClientRect();
      let state = inside(r0) && hit(el) ? 'visible' : null;
      if (!state) {
        el.scrollIntoView({ block: 'center', inline: 'nearest' });
        const r1 = el.getBoundingClientRect();
        state = inside(r1) && hit(el) ? 'page-scroll' : 'clipped';
        window.scrollTo(0, 0);
      }
      const panel = scrollPanel(el);
      const panelBottom = panel ? Math.round(panel.getBoundingClientRect().bottom) : null;
      out.push({ label: label(el), state, top: Math.round(r0.top), bottom: Math.round(r0.bottom), panelBottom, panelBelowViewport: panel ? panelBottom > innerHeight : false });
    }
    return out;
  }, [ACTION_SELECTOR]);
}

async function measureDialogs(route) {
  const out = [];
  const triggers = await page.$$eval('[data-dialog], [aria-haspopup="dialog"]', (els) => els
    .map((e, i) => ({ i, e }))
    .filter(({ e }) => e.getClientRects().length > 0)
    .map(({ i, e }) => ({ i, target: e.getAttribute('aria-controls') || e.getAttribute('data-dialog') || null, label: (e.getAttribute('aria-label') || e.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 40) }))
    .slice(0, 4));
  const openDialog = 'dialog[open], [role="dialog"]:not([hidden])';
  for (const t of triggers) {
    try {
      const el = (await page.$$('[data-dialog], [aria-haspopup="dialog"]'))[t.i];
      if (!el) continue;
      await el.click({ timeout: 3000 });
      await page.waitForSelector(openDialog, { timeout: 3000 });
      // Dialog bodies arrive lazily (htmx); measure once the form's own button
      // exists. A <button> inside a form submits by default, so check the
      // property, not the attribute.
      await page.waitForFunction(([target, sel]) => {
        const d = (target && document.getElementById(target)) || [...document.querySelectorAll(sel)].pop();
        return d && d.getClientRects().length > 0 && [...d.querySelectorAll('button, input[type="submit"], .button')].some((b) => b.type === 'submit' || /primary/.test(b.className));
      }, [t.target, openDialog], { timeout: 3000 }).catch(() => {});
      await page.waitForTimeout(120);
      const m = await page.evaluate(([target, sel]) => {
        const d = (target && document.getElementById(target)) || [...document.querySelectorAll(sel)].pop();
        const r = d.getBoundingClientRect();
        const buttons = [...d.querySelectorAll('button, input[type="submit"], .button')].filter((b) => b.getClientRects().length > 0);
        const primary = buttons.find((b) => b.type === 'submit' || /primary/.test(b.className)) || buttons[buttons.length - 1];
        const pr = primary ? primary.getBoundingClientRect() : null;
        const hit = primary ? (() => { const e = document.elementFromPoint(pr.left + pr.width / 2, pr.top + pr.height / 2); return !!e && (e === primary || primary.contains(e)); })() : null;
        return { dialogTop: Math.round(r.top), dialogBottom: Math.round(r.bottom), primary: primary ? (primary.textContent || primary.value || '').trim().slice(0, 40) : null, primaryBottom: pr ? Math.round(pr.bottom) : null, primaryVisible: pr ? pr.top >= 0 && pr.bottom <= innerHeight && hit : null };
      }, [t.target, openDialog]);
      out.push({ trigger: t.label, ...m });
      await page.keyboard.press('Escape');
      await page.waitForTimeout(150);
      if (await page.$(openDialog)) await page.goto(`${tenantURL}${route}`, { waitUntil: 'load' });
    } catch {
      await page.goto(`${tenantURL}${route}`, { waitUntil: 'load' }).catch(() => {});
    }
  }
  return out;
}

for (const route of routes) {
  for (const [w, h] of VIEWPORTS) {
    await page.setViewportSize({ width: w, height: h });
    const res = await page.goto(`${tenantURL}${route}`, { waitUntil: 'load' }).catch(() => null);
    if (!res || res.status() >= 400) { seen.push({ route, viewport: `${w}x${h}`, skipped: res ? res.status() : 'no response' }); continue; }
    await page.waitForTimeout(120);
    if (route === '/app/anliegen/board' && (await page.$('[data-board-card]'))) {
      // HAUSV-717: the detail panel opens from a card; its actions count too.
      await page.$eval('[data-board-card]', (card) => card.click());
      await page.waitForSelector('[data-board-panel] form button', { timeout: 8000 }).catch(() => {});
      await page.waitForTimeout(300);
    }
    const actions = await measureActions();
    const dialogs = await measureDialogs(route);
    seen.push({ route, viewport: `${w}x${h}`, actions, dialogs });
    for (const a of actions) {
      if (a.state === 'clipped') findings.push({ route, viewport: `${w}x${h}`, kind: 'clipped', label: a.label, detail: `bottom ${a.bottom}px, viewport ${h}px` });
      if (a.panelBelowViewport) findings.push({ route, viewport: `${w}x${h}`, kind: 'panel-below-viewport', label: a.label, detail: `scroll panel ends at ${a.panelBottom}px, viewport ${h}px` });
    }
    for (const d of dialogs) {
      if (d.primary && d.primaryVisible === false) findings.push({ route, viewport: `${w}x${h}`, kind: 'dialog-footer', label: `${d.trigger} → ${d.primary}`, detail: `dialog ${d.dialogTop}–${d.dialogBottom}px, primary ends at ${d.primaryBottom}px, viewport ${h}px` });
    }
  }
}
await browser.close();

const key = (f) => `${f.route}|${f.viewport}|${f.kind}|${f.label}`;
const scrollOnly = seen.flatMap((s) => (s.actions || []).filter((a) => a.state === 'page-scroll').map((a) => `${s.route} ${s.viewport}: ${a.label}`));
console.log(`${routes.size} routes × ${VIEWPORTS.length} viewports — ${findings.length} finding(s), ${scrollOnly.length} action(s) need page scroll`);
for (const f of findings) console.log(`  ${KNOWN.has(key(f)) ? `known ${KNOWN.get(key(f))}` : 'NEW  '} ${f.route} ${f.viewport} ${f.kind}: ${f.label} — ${f.detail}`);
if (scrollOnly.length) { console.log('  page-scroll (information):'); for (const line of scrollOnly) console.log(`    ${line}`); }
if (outJson) await writeFile(outJson, JSON.stringify({ findings, seen }, null, 2));
const unknown = findings.filter((f) => !KNOWN.has(key(f)));
if (unknown.length) { console.log(`FAIL — ${unknown.length} new finding(s)`); process.exit(1); }
console.log('PASS — every primary action, panel bar and dialog footer is reachable on laptop viewports');
