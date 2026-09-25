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
  // The isolated demo rig starts with its management shell; explicitly seed
  // once if this fresh database has no home-unit form yet.
  await page.goto(`${baseURL}/janusbergweg-123/app/settings/building?section=units`);
  if (!(await page.locator('#unit-top-3-form').count())) {
    await page.goto(`${baseURL}/app/verwaltung/einstellungen/demo`);
    await page.locator('form.demo-reset-card button[type="submit"]').click();
    await page.getByText('Demodaten wurden initialisiert').waitFor({ timeout: 180000 });
  }
  const results = [];
  for (const scenario of [
    { house: 'janusbergweg-123', unit: 'top-3', label: 'Top 3', previous: 'clara.berger', current: 'daniel.leitner', oldName: 'Clara Berger', newName: 'Daniel Leitner', count: 2, excluded: 'matthias.mieter' },
    { house: 'musterstrasse-12', unit: 'top-2', label: 'Top 2', previous: 'theresa.aichner', current: 'lena.krainer', oldName: 'Theresa Aichner', newName: 'Lena Krainer', count: 3 },
  ]) {
    await page.setViewportSize({ width: 1440, height: 1000 });
    if (scenario.house === 'musterstrasse-12') {
      await page.locator('#context-property > summary').click();
      await page.locator('#context-property-search').fill('Musterstraße 12');
      await page.locator('#context-property [role="option"]', { hasText: 'Musterstraße 12 · Zinshaus' }).click();
      await page.waitForURL('**/musterstrasse-12/**');
    }
    const buildingURL = `${baseURL}/${scenario.house}/app/settings/building?section=units#unit-${scenario.unit}`;
    await page.goto(buildingURL);
    const form = page.locator(`#unit-${scenario.unit}-form`);
    if (!(await form.count())) {
      await page.screenshot({ path: `${out}/${scenario.house}-missing-form.png`, fullPage: true });
      throw new Error(JSON.stringify({ house: scenario.house, route: new URL(page.url()).pathname, headings: await page.locator('h1,h2').allTextContents(), forms: await page.locator('form[id]').evaluateAll(nodes => nodes.map(n => n.id)) }));
    }
    const emails = await form.locator('[name="party_email"]').evaluateAll(nodes => nodes.map(n => n.value));
    const old = emails.indexOf(`${scenario.previous}@musterstadt.example`);
    const next = emails.indexOf(`${scenario.current}@musterstadt.example`);
    assert(old >= 0 && next >= 0, JSON.stringify({ emails }));
    assert.equal(await form.locator('[name="party_valid_to"]').nth(old).inputValue(), '2025-06-30');
    assert.equal(await form.locator('[name="party_valid_from"]').nth(next).inputValue(), '2025-07-01');
    await page.locator(`#unit-${scenario.unit}`).screenshot({ path: `${out}/${scenario.house}-party-dates.png` });
    await Promise.all([page.waitForURL(url => url.searchParams.has('unit')), page.locator(`#unit-${scenario.unit}`).getByRole('button', { name: 'Änderungen speichern', exact: true }).click()]);
    assert.equal(new URL(page.url()).searchParams.get('unit'), 'saved');
    await page.goto(buildingURL);
    assert.equal(await form.locator('[name="party_valid_from"]').nth(next).inputValue(), '2025-07-01');
    await page.goto(`${baseURL}/${scenario.house}/app/settings/annual-statement?year=2025`);
    const calculate = page.getByRole('button', { name: 'Für alle Einheiten berechnen', exact: true });
    assert(await calculate.isEnabled());
    await Promise.all([page.waitForURL(url => url.searchParams.get('run-status') === 'created'), calculate.click()]);
    const row = page.locator(`.annual-unit-group[aria-label="${scenario.label}"]`);
    assert.equal(await row.locator('.annual-party').count(), scenario.count);
    const names = await row.locator('.annual-party').allTextContents();
    assert.equal(await row.locator('.annual-unit-row').count(), scenario.count);
    for (const partyRow of await row.locator('.annual-unit-row').all()) {
      assert.equal(await partyRow.locator('.annual-party-name').count(), 1);
      assert.equal(await partyRow.locator('.annual-money').count(), 3);
      assert.equal(await partyRow.locator('.annual-pdf a:not([download])').count(), 1);
      assert.match(await partyRow.locator('[data-label=Saldo]').innerText(), /Nachzahlung|Guthaben|Ausgeglichen/);
    }
    assert(names.some(n => n.includes(scenario.oldName) && n.includes('bis 30.06.2025')));
    assert(names.some(n => n.includes(scenario.newName) && n.includes('ab 01.07.2025')));
    assert(names.every(n => !/Nachzahlung|Guthaben|Ausgeglichen|Beginn offen/.test(n)));
    const oldRow = row.locator('.annual-unit-row', {hasText: scenario.oldName});
    const newRow = row.locator('.annual-unit-row', {hasText: scenario.newName});
    assert.match(await oldRow.locator('[data-label=Saldo]').innerText(), scenario.house === 'janusbergweg-123' ? /143,52/ : /193,36/);
    assert.match(await newRow.locator('[data-label=Saldo]').innerText(), scenario.house === 'janusbergweg-123' ? /384,23/ : /456,58/);
    const links = await row.locator('.annual-pdf a').evaluateAll(nodes => nodes.map(n => n.href));
    for (const local of [scenario.previous, scenario.current]) {
      const link = links.find(href => new URL(href).searchParams.get('party') === `${local}@musterstadt.example`);
      assert(link);
      const response = await context.request.get(link);
      assert.equal(response.status(), 200);
      const raw = await response.body();
      const text = raw.toString('latin1');
      assert(text.startsWith('%PDF-'));
      assert(text.includes('Parteienwechsel'));
      assert(text.includes('monatliche Anteile'));
      await writeFile(`${out}/${local}.pdf`, raw);
    }
    if (scenario.excluded) {
      assert(!links.some(href => new URL(href).searchParams.get('party') === `${scenario.excluded}@musterstadt.example`));
      const denied = new URL(links[0]);
      denied.searchParams.set('party', `${scenario.excluded}@musterstadt.example`);
      assert.equal((await context.request.get(denied.href)).status(), 404, 'WEG tenant must have no statement PDF');
      const allNames = await page.locator('.annual-party-name').allTextContents();
      assert(!allNames.some(name => /Matthias Dorn|Sophie Berger|Mia Berger|Omar Kostic|Ines Winkler|Farid Aslan/.test(name)), 'WEG run bills only owners');
    }
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await row.scrollIntoViewIfNeeded();
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), `${width}px horizontal overflow`);
      await page.screenshot({ path: `${out}/${scenario.house}-party-run-${width}.png` });
    }
    if (scenario.house === 'musterstrasse-12') {
      const shop = page.locator('.annual-unit-group[aria-label="Geschäft"]');
      assert(await shop.locator('.annual-unit-row').count() >= 1);
      for (const width of [1440, 390]) {
        await page.setViewportSize({width, height:1000});
        await shop.scrollIntoViewIfNeeded();
        await page.screenshot({path:`${out}/geschaeft-${width}.png`});
      }
    }
    results.push({ house: scenario.house, names });
  }
  assert.deepEqual(errors, []);
  await writeFile(`${out}/result.json`, JSON.stringify({ passed: true, results, checks: ['date persistence', 'WEG owners only', 'MRG tenant change', 'WEG tenant PDF denied', 'party PDF split note', 'desktop and mobile overflow', 'no page errors'] }, null, 2));
  console.log('party-change dates, run recipients and PDFs: ok');
} finally {
  await context.request.dispose();
  await context.close();
  await browser.close();
}
