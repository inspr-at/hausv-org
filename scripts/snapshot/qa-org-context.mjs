#!/usr/bin/env node
// Organisation views must switch house before opening another house's work.
// Demo seed, headless Chrome:
//   node qa-org-context.mjs http://localhost:8292 /path/to/shots

import { existsSync, mkdirSync } from 'node:fs';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
const artifactDir = process.argv[3] || '';
if (!baseURL) {
  console.error('usage: qa-org-context.mjs <baseURL> [artifact-dir]');
  process.exit(1);
}
const parsedBaseURL = new URL(baseURL);
if (!['localhost', '127.0.0.1', '::1'].includes(parsedBaseURL.hostname)) {
  throw new Error('qa-org-context only runs against localhost');
}
if (artifactDir) mkdirSync(artifactDir, { recursive: true });

const executableCandidates = [
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Chromium.app/Contents/MacOS/Chromium',
].filter(Boolean);
const executablePath = executableCandidates.find((path) => existsSync(path));
const browser = await chromium.launch({
  headless: true,
  args: ['--no-proxy-server'],
  ...(executablePath ? { executablePath } : {}),
});

function fail(message) {
  throw new Error(message);
}

async function shot(page, name) {
  if (!artifactDir) return;
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: `${artifactDir}/${name}.png`, fullPage: true });
  process.stdout.write(`  screenshot ${artifactDir}/${name}.png\n`);
}

async function login() {
  const context = await browser.newContext({ locale: 'de-AT', timezoneId: 'Europe/Vienna' });
  const page = await context.newPage();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  const details = page.locator('details:has(form[action$="/auth/request"])');
  if (await details.count()) await details.evaluate((node) => { node.open = true; });
  await page.locator('input[name="email"]').fill('paul.verwalter@musterstadt.example');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible', timeout: 15000 });
  const href = await devLink.getAttribute('href');
  if (!href) fail('Kein Anmeldelink');
  const target = new URL(href, baseURL);
  target.protocol = parsedBaseURL.protocol;
  target.hostname = parsedBaseURL.hostname;
  target.port = parsedBaseURL.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  if (!page.url().includes('/app')) fail(`Anmeldung endete auf ${page.url()}`);
  await page.close();
  return context;
}

async function houseSlug(page) {
  return page.locator('[data-tenant-slug]').first().getAttribute('data-tenant-slug');
}

async function ensureJanusbergweg(page) {
  if (!page.url().startsWith(baseURL)) {
    await page.goto(`${baseURL}/app`, { waitUntil: 'networkidle' });
  }
  const slug = await houseSlug(page);
  if (slug === 'janusbergweg-123') return;
  if (!slug) fail(`Keine Liegenschaft auf ${page.url()}`);
  await page.evaluate((current) => {
    const form = document.createElement('form');
    form.method = 'post';
    form.action = `/${current}/app/context`;
    for (const [name, value] of [['tenant', 'janusbergweg-123'], ['role', 'Verwalter'], ['next', '/app']]) {
      const input = document.createElement('input');
      input.type = 'hidden';
      input.name = name;
      input.value = value;
      form.appendChild(input);
    }
    document.body.appendChild(form);
    form.submit();
  }, slug);
  await page.waitForLoadState('networkidle');
  if (await houseSlug(page) !== 'janusbergweg-123') fail(`Rückwechsel landete auf ${page.url()}`);
}

