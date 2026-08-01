#!/usr/bin/env node
// Capture HTML + PNG snapshots of every page, for one running build.
//
// Why both: screenshots catch visual regressions a human would see; the HTML is
// the precise oracle — a pure code-motion refactor must render byte-identical
// markup, and a diff tells you exactly which element changed rather than "some
// pixels moved".
//
//   node capture.mjs <baseURL> <outDir>
//
// Chromium: uses the system browser via PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH
// (same approach as nixcfg templates/playwright-qa), so no 150 MB download.

import { chromium } from 'playwright';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';

const baseURL = process.argv[2];
const outDir = process.argv[3];
if (!baseURL || !outDir) {
  console.error('usage: capture.mjs <baseURL> <outDir>');
  process.exit(1);
}

// Personas cover the role-gated UI: an Admin sees nav items an Eigentümer never
// does, so screenshotting only one role would leave most templates uncovered.
const PERSONAS = [
  { name: 'admin', email: 'admin@example.com' },
  { name: 'verwalter', email: 'verwalter@example.com' },
  { name: 'owner', email: 'owner@example.com' },
  { name: 'resident', email: 'resident@example.com' },
];

const ROUTES = [
  ['portal', '/app'],
  ['announcements', '/app/announcements'],
  ['events', '/app/events'],
  ['issues', '/app/anliegen'],
  ['documents', '/app/dokumente'],
  ['ballots', '/app/abstimmungen'],
  ['handovers', '/app/uebergaben'],
  ['contacts', '/app/kontakte'],
  ['settings', '/app/settings'],
  ['settings-profile', '/app/settings/profile'],
  ['settings-notifications', '/app/settings/notifications'],
  ['settings-building', '/app/settings/building'],
  ['settings-users', '/app/settings/users'],
  ['audit', '/app/audit'],
];

// Volatile by design — must be neutralised or every run diffs against itself.
function normalise(html) {
  return html
    // assetVersion() mixes in a process-start nonce
    .replace(/\?v=[^"'&\s]+/g, '?v=NONCE')
    // magic-link / CSRF-ish tokens
    .replace(/token=[A-Za-z0-9_\-.]+/g, 'token=TOKEN')
    // absolute timestamps and relative "vor x Minuten"
    .replace(/\b\d{1,2}\.\s?\w+\s?\d{4},?\s?\d{1,2}:\d{2}\b/g, 'TIMESTAMP')
    .replace(/\b\d{4}-\d{2}-\d{2}T[\d:.+Z-]+/g, 'TIMESTAMP')
    .replace(/vor\s+\d+\s+\w+/g, 'vor N EINHEIT')
    .replace(/localhost:\d+/g, 'localhost:PORT')
    // Calendar feed URLs embed a SIGNED token whose payload carries iat
    // (issued-at), so the whole path changes every run.
    .replace(/\/calendar\/[^"'\s]+\.ics/g, '/calendar/FEEDTOKEN.ics')
    // Clock times: logging in writes audit rows stamped with the current
    // minute, so two runs a minute apart differ for no code reason.
    .replace(/\b\d{1,2}:\d{2}(:\d{2})?\b/g, 'HH:MM');
}

const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;
const headless = process.env.CI === 'true' || process.env.HV_QA_HEADLESS !== 'false';
const browser = await chromium.launch({
  headless,
  ...(executablePath ? { executablePath } : {}),
});

let captured = 0;
for (const persona of PERSONAS) {
  const ctx = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
    // Freeze anything locale/tz dependent.
    locale: 'de-AT',
    timezoneId: 'Europe/Vienna',
  });
  const page = await ctx.newPage();

  // LOCAL_DEV_LOGIN: submitting the email renders a dev magic-link inline
  // instead of mailing it; following that link establishes the session.
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  await page.fill('input[name="email"]', persona.email);
  await page.click('form[action="/auth/request"] button[type="submit"], form[action="/auth/request"] button');
  await page.waitForSelector('a.dev-link', { timeout: 10_000 });
  const link = await page.getAttribute('a.dev-link', 'href');
  await page.goto(link, { waitUntil: 'networkidle' });

  const dir = path.join(outDir, persona.name);
  await mkdir(dir, { recursive: true });

  for (const [name, route] of ROUTES) {
    const res = await page.goto(`${baseURL}${route}`, { waitUntil: 'networkidle' });
    const status = res ? res.status() : 0;
    const html = normalise(await page.content());
    await writeFile(path.join(dir, `${name}.html`), `<!-- status:${status} -->\n${html}\n`);

    // Neutralise volatile TEXT in the DOM before the screenshot. The HTML
    // normaliser only fixes the saved markup — the pixels would still render a
    // real clock ("14:41" vs "14:42" one minute later) and diff forever.
    await page.evaluate(() => {
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      const times = /\b\d{1,2}:\d{2}(:\d{2})?\b/g;
      for (let n = walker.nextNode(); n; n = walker.nextNode()) {
        if (times.test(n.nodeValue)) n.nodeValue = n.nodeValue.replace(times, 'HH:MM');
      }
    });
    await page.screenshot({ path: path.join(dir, `${name}.png`), fullPage: true });
    captured++;
    process.stdout.write(`  ${persona.name}/${name} → ${status}\n`);
  }
  await ctx.close();
}

await browser.close();
console.log(`captured ${captured} pages into ${outDir}`);
