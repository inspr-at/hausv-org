#!/usr/bin/env node
// HAUSV-801: organisation modes, lease overrides and legal-help navigation.
import { chromium } from 'playwright';
import { mkdirSync } from 'node:fs';

const [base, out] = process.argv.slice(2);
if (!base || !out || !['localhost','127.0.0.1'].includes(new URL(base).hostname)) throw new Error('Use a local demo rig and an absolute artifact directory');
mkdirSync(out, { recursive: true });
const browser = await chromium.launch({headless:true});
const context = await browser.newContext({locale:'de-AT',timezoneId:'Europe/Vienna'});
const page = await context.newPage();
const errors=[];
page.on('pageerror',error=>errors.push(error.message));
async function go(path) {
 const response=await page.goto(`${base}${path}`,{waitUntil:'networkidle'});
 if(response.status()!==200) throw new Error(`${path}: HTTP ${response.status()}`);
}
async function checkWidth(name, locator=page.locator('main').first()) {
 if(await page.evaluate(()=>document.documentElement.scrollWidth-document.documentElement.clientWidth)>1) throw new Error(`${name}: horizontal overflow`);
 await locator.screenshot({path:`${out}/${name}.png`});
}
async function submit(form) {
 await form.locator('button[type="submit"]').first().click();
 await page.waitForLoadState('networkidle');

}
try {
 await go('/');
 const login=page.locator('details:has(form[action$="/auth/request"])');
 if(await login.count()) await login.evaluate(n=>n.open=true);
 await page.locator('[name="email"]').fill('vera.verwalter@musterstadt.example');
 await submit(page.locator('form[action$="/auth/request"]'));
 await page.locator('a.dev-link').waitFor({state:'visible'});
 const url=new URL(await page.locator('a.dev-link').getAttribute('href'),base);
 url.host=new URL(base).host; url.protocol=new URL(base).protocol;
 await page.goto(url.href,{waitUntil:'networkidle'});
 for(const width of [1440,390]) {
  await page.setViewportSize({width,height:900});
  await go('/app/verwaltung/einstellungen');
  if(await page.locator('[name="valorisation_mode"] option').count()!==3) throw new Error('Expected three organisation modes');
  for(const mode of ['oevi','contract','wko']) {
   await page.locator('[name="valorisation_mode"]').selectOption(mode);
   await submit(page.locator('form.settings-form'));
   await go('/app/verwaltung/einstellungen');
   if(await page.locator('[name="valorisation_mode"]').inputValue()!==mode) throw new Error(`Organisation ${mode} not saved`);
  }
  await checkWidth(`settings-${width}`,page.locator('#wertsicherung'));
  await go('/app/settings/building/units/top-7/lease?bearbeiten=1');
  const selector=page.locator('[name="wirksamwerden_mode"]');
  if(await selector.locator('option').count()!==4) throw new Error('Expected inheritance plus three lease modes');
  await selector.selectOption('oevi');
  await checkWidth(`lease-edit-${width}`,page.locator('label:has(select[name="wirksamwerden_mode"])'));
  await page.getByRole('button',{name:'Speichern',exact:true}).click();
  await page.waitForLoadState('networkidle');
  if(!(await page.locator('main').innerText()).includes('ab endgültiger Verlautbarung (ÖVI)')) throw new Error('Lease override summary missing');
  await checkWidth(`lease-summary-${width}`);
  await go('/app/settings/building/units/top-7/lease?bearbeiten=1');
  if(await page.locator('[name="wirksamwerden_mode"]').inputValue()!=='oevi') throw new Error('Override did not persist');
  await page.locator('[name="wirksamwerden_mode"]').selectOption('');
  await page.getByRole('button',{name:'Speichern',exact:true}).click();
  await page.waitForLoadState('networkidle');
  if(!(await page.locator('main').innerText()).includes('Standard der Hausverwaltung')) throw new Error('Inheritance not restored');
  await go('/app/settings/valorisation');
  const item=page.locator('details.vr-item').filter({has:page.locator('summary strong',{hasText:'Top 7 ·'})}).first();
  await item.locator('summary').first().click();
  const timing=await item.innerText();
  for(const label of ['Indexmonat:', 'Endgültige Veröffentlichung:', 'Wirksamwerden (angenommen):', 'vorsichtig (WKO)', 'Erster zahlbarer Zinstermin']) {
   if(!timing.includes(label)) throw new Error(`Separate timing evidence missing: ${label}`);
  }
  await checkWidth(`valorisation-dates-${width}`,item.locator('p.vr-note').filter({hasText:'Indexmonat:'}));
  for(const [route,anchor] of [['/app/settings/valorisation','recht-wirksamwerden'],['/app/settings/annual-statement','recht-weg']]) {
   await go(route);
   await page.getByRole('link',{name:'Warum?',exact:true}).first().click();
   await page.waitForLoadState('networkidle');
   if(new URL(page.url()).hash!==`#${anchor}`) throw new Error(`Wrong help destination from ${route}`);
   const topic=page.locator(`#${anchor}`);
   await topic.locator('details').evaluateAll(nodes=>nodes.forEach(n=>n.open=true));
   if(!(await page.locator('main').innerText()).includes('Keine Rechtsberatung')) throw new Error('Legal note missing');
   if(await topic.locator('a[href^="https://"]').count()<3) throw new Error('Legal sources missing');
   if(anchor==='recht-wirksamwerden') {
    const text=await topic.innerText();
    for(const phrase of ['Zugang spätestens 14 Tage', 'spätestens am 21.04.2026 zugehen', 'Postlaufzeit einplanen.']) {
     if(!text.includes(phrase)) throw new Error(`Receipt guidance missing: ${phrase}`);
    }
   }
   await checkWidth(`help-${anchor}-${width}`,topic);
  }
  const reserve=page.locator('#recht-mindestruecklage');
  await reserve.locator('details').evaluateAll(nodes=>nodes.forEach(n=>n.open=true));
  const reserveText=await reserve.innerText();
  for(const text of ['25.09.2026', '0,90 €', '1,06 €', '1,12 €', 'Noch nicht verlautbart', 'Keine Rechtsberatung', 'entscheidet keine Ausnahme']) {
   if(!reserveText.includes(text)) throw new Error(`Reserve guidance missing: ${text}`);
  }
  await checkWidth(`help-recht-mindestruecklage-${width}`,reserve);
 }
 if(errors.length) throw new Error(errors.join('\n'));
 console.log('qa-wirksamwerden ok: settings save, override/inheritance, help links, disclosures, 390/1440');
} finally {await browser.close();}
