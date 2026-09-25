// HAUSV-780. Adds a Rücklage booking and checks the balance and the PDF block.
// Usage: node qa-annual-reserve.mjs http://localhost:8307 <artifact-dir>
import assert from 'node:assert/strict';
import { mkdir, writeFile, access } from 'node:fs/promises';
import { chromium } from 'playwright';

const [baseURL, out = '/Users/markus/Code/hausv-worktrees/logs/w-ja7-reserve'] = process.argv.slice(2);
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
  await page.locator('details.annual-preparation').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
  const section = page.locator('#ruecklage');
  await section.scrollIntoViewIfNeeded();
  const sameDayKinds = await section.locator('.reserve-entry').evaluateAll(rows => rows.filter(row => row.firstElementChild.textContent === '01.01.2025').map(row => row.querySelector('strong').textContent));
  assert.equal(sameDayKinds[0], 'Anfangsstand');
  assert(sameDayKinds.includes('Zuführung'));
  const withdrawal = section.locator('.reserve-entry').filter({ hasText: 'Entnahme' }).locator('.reserve-note');
  assert.equal(await withdrawal.innerText(), 'Dachrinnenreparatur · Rechnung Dachrinnenreparatur 2025 (Spenglerei Holzer)');
  assert(!/\.pdf|demo-document/i.test(await withdrawal.innerText()), 'Use the document title without repeating its filename');
  const before = await section.locator('[data-reserve-closing]').getAttribute('data-reserve-closing');
  await section.locator('[name="kind"]').selectOption('contribution');
  await section.locator('[name="entry_date"]').fill('2025-03-15');
  await section.locator('[name="amount"]').fill('1,00');
  await section.locator('[name="note"]').fill('Prüfbuchung');
  await section.getByRole('button', { name: 'Buchung hinzufügen', exact: true }).click();
  await section.waitFor();
  const after = await section.locator('[data-reserve-closing]').getAttribute('data-reserve-closing');
  assert.notEqual(after, before);
  assert.match(await section.innerText(), /Prüfbuchung/);
  assert.match(await section.innerText(), /Endstand/);
  const existingRun = page.locator('[data-annual-statement-run]');
  const previousRun = await existingRun.count() ? await existingRun.getAttribute('data-annual-statement-run') : null;
  await Promise.all([
    page.waitForURL(url => url.searchParams.get('run-status') === 'created' && url.searchParams.get('run') !== previousRun),
    page.getByRole('button', { name: 'Für alle Einheiten berechnen', exact: true }).click(),
  ]);
  const run = page.locator('[data-annual-statement-run]');
  await run.waitFor();
  const pdfURL = await run.locator('.annual-pdf a').first().getAttribute('href');
  const response = await context.request.get(new URL(pdfURL, baseURL).href, { headers: { Connection: 'close' } });
  assert.equal(response.status(), 200);
  const pdf = await response.body();
  await writeFile(`${out}/statement.pdf`, pdf);
  for (const text of ['R\\374cklage', 'Anfangsstand', 'Entnahmen', 'Zinsen', 'Endstand', 'Anteil dieser Einheit']) {
    assert(pdf.includes(Buffer.from(text)), `PDF misses ${text}`);
  }
  for (const width of [390, 1440]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.locator('details.annual-preparation').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
    await section.scrollIntoViewIfNeeded();
    const notes = await section.locator('.reserve-note').evaluateAll(nodes=>nodes.map(n=>({width:n.clientWidth,scroll:n.scrollWidth,white:getComputedStyle(n).whiteSpace})));
    assert(notes.every(n=>n.scroll<=n.width+1&&n.white==='normal'),'reserve receipt must wrap inside its row');
    await page.screenshot({ path: `${out}/reserve-${width}.png`, fullPage: false });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1));
    await section.screenshot({ path: `${out}/reserve-section-${width}.png` });
  }
  assert.deepEqual(pageErrors, []);
} finally {
  await context.request.dispose();
  await context.close();
  await browser.close();
}
