// HAUSV-795. Smoke the seeded MRG Zinshaus: switch house, annual statement, valorisation preview.
// Usage: node qa-demo-zinshaus.mjs http://localhost:8292 <artifact-dir>
import assert from 'node:assert/strict';
import { access, mkdir, writeFile } from 'node:fs/promises';
import { chromium } from 'playwright';

const [baseURL, out = '../../tmp/qa-demo-zinshaus'] = process.argv.slice(2);
assert(['localhost', '127.0.0.1', '[::1]'].includes(new URL(baseURL).hostname));
await mkdir(out, { recursive: true });
const state = `${out}/session.json`;
const saved = await access(state).then(() => true, () => false);
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : {}) });
const context = await browser.newContext({ locale: 'de-AT', timezoneId: 'Europe/Vienna', ...(saved ? { storageState: state } : {}) });
const page = await context.newPage();
const pageErrors = [];
page.on('pageerror', error => pageErrors.push(error.message));

async function login() {
  await page.goto(`${baseURL}/janusbergweg-123/app`);
  if (await page.locator('input[name="email"]').count()) {
    await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
    await page.locator('input[name="email"]').fill('vera.verwalter@musterstadt.example');
    await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
    await page.locator('a.dev-link').click();
    await page.waitForURL('**/app**');
    await context.storageState({ path: state });
  }
}

