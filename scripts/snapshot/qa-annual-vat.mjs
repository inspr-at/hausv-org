// HAUSV-786. Usage: node qa-annual-vat.mjs http://localhost:8301 <artifact-dir>
import assert from 'node:assert/strict';
import { mkdir, writeFile, access } from 'node:fs/promises';
import { chromium } from 'playwright';

const [baseURL, out = '/Users/markus/Code/hausv-worktrees/logs/w-ja8-vat'] = process.argv.slice(2);
assert(['localhost', '127.0.0.1', '[::1]'].includes(new URL(baseURL).hostname));
await mkdir(out, { recursive: true });
const state = `${out}/session.json`;
const saved = await access(state).then(() => true, () => false);
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : {}) });
const context = await browser.newContext({ locale: 'de-AT', timezoneId: 'Europe/Vienna', ...(saved ? { storageState: state } : {}) });
const page = await context.newPage();
const route = `${baseURL}/janusbergweg-123/app/settings/annual-statement?year=2025`;
const pdfText = (data) => data.toString('latin1');
const calculate = async () => {
  const run = page.locator('[data-annual-statement-run]');
  const previousRun = await run.count() ? await run.getAttribute('data-annual-statement-run') : null;
  await Promise.all([
    page.waitForURL(url => url.searchParams.get('run-status') === 'created' && url.searchParams.get('run') !== previousRun),
    page.getByRole('button', { name: 'Für alle Einheiten berechnen', exact: true }).click(),
  ]);
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
  await page.locator('#rechtsgrundlage').evaluate(node => {
    for (let parent = node.parentElement; parent; parent = parent.parentElement) if (parent.tagName === 'DETAILS') parent.open = true;
  });
  const vatSwitch = page.locator('input[name="show_vat"]');
  assert.equal(await vatSwitch.isChecked(), false);
  const rate = (key) => page.locator(`form:has(input[name="key"][value="${key}"]) select[name="vat_rate"]`);
  assert.equal(await rate('heizung').inputValue(), '20');
  assert.equal(await rate('versicherung').inputValue(), '10');
  await page.screenshot({ path: `${out}/vat-off.png`, fullPage: true });
  await calculate();
  const offHref = await page.locator('.annual-pdf a').first().getAttribute('href');
  const offPDF = await (await context.request.get(new URL(offHref, baseURL).href, { headers: { Connection: 'close' } })).body();
  assert(!pdfText(offPDF).includes('USt-Satz'));
  await writeFile(`${out}/vat-off.pdf`, offPDF);
  await page.goto(route);
  await page.locator('#rechtsgrundlage').evaluate(node => {
    for (let parent = node.parentElement; parent; parent = parent.parentElement) if (parent.tagName === 'DETAILS') parent.open = true;
  });
  await vatSwitch.check();
  await page.getByRole('button', { name: 'Rechtsgrundlage speichern', exact: true }).click();
  await page.locator('#rechtsgrundlage').evaluate(node => {
    for (let parent = node.parentElement; parent; parent = parent.parentElement) if (parent.tagName === 'DETAILS') parent.open = true;
  });
  assert.equal(await page.locator('input[name="show_vat"]').isChecked(), true);
  await page.screenshot({ path: `${out}/vat-on.png`, fullPage: true });
  await calculate();
  const onHref = await page.locator('.annual-pdf a').first().getAttribute('href');
  const onPDF = await (await context.request.get(new URL(onHref, baseURL).href, { headers: { Connection: 'close' } })).body();
  const text = pdfText(onPDF);
  for (const want of ['Netto', 'USt-Satz', 'Brutto', '10 %', '20 %']) {
    assert(text.includes(want), `pdf missing ${want}`);
  }
  await writeFile(`${out}/vat-on.pdf`, onPDF);
  console.log('vat switch and pdf text ok');
} finally {
  await context.request.dispose();
  await context.close();
  await browser.close();
}
