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
 assert.equal(await page.locator('input[name="show_vat"]').isChecked(),true,'Demo VAT basis must already be enabled');
 await page.getByRole('button',{name:'Für alle Einheiten berechnen',exact:true}).click();
 await page.locator('[data-annual-statement-run]').waitFor();
 await page.getByRole('button',{name:'Abrechnung freigeben',exact:true}).click();
 const panel=page.locator('#mieterabrechnungen');await panel.waitFor();
 assert.equal(await panel.locator('input[name="active"]:checked').count(),3);
 assert((await panel.textContent()).includes('Matthias Dorn'));
 const unit = id => panel.locator('.ts-unit').filter({has:page.locator(`input[name="unit_id"][value="${id}"]`)});
 const third = unit('top-3');
 const thirdSummary = await third.locator('summary').innerText();
 for (const text of ['Clara Berger', '01.01.2025 – 30.06.2025', 'Daniel Leitner', '01.07.2025 – 31.12.2025', 'Matthias Dorn', 'Teilanwendung']) assert(thirdSummary.includes(text), `owner-change summary missing ${text}`);
 assert.equal(await third.getByRole('button',{name:'Mieterabrechnung erstellen',exact:true}).count(),0);
 assert.match(await third.textContent(), /Bei Eigentümerwechsel/);
 const fourth = unit('top-4');
 assert.match(await fourth.locator('summary').innerText(), /Clara Novak.*Vollanwendung/);
 for (const key of ['abfall','reinigung']) assert.equal(await fourth.locator(`input[name=passable][value=${key}]`).isChecked(),true);
 for (const key of ['versicherung','lift']) assert.equal(await fourth.locator(`input[name=passable][value=${key}]`).isChecked(),false);
 for (const [label, costs, saldo] of [['Top 1','871,69 €','271,69 €'],['Top 2','999,71 €','399,71 €'],['Top 5','1.312,37 €','712,37 €'],['Top 8','1.158,54 €','558,54 €']]) {
   const row=page.locator(`.annual-unit-group[aria-label="${label}"] .annual-unit-row`).first();
   assert.deepEqual(await row.locator('.annual-money').allTextContents(),[costs,'600,00 €',`Nachzahlung ${saldo}`]);
 }
 const runID=await page.locator('[data-annual-statement-run]').getAttribute('data-annual-statement-run');
 const ownerChange=await context.request.post(`${baseURL}/janusbergweg-123/app/settings/annual-statement/runs/${runID}/tenant-statements`,{form:{unit_id:'top-3',statement_on:'2026-06-20'},headers:{Origin:baseURL},maxRedirects:0});
 assert.equal(ownerChange.status(),303);
 assert.match(new URL(ownerChange.headers().location,baseURL).searchParams.get('tenant-message'),/Eigentümerwechsel/);
 const top=panel.locator('details').filter({has:page.locator('input[name="unit_id"][value="top-2"]')});
 await top.evaluate(n=>n.open=true);
 await top.locator('input[name="heating_monthly"]').check();
 await top.getByRole('button',{name:'Mietverwaltung speichern',exact:true}).click();
 await top.evaluate(n=>n.open=true);
 await top.locator('input[name="statement_on"]').fill('2026-06-20');
 await top.getByRole('button',{name:'Mieterabrechnung erstellen',exact:true}).click();
 const statement=panel.locator('article').first();await statement.waitFor();
 assert.equal(await statement.locator('h3').innerText(), 'Top\u00a02');
 assert.match(await statement.innerText(),/Guthaben\s+58,30 €/);
 const pdfHref=await statement.getByRole('link',{name:'PDF-Vorschau'}).first().getAttribute('href');
 const draft=await context.request.get(new URL(pdfHref,baseURL).href);assert.equal(draft.status(),200);await writeFile(`${out}/tenant-draft.pdf`,await draft.body());
 await statement.getByRole('button',{name:'Mieterabrechnung freigeben',exact:true}).click();
 await statement.getByRole('button',{name:'Mieterabrechnung archivieren',exact:true}).click();
 assert((await statement.innerText()).includes('Archiviert'));
 await statement.getByRole('button',{name:'An Mietparteien und Eigentümer senden',exact:true}).click();
 await statement.getByRole('button',{name:'An Mietparteien und Eigentümer senden',exact:true}).click();
 assert((await panel.innerText()).includes('2 übersprungen'));
 const approved=await context.request.get(new URL(pdfHref,baseURL).href);assert.equal(approved.status(),200);await writeFile(`${out}/tenant-approved.pdf`,await approved.body());
 for (const [id, expected] of [['top-1','53,29 €'],['top-5','58,37 €'],['top-8','51,17 €']]) {
   const row = unit(id);
   await row.evaluate(n=>n.open=true);
   if (!await row.locator('input[name=active]').isChecked()) {
     await row.locator('input[name=active]').check();
     await row.locator('input[name=heating_monthly]').check();
     const owner=await row.locator('select[name=owner_email] option').nth(1).getAttribute('value');
     await row.locator('select[name=owner_email]').selectOption(owner);
     for (const field of await row.locator(`input[name=contract_lease-${id}]`).all()) await field.check();
     await row.getByRole('button',{name:'Mietverwaltung speichern',exact:true}).click();
     await row.evaluate(n=>n.open=true);
   }
   await row.locator('input[name=statement_on]').fill('2026-06-20');
   if (await row.locator('input[name=contract_due_on]').count()) await row.locator('input[name=contract_due_on]').fill('2026-08-05');
   await row.getByRole('button',{name:'Mieterabrechnung erstellen',exact:true}).click();
   const card=panel.locator('article').filter({has:page.getByRole('heading',{name:id.replace('top-','Top '),exact:true})});
   assert((await card.innerText()).includes(expected), `${id} tenant amount`);
   assert((await card.innerText()).includes('Nachzahlung'));
 }
 for(const width of [1440,390]){
  await page.setViewportSize({width,height:1000});await panel.locator('details').evaluateAll(nodes=>nodes.forEach(n=>n.open=false));await panel.evaluate(n=>n.scrollIntoView({block:'start'}));await page.evaluate(()=>scrollBy(0,-160));
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1);assert(!overflow,`overflow at ${width}`);
  await page.screenshot({path:`${out}/tenant-statements-${width}.png`});
  for (const [id, name] of [['top-2','tenant-panel-open'],['top-3','owner-change'],['top-4','new-mandate']]) {
    await panel.locator('.ts-unit').evaluateAll(nodes=>nodes.forEach(n=>n.open=false));
    const row=unit(id);await row.evaluate(n=>n.open=true);
    await page.setViewportSize({width,height:Math.ceil(await row.evaluate(n=>n.getBoundingClientRect().height))+200});
    await row.evaluate(n=>window.scrollTo(0,n.getBoundingClientRect().top+window.scrollY-150));
    await row.screenshot({path:`${out}/${name}-${width}.png`});
  }
  const card=panel.locator('article').filter({has:page.getByRole('heading',{name:'Top 2',exact:true})});
  await card.screenshot({path:`${out}/top2-card-${width}.png`});
 }
 // Existing non-VAT runs explain the prerequisite before any creation click.
 await page.locator('#rechtsgrundlage').evaluate(n=>{for(let p=n.parentElement;p;p=p.parentElement)if(p.tagName==='DETAILS')p.open=true;});
 await page.locator('input[name=show_vat]').uncheck();
 await page.getByRole('button',{name:'Rechtsgrundlage speichern',exact:true}).click();
 await page.getByRole('button',{name:'Für alle Einheiten berechnen',exact:true}).click();
 await page.getByRole('button',{name:'Abrechnung freigeben',exact:true}).click();
 await unit('top-2').evaluate(n=>n.open=true);
 assert.match(await unit('top-2').innerText(),/ausgewiesene Netto-\/USt-Basis/);
 assert.equal(await unit('top-2').getByRole('button',{name:'Mieterabrechnung erstellen',exact:true}).count(),0);
 await unit('top-2').getByRole('link',{name:'Umsatzsteuer in der Rechtsgrundlage ausweisen'}).click();
 await page.locator('input[name=show_vat]').waitFor({state:'visible'});
 assert.equal(await page.locator('input[name=show_vat]').isVisible(),true,'Setting link must open Grundlagen');
 assert.deepEqual(errors,[]);console.log('HAUSV-798: mandate, three demo units, preview, approve, archive, tenant + owner delivery, retry, mobile widths passed');
} catch(error) { console.log('Page title:', await page.title()); console.log((await page.locator('body').innerText()).slice(0,2200)); await page.screenshot({path:`${out}/failure.png`,fullPage:true}); throw error; } finally {await context.close();await browser.close();}
