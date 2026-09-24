// HAUSV-794. Demo fixture, own lane port. Run with the coordinator's pw.sh.
import assert from 'node:assert/strict';
import { mkdir, access, writeFile } from 'node:fs/promises';
import { chromium } from 'playwright';

const [baseURL, out = '/Users/markus/Code/hausv-worktrees/logs/w-ja10-browser'] = process.argv.slice(2);
assert(['localhost', '127.0.0.1', '[::1]'].includes(new URL(baseURL).hostname));
await mkdir(out, { recursive: true });
const state = `${out}/session.json`;
const saved = await access(state).then(() => true, () => false);
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : {}) });
const context = await browser.newContext({ locale: 'de-AT', timezoneId: 'Europe/Vienna', viewport: { width: 1440, height: 1000 }, ...(saved ? { storageState: state } : {}) });
const page = await context.newPage();
page.setDefaultTimeout(15000);
page.setDefaultNavigationTimeout(30000);
const errors = [];
page.on('pageerror', error => errors.push(error.message));
const annual = `${baseURL}/janusbergweg-123/app/settings/annual-statement?year=2025`;
const building = `${baseURL}/janusbergweg-123/app/settings/building?section=units#unit-top-3`;
try {
  await page.goto(annual);
  if (await page.locator('input[name="email"]').count()) {
    await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes => nodes.forEach(node => { node.open = true; }));
    await page.locator('input[name="email"]').fill('vera.verwalter@musterstadt.example');
    await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
    await page.locator('a.dev-link').click();
    await page.waitForURL('**/app**');
    await context.storageState({ path: state });
  }
  await page.goto(building);
  const form = page.locator('#unit-top-3-form');
  if (!(await form.count())) { await page.screenshot({ path: `${out}/missing-form.png`, fullPage: true }); throw new Error(JSON.stringify({ route: new URL(page.url()).pathname, headings: await page.locator('h1,h2').allTextContents(), forms: await page.locator('form[id]').evaluateAll(nodes => nodes.map(n => n.id)) })); }
  const emails = await form.locator('[name="party_email"]').evaluateAll(nodes => nodes.map(n => n.value));
  const old = emails.indexOf('sophie.bewohner@musterstadt.example');
  const next = emails.indexOf('matthias.mieter@musterstadt.example');
  assert(old >= 0 && next >= 0, JSON.stringify({ emails }));
  assert.equal(await form.locator('[name="party_valid_to"]').nth(old).inputValue(), '2025-06-30');
  assert.equal(await form.locator('[name="party_valid_from"]').nth(next).inputValue(), '2025-07-01');
  await page.locator('#unit-top-3').screenshot({ path: `${out}/party-dates.png` });
  // Exercise the existing unit save path, preserving both dated assignments.
  await Promise.all([page.waitForURL(url => url.searchParams.has('unit')), page.locator('#unit-top-3').getByRole('button', { name: 'Änderungen speichern', exact: true }).click()]);
  assert.equal(new URL(page.url()).searchParams.get('unit'), 'saved');
  await page.goto(building);
  assert.equal(await form.locator('[name="party_valid_from"]').nth(next).inputValue(), '2025-07-01');
  await page.goto(annual);
  const calculate = page.getByRole('button', { name: 'Für alle Einheiten berechnen', exact: true });
  assert(await calculate.isEnabled());
  await Promise.all([page.waitForURL(url => url.searchParams.get('run-status') === 'created'), calculate.click()]);
  const row = page.locator('.annual-unit-row').filter({ has: page.getByRole('rowheader', { name: 'Top 3', exact: true }) });
  assert.equal(await row.locator('.annual-party').count(), 3);
  const names = await row.locator('.annual-party-name').allTextContents();
  assert(names.some(n => n.includes('Sophie Berger') && n.includes('30.06.2025')));
  assert(names.some(n => n.includes('Matthias Dorn') && n.includes('01.07.2025')));
  assert(names.every(n => /Nachzahlung|Guthaben|Ausgeglichen/.test(n)));
  for (const email of ['sophie.bewohner@musterstadt.example', 'matthias.mieter@musterstadt.example', 'alina.eigentuemer@musterstadt.example']) {
    const links = await row.locator('.annual-pdf a').evaluateAll(nodes => nodes.map(n => n.href));
    const link = links.find(href => new URL(href).searchParams.get('party') === email);
    assert(link);
    const response = await context.request.get(link);
    assert.equal(response.status(), 200);
    const raw = await response.body();
    const text = raw.toString('latin1');
    assert(text.startsWith('%PDF-'));
    assert(text.includes('Parteienwechsel'));
    assert(text.includes('monatliche Anteile'));
    await writeFile(`${out}/${email.split('@')[0]}.pdf`, raw);
  }
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    await row.scrollIntoViewIfNeeded();
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), `${width}px horizontal overflow`);
    await page.screenshot({ path: `${out}/party-run-${width}.png` });
  }
  assert.deepEqual(errors, []);
  await writeFile(`${out}/result.json`, JSON.stringify({ passed: true, names, checks: ['date persistence', 'three recipients', 'party PDF split note', 'desktop and mobile overflow', 'no page errors'] }, null, 2));
  console.log('party-change dates, run recipients and PDFs: ok');
} finally {
  await context.request.dispose();
  await context.close();
  await browser.close();
}
