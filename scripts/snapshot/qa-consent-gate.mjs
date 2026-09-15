#!/usr/bin/env node
// Browser regression suite for the Google Ads consent gate (HAUSV-742).
//
// Runs against an isolated instance started with GOOGLE_ADS_TAG_ID and
// GOOGLE_ADS_LEAD_CONVERSION set (scripts/qa-consent-gate.sh). Google's host
// is never reached: every request to it is intercepted and answered locally,
// so the suite only observes whether the page *tried* to load the tag.
import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from 'playwright';

const baseURL = process.argv[2];
if (!baseURL) {
  console.error('usage: qa-consent-gate.mjs <base-url>');
  process.exit(1);
}
const artifactDir = process.env.HV_QA_ARTIFACT_DIR?.trim() || '';
if (artifactDir) mkdirSync(artifactDir, { recursive: true });
const shot = (name) => (artifactDir ? { path: join(artifactDir, `consent-${name}.png`) } : null);

const executableCandidates = [
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
  ...(process.env.CI === 'true' ? [] : [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Chromium.app/Contents/MacOS/Chromium',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
    '/usr/bin/google-chrome',
  ]),
].filter(Boolean);
const executablePath = executableCandidates.find(existsSync);
const launchOptions = { headless: true, args: ['--no-proxy-server'] };
if (executablePath) launchOptions.executablePath = executablePath;

// The gate hides the bar from crawlers; visitor scenarios therefore use a
// regular browser user agent and the bot scenario the default headless one.
const VISITOR_UA = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36';
const GOOGLE = /googletagmanager\.com|google\.com|google\.at|doubleclick\.net|googlesyndication\.com|google-analytics\.com/;
const TAG_URL = 'googletagmanager.com/gtag/js?id=AW-';

const browser = await chromium.launch(launchOptions);
const results = [];
const check = (name, ok, extra = '') => {
  results.push({ name, ok });
  console.log(`${ok ? 'ok  ' : 'FAIL'} ${name}${extra ? ` — ${extra}` : ''}`);
};

async function open(opts = {}) {
  const ctx = await browser.newContext({
    viewport: opts.mobile ? { width: 390, height: 780 } : { width: 1280, height: 860 },
    userAgent: opts.bot ? undefined : VISITOR_UA,
  });
  // Answer Google's hosts locally: the suite must not depend on the network
  // and must never send anything to Google from CI.
  await ctx.route(GOOGLE, (route) => route.fulfill({ status: 200, contentType: 'application/javascript', body: '/* stubbed by qa-consent-gate */' }));
  if (opts.gpc) await ctx.addInitScript(() => { Object.defineProperty(navigator, 'globalPrivacyControl', { get: () => true }); });
  if (opts.dnt) await ctx.addInitScript(() => { Object.defineProperty(navigator, 'doNotTrack', { get: () => '1' }); });
  // Storage failure modes: a cookie jar that silently drops writes, or one
  // that throws on every access. Both must keep the gate closed.
  if (opts.cookies === 'reject') await ctx.addInitScript(() => {
    const desc = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie');
    Object.defineProperty(document, 'cookie', { configurable: true, get() { return desc.get.call(document); }, set(v) { if (!String(v).startsWith('hausv_consent=')) desc.set.call(document, v); } });
  });
  if (opts.cookies === 'throw') await ctx.addInitScript(() => {
    Object.defineProperty(document, 'cookie', { configurable: true, get() { throw new Error('cookie jar unavailable'); }, set() { throw new Error('cookie jar unavailable'); } });
  });
  const page = await ctx.newPage();
  const requests = [];
  page.on('request', (r) => requests.push(r.url()));
  return { ctx, page, requests };
}
const consentCookie = async (ctx) => (await ctx.cookies(baseURL)).find((c) => c.name === 'hausv_consent');
const googleHits = (requests) => requests.filter((u) => GOOGLE.test(u));
const tagLoads = (requests) => requests.filter((u) => u.includes(TAG_URL));
const layerOf = (page) => page.evaluate(() => (window.dataLayer || []).map((a) => Array.from(a).map((x) => (typeof x === 'object' && !(x instanceof Date) ? JSON.stringify(x) : String(x))).join(' ')));