try {
  await login();
  await page.goto(`${baseURL}/app/verwaltung/einstellungen/demo`);
  await page.locator('form.demo-reset-card button[type="submit"]').click();
  await page.getByText('Demodaten wurden initialisiert').waitFor({ timeout: 180000 });
  await page.locator('#context-property > summary').click();
  const search = page.locator('#context-property-search');
  await search.fill('Musterstraße 12');
  const option = page.locator('#context-property [role="option"]', { hasText: 'Musterstraße 12 · Zinshaus' });
  await option.waitFor();
  await option.click();
  await page.waitForURL('**/musterstrasse-12/**');
  assert.match(await page.locator('#context-property strong').first().innerText(), /Musterstraße 12/);

  const route = `${baseURL}/musterstrasse-12/app/settings/annual-statement?year=2025`;
  await page.goto(route);
  await page.locator('#rechtsgrundlage, #verteilerschluessel').evaluateAll(nodes => {
    for (const node of nodes) {
      for (let parent = node; parent; parent = parent.parentElement) if (parent.tagName === 'DETAILS') parent.open = true;
    }
  });
  assert.equal(await page.locator('[name="regime"]').inputValue(), 'mrg_voll');
  assert.equal(await page.locator('[name="heating_consumption_percent"]').inputValue(), '65');
  assert.equal(await page.locator('[name="show_vat"]').isChecked(), true);
  const vacant = page.locator('.allocation-row', { hasText: 'Top 6' }).first();
  assert.equal(await vacant.locator('input[name="vacant_from"]').inputValue(), '2025-03-01');
  assert.equal(await vacant.locator('input[name="vacant_to"]').inputValue(), '2025-08-31');
  await page.getByRole('button', { name: 'Für alle Einheiten berechnen', exact: true }).click();
  const owner = page.locator('[data-vacancy-owner]');
  await owner.first().waitFor();
  assert.match(await owner.first().innerText(), /Leerstand · Eigentümer trägt/);
  assert.match(await owner.first().innerText(), /504,33/);
  const result = page.locator('[data-annual-statement-run]');
  const group = label => result.locator(`.annual-unit-group[aria-label="${label}"]`);
  const cents = text => Math.round(Number(text.replace(/\./g, '').replace(',', '.').replace(/[^0-9.-]/g, '')) * 100);
  const amounts = async row => ({
    cost: cents(await row.locator('[data-label="Kostenanteil"]').innerText()),
    prepaid: cents(await row.locator('[data-label="Akonto"]').innerText()),
    balance: cents(await row.locator('[data-label="Saldo"]').innerText()),
  });
  for (const label of ['Top 1', 'Top 3']) {
    assert.equal(await group(label).locator('.annual-unit-row').count(), 1, 'landlord is not a second debtor');
    assert.equal(await group(label).getByRole('link', {name:/^Kopie für Vermieter/}).count(), 1);
    assert.equal(await group(label).locator('.annual-copy-row .annual-money').count(), 0);
  }
  assert.equal(await group('Top 2').locator('.annual-unit-row').count(), 2, 'no zero landlord row');
  const top6 = group('Top 6');
  const occupied = top6.locator('.annual-unit-row:not([data-vacancy-owner])');
  const vacancy = top6.locator('[data-vacancy-owner]');
  assert.equal(await occupied.count(), 1);
  assert.equal(await vacancy.count(), 1);
  assert.equal(await result.locator('[data-vacancy-owner]').count(), 1);
  assert.match(await occupied.innerText(), /Mietzeitraum · keine Mietpartei gespeichert/);
  assert.match(await occupied.innerText(), /01\.01\.2025 – 28\.02\.2025 · 01\.09\.2025 – 31\.12\.2025/);
  assert.match(await vacancy.innerText(), /01\.03\.2025 – 31\.08\.2025/);
  const occupiedAmounts = await amounts(occupied), vacancyAmounts = await amounts(vacancy);
  assert.deepEqual(occupiedAmounts, {cost:49617, prepaid:42000, balance:7617});
  assert.deepEqual(vacancyAmounts, {cost:50433, prepaid:0, balance:50433});
  assert.equal(occupiedAmounts.cost + vacancyAmounts.cost, 100050, 'unit total including vacancy');
  assert.equal(occupiedAmounts.balance + vacancyAmounts.balance, 58050);
  const rows = await result.locator('.annual-unit-row').all();
  let houseCosts = 0;
  for (const row of rows) houseCosts += (await amounts(row)).cost;
  assert.equal(houseCosts, 1959600, 'charge rows reconcile to the house total');
  const landlord = await context.request.get(new URL(await group('Top 1').getByRole('link', {name:/^Kopie für Vermieter/}).getAttribute('href'), baseURL).href);
  assert.equal(landlord.status(), 200);
  await writeFile(`${out}/top1-landlord.pdf`, await landlord.body());
  await writeFile(`${out}/mrg-amounts.json`, JSON.stringify({occupied:occupiedAmounts, vacancy:vacancyAmounts, houseCosts}, null, 2));
  for (const width of [1440, 390]) {
    await page.setViewportSize({width, height:1000});
    await result.locator('.annual-units').screenshot({path:`${out}/zinshaus-result-${width}.png`, style: '[data-context-bar], .skip-link { visibility: hidden !important; }'});
    await top6.screenshot({path:`${out}/top6-rows-${width}.png`});
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${width}px overflow`);
  }
  await page.locator('#rechtsgrundlage').evaluateAll(nodes => {
    for (const node of nodes) {
      for (let parent = node; parent; parent = parent.parentElement) if (parent.tagName === 'DETAILS') parent.open = true;
    }
  });
  assert.equal(await page.locator('[name="regime"] option:checked').innerText(), 'MRG Vollanwendung');
  await page.locator('a[href*="aushang=1"]').waitFor();
  await page.locator('[data-annual-statement-run]').screenshot({ path: `${out}/zinshaus-annual.png` });
  await page.locator('[name="regime"]').evaluate(node => node.scrollIntoView({ block: 'center' }));
  await page.screenshot({ path: `${out}/zinshaus-annual-legal.png`, fullPage: false });

  const previewURL = `${baseURL}/musterstrasse-12/app/settings/valorisation?new=1&preview=1&effective_on=2026-04-01&house=musterstrasse-12`;
  const response = await page.goto(previewURL);
  assert.equal(response?.status(), 200);
  const preview = page.locator('article[data-run-id=""]');
  await preview.waitFor();
  const counts = { ready: 8, unchanged: 1, exception: 1 };
  for (const [group, count] of Object.entries(counts)) {
    assert.equal(await preview.locator(`[data-group="${group}"]`).count(), 1, group);
    assert.equal(await preview.locator(`[data-group="${group}"] > .vr-item`).count(), count, group);
  }
  await preview.locator('[data-group="ready"]').scrollIntoViewIfNeeded();
  await page.screenshot({ path: `${out}/zinshaus-valorisation.png`, fullPage: false });
  assert.deepEqual(pageErrors, []);
  process.stdout.write('qa-demo-zinshaus ok\n');
} finally {
  await browser.close();
}
