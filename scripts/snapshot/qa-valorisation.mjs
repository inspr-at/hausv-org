#!/usr/bin/env node
// Valorisation demo: preview, approval, archived PDF and owner denial. Headless, against one local rig.
//   node qa-valorisation.mjs http://localhost:8309 /path/to/artifacts

import { existsSync, mkdirSync } from 'node:fs';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
const artifactDir = process.argv[3] || '';
if (!baseURL) {
  console.error('usage: qa-valorisation.mjs <baseURL> [artifact-dir]');
  process.exit(1);
}
const parsedBaseURL = new URL(baseURL);
if (!['localhost', '127.0.0.1', '::1'].includes(parsedBaseURL.hostname)) {
  throw new Error('qa-valorisation only runs against localhost');
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

async function login(email) {
  const context = await browser.newContext({ locale: 'de-AT', timezoneId: 'Europe/Vienna' });
  const page = await context.newPage();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  const details = page.locator('details:has(form[action$="/auth/request"])');
  if (await details.count()) await details.evaluate((node) => { node.open = true; });
  await page.locator('input[name="email"]').fill(email);
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible', timeout: 15000 });
  const href = await devLink.getAttribute('href');
  if (!href) fail(`Kein Anmeldelink für ${email}`);
  const target = new URL(href, baseURL);
  target.protocol = parsedBaseURL.protocol;
  target.hostname = parsedBaseURL.hostname;
  target.port = parsedBaseURL.port;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  if (!page.url().includes('/app')) fail(`Anmeldung für ${email} endete auf ${page.url()}`);
  await page.close();
  return context;
}

async function shot(page, name) {
  if (!artifactDir) return;
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: `${artifactDir}/${name}.png`, fullPage: true });
  process.stdout.write(`  screenshot ${artifactDir}/${name}.png\n`);
}

try {
  const manager = await login('vera.verwalter@musterstadt.example');
  const page = await manager.newPage();
  const route = `${baseURL}/app/settings/valorisation`;
  await page.setViewportSize({ width: 1440, height: 900 });
  let response = await page.goto(`${route}?new=1&preview=1&effective_on=2026-04-01&house=janusbergweg-123`, { waitUntil: 'networkidle' });
  if (response?.status() !== 200) fail(`Vorschau HTTP ${response?.status()}`);
  const preview = page.locator('article[data-run-id=""]');
  for (const group of ['ready', 'unchanged', 'exception']) {
    if (await preview.locator(`[data-group="${group}"]`).count() !== 1) fail(`Gruppe ${group} fehlt`);
  }
  if (await preview.locator('[data-group="ready"] > .vr-item').count() < 6) fail('Weniger als sechs bereite Verträge');
  const top1 = preview.locator('[data-lease-id="lease-top-1"]');
  await top1.locator(':scope > summary').click();
  const used = top1.getByRole('table', { name: 'Verwendete Indexwerte', exact: true });
  if (await used.locator('tbody tr').count() !== 5) fail('E1: erwartete Basis, Auslöser und drei Jahresmittel fehlen');
  if (await top1.getByRole('table', { name: 'Geprüfte Monatsreihe und Jahresmittel' }).isVisible()) fail('Volle Indexreihe ist nicht eingeklappt');
  await top1.locator('summary').filter({ hasText: 'Alle Indexwerte anzeigen' }).click();
  if (await top1.getByRole('table', { name: 'Geprüfte Monatsreihe und Jahresmittel' }).locator('tbody tr').count() <= 5) fail('Volle Indexreihe fehlt');
  const body = await preview.innerText();
  for (const value of ['1.040,28', '1.017,35', '21.04.2026', '05.05.2026', 'Top 1 · Eva Huber', 'Schwelle 5 % überschritten: +5,02 %', '+3,55412 %']) {
    if (!body.includes(value)) fail(`Vorschau: ${value} fehlt`);
  }
  if (/\btop-\d|\d+\/\d+ %|\d+\.\d+ %|percent|Kurve exakt/.test(body)) fail('Technische Zahlen oder IDs in der Vorschau');
  // The seeded April draft is reviewable even when the system clock is later.
  await page.goto(route, { waitUntil: 'networkidle' });
  const first = page.locator('article[data-run-id]').first();
  const runID = await first.getAttribute('data-run-id');
  if (!runID) fail('Demo-Lauf fehlt');
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await first.locator('.vr-item').first().evaluate(node => { node.open = true; });
    await shot(page, `valorisation-${width}`);
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    if (overflow > 1) fail(`${width}px: Überlauf ${overflow}px`);
  }
  const itemIDs = await first.locator('[data-group="exception"] > .vr-item').evaluateAll(nodes => nodes.map(node => node.id.slice(5)));
  for (const itemID of itemIDs) {
    const detail = page.locator(`#item-${itemID}`);
    await detail.evaluate(node => { node.open = true; });
    await detail.locator('.vr-manual > summary').click();
    await detail.getByLabel('Begründung für Ausschluss').fill('Browserprüfung: gesonderte Vertragsprüfung');
    await detail.getByRole('button', { name: 'Ausschließen', exact: true }).click();
    await page.waitForLoadState('networkidle');
  }
  await page.locator(`#run-${runID}`).getByRole('button', { name: 'Freigeben und archivieren' }).click();
  await page.waitForLoadState('networkidle');
  const approved = page.locator(`#run-${runID}`);
  if (!(await approved.innerText()).includes('Freigegeben')) fail('Lauf nicht freigegeben');
  await approved.locator('.vr-item').evaluateAll(nodes => nodes.forEach(node => { node.open = false; }));
  const pdfLink = approved.getByRole('link', { name: 'Schreiben (PDF)', exact: true }).first();
  if (!(await pdfLink.isVisible())) fail('Schreiben ist bei eingeklappter Zeile nicht erreichbar');
  if (await approved.locator('input[name="reason"]').isVisible()) fail('Stornogrund wird ohne Bestätigungsschritt angezeigt');
  await approved.locator('.vr-cancel > summary').click();
  const cancelReason = approved.getByLabel('Stornogrund');
  if (!(await cancelReason.isVisible()) || !(await cancelReason.evaluate(node => node.required && node.validity.valueMissing))) fail('Stornierung benötigt keinen sichtbaren Pflichtgrund');
  await approved.locator('.vr-cancel > summary').click();
  const href = await pdfLink.getAttribute('href');
  if (!href) fail('PDF-Link fehlt');
  const pdf = await manager.request.get(new URL(href, baseURL).href, { headers: { Connection: 'close' } });
  if (pdf.status() !== 200 || !(await pdf.body()).subarray(0, 5).equals(Buffer.from('%PDF-'))) fail('Archiv-PDF fehlt');
  await shot(page, 'valorisation-approved-390');
  await manager.close();
  const owner = await login('alina.eigentuemer@musterstadt.example');
  const ownerPage = await owner.newPage();
  response = await ownerPage.goto(route, { waitUntil: 'networkidle' });
  if (response?.status() !== 403) fail(`Eigentümer HTTP ${response?.status()}, erwartet 403`);
  await shot(ownerPage, 'valorisation-owner-denied');
  await owner.close();
  process.stdout.write('qa-valorisation ok\n');
} finally {
  await browser.close();
}
