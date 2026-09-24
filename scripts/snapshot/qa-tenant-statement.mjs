// HAUSV-798: use the demo rig and the coordinator's pw.sh container wrapper.
import assert from 'node:assert/strict';
import { mkdir,writeFile,access } from 'node:fs/promises';
import { chromium } from 'playwright';
const [baseURL,out]=process.argv.slice(2);
assert(['localhost','127.0.0.1'].includes(new URL(baseURL).hostname));
await mkdir(out,{recursive:true});
const state=`${out}/session.json`;
const saved=await access(state).then(()=>true,()=>false);
const browser=await chromium.launch({headless:true});
const context=await browser.newContext({locale:'de-AT',timezoneId:'Europe/Vienna',...(saved?{storageState:state}:{})});
const page=await context.newPage();
const route=`${baseURL}/janusbergweg-123/app/settings/annual-statement?year=2025`;
const errors=[];page.on('pageerror',e=>errors.push(e.message));
try {
 await page.goto(route);
 if(await page.locator('input[name="email"]').count()){
  await page.locator('details:has(form[action$="/auth/request"])').evaluateAll(nodes=>nodes.forEach(n=>n.open=true));
  await page.locator('input[name="email"]').fill('vera.verwalter@musterstadt.example');
  await page.locator('form[action$="/auth/request"] button[type="submit"]').click();
  await page.locator('a.dev-link').click();await page.waitForURL('**/app**');
  await context.storageState({path:state});await page.goto(route);
 }
 if (!await page.locator('#rechtsgrundlage').count()) {
  const seeded=await context.request.post(`${baseURL}/janusbergweg-123/app/verwaltung/einstellungen/demo`,{headers:{Origin:baseURL}});assert.equal(seeded.status(),200);await page.goto(route);
 }
 await page.locator('#rechtsgrundlage').evaluate(n=>{for(let p=n.parentElement;p;p=p.parentElement)if(p.tagName==='DETAILS')p.open=true;});
 await page.locator('input[name="show_vat"]').check();
 await page.getByRole('button',{name:'Rechtsgrundlage speichern',exact:true}).click();
 await page.getByRole('button',{name:'Für alle Einheiten berechnen',exact:true}).click();
 await page.locator('[data-annual-statement-run]').waitFor();
 await page.getByRole('button',{name:'Abrechnung freigeben',exact:true}).click();
 const panel=page.locator('#mieterabrechnungen');await panel.waitFor();
 assert.equal(await panel.locator('input[name="active"]:checked').count(),3);
 assert((await panel.textContent()).includes('Matthias Dorn'));
 const top=panel.locator('details').filter({has:page.locator('input[name="unit_id"][value="top-2"]')});
 await top.evaluate(n=>n.open=true);
 await top.locator('input[name="heating_monthly"]').check();
 await top.getByRole('button',{name:'Mietverwaltung speichern',exact:true}).click();
 await top.evaluate(n=>n.open=true);
 await top.locator('input[name="statement_on"]').fill('2026-06-20');
 await top.getByRole('button',{name:'Mieterabrechnung erstellen',exact:true}).click();
 const statement=panel.locator('article').first();await statement.waitFor();
 assert((await statement.innerText()).includes('Top 2'));
 const pdfHref=await statement.getByRole('link',{name:'PDF-Vorschau'}).first().getAttribute('href');
 const draft=await context.request.get(new URL(pdfHref,baseURL).href);assert.equal(draft.status(),200);await writeFile(`${out}/tenant-draft.pdf`,await draft.body());
 await statement.getByRole('button',{name:'Mieterabrechnung freigeben',exact:true}).click();
 await statement.getByRole('button',{name:'Mieterabrechnung archivieren',exact:true}).click();
 assert((await statement.innerText()).includes('Archiviert'));
 await statement.getByRole('button',{name:'An Mietparteien und Eigentümer senden',exact:true}).click();
 await statement.getByRole('button',{name:'An Mietparteien und Eigentümer senden',exact:true}).click();
 assert((await panel.innerText()).includes('2 übersprungen'));
 const approved=await context.request.get(new URL(pdfHref,baseURL).href);assert.equal(approved.status(),200);await writeFile(`${out}/tenant-approved.pdf`,await approved.body());
 for(const width of [1440,390]){
  await page.setViewportSize({width,height:1000});await panel.locator('details').evaluateAll(nodes=>nodes.forEach(n=>n.open=false));await panel.evaluate(n=>n.scrollIntoView({block:'start'}));await page.evaluate(()=>scrollBy(0,-160));
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1);assert(!overflow,`overflow at ${width}`);
  await page.screenshot({path:`${out}/tenant-statements-${width}.png`});
 }
 assert.deepEqual(errors,[]);console.log('HAUSV-798: mandate, three demo units, preview, approve, archive, tenant + owner delivery, retry, mobile widths passed');
} catch(error) { console.log('Page title:', await page.title()); console.log((await page.locator('body').innerText()).slice(0,2200)); await page.screenshot({path:`${out}/failure.png`,fullPage:true}); throw error; } finally {await context.close();await browser.close();}