// 1. Fresh visit: bar, no Google, equivalent first-layer choices, Escape = refusal.
{
  const { ctx, page, requests } = await open();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  const bar = page.locator('.hv-consent');
  check('fresh visit shows the bar', await bar.isVisible());
  check('fresh visit sends nothing to Google', googleHits(requests).length === 0, googleHits(requests).join(','));
  check('fresh visit sets no consent cookie', !(await consentCookie(ctx)));
  const styles = await page.$$eval('.hv-consent .hv-consent-btn', (els) => els.map((e) => { const s = getComputedStyle(e); return [s.fontSize, s.fontWeight, s.color, s.backgroundColor, s.borderColor, s.paddingLeft, s.borderRadius, e.offsetWidth, e.offsetHeight].join('|'); }));
  check('reject and accept are rendered identically', styles.length === 2 && styles[0] === styles[1], styles.join(' vs '));
  const s = shot('bar-desktop'); if (s) await page.screenshot(s);
  await page.keyboard.press('Escape');
  await page.waitForTimeout(200);
  const c = await consentCookie(ctx);
  check('Escape records a refusal and removes the bar', !!c && c.value.includes('m%3D0') && !(await bar.isVisible()));
  check('consent cookie is host-only, Lax and carries no identifier', !!c && !c.domain.startsWith('.') && c.sameSite === 'Lax' && /^v1%3Br%3D\d+%3Bm%3D0%3Bt%3D\d+$/.test(c.value), c && JSON.stringify({ d: c.domain, s: c.sameSite, v: c.value }));
  await page.reload({ waitUntil: 'networkidle' });
  check('refusal is remembered on reload', !(await bar.isVisible()) && googleHits(requests).length === 0);
  // Settings after a refusal must not resurrect a draft grant.
  await page.locator('[data-consent-open]').first().click();
  await page.waitForSelector('dialog.hv-consent-sheet[open]');
  check('settings after refusal show marketing off', !(await page.locator('#hv-consent-marketing').isChecked()));
  await page.getByRole('button', { name: 'Abbrechen' }).click();
  check('cancel keeps the refusal', (await consentCookie(ctx)).value.includes('m%3D0') && googleHits(requests).length === 0);
  await ctx.close();
}

// 2. Mobile: reject button.
{
  const { ctx, page, requests } = await open({ mobile: true });
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  const s = shot('bar-mobile'); if (s) await page.screenshot(s);
  await page.getByRole('button', { name: 'Ablehnen' }).click();
  await page.waitForTimeout(200);
  check('Ablehnen stores a refusal', (await consentCookie(ctx))?.value.includes('m%3D0') === true);
  check('nothing reaches Google after Ablehnen', googleHits(requests).length === 0);
  await ctx.close();
}

// 3. Settings draft + bar reject: reopened settings reflect the persisted refusal.
{
  const { ctx, page, requests } = await open();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Einstellungen' }).click();
  await page.waitForSelector('dialog.hv-consent-sheet[open]');
  await page.locator('#hv-consent-marketing').check();
  await page.getByRole('button', { name: 'Abbrechen' }).click();
  await page.waitForTimeout(200);
  check('cancelling the sheet during the first prompt counts as refusal', (await consentCookie(ctx))?.value.includes('m%3D0') === true && !(await page.locator('.hv-consent').isVisible()));
  await page.locator('[data-consent-open]').first().click();
  await page.waitForSelector('dialog.hv-consent-sheet[open]');
  check('reopened settings drop the unsaved draft', !(await page.locator('#hv-consent-marketing').isChecked()));
  await page.keyboard.press('Escape');
  check('no Google load throughout', googleHits(requests).length === 0);
  await ctx.close();
}

