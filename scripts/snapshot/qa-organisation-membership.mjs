#!/usr/bin/env node
// HAUSV-782: the roster form drives the transactional membership service.
// Run against an isolated snapshot fixture, never production.
import assert from 'node:assert/strict';
import { mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const base = process.argv[2];
const artifacts = process.argv[3];
assert(base && artifacts, 'usage: qa-organisation-membership.mjs <local-url> <artifact-dir>');
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
  await page.locator('input[name="email"]').fill('admin@example.com');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  const devLink = page.locator('a.dev-link');
  await devLink.waitFor({ state: 'visible' });
  const target = new URL(await devLink.getAttribute('href'), base);
  target.host = url.host;
  target.protocol = url.protocol;
  await page.goto(target.href, { waitUntil: 'networkidle' });
  await page.goto(`${base}/demo/app/settings/users`, { waitUntil: 'networkidle' });
  const invite = page.locator('form[action$="/app/settings/users"]');
  await invite.locator('input[name="email"]').waitFor({ state: 'attached' });
  await invite.evaluate(form => { const parent = form.closest('details'); if (parent) parent.open = true; });
  await invite.locator('input[name="email"]').fill(email);
  await invite.locator('select[name="role"]').selectOption('Mieter');
  await submit(invite, '/settings/users');
  await page.goto(`${base}/demo/app/verwaltung/einstellungen`, { waitUntil: 'networkidle' });
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
  console.log('✓ Organisation roster: add clerk, change to admin, remove; one committed row per member');
} finally {
  await browser.close();
}
