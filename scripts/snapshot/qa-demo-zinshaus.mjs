// HAUSV-795. Smoke the seeded MRG Zinshaus: switch house, annual statement, valorisation preview.
// Usage: node qa-demo-zinshaus.mjs http://localhost:8292 <artifact-dir>
import assert from 'node:assert/strict';
import { access, mkdir } from 'node:fs/promises';
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
  assert.match(await owner.first().innerText(), /Leerstand – Eigentümeranteil/);
  assert.match(await owner.first().innerText(), /504,33/);
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
  const counts = { ready: 7, unchanged: 1, exception: 2 };
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