// 4. Accept: exactly one tag load, Consent Mode basic, measurement only; withdrawal cleans up.
{
  const { ctx, page, requests } = await open();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Akzeptieren' }).click();
  await page.waitForTimeout(800);
  check('Akzeptieren stores a grant', (await consentCookie(ctx))?.value.includes('m%3D1') === true);
  check('the tag is requested exactly once after accept', tagLoads(requests).length === 1, tagLoads(requests).join(','));
  const layer = await layerOf(page);
  check('Consent Mode: default denied, then update', layer[0]?.startsWith('consent default') && layer[0].includes('"ad_storage":"denied"') && layer[1]?.startsWith('consent update'), layer.slice(0, 2).join(' || '));
  check('update grants measurement only, personalisation stays denied', layer[1]?.includes('"ad_storage":"granted"') && layer[1].includes('"ad_user_data":"granted"') && layer[1].includes('"ad_personalization":"denied"'), layer[1]);
  check('config pins cookies to this host and accepts the incoming linker', layer.some((l) => l.startsWith('config AW-') && l.includes('"cookie_domain"') && l.includes('"accept_incoming":true')));
  check('no conversion on the landing page', !layer.some((l) => l.startsWith('event conversion')));
  const s = shot('after-accept'); if (s) await page.screenshot(s);
  // Seed what Google's tag would have stored, then withdraw.
  await ctx.addCookies([{ name: '_gcl_au', value: 'seeded', url: baseURL }]);
  await page.evaluate(() => { localStorage.setItem('_gcl_ls', 'seeded'); sessionStorage.setItem('hausv_ads_lead_fired', '1'); });
  await page.locator('[data-consent-open]').first().click();
  await page.waitForSelector('dialog.hv-consent-sheet[open]');
  check('settings reflect the grant', await page.locator('#hv-consent-marketing').isChecked());
  const s2 = shot('sheet'); if (s2) await page.screenshot(s2);
  await page.locator('#hv-consent-marketing').uncheck();
  const before = requests.length;
  await page.getByRole('button', { name: 'Speichern' }).click();
  await page.waitForLoadState('networkidle');
  check('withdrawal stores a refusal and reloads', (await consentCookie(ctx))?.value.includes('m%3D0') === true);
  check('no Google request after withdrawal', googleHits(requests.slice(before)).length === 0);
  const leftovers = (await ctx.cookies(baseURL)).filter((k) => /^(_gcl_|_gac_|_ga)/.test(k.name)).map((k) => k.name);
  const local = await page.evaluate(() => [localStorage.getItem('_gcl_ls'), sessionStorage.getItem('hausv_ads_lead_fired')]);
  check('withdrawal clears Google cookies and storage', leftovers.length === 0 && local.every((v) => v === null), `${leftovers.join(',')} ${JSON.stringify(local)}`);
  await ctx.close();
}

// 5. GPC: no bar, refusal persisted over an older grant, settings cannot re-enable.
{
  const { ctx, page, requests } = await open({ gpc: true });
  await ctx.addCookies([{ name: 'hausv_consent', value: `v1%3Br%3D1%3Bm%3D1%3Bt%3D${Math.floor(Date.now() / 1000)}`, url: baseURL }]);
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  check('GPC: no bar', !(await page.locator('.hv-consent').isVisible()));
  check('GPC: no Google request despite an older grant', googleHits(requests).length === 0, googleHits(requests).join(','));
  check('GPC: refusal persisted over the older grant', (await consentCookie(ctx))?.value.includes('m%3D0') === true);
  await page.locator('[data-consent-open]').first().click();
  await page.waitForSelector('dialog.hv-consent-sheet[open]');
  const box = page.locator('#hv-consent-marketing');
  check('GPC: marketing cannot be enabled in settings', (await box.isDisabled()) && !(await box.isChecked()));
  await page.getByRole('button', { name: 'Speichern' }).click();
  await page.waitForTimeout(300);
  check('GPC: saving settings loads nothing', googleHits(requests).length === 0);
  await ctx.close();
}

// 5b. DNT: an existing grant is overridden, nothing loads, refusal persisted.
{
  const { ctx, page, requests } = await open({ dnt: true });
  await ctx.addCookies([{ name: 'hausv_consent', value: `v1%3Br%3D1%3Bm%3D1%3Bt%3D${Math.floor(Date.now() / 1000)}`, url: baseURL }]);
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  check('DNT: no bar, no Google, refusal persisted over the older grant', !(await page.locator('.hv-consent').isVisible()) && googleHits(requests).length === 0 && (await consentCookie(ctx))?.value.includes('m%3D0') === true);
  await ctx.close();
}

// 5c. Storage failure: silently rejected consent-cookie writes keep the gate closed.
{
  const { ctx, page, requests } = await open({ cookies: 'reject' });
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Akzeptieren' }).click();
  await page.waitForTimeout(500);
  check('rejected cookie writes: accept loads nothing and stores nothing', googleHits(requests).length === 0 && !(await consentCookie(ctx)));
  await page.reload({ waitUntil: 'networkidle' });
  check('rejected cookie writes: the bar returns on reload', await page.locator('.hv-consent').isVisible());
  await ctx.close();
}

