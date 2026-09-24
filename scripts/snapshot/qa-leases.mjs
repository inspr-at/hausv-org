#!/usr/bin/env node
// Lease tab, import dry-run and owner denial. Headless, against one local rig.
//   node qa-leases.mjs http://localhost:8315 /path/to/artifacts

import { existsSync, mkdirSync } from 'node:fs';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
const artifactDir = process.argv[3] || '';
if (!baseURL) {
  console.error('usage: qa-leases.mjs <baseURL> [artifact-dir]');
  process.exit(1);
}
const parsedBaseURL = new URL(baseURL);
if (!['localhost', '127.0.0.1', '::1'].includes(parsedBaseURL.hostname)) {
  throw new Error('qa-leases only runs against localhost');
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

const csv = [
  'einheit;mieter_name;vertragsabschluss;beginn;nutzung;mrg;regime;hmz_netto;klausel_typ;index;basis_monat;basis_wert;schwelle;schwelle_art',
  'Top 1;Eva Huber;15.03.2018;01.04.2018;Wohnung;teil;frei;1000,00;vpi_threshold;vpi2020;2024-09;123,6;5;prozent',
  'Gibt es nicht;Niemand;01.01.2020;01.02.2020;Wohnung;voll;frei;800;none;;;;;',
].join('\n');

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
  await page.screenshot({ path: `${artifactDir}/${name}.png`, fullPage: true });
  process.stdout.write(`  screenshot ${artifactDir}/${name}.png\n`);
}

const manager = await login('vera.verwalter@musterstadt.example');
const page = await manager.newPage();
await page.setViewportSize({ width: 1440, height: 900 });
const leaseURL = `${baseURL}/app/settings/building/units/top-1/lease`;
let response = await page.goto(leaseURL, { waitUntil: 'networkidle' });
if (!response || response.status() !== 200) fail(`Mietvertrag HTTP ${response?.status()}`);
const body = await page.locator('body').innerText();
for (const text of ['MieWeG', 'ja', 'Teilanwendung', '1.000,00', 'VPI mit Schwelle', 'Noch keine Wertsicherung', 'Eva Huber']) {
  if (!body.includes(text)) fail(`Mietvertrag zeigt „${text}“ nicht`);
}
await shot(page, 'lease-1440');
const wide = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
if (wide > 1) fail(`1440 px überläuft um ${wide} px`);

await page.setViewportSize({ width: 390, height: 844 });
response = await page.goto(leaseURL, { waitUntil: 'networkidle' });
if (!response || response.status() !== 200) fail(`Mietvertrag mobil HTTP ${response?.status()}`);
await shot(page, 'lease-390');
const narrow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
if (narrow > 1) fail(`390 px überläuft um ${narrow} px`);

await page.setViewportSize({ width: 1280, height: 800 });
await page.goto(`${baseURL}/app/settings/building/leases/import`, { waitUntil: 'networkidle' });
await page.locator('input[name="leases_file"]').setInputFiles({ name: 'mietvertraege.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) });
await page.getByRole('button', { name: 'Prüfen' }).click();
await page.waitForLoadState('networkidle');
const imported = await page.locator('body').innerText();
if (!imported.includes('Einheit ist nicht vorhanden')) fail('Import nennt die unbekannte Einheit nicht');
if (!imported.includes('Indexwert wurde nicht geprüft')) fail('Import nennt index_check_unavailable nicht');
if (imported.includes('wurden übernommen')) fail('Die Prüfung hat gespeichert');
await shot(page, 'import-dry-run');
await manager.close();

const owner = await login('alina.eigentuemer@musterstadt.example');
const ownerPage = await owner.newPage();
response = await ownerPage.goto(leaseURL, { waitUntil: 'networkidle' });
if (!response || response.status() !== 403) fail(`Eigentümer HTTP ${response?.status()}, erwartet 403`);
const denied = await ownerPage.locator('body').innerText();
if (!denied.includes('Verwaltung')) fail('Die Absage nennt die Verwaltung nicht');
await shot(ownerPage, 'owner-denied');
await owner.close();

await browser.close();
process.stdout.write('qa-leases ok\n');
