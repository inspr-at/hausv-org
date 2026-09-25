// HAUSV-802: agreed shares and HeizKG § 18. Run against a demo rig.
import assert from 'node:assert/strict';
import { access, mkdir, writeFile } from 'node:fs/promises';
import { chromium } from 'playwright';
const [baseURL, out, resetDemo] = process.argv.slice(2);
assert(['localhost', '127.0.0.1'].includes(new URL(baseURL).hostname));
assert(out?.startsWith('/'));
await mkdir(out, { recursive: true });
const state = `${out}/session.json`;
const saved = await access(state).then(() => true, () => false);
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ locale: 'de-AT', timezoneId: 'Europe/Vienna', viewport: {width:1440,height:1200}, ...(saved ? {storageState:state}: {}) });
const page = await context.newPage();
const errors = [];
page.on('pageerror', e => errors.push(e.message));
const route = `${baseURL}/janusbergweg-123/app/settings/annual-statement?year=2025`;
const openSection = async id => {
  await page.goto(`${route}#${id}`);
  await page.locator(`#${id}`).evaluate(node => {
    for (let p=node.parentElement;p;p=p.parentElement) if(p.tagName==='DETAILS')p.open=true;
  });
};
const submit = async form => {
  await Promise.all([page.waitForNavigation(), form.locator('button[type=submit]').click()]);
};
try {
  await page.goto(route);
  if (await page.locator('input[name=email]').count()) {
    await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes=>nodes.forEach(n=>n.open=true));
    await page.locator('input[name=email]').fill('vera.verwalter@musterstadt.example');
    await page.locator('form[action$="/auth/request"] button[type=submit]').click();
    await page.locator('a.dev-link').click();
    await page.waitForURL('**/app**');
    await context.storageState({path:state});
  }
  await page.goto(route);
  if (resetDemo === '--reset-demo' || !await page.locator('#rechtsgrundlage').count()) {
    await page.goto(`${baseURL}/janusbergweg-123/app/verwaltung/einstellungen/demo`);
    await Promise.all([page.waitForNavigation(),page.getByRole('button',{name:'Ja, initialisieren',exact:true}).click()]);
  }
  await openSection('vereinbarte-anteile');
  const shares = page.locator('form[data-agreed-shares]:has(input[value=lift])');
  assert.equal(await shares.locator('input[data-agreed-share]').count(),24);
  assert(!/PPM|Millionstel/.test(await page.locator('#vereinbarte-anteile').innerText()));
  assert.match(await shares.locator('[data-agreed-sum]').innerText(), /100 % \/ 100 %/);
  const exactShares = ['33,3333', '33,3333', '33,3334'];
  const savedShares = await shares.locator('[data-agreed-share]').evaluateAll(fields => fields.map(field => field.value));
  for (let i=0; i<24; i++) await shares.locator('[data-agreed-share]').nth(i).fill(exactShares[i] ?? '0');
  assert.equal(await shares.locator('[data-agreed-sum]').getAttribute('data-valid'),'true');
  await submit(shares);
  for (let i=0; i<3; i++) assert.equal(await shares.locator('[data-agreed-share]').nth(i).inputValue(), exactShares[i]);
  for (let i=0; i<24; i++) await shares.locator('[data-agreed-share]').nth(i).fill(savedShares[i]);
  await submit(shares);
  for(const id of ['top-1','top-2','top-3','top-4','stellplatz-1'])assert.equal(await shares.locator(`input[name=share_${id}]`).inputValue(),'0');
  assert.equal(await shares.locator('[data-agreed-sum]').getAttribute('data-valid'),'true');
  await shares.locator('input[name=share_top-1]').fill('1');
  assert.equal(await shares.locator('[data-agreed-sum]').getAttribute('data-valid'),'false');
  assert.equal(await shares.locator('button[type=submit]').isDisabled(),true);
  const invalid = await shares.evaluate(form=>Object.fromEntries(new FormData(form)));
  const rejected = await context.request.post(`${baseURL}/janusbergweg-123/app/settings/annual-statement/agreed-shares`,{form:invalid,headers:{Origin:baseURL},maxRedirects:0});
  assert.equal(rejected.status(),400);
  await shares.locator('input[name=share_top-1]').fill('0');
  await submit(shares);
  assert.equal(await shares.locator('[data-agreed-sum]').getAttribute('data-valid'),'true');
  await page.locator('#vereinbarte-anteile').screenshot({path:`${out}/agreed-desktop.png`});
  await page.locator('#vereinbarte-anteile').scrollIntoViewIfNeeded();
  await page.screenshot({path:`${out}/agreed-1440.png`});
  await page.setViewportSize({width:390,height:844});
  await page.locator('#vereinbarte-anteile').scrollIntoViewIfNeeded();
  assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'mobile overflow');
  await page.screenshot({path:`${out}/agreed-mobile.png`});
  await page.setViewportSize({width:1440,height:1200});
  await openSection('heizinformationen');
  const energy=page.locator('form[action$="/heating-information"]');
  assert.equal(await energy.locator('input[name=purchase_0_quantity]').inputValue(),'100000');
  assert.equal(await energy.locator('input[name=purchase_0_price]').inputValue(),'0,09');
  await energy.locator('input[name=purchase_0_quantity]').fill('100000,000001');
  await energy.locator('input[name=purchase_0_price]').fill('0,09');
  await submit(energy);
  assert.equal(await energy.locator('input[name=purchase_0_quantity]').inputValue(),'100000,000001');
  await energy.locator('input[name=purchase_0_quantity]').fill('100000');
  await submit(energy);
  await page.locator('#heizinformationen').screenshot({path:`${out}/heating-information.png`});
  for (const width of [1440,390]) {
    await page.setViewportSize({width,height:1000});
    await page.locator('#heizinformationen').scrollIntoViewIfNeeded();
    const disclosure=energy.locator('summary').first();
    assert.equal(await disclosure.locator('.disclosure-chevron').count(),1);
    assert.equal(await disclosure.evaluate(n=>getComputedStyle(n).listStyleType),'none');
    assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
    await page.screenshot({path:`${out}/heating-information-${width}.png`});
  }
  await page.goto(route);
  const runPanel = page.locator('[data-annual-statement-run]');
  const previous = await runPanel.count() ? await runPanel.getAttribute('data-annual-statement-run') : null;
  await Promise.all([
    page.waitForURL(url=>url.searchParams.get('run-status')==='created'&&url.searchParams.get('run')!==previous),
    page.getByRole('button',{name:'Für alle Einheiten berechnen',exact:true}).click(),
  ]);
  const hrefs=await page.locator('.annual-pdf a').evaluateAll(nodes=>nodes.map(n=>n.getAttribute('href')));
  const href=hrefs.find(h=>h.includes('top-1')&&!h.includes('top-10')&&!h.includes('download=1'));
  assert(href, 'top-1 PDF');
  const pdfURL=new URL(href,baseURL).href;
  const response=await context.request.get(pdfURL);assert.equal(response.status(),200);
  const pdf=await response.body();
  for(const text of ['0,09 \\200/kWh','100.000 kWh','Vorperiodenvergleich','www.topprodukte.at','www.verbraucherschlichtung.at/antrag/','Monatliche Verbrauchsinformation']) assert(pdf.toString('latin1').includes(text),`PDF missing ${text}`);
  await writeFile(`${out}/heizkg18-top1.pdf`,pdf);
  await openSection('heizinformationen');
  await energy.locator('input[name=purchase_0_price]').fill('0,123456');
  await submit(energy);
  assert.equal(await energy.locator('input[name=purchase_0_price]').inputValue(),'0,123456');
  const unchanged=await(await context.request.get(pdfURL)).body();
  assert(pdf.equals(unchanged),'existing PDF changed after energy edit');
  await energy.locator('input[name=purchase_0_price]').fill('0,09');
  await submit(energy);
  await openSection('vereinbarte-anteile');
  assert.equal(await shares.locator('[data-agreed-sum]').getAttribute('data-valid'),'true');
  assert.deepEqual(errors,[]);
  console.log('ok: share sum, explicit zeros, invalid POST, desktop/mobile, energy persistence, PDF content and immutable snapshot');
} catch (error) {
  await page.screenshot({path:`${out}/failure.png`,fullPage:true});
  console.log('Page heading:',await page.locator('h1').allTextContents());
  console.log('Page sections:',await page.locator('h2').allTextContents());
  throw error;
} finally {await context.close();await browser.close();}
