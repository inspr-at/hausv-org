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
