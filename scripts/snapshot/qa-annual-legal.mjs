// HAUSV-767 round 3. Coordinator runs against the isolated w-ja demo rig.
// Usage: node qa-annual-legal.mjs http://localhost:8313 <artifact-dir>
import assert from 'node:assert/strict';
import { mkdir, writeFile, access } from 'node:fs/promises';
import { chromium } from 'playwright';

const [baseURL, out = '../../tmp/qa-annual-legal'] = process.argv.slice(2);
assert(['localhost', '127.0.0.1', '[::1]'].includes(new URL(baseURL).hostname));
await mkdir(out, { recursive: true });
const state = `${out}/session.json`;
const saved = await access(state).then(() => true, () => false);
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : {}) });
const context = await browser.newContext({ locale: 'de-AT', timezoneId: 'Europe/Vienna', ...(saved ? { storageState: state } : {}) });
const page = await context.newPage();
const pageErrors = [];
page.on('pageerror', error => pageErrors.push(error.message));
const route = `${baseURL}/janusbergweg-123/app/settings/annual-statement?year=2025`;
const getPDF = async (href, name) => {
  const response = await context.request.get(new URL(href, baseURL).href);
  assert.equal(response.status(), 200);
  assert.match(response.headers()['content-type'], /application\/pdf/);
  const data = await response.body();
  assert(data.subarray(0, 5).equals(Buffer.from('%PDF-')));
  await writeFile(`${out}/${name}.pdf`, data);
  return data;
};
const prepare = async () => {
  await page.locator('#annual-preparation, details.annual-preparation').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
  await page.locator('#rechtsgrundlage').evaluate(node => {
    for (let parent = node.parentElement; parent; parent = parent.parentElement) if (parent.tagName === 'DETAILS') parent.open = true;
  });
};
try {
  await page.goto(route);
  if (await page.locator('input[name="email"]').count()) {
    await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
    await page.locator('input[name="email"]').fill('vera.verwalter@musterstadt.example');
    await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
    await page.locator('a.dev-link').click();
    await page.waitForURL('**/app**');
    await context.storageState({ path: state });
    await page.goto(route);
  }
  await prepare();
  assert.equal(await page.locator('[name="regime"]').inputValue(), 'weg');
  assert(await page.locator('[name="heizkg_applies"]').isChecked());
  assert.equal(await page.locator('[name="heating_consumption_percent"]').inputValue(), '70');
  assert.match(await page.locator('[name="inspection_place"]').inputValue(), /Musterstadt/);
  assert(await page.locator('[name="monthly_proposal_versicherung_top-1"]').count());
  await page.getByRole('button', { name: 'Für alle Einheiten berechnen', exact: true }).click();
  const run = page.locator('[data-annual-statement-run]');
  await run.waitFor();
  const runID = await run.getAttribute('data-annual-statement-run');
  const pdfURL = await run.locator('.annual-pdf a').first().getAttribute('href');
  const draft = await getPDF(pdfURL, 'draft');
  assert(draft.includes(Buffer.from('Entwurf')));
  const sendURL = `${baseURL}/janusbergweg-123/app/settings/annual-statement/runs/${runID}/send`;
  assert.equal((await context.request.post(sendURL, { headers: { Origin: baseURL }, maxRedirects: 0 })).status(), 409);
  await page.getByRole('button', { name: 'Abrechnung freigeben', exact: true }).click();
  assert.match(await run.innerText(), /Freigegeben am/);
  const final = await getPDF(pdfURL, 'final');
  assert(!final.includes(Buffer.from('Entwurf')));
  for (const text of ['Freigegeben:', 'Einsicht in die Belege', 'Belegverzeichnis', 'Energiekosten gesamt:', '70 % Verbrauch', 'Neue monatliche Vorauszahlung']) {
    assert(final.includes(Buffer.from(text)), `Final PDF misses ${text}`);
  }
  for (const width of [390, 1440]) {
    await page.setViewportSize({ width, height: 1000 });
    const row = run.locator('.annual-unit-row').first();
    const toggle = row.getByRole('button', { name: 'Details · Top 1', exact: true });
    assert.equal(await toggle.getAttribute('aria-expanded'), 'false');
    const box = await toggle.boundingBox();
    assert(box.height >= 44 && box.width >= 44);
    if (width === 1440) assert((await row.boundingBox()).height <= 60, 'Collapsed unit row stays near 56px');
    await toggle.focus();
    await page.keyboard.press('Space');
    assert.equal(await toggle.getAttribute('aria-expanded'), 'true');
    await page.keyboard.press('Space');
    assert.equal(await toggle.getAttribute('aria-expanded'), 'false');
    await run.scrollIntoViewIfNeeded();
    await page.screenshot({ path: `${out}/approved-${width}.png` });
    await prepare();
    await page.locator('#rechtsgrundlage').scrollIntoViewIfNeeded();
    await page.screenshot({ path: `${out}/legal-settings-${width}.png` });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1));
  }
  // A later settings change cannot alter the approved PDF.
  await page.locator('[name="regime"]').selectOption('mrg_voll');
  await page.getByRole('button', { name: 'Rechtsgrundlage speichern', exact: true }).click();
  assert((await getPDF(pdfURL, 'final-after-settings-change')).equals(final));
  await page.getByRole('button', { name: 'Für alle Einheiten berechnen', exact: true }).click();
  await page.getByRole('button', { name: 'Abrechnung freigeben', exact: true }).click();
  const noticeURL = await page.getByRole('link', { name: 'Aushang (PDF)', exact: true }).getAttribute('href');
  const notice = await getPDF(noticeURL, 'aushang');
  assert(notice.includes(Buffer.from('Aushang')));
  assert(!notice.includes(Buffer.from('alina.eigentuemer@')));
  await prepare();
  await page.locator('[name="regime"]').selectOption('weg');
  await page.getByRole('button', { name: 'Rechtsgrundlage speichern', exact: true }).click();
  assert.deepEqual(pageErrors, []);
  await writeFile(`${out}/report.json`, JSON.stringify({ ok: true, runID, widths: [390, 1440], checks: ['draft gate', 'approval', 'final PDF', 'inspection', 'heating split', 'Akonto', 'immutable PDF', 'MRG Aushang'] }, null, 2));
} finally {
  await browser.close();
}
