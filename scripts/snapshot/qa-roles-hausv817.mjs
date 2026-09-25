#!/usr/bin/env node
// HAUSV-817: run on a fresh seeded demo rig; mutates disposable fixture data only.
import assert from 'node:assert/strict';
import {mkdirSync,writeFileSync} from 'node:fs';
import {join} from 'node:path';
import {chromium} from 'playwright';
const [base,out]=process.argv.slice(2);
assert(base && out,'usage: qa-roles-hausv817.mjs <local demo baseURL> <artifact dir>');
const origin=new URL(base).origin;
assert(['localhost','127.0.0.1'].includes(new URL(base).hostname),'local disposable demo rig required');
mkdirSync(out,{recursive:true});
const tenant=`${origin}/janusbergweg-123`;
const browser=await chromium.launch({headless:true});
const sessions=new Map(),results=[];
async function login(name){
 const context=await browser.newContext({locale:'de-AT',timezoneId:'Europe/Vienna',viewport:{width:1440,height:1000}});
 const page=await context.newPage();
 await page.goto(`${origin}/`,{waitUntil:'networkidle'});
 await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(es=>es.forEach(e=>e.open=true));
 await page.locator('input[name="email"]').fill(`${name}@musterstadt.example`);
 await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
 const dev=page.locator('a.dev-link');await dev.waitFor();
 const target=new URL(await dev.getAttribute('href'),origin);target.host=new URL(origin).host;target.protocol=new URL(origin).protocol;
 await page.goto(target.href,{waitUntil:'networkidle'});
 assert(page.url().includes('/app'),`login ${name}`);
 const session={context,page};sessions.set(name,session);return session;
}
async function capture(page,name,path){
 for(const width of [1440,390]){
  await page.setViewportSize({width,height:width===390?844:1000});
  await page.goto(`${tenant}${path}`,{waitUntil:'networkidle'});
  assert(!(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1)),`${name} overflow at ${width}`);
  await page.screenshot({path:join(out,`${name}-${width}.png`),fullPage:true});
 }
}
async function create(session,{title,location='own-unit',detail='Top 7',category='Reparatur',unit='',want=303}){
 const {page,context}=session;
 await page.goto(`${tenant}/app/anliegen?new=1`,{waitUntil:'networkidle'});
 const form=page.locator('form[data-issue-wizard]');
 const fields=await form.evaluate(f=>Object.fromEntries([...new FormData(f)].filter(([,v])=>typeof v==='string')));
 const response=await context.request.post(new URL(await form.getAttribute('action'),page.url()).href,{headers:{Origin:origin},multipart:{...fields,title,category,body:'Lokaler Regressionstest für die Anliegen-Sichtbarkeit.',location_type:location,location_detail:detail,unit_id:unit},maxRedirects:0});
 assert.equal(response.status(),want,`create ${title}`);
 if(want!==303)return;
 assert.match(response.headers().location,/\/app\/anliegen\/.+\?created=1$/);
 return new URL(response.headers().location,origin).pathname.split('/').at(-1);
}
try{
 const sophie=await login('sophie.bewohner');
 const hedwig=await login('hedwig.beirat');
 const alina=await login('alina.eigentuemer');
 const vera=await login('vera.verwalter');
 const matthias=await login('matthias.mieter');
 const cases=[
  {title:'QA817 Privat Top 7',visible:false},
  {title:'QA817 Beirat eigene Einheit',detail:'Top 2',visible:true},
  {title:'QA817 Allgemeinflächen',detail:'Stiegenhaus',location:'common',visible:true},
 ];
 for(const item of cases)item.id=await create(sophie,item);
 // Capture the home before the Beirat creates newer reports, so its newest rows exercise privacy.
 for(const route of ['/app','/app/anliegen']){
  await hedwig.page.goto(`${tenant}${route}`,{waitUntil:'networkidle'});
  const text=await hedwig.page.locator('body').innerText();
  assert(!text.includes(cases[0].title),`Beirat private title leaked on ${route}`);
  for(const item of cases.slice(1))assert(text.includes(item.title),`Beirat missing ${item.title} on ${route}`);
 }
 for(const item of cases){
  const response=await hedwig.context.request.get(`${tenant}/app/anliegen/${item.id}`);
  assert.equal(response.status(),item.visible?200:403,`Beirat detail ${item.title}`);
  assert.equal((await sophie.context.request.get(`${tenant}/app/anliegen/${item.id}`)).status(),200,'author access');
  assert.equal((await vera.context.request.get(`${tenant}/app/anliegen/${item.id}`)).status(),200,'management access');
 }
 for(const session of [alina,matthias])assert.equal((await session.context.request.get(`${tenant}/app/anliegen/${cases[0].id}`)).status(),403,'other resident private access');
 await capture(hedwig.page,'hedwig-home','/app');
 await capture(hedwig.page,'hedwig-anliegen','/app/anliegen');
 await capture(sophie.page,'sophie-anliegen','/app/anliegen');
 const labels=await sophie.page.locator('.issue-meta .pill').allTextContents();
 for(const label of ['Betriebskosten/Vorschreibung','Reparatur'])assert(labels.includes(label),`category ${label}`);
 for(const raw of ['versicherung','betriebskosten','reparatur'])assert(!labels.includes(raw),`raw category ${raw}`);
 await create(hedwig,{title:'QA817 Beirat meldet eigene Einheit',detail:'Badezimmer',unit:'top-2'});
 await create(hedwig,{title:'QA817 Beirat meldet Allgemeinfläche',location:'common',detail:'Keller'});
 await create(hedwig,{title:'QA817 Fremde Einheit abgewiesen',unit:'top-7',want:403});
 results.push({check:'Beirat list, home, detail, author/management access, creation scope and category labels',pass:true});
 await alina.page.goto(`${tenant}/app/abstimmungen`,{waitUntil:'networkidle'});
 const ballot=alina.page.locator('#ballot-demo-ballot-fassade-2027');
 const stats=await ballot.locator('.vote-result-stat').allTextContents();
 const participation=Number(stats.find(s=>s.includes('Teilnahme')).match(/[\d]+,[\d]+/)[0].replace(',','.'));
 assert(participation>=60 && participation<=75,`realistic participation ${participation}`);
 assert(stats.some(s=>/Stimmen\s*9/.test(s)),`nine voters ${stats}`);
 assert(stats.some(s=>s.includes('Quorum erreicht')),'quorum reached by multiple owners');
 await capture(alina.page,'alina-abstimmungen','/app/abstimmungen');
 results.push({check:'closed ballot seed',participation,stats,pass:true});
 await vera.page.goto(`${tenant}/app/parking/settings?section=charging`,{waitUntil:'networkidle'});
 assert((await vera.page.locator('.pk-rule-summary').innerText()).includes('3,3 kW'),'start summary');
 assert((await vera.page.locator('.pk-rule-summary').innerText()).includes('1,5 kW'),'stop summary');
 await vera.page.getByText('Erweiterte Grenzwerte',{exact:true}).click();
 assert.equal(await vera.page.locator('[name=start_feed_in_kw]').inputValue(),'3,3');
 assert.equal(await vera.page.locator('[name=stop_feed_in_kw]').inputValue(),'1,5');
 await vera.page.locator('[name=start_feed_in_kw]').fill('3,4');
 await vera.page.locator('[name=stop_feed_in_kw]').fill('1,6');
 await vera.page.getByRole('button',{name:'Laderegeln speichern'}).click();
 await vera.page.waitForURL(/charging=saved/);
 assert((await vera.page.locator('.pk-rule-summary').innerText()).includes('3,4 kW'),'saved kW round trip');
 await vera.page.getByText('Erweiterte Grenzwerte',{exact:true}).click();
 await vera.page.locator('[name=start_feed_in_kw]').fill('3,3');
 await vera.page.locator('[name=stop_feed_in_kw]').fill('1,5');
 await vera.page.getByRole('button',{name:'Laderegeln speichern'}).click();
 await vera.page.waitForURL(/charging=saved/);
 await capture(vera.page,'vera-laderegeln','/app/parking/settings?section=charging');
 for(const width of [1440,390]){
  await vera.page.setViewportSize({width,height:width===390?844:1000});
  await vera.page.goto(`${tenant}/app/parking/settings?section=charging`,{waitUntil:'networkidle'});
  await vera.page.getByText('Erweiterte Grenzwerte',{exact:true}).click();
  await vera.page.evaluate(()=>window.scrollTo(0,0));
  await vera.page.screenshot({path:join(out,`vera-laderegeln-inputs-${width}.png`),fullPage:true});
 }
 results.push({check:'charging kW display, localized inputs and save/reload',pass:true});
 writeFileSync(join(out,'oracle.json'),JSON.stringify(results,null,2));
 console.log(JSON.stringify(results));
}finally{await browser.close();}