try {
  const manager = await login();
  const page = await manager.newPage();
  await page.setViewportSize({ width: 1440, height: 900 });
  await ensureJanusbergweg(page);
  await page.goto(`${baseURL}/app/verwaltung/wertsicherung`, { waitUntil: 'networkidle' });
  if (await page.locator('article[data-run-id]').count() === 0) {
    await page.goto(`${baseURL}/app/verwaltung/einstellungen/demo`, { waitUntil: 'networkidle' });
    await page.getByRole('button', { name: 'Ja, initialisieren', exact: true }).click();
    await page.getByText('Demodaten wurden initialisiert').waitFor({ timeout: 120000 });
    console.log('Demodaten initialisiert');
    await page.goto(`${baseURL}/app/verwaltung/wertsicherung`, { waitUntil: 'networkidle' });
  }
  const run = page.locator('article[data-run-id]').filter({ hasText: 'Musterstraße' }).first();
  if (await run.count() !== 1) fail(`Musterstraße-Wertsicherung fehlt (${await run.count()})`);
  const html = await run.innerHTML();
  if (html.includes('/janusbergweg-123/musterstrasse-12/')) fail('Doppeltes Liegenschafts-Präfix in der Wertsicherung');
  if (await run.locator('form[action*="/approve"]').count()) fail('Freigabeformular einer anderen Liegenschaft wird gerendert');
  const openRun = run.getByRole('button', { name: 'In Liegenschaft öffnen', exact: true });
  if (!(await openRun.isVisible())) fail('Wertsicherung bietet keinen Wechsel in die Liegenschaft');
  await openRun.click();
  await page.waitForLoadState('networkidle');
  if (!page.url().includes('/musterstrasse-12/app/settings/valorisation')) fail(`Wertsicherung landete auf ${page.url()}`);
  if (await houseSlug(page) !== 'musterstrasse-12') fail('Wertsicherung: falsche Liegenschaft gewählt');
  if (!(await page.locator('[data-house-picker]').first().innerText()).includes('Musterstraße')) fail('Wertsicherung: Umschalter zeigt Musterstraße nicht');
  await shot(page, 'wertsicherung-musterstrasse');
  console.log('Wertsicherung: Musterstraße aus Janusbergweg geöffnet');

  await ensureJanusbergweg(page);
  await page.goto(`${baseURL}/app/verwaltung/jahresabrechnung`, { waitUntil: 'networkidle' });
  const statement = page.locator('li').filter({ hasText: 'Musterstraße' }).first();
  const openStatement = statement.getByRole('button', { name: 'In Liegenschaft öffnen', exact: true });
  if (!(await openStatement.isVisible())) {
    const rows = await page.locator('main li').allInnerTexts();
    fail(`Jahresabrechnung bietet keinen Wechsel. url=${page.url()} rows=${rows.slice(0, 4).join(' || ')}`);
  }
  await openStatement.click();
  await page.waitForLoadState('networkidle');
  if (!page.url().includes('/musterstrasse-12/app/settings/annual-statement')) fail(`Jahresabrechnung landete auf ${page.url()}`);
  if (await houseSlug(page) !== 'musterstrasse-12') fail('Jahresabrechnung: falsche Liegenschaft gewählt');
  await shot(page, 'jahresabrechnung-musterstrasse');
  console.log('Jahresabrechnung: Musterstraße aus Janusbergweg geöffnet');

  await ensureJanusbergweg(page);
  await page.goto(`${baseURL}/janusbergweg-123/app/verwaltung/posteingang`, { waitUntil: 'networkidle' });
  const foreign = page.locator('form.queue-row-form').first();
  if (await foreign.count() !== 1) fail('Posteingang zeigt keinen Fall einer anderen Liegenschaft');
  const tenant = await foreign.locator('input[name="tenant"]').inputValue();
  if (!tenant || tenant === 'janusbergweg-123') fail(`Posteingang-Fall gehört nicht zu einer anderen Liegenschaft: ${tenant}`);
  await foreign.getByRole('button').click();
  await page.waitForLoadState('networkidle');
  if (!page.url().includes(`/${tenant}/app/verwaltung/posteingang/`)) fail(`Posteingang landete auf ${page.url()}`);
  if (await houseSlug(page) !== tenant) fail(`Posteingang: Liegenschaft ${await houseSlug(page)}, erwartet ${tenant}`);
  await shot(page, 'posteingang-andere-liegenschaft');
  console.log(`Posteingang: Fall von ${tenant} geöffnet`);
} finally {
  await browser.close();
}