// 5d. Storage failure: a throwing cookie jar keeps the gate closed and the page usable.
{
  const { ctx, page, requests } = await open({ cookies: 'throw' });
  const errors = [];
  page.on('pageerror', (e) => errors.push(String(e)));
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  check('throwing cookie jar: bar shows, no script error', (await page.locator('.hv-consent').isVisible()) && errors.length === 0, errors.join(' | '));
  await page.getByRole('button', { name: 'Akzeptieren' }).click();
  await page.waitForTimeout(500);
  check('throwing cookie jar: accept loads nothing', googleHits(requests).length === 0 && errors.length === 0, errors.join(' | '));
  await ctx.close();
}

// 5e. Withdrawal when the refusal cannot be written: the grant must not survive into the next page.
{
  const { ctx, page, requests } = await open();
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Akzeptieren' }).click();
  await page.waitForTimeout(500);
  check('withdrawal-failure setup: tag loaded once', tagLoads(requests).length === 1);
  // From now on the cookie jar drops our writes: the refusal falls back to the session flag.
  await page.evaluate(() => {
    const desc = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie');
    Object.defineProperty(document, 'cookie', { configurable: true, get() { return desc.get.call(document); }, set(v) { if (!String(v).startsWith('hausv_consent=')) desc.set.call(document, v); } });
  });
  const before = requests.length;
  await page.locator('[data-consent-open]').first().click();
  await page.waitForSelector('dialog.hv-consent-sheet[open]');
  await page.locator('#hv-consent-marketing').uncheck();
  await page.getByRole('button', { name: 'Speichern' }).click();
  await page.waitForLoadState('load');
  await page.waitForTimeout(500);
  const flag = await page.evaluate(() => sessionStorage.getItem('hausv_consent_revoked'));
  check('withdrawal with a blocked cookie jar records a session revocation', flag === '1');
  check('withdrawal with a blocked cookie jar loads nothing afterwards', googleHits(requests.slice(before)).length === 0, googleHits(requests.slice(before)).join(','));
  await page.goto(`${baseURL}/impressum`, { waitUntil: 'networkidle' });
  check('the surviving grant cookie does not authorise the next page', googleHits(requests.slice(before)).length === 0 && !(await page.locator('.hv-consent').isVisible()));
  await ctx.close();
}

// 5f. Throwing cookie access after acceptance: withdrawal still clears local and session storage and closes the next page.
{
  const { ctx, page, requests } = await open();
  // Gated by a session flag so the same tab can flip into the failure mode later.
  await page.addInitScript(() => {
    if (sessionStorage.getItem('qa_cookie_throw') === '1') {
      Object.defineProperty(document, 'cookie', { configurable: true, get() { throw new Error('cookie jar unavailable'); }, set() { throw new Error('cookie jar unavailable'); } });
    }
  });
  const errors = [];
  page.on('pageerror', (e) => errors.push(String(e)));
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Akzeptieren' }).click();
  await page.waitForTimeout(500);
  check('throwing-after-accept setup: tag loaded once', tagLoads(requests).length === 1);
  await page.evaluate(() => {
    localStorage.setItem('_gcl_ls', 'seeded');
    sessionStorage.setItem('hausv_ads_lead_fired', '1');
    sessionStorage.setItem('qa_cookie_throw', '1');
    Object.defineProperty(document, 'cookie', { configurable: true, get() { throw new Error('cookie jar unavailable'); }, set() { throw new Error('cookie jar unavailable'); } });
  });
  const before = requests.length;
  await page.locator('[data-consent-open]').first().click();
  await page.waitForSelector('dialog.hv-consent-sheet[open]');
  await page.locator('#hv-consent-marketing').uncheck();
  await page.getByRole('button', { name: 'Speichern' }).click();
  await page.waitForLoadState('load');
  await page.waitForTimeout(500);
  const state = await page.evaluate(() => [localStorage.getItem('_gcl_ls'), sessionStorage.getItem('hausv_ads_lead_fired'), sessionStorage.getItem('hausv_consent_revoked')]);
  check('throwing cookie jar: withdrawal still clears local and session storage and records the revocation', state[0] === null && state[1] === null && state[2] === '1', JSON.stringify(state));
  check('throwing cookie jar: no page error during withdrawal', errors.length === 0, errors.join(' | '));
  await page.goto(`${baseURL}/impressum`, { waitUntil: 'networkidle' });
  check('throwing cookie jar: the next page loads nothing from Google', googleHits(requests.slice(before)).length === 0 && errors.length === 0, errors.join(' | '));
  await ctx.close();
}

