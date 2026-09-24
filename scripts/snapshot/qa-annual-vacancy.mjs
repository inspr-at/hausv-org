// HAUSV-785. The demo house is WEG, so this script switches the period to
// MRG Vollanwendung before it expects the landlord line.
// Usage: node qa-annual-vacancy.mjs http://localhost:8302 <artifact-dir>
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';

const [baseURL, out = '../../tmp/qa-annual-vacancy'] = process.argv.slice(2);
assert(['localhost', '127.0.0.1', '[::1]'].includes(new URL(baseURL).hostname));
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : {}) });
const context = await browser.newContext({ locale: 'de-AT', timezoneId: 'Europe/Vienna' });
const page = await context.newPage();
const pageErrors = [];
page.on('pageerror', error => pageErrors.push(error.message));
const route = `${baseURL}/janusbergweg-123/app/settings/annual-statement?year=2025#verteilerschluessel`;
try {
  await page.goto(route);
  if (await page.locator('input[name="email"]').count()) {
    await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
    await page.locator('input[name="email"]').fill('vera.verwalter@musterstadt.example');
    await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
    await page.locator('a.dev-link').click();
    await page.waitForURL('**/app**');
    await page.goto(route);
  }
  await page.locator('#rechtsgrundlage, #verteilerschluessel').evaluateAll(nodes => {
    for (const node of nodes) {
      for (let parent = node; parent; parent = parent.parentElement) if (parent.tagName === 'DETAILS') parent.open = true;
    }
  });
  await page.locator('[name="regime"]').selectOption('mrg_voll');
  await page.getByRole('button', { name: 'Rechtsgrundlage speichern', exact: true }).click();
  await page.locator('#verteilerschluessel').evaluate(node => {
    for (let parent = node; parent; parent = parent.parentElement) if (parent.tagName === 'DETAILS') parent.open = true;
  });
  await page.locator('input[name="vacant_from"]').first().fill('2025-03-01');
  await page.locator('input[name="vacant_to"]').first().fill('2025-05-31');
  await page.getByRole('button', { name: 'Verteilerbasis speichern', exact: true }).click();
  await page.locator('input[name="vacant_from"]').first().waitFor();
  assert.equal(await page.locator('input[name="vacant_from"]').first().inputValue(), '2025-03-01');
  assert.equal(await page.locator('input[name="vacant_to"]').first().inputValue(), '2025-05-31');
  await page.getByRole('button', { name: 'Für alle Einheiten berechnen', exact: true }).click();
  const owner = page.locator('[data-vacancy-owner]');
  await owner.first().waitFor();
  assert.match(await owner.first().innerText(), /Leerstand – Eigentümeranteil/);
  for (const width of [390, 1440]) {
    await page.setViewportSize({ width, height: 1000 });
    await owner.first().scrollIntoViewIfNeeded();
    await page.screenshot({ path: `${out}/vacancy-owner-${width}.png`, fullPage: false });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `overflow at ${width}`);
  }
  assert.deepEqual(pageErrors, []);
} finally {
  await browser.close();
}
