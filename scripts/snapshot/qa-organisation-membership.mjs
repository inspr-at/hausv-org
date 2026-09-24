#!/usr/bin/env node
// HAUSV-782: the roster form drives the transactional membership service.
// Run against an isolated snapshot fixture, never production.
import assert from 'node:assert/strict';
import { mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const base = process.argv[2];
const artifacts = process.argv[3];
const tenant = process.argv[4] || 'demo';
const demo = tenant === 'janusbergweg-123';
assert(base && artifacts, 'usage: qa-organisation-membership.mjs <local-url> <artifact-dir> [tenant]');
const email = `membership-qa-${Date.now()}@example.com`;
const url = new URL(base);
assert(['localhost', '127.0.0.1'].includes(url.hostname), 'isolated local fixture required');
mkdirSync(artifacts, { recursive: true });
const browser = await chromium.launch({ headless: true });
try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 }, locale: 'de-AT' });
  await page.goto(base, { waitUntil: 'networkidle' });
  const details = page.locator('details:has(form[action$="/auth/request"])');
  if (await details.count()) await details.evaluate(node => { node.open = true; });
  await page.locator('input[name="email"]').fill(demo ? 'vera.verwalter@musterstadt.example' : 'admin@example.com');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible' });
  const target = new URL(await devLink.getAttribute('href'), base);
  target.host = url.host;
  target.protocol = url.protocol;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  const settingsURL = `${base}/${tenant}/app/verwaltung/einstellungen`;
  async function seededMemberRoundTrip(suffix) {
    await page.goto(settingsURL, { waitUntil: 'networkidle' });
    const paul = () => page.locator('.member-table tbody tr').filter({ hasText: 'paul.verwalter@musterstadt.example' });
    assert.equal(await paul().count(), 1, 'seeded Paul must be present');
    await submit(paul().locator('form'), '/mitarbeiter/entfernen');
    assert.equal(await paul().count(), 0, 'seeded Paul can be removed without reconciliation');
    const form = page.locator('form.member-add');
    await form.locator('input[name="member_email"]').fill('paul.verwalter@musterstadt.example');
    await form.locator('select[name="member_role"]').selectOption('sachbearbeiter');
    await submit(form, '/mitarbeiter');
    assert.equal(await paul().count(), 1, 'seeded Paul can be re-added');
    assert.equal(await paul().locator('[data-label="Rolle"]').innerText(), 'Sachbearbeiter');
    await page.screenshot({ path: join(artifacts, `seeded-member-${suffix}.png`), fullPage: true });
  }
  if (demo) {
    await seededMemberRoundTrip('initial');
    await page.goto(`${settingsURL}/demo`, { waitUntil: 'networkidle' });
    const reset = page.locator('form[action$="/einstellungen/demo"]');
    await reset.locator('button[type="submit"]').click();
    await page.waitForLoadState('networkidle');
    assert.match(await page.locator('body').innerText(), /Demodaten (?:sind |wurden )?initialisiert/);
    await seededMemberRoundTrip('settings-reset');
  }
  await page.goto(`${base}/${tenant}/app/settings/users`, { waitUntil: 'networkidle' });
  const invite = page.locator('form[action$="/app/settings/users"]');
  await invite.locator('input[name="email"]').waitFor({ state: 'attached' });
  await invite.evaluate(form => { const parent = form.closest('details'); if (parent) parent.open = true; });
  await invite.locator('input[name="email"]').fill(email);
  await invite.locator('select[name="role"]').selectOption('Mieter');
  await submit(invite, '/settings/users');
  await page.goto(settingsURL, { waitUntil: 'networkidle' });
  const member = () => page.locator('.member-table tbody tr').filter({ hasText: email });
  assert.equal(await member().count(), 0, 'fixture must start without the employee');
  async function submit(form, suffix) {
    const response = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith(suffix));
    await form.locator('button[type="submit"]').click();
    const result = await response;
    assert.equal(result.status(), 303, `membership mutation must commit and redirect: ${result.status() === 303 ? '' : await result.text()}`);
    await page.waitForLoadState('networkidle');
  }
  for (const [role, label] of [['sachbearbeiter', 'Sachbearbeiter'], ['admin', 'Verwaltungs-Admin']]) {
    const form = page.locator('form.member-add');
    await form.locator('input[name="member_email"]').fill(email);
    await form.locator('select[name="member_role"]').selectOption(role);
    await submit(form, '/mitarbeiter');
    assert.equal(await member().count(), 1, 'one roster row after add/change');
    assert.equal(await member().locator('[data-label="Rolle"]').innerText(), label);
    await page.screenshot({ path: join(artifacts, `roster-${role}.png`), fullPage: true });
  }
  await submit(member().locator('form'), '/mitarbeiter/entfernen');
  assert.equal(await member().count(), 0, 'removed employee must disappear from roster');
  await page.screenshot({ path: join(artifacts, 'roster-removed.png'), fullPage: true });
  console.log(`✓ Organisation roster: ${demo ? 'seeded employee remove/re-add before and after settings reset; ' : ''}add clerk, change to admin, remove; one committed row per member`);
} finally {
  await browser.close();
}
