// HAUSV-662: real stored annual-statement results in the isolated demo fixture.
// Run with HV_CAPTURE=qa-annual-costs.mjs scripts/snapshot/run.sh WORKTREE <out>.
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { chromium } from 'playwright';

const [baseURL, out] = process.argv.slice(2);
assert(['localhost', '127.0.0.1', '[::1]'].includes(new URL(baseURL).hostname), 'Fixture writes require localhost');
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ locale: 'de-AT', timezoneId: 'Europe/Vienna' });
const errors = [];
page.on('pageerror', error => errors.push(error.message));
const report = [];
try {
  await page.goto(baseURL);
  const login = page.locator('details:has(form[action$="/auth/request"])');
  if (await login.count()) await login.evaluate(node => { node.open = true; });
  await page.locator('input[name="email"]').fill('vera.verwalter@musterstadt.example');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  await page.locator('a.dev-link').click();
  await page.waitForURL('**/app**');
  await page.goto(`${baseURL}/janusbergweg-123/app/settings/annual-statement?year=2025`);
  const calculate = page.getByRole('button', { name: 'Für alle Einheiten berechnen', exact: true });
  assert(await calculate.isEnabled(), 'Seed must provide calculable annual costs');
  await calculate.click();
  await page.locator('[data-annual-statement-run]').waitFor();
  const run = page.locator('[data-annual-statement-run]');
  const runID = await run.getAttribute('data-annual-statement-run');
  const runDetails = run.locator('.annual-run-details');
  const unitRows = page.locator('.annual-unit-row');
  assert.equal(await unitRows.first().locator('th').innerText(), 'Top 1');
  assert.deepEqual(await unitRows.first().locator('.annual-money').allTextContents(), ['428,04 €', '600,00 €', 'Guthaben 171,96 €']);
  assert.equal(await unitRows.nth(2).locator('.annual-party').count(), 2, 'Fixture covers multiple parties');
  const originalRows = await unitRows.evaluateAll(rows => rows.map(row => ({
    text: row.textContent, links: [...row.querySelectorAll('a')].map(a => a.getAttribute('href')),
  })));
  const details = page.locator('.annual-costs').first();
  const labels = ['Kostenart', 'Verteilerschlüssel', 'Anteil', 'Betrag'];
  const expected = [
    ['Abfallentsorgung', 'Personen', '2,78 %', '66,67 €'],
    ['Hausreinigung', 'Nutzfläche', '4,48 %', '161,37 €'],
    ['Gebäudeversicherung', 'Nutzwert', '4,17 %', '200,00 €'],
  ];
  for (const width of [320, 390, 820, 1050, 1051, 1440]) {
    await page.setViewportSize({ width, height: 1000 });
    assert.equal(await runDetails.getAttribute('open'), null, 'Run details initially collapsed');
    assert.equal(await runDetails.locator('code').isVisible(), false, 'Internal ID hidden initially');
    await runDetails.locator('summary').focus();
    await page.keyboard.press('Enter');
    assert.equal(await runDetails.locator('code').innerText(), runID, 'Exact ID available as selectable text');
    assert.equal(await runDetails.locator('code').isVisible(), true);
    await runDetails.screenshot({ path: `${out}/run-details-${width}.png` });
    await page.keyboard.press('Enter');
    assert.equal(await runDetails.locator('code').isVisible(), false, 'Keyboard closes run details');
    const alignment = await unitRows.evaluateAll((rows, width) => {
      const issues = [];
      const textCenter = node => {
        const range = document.createRange();
        range.selectNodeContents(node);
        const rect = range.getBoundingClientRect();
        return rect.top + rect.height / 2;
      };
      for (const row of rows) {
        const names = [...row.querySelectorAll('.annual-party')];
        const documents = [...row.querySelectorAll('.annual-pdf')];
        if (names.length !== documents.length) issues.push('Party/document count differs');
        names.forEach((name, i) => {
          if (Math.abs(name.getBoundingClientRect().top - documents[i].getBoundingClientRect().top) > 1) issues.push('Party/document rows differ');
          if (documents[i].querySelector('a').getBoundingClientRect().height < 44) issues.push('PDF touch target shrunk');
        });
        if (row.querySelectorAll('.annual-money').length !== 3) issues.push('Unit totals duplicated');
        if (width > 1050 && names.length) {
          const center = textCenter(names[0].querySelector('.annual-party-name'));
          for (const value of row.querySelectorAll('.annual-unit-value')) {
            if (Math.abs(textCenter(value) - center) > 1) issues.push('Unit/name/amount text misaligned');
          }
        }
        if (width <= 1050) for (const cell of row.querySelectorAll('td')) {
          if (getComputedStyle(cell, '::before').content !== JSON.stringify(cell.dataset.label)) issues.push('Unit field label missing');
        }
        if (row.scrollWidth > row.clientWidth + 1) issues.push('Unit row overflows');
      }
      return issues;
    }, width);
    assert.deepEqual(alignment, [], `${width}px party/unit/amount alignment`);
    assert.deepEqual(await unitRows.evaluateAll(rows => rows.map(row => ({
      text: row.textContent, links: [...row.querySelectorAll('a')].map(a => a.getAttribute('href')),
    }))), originalRows, 'Stored unit amounts and document targets unchanged');
    await unitRows.first().evaluate(node => node.scrollIntoView({ block: 'center' }));
    await page.screenshot({ path: `${out}/units-${width}.png` });
    assert.equal(await details.getAttribute('open'), null, 'Costs initially collapsed');
    await details.locator('summary').focus();
    await page.keyboard.press('Enter');
    const table = details.getByRole('table', { name: 'Kostenarten und Anteile · Top 1', exact: true });
    await table.waitFor({ state: 'visible' });
    assert.deepEqual(await table.locator('thead th').allTextContents(), labels);
    const rows = table.locator('tbody tr');
    assert.equal(await rows.count(), expected.length);
    for (let i = 0; i < expected.length; i++) {
      assert.deepEqual(await rows.nth(i).locator('[data-label]').allTextContents(), expected[i], 'Stored costs unchanged');
    }
    const defects = await table.evaluate((node, { width, labels }) => {
      const issues = [];
      const head = node.querySelector('thead').getBoundingClientRect();
      if (width > 1050 && head.height < 20) issues.push('Desktop headings hidden');
      if (width <= 1050 && head.width > 1) issues.push('Compact heading not visually hidden');
      const headings = [...node.querySelectorAll('thead th')].map(cell => cell.getBoundingClientRect());
      for (const row of node.querySelectorAll(':scope > tbody > tr')) {
        const cells = [...row.children];
        cells.forEach((cell, index) => {
          const box = cell.getBoundingClientRect();
          const container = node.getBoundingClientRect();
          if (box.left < container.left - 1 || box.right > container.right + 1 || cell.scrollWidth > cell.clientWidth + 1) issues.push('Cost cell overflows');
          if (width <= 1050) {
            const label = getComputedStyle(cell, '::before');
            if (label.content !== JSON.stringify(labels[index]) || label.display === 'none') issues.push('Mobile label missing');
          } else if (Math.abs(box.left - headings[index].left) > 1 || Math.abs(box.width - headings[index].width) > 1) issues.push('Header/value columns misaligned');
        });
        const tops = cells.map(cell => Math.round(cell.getBoundingClientRect().top));
        if (width > 1050 && new Set(tops).size !== 1) issues.push('Desktop row wrapped');
        if (width <= 760 && new Set(tops).size !== 4) issues.push('Phone values not stacked');
        if (width > 760 && width <= 1050 && new Set(tops).size !== 2) issues.push('Tablet values not paired');
      }
      if (document.documentElement.scrollWidth > innerWidth + 1) issues.push('Page overflow');
      return issues;
    }, { width, labels });
    assert.deepEqual(defects, [], `${width}px layout`);
    await details.scrollIntoViewIfNeeded();
    await details.screenshot({ path: `${out}/costs-${width}.png` });
    report.push({ width, rows: expected.length, headings: labels, valuesUnchanged: true, layout: 'passed', unitAlignment: 'passed', runDetailsKeyboard: 'passed' });
    await details.locator('summary').focus();
    await page.keyboard.press('Enter');
    assert.equal(await table.isVisible(), false, 'Keyboard closes cost details');
    console.log(`Annual costs ${width}px passed`);
  }
  assert.deepEqual(errors, [], 'Browser JavaScript errors');
  await writeFile(`${out}/evidence.json`, JSON.stringify({ report, errors }, null, 2));
} finally {
  await browser.close();
}