// 5g. Signal refusal with rejected cookie writes, then the signal disappears: the superseded grant must stay dead.
{
  const { ctx, page, requests } = await open();
  await page.addInitScript(() => {
    if (sessionStorage.getItem('qa_gpc_off') !== '1') {
      Object.defineProperty(navigator, 'globalPrivacyControl', { configurable: true, get: () => true });
    }
    const desc = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie');
    Object.defineProperty(document, 'cookie', { configurable: true, get() { return desc.get.call(document); }, set(v) { if (!String(v).startsWith('hausv_consent=')) desc.set.call(document, v); } });
  });
  await ctx.addCookies([{ name: 'hausv_consent', value: `v1%3Br%3D1%3Bm%3D1%3Bt%3D${Math.floor(Date.now() / 1000)}`, url: baseURL }]);
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  check('signal with rejected writes: nothing loads and a session revocation is recorded', googleHits(requests).length === 0 && (await page.evaluate(() => sessionStorage.getItem('hausv_consent_revoked'))) === '1');
  await page.evaluate(() => sessionStorage.setItem('qa_gpc_off', '1'));
  await page.goto(`${baseURL}/impressum`, { waitUntil: 'networkidle' });
  check('after the signal disappears the superseded grant still loads nothing', googleHits(requests).length === 0 && (await page.evaluate(() => navigator.globalPrivacyControl)) !== true, googleHits(requests).join(','));
  await ctx.close();
}

// 6. Bots (headless UA): no bar, no Google, nothing stored.
{
  const { ctx, page, requests } = await open({ bot: true });
  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' });
  check('crawler UA: no bar, no Google, no cookie', !(await page.locator('.hv-consent').isVisible()) && googleHits(requests).length === 0 && !(await consentCookie(ctx)));
  await ctx.close();
}

// 7. Lead conversion: only on the sent view of /start, once per session.
{
  const { ctx, page } = await open();
  await page.goto(`${baseURL}/start`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: 'Akzeptieren' }).click();
  await page.waitForTimeout(400);
  let layer = await layerOf(page);
  check('plain /start fires no conversion', !layer.some((l) => l.startsWith('event conversion')));
  await page.goto(`${baseURL}/start?sent=1`, { waitUntil: 'load' });
  await page.waitForTimeout(400);
  layer = await layerOf(page);
  check('/start?sent=1 fires the conversion once', layer.filter((l) => l.startsWith('event conversion')).length === 1);
  await page.reload({ waitUntil: 'load' });
  await page.waitForTimeout(400);
  layer = await layerOf(page);
  check('a reload of the sent view does not fire again', layer.filter((l) => l.startsWith('event conversion')).length === 0);
  // Withdraw and re-consent on the sent view: the new grant is a new decision
  // and fires at most once again; nothing fires while withdrawn.
  await page.locator('[data-consent-open]').first().click();
  await page.waitForSelector('dialog.hv-consent-sheet[open]');
  await page.locator('#hv-consent-marketing').uncheck();
  await page.getByRole('button', { name: 'Speichern' }).click();
  await page.waitForLoadState('networkidle');
  layer = await layerOf(page);
  check('after withdrawal the sent view fires nothing', !layer.some((l) => l.startsWith('event conversion')));
  await page.locator('[data-consent-open]').first().click();
  await page.waitForSelector('dialog.hv-consent-sheet[open]');
  await page.locator('#hv-consent-marketing').check();
  await page.getByRole('button', { name: 'Speichern' }).click();
  await page.waitForTimeout(400);
  layer = await layerOf(page);
  check('re-consent fires the conversion exactly once', layer.filter((l) => l.startsWith('event conversion')).length === 1);
  await page.reload({ waitUntil: 'load' });
  await page.waitForTimeout(400);
  layer = await layerOf(page);
  check('and stays deduplicated on reload', layer.filter((l) => l.startsWith('event conversion')).length === 0);
  await ctx.close();
}

// 8. Privacy page: disclosure and control present with a configured tag.
{
  const { ctx, page } = await open();
  await page.goto(`${baseURL}/datenschutz`, { waitUntil: 'networkidle' });
  check('privacy page discloses Google Ads', (await page.locator('h2', { hasText: 'Werbe-Cookies (Google Ads)' }).count()) === 1);
  check('privacy page offers the Privatsphäre control', (await page.locator('footer [data-consent-open]').count()) === 1);
  await ctx.close();
}

await browser.close();
const failed = results.filter((r) => !r.ok);
console.log(failed.length ? `\nconsent gate: ${failed.length} check(s) failed` : `\nconsent gate: all ${results.length} checks passed`);
process.exit(failed.length ? 1 : 0);
