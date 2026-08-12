// Focused browser contract for the reduced energy-cockpit lead.
//
// Keep this module limited to the content above #energieverlauf. The chart and
// every section after it retain their established checks in qa-main-flows.mjs.

import { mkdirSync } from 'node:fs';
import { join } from 'node:path';

export const energyRedesignWidths = new Set([
  320, 359, 360, 361, 390, 430, 559, 560, 561, 619, 620, 621,
  768, 899, 900, 901, 1023, 1024, 1280, 1366, 1439, 1440, 1441,
  1600, 1672, 1920, 2048,
]);

const ownerSidebar = [
  { path: '/app', label: 'Hausüberblick' },
  { path: '/app/energie', label: 'QA Zuhause', secondary: 'Top 11' },
  { path: '/app/announcements', label: 'Aushang' },
  { path: '/app/events', label: 'Termine' },
  { path: '/app/kontakte', label: 'Kontakte' },
  { path: '/app/dokumente', label: 'Dokumente' },
  { path: '/app/anliegen', label: 'Anliegen' },
  { path: '/app/abstimmungen', label: 'Abstimmungen' },
  { path: '/app/audit', label: 'Verlauf' },
  { path: '/app/settings', label: 'Einstellungen' },
];

// The flow is rendered client-side (assets/energy-flow.js) into [data-edge]
// tiles; visual symbols are locally vendored Lucide assets while the ribbons
// remain data-driven SVG geometry.
const flowNodes = [
  { id: 'hub', patterns: [/Hausverbrauch/i, /1,8/, /kW/] },
  { id: 'producer-0', patterns: [/PV(?:-Leistung|-Erzeugung)?/i, /3,1/, /kW/] },
  { id: 'grid', patterns: [/Netz(?:bezug)?/i, /2,4/, /kW/] },
  { id: 'storage', patterns: [/Speicher/i, /78\s*%/i, /lädt|entlädt|wartet/i] },
];

const disclosures = [
  {
    selector: '.energy-info-disclosure',
    label: 'Live-Info',
    patterns: [/Home Assistant|nur gelesen|weitere Messwerte|Messwert/i],
  },
  {
    selector: '.energy-tariff-disclosure',
    label: 'Tarif-Erklärung',
    patterns: [/Viertelstunde|Netztarif|Begutachtungsentwurf|Modell/i],
  },
  {
    selector: '.energy-next-why',
    label: 'Empfehlungsbegründung',
    patterns: [/Aufwand|Wirkung|Empfehlung|Mess|Schwankung/i],
  },
];

const metricDisclosures = [
  { id: 'tariff', patterns: [/Höchste Viertelstunde/i, /Verrechnet/i, /Kalendermonat/i, /2-kW-Sockel/i] },
  { id: 'annual', patterns: [/Jahreswert/i, /Netztarif|Stromrechnung/i] },
];

function fail(label, message, evidence) {
  const suffix = evidence === undefined ? '' : ` (${JSON.stringify(evidence)})`;
  throw new Error(`${label}: ${message}${suffix}`);
}

export async function assertOwnerEnergySidebarOrder(page, label) {
  const actual = await page.locator('#portal-navigation .nav-item').evaluateAll((links) => links.map((link) => {
    const homeName = link.querySelector('[data-home-display-name]');
    const unit = link.querySelector('[data-home-unit-label]');
    const navLabel = link.querySelector('.nav-label');
    return {
      path: new URL(link.href, document.baseURI).pathname,
      label: (homeName || navLabel)?.textContent?.trim().replace(/\s+/g, ' ') || '',
      secondary: unit?.textContent?.trim().replace(/\s+/g, ' ') || '',
    };
  }));
  const expected = ownerSidebar.map((item) => ({ ...item, secondary: item.secondary || '' }));
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    fail(label, 'Reihenfolge oder Beschriftung der bestehenden Eigentümer-Navigation wurde verändert', { expected, actual });
  }
}

export async function assertEnergyTopContent(page, label) {
  const diagram = page.locator('[data-energy-flow-diagram]');
  if ((await diagram.count()) !== 1 || !(await diagram.isVisible())) {
    fail(label, 'genau ein sichtbares [data-energy-flow-diagram] fehlt');
  }

  const readingCount = Number(await diagram.getAttribute('data-energy-reading-count'));
  if (!Number.isFinite(readingCount) || readingCount < 8) {
    fail(label, 'deterministische Live-Daten sind am Diagramm nicht vollständig ausgewiesen', { readingCount });
  }

  for (const expected of flowNodes) {
    const node = diagram.locator(`[data-edge="${expected.id}"]`);
    if ((await node.count()) !== 1 || !(await node.isVisible())) {
      fail(label, `sichtbarer Energie-Knoten „${expected.id}“ fehlt`);
    }
    const text = await node.evaluate((element) => [
      element.textContent,
      element.getAttribute('aria-label'),
      ...[...element.querySelectorAll('[aria-label]')].map((child) => child.getAttribute('aria-label')),
    ].filter(Boolean).join(' ').trim().replace(/\s+/g, ' '));
    const missing = expected.patterns.filter((pattern) => !pattern.test(text)).map(String);
    if (missing.length) {
      fail(label, `Energie-Knoten „${expected.id}“ verliert verständliche Live-Information`, { text, missing });
    }
  }

  // Ribbons: one filled polygon per active flow, gradient-filled. The
  // deterministic fixture always has PV production plus a grid flow.
  const ribbons = await diagram.locator('svg.energy-flow-ribbons path').evaluateAll((paths) =>
    paths.map((path) => path.getAttribute('fill') || ''));
  if (ribbons.length < 2 || ribbons.some((fill) => !fill.startsWith('url('))) {
    fail(label, 'Energiefluss-Ribbons fehlen oder sind nicht als Verlaufs-Linienzüge gefüllt', ribbons);
  }
  const railTiles = await diagram.locator('.energy-flow-rail .energy-flow-big:not(.ghost)').count();
  const railGhost = await diagram.locator('.energy-flow-rail .energy-flow-big.ghost').count();
  const railPrios = await diagram.locator('.energy-flow-rail .energy-flow-big .prio').allTextContents();
  const priosSequential = railPrios.every((text, index) => text.trim() === String(index + 1));
  if (!railTiles || railGhost !== 1 || !priosSequential) {
    fail(label, 'Verbraucher-Spalte fehlt, hat keine fortlaufenden Prioritäten oder keine Hinzufügen-Kachel', { railTiles, railGhost, railPrios });
  }

  const mapping = diagram.locator('a[href^="/app/zuhause/onboarding"]').filter({ hasText: 'Messwerte zuordnen' });
  if ((await mapping.count()) !== 1) {
    fail(label, 'bestehender Einstieg „Messwerte zuordnen“ fehlt im neuen Kopf');
  }
  const mappingURL = new URL(await mapping.getAttribute('href'), page.url());
  if (mappingURL.pathname !== '/app/zuhause/onboarding' ||
      mappingURL.searchParams.get('step') !== '4') {
    fail(label, 'bestehender Einstieg „Messwerte zuordnen“ fehlt im neuen Kopf');
  }
  if (await diagram.getByText('Home Current Consumption', { exact: true }).isVisible().catch(() => false)) {
    fail(label, 'technische Home-Assistant-Rohbezeichnung konkurriert mit dem Diagramm');
  }

  const tariffStatus = page.locator('.energy-tariff > .energy-card-head .pill');
  if ((await tariffStatus.count()) !== 1 || (await tariffStatus.innerText()).trim() !== 'Entwurf' ||
      (await tariffStatus.getAttribute('aria-label')) !== 'Entwurf · nicht verbindlich' ||
      (await tariffStatus.getAttribute('title')) !== 'Entwurf · nicht verbindlich') {
    fail(label, 'Tarifstatus ist nicht kompakt sichtbar und vollständig als Hover-/Hilfetext erhalten');
  }

  const iconContract = await page.evaluate(() => {
    const scope = document.querySelector('.energy-cockpit-main');
    const icons = [...(scope?.querySelectorAll('.energy-mode-strip .energy-ui-icon, .energy-cockpit-top .energy-ui-icon') || [])];
    return {
      count: icons.length,
      inlineSVGs: [...(scope?.querySelectorAll('.energy-mode-strip svg, .energy-cockpit-top svg') || [])]
        .filter((svg) => !svg.closest('.energy-flow-area')).length,
      inlineFlowIconSVGs: scope?.querySelectorAll('.energy-flow-tile .ico svg, .energy-flow-big svg').length || 0,
      missingExternalAsset: icons.map((icon) => {
        const style = getComputedStyle(icon);
        const mask = style.maskImage || style.webkitMaskImage || '';
        return {
          className: icon.className,
          mask,
        };
      }).filter(({ mask }) => !/\/assets\/icons\/lucide\/[a-z0-9-]+\.svg(?:\?|["')]|$)/i.test(mask)),
    };
  });
  if (iconContract.count < 6 || iconContract.inlineSVGs || iconContract.inlineFlowIconSVGs || iconContract.missingExternalAsset.length) {
    fail(label, 'Redesign verwendet nicht durchgängig die etablierten externen Lucide-SVGs', iconContract);
  }
}

async function assertEnergyConsumerManagement(page, { label, width }) {
  const cards = page.locator('.energy-flow-rail .energy-flow-big[data-consumer-id]');
  if (!(await cards.count())) fail(label, 'kein direkt bearbeitbarer Verbraucher vorhanden');
  const first = cards.first();
  const editTrigger = first.locator('button.energy-flow-main');
  const before = await cards.evaluateAll((nodes) => nodes.map((node) => {
    const box = node.getBoundingClientRect();
    return { top: box.top, height: box.height };
  }));
  if (width >= 1180) {
    await editTrigger.hover();
    await page.waitForTimeout(180);
    const after = await cards.evaluateAll((nodes) => nodes.map((node) => {
      const box = node.getBoundingClientRect();
      return { top: box.top, height: box.height };
    }));
    if (JSON.stringify(before) !== JSON.stringify(after) ||
        !(await first.getByText('Klicken zum Bearbeiten', { exact: true }).isVisible())) {
      fail(label, 'Hover verändert weiterhin die Verbraucher-Geometrie oder zeigt den Bearbeitungshinweis nicht', { before, after });
    }
  }

  await editTrigger.click();
  const dialog = page.locator('#energy-consumer-dialog');
  if (!(await dialog.isVisible()) || !(await dialog.evaluate((node) => node.open))) {
    fail(label, 'Klick auf Verbraucher öffnet den Bearbeitungsdialog nicht');
  }
  const state = await dialog.evaluate((node) => {
    const box = node.getBoundingClientRect();
    const choices = [...node.querySelectorAll('[data-consumer-icon-choice]')];
    return {
      title: node.querySelector('[data-consumer-dialog-title]')?.textContent?.trim(),
      name: node.querySelector('[name="name"]')?.value,
      priorityOptions: node.querySelector('[name="priority"]')?.options.length || 0,
      iconChoices: choices.length,
      inlineSVGs: node.querySelectorAll('svg').length,
      missingMasks: choices.filter((choice) => {
        const icon = choice.querySelector('.energy-ui-icon');
        const style = getComputedStyle(icon);
        return !/\/assets\/icons\/lucide\/[a-z0-9-]+\.svg/i.test(style.maskImage || style.webkitMaskImage || '');
      }).length,
      box: { left: box.left, right: box.right, top: box.top, bottom: box.bottom, width: box.width, height: box.height },
      documentWidth: document.documentElement.scrollWidth,
    };
  });
  if (state.title !== 'Verbraucher bearbeiten' || !state.name || !state.priorityOptions ||
      state.iconChoices < 10 || state.inlineSVGs || state.missingMasks ||
      state.box.left < -1 || state.box.right > width + 1 || state.documentWidth > width + 1) {
    fail(label, 'Bearbeitungsdialog ist unvollständig, läuft über oder verwendet nicht-lokale Symbole', state);
  }
  if (process.env.HV_QA_SCREENSHOT_DIR) {
    mkdirSync(process.env.HV_QA_SCREENSHOT_DIR, { recursive: true });
    await dialog.screenshot({
      path: join(process.env.HV_QA_SCREENSHOT_DIR, `energy-consumer-dialog-${width}.png`),
    });
  }

  const search = dialog.locator('[data-consumer-icon-search]');
  await search.fill('Klima');
  const visibleChoices = await dialog.locator('[data-consumer-icon-choice]:visible').count();
  if (visibleChoices !== 1) fail(label, 'Lucide-Symbolsuche filtert die Auswahl nicht eindeutig', { visibleChoices });
  await search.fill('cable-car');
  const completeLibraryChoice = dialog.locator('label:has([name="icon_choice"][value="cable-car"])');
  if (!(await completeLibraryChoice.isVisible())) {
    fail(label, 'vollständige Lucide-Library ist nicht durchsuchbar');
  }
  await completeLibraryChoice.click();
  if (await dialog.locator('[data-consumer-icon-value]').inputValue() !== 'cable-car') {
    fail(label, 'Symbol aus vollständiger Lucide-Library lässt sich nicht auswählen');
  }
  await search.fill('');
  await dialog.locator('label:has([name="icon_choice"][value="drill"])').click();
  if (!(await dialog.locator('[name="icon_choice"][value="drill"]').isChecked()) ||
      await dialog.locator('[data-consumer-icon-value]').inputValue() !== 'drill') {
    fail(label, 'Lucide-Symbol lässt sich nicht auswählen');
  }
  await page.waitForFunction(() => {
    const power = document.querySelector('#energy-consumer-dialog [name="consumer_power_entity"]');
    const status = document.querySelector('#energy-consumer-dialog [data-consumer-measurement-status]');
    return power?.options.length > 1 && !/geladen/.test(status?.textContent || '');
  });
  if (await dialog.locator('[name="consumer_energy_entity"] option').count() < 2) {
    fail(label, 'Home-Assistant-Leistung und Energiezähler werden nicht getrennt angeboten');
  }
  const remove = dialog.locator('[data-consumer-delete]');
  if (!(await remove.isVisible())) fail(label, 'Löschoption fehlt beim bestehenden Verbraucher');
  await remove.click();
  if (!(await dialog.locator('[data-consumer-delete-confirm]').isVisible())) {
    fail(label, 'Löschen verlangt keine explizite zweite Bestätigung');
  }
  await dialog.locator('[data-consumer-delete-cancel]').click();
  await page.keyboard.press('Escape');
  if (await dialog.isVisible() || !(await editTrigger.evaluate((node) => document.activeElement === node))) {
    fail(label, 'Escape schließt den Dialog nicht mit Fokus-Rückgabe');
  }

  const addTrigger = page.locator('.energy-flow-big.ghost button.energy-flow-main');
  await addTrigger.click();
  const addState = await dialog.evaluate((node) => ({
    title: node.querySelector('[data-consumer-dialog-title]')?.textContent?.trim(),
    name: node.querySelector('[name="name"]')?.value,
    removeHidden: node.querySelector('[data-consumer-delete]')?.hidden,
  }));
  if (addState.title !== 'Verbraucher hinzufügen' || addState.name !== '' || !addState.removeHidden) {
    fail(label, 'Hinzufügen-Kachel verwendet nicht denselben leeren Dialog', addState);
  }
  await page.keyboard.press('Escape');
}

async function disclosureState(root) {
  return root.evaluate((element) => {
    const trigger = element.querySelector(':scope > summary, :scope > button');
    const panel = [...element.children].find((child) => child !== trigger) || null;
    const panelStyle = panel ? getComputedStyle(panel) : null;
    const panelRect = panel?.getBoundingClientRect();
    const triggerRect = trigger?.getBoundingClientRect();
    const isDetails = element.tagName === 'DETAILS';
    const visiblyRendered = (node) => {
      const closedDetails = node.closest('details:not([open])');
      if (closedDetails) {
        const summary = closedDetails.querySelector(':scope > summary');
        if (node !== summary && !summary?.contains(node)) return false;
      }
      const style = getComputedStyle(node);
      const box = node.getBoundingClientRect();
      return style.display !== 'none' && style.visibility !== 'hidden' && box.width > 0 && box.height > 0;
    };
    const panelOutliers = panel && panelRect ? [...panel.querySelectorAll('*')]
      .filter(visiblyRendered)
      .map((node) => ({
        node: `${node.tagName.toLowerCase()}.${String(node.className || '').trim().replace(/\s+/g, '.')}`,
        text: (node.getAttribute('aria-label') || node.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 60),
        box: (() => {
          const box = node.getBoundingClientRect();
          return { left: box.left, right: box.right, width: box.width };
        })(),
      }))
      .filter(({ box }) => box.left < panelRect.left - 1 || box.right > panelRect.right + 1) : [];
    return {
      isDetails,
      open: isDetails ? element.open : trigger?.getAttribute('aria-expanded') === 'true',
      triggerTag: trigger?.tagName || '',
      triggerName: (trigger?.getAttribute('aria-label') || trigger?.textContent || '').trim().replace(/\s+/g, ' '),
      triggerRect: triggerRect ? {
        left: triggerRect.left,
        right: triggerRect.right,
        width: triggerRect.width,
        height: triggerRect.height,
      } : null,
      controls: trigger?.getAttribute('aria-controls') || '',
      panelID: panel?.id || '',
      panelVisible: Boolean((!isDetails || element.open) && panel &&
        panelStyle?.display !== 'none' && panelStyle?.visibility !== 'hidden' && panelRect?.width && panelRect?.height),
      panelText: (panel?.textContent || '').trim().replace(/\s+/g, ' '),
      panelRect: panelRect ? {
        left: panelRect.left,
        right: panelRect.right,
        width: panelRect.width,
        height: panelRect.height,
      } : null,
      rootClientWidth: element.clientWidth,
      rootScrollWidth: element.scrollWidth,
      panelOutliers,
      documentWidth: document.documentElement.scrollWidth,
      viewportWidth: window.innerWidth,
    };
  });
}

async function waitForDisclosure(root, expected) {
  await root.page().waitForFunction(({ selector, open }) => {
    const element = document.querySelector(selector);
    if (!element) return false;
    const trigger = element.querySelector(':scope > summary, :scope > button');
    return element.tagName === 'DETAILS'
      ? element.open === open
      : trigger?.getAttribute('aria-expanded') === String(open);
  }, { selector: await root.evaluate((element) => {
    if (element.id) return `#${CSS.escape(element.id)}`;
    if (element.dataset.energyHelp) return `[data-energy-help="${CSS.escape(element.dataset.energyHelp)}"]`;
    if (element.dataset.energyDisclosure) return `[data-energy-disclosure="${CSS.escape(element.dataset.energyDisclosure)}"]`;
    const className = [...element.classList].find((value) => value.startsWith('energy-'));
    return className ? `.${CSS.escape(className)}` : element.tagName.toLowerCase();
  }), open: expected });
}

export async function assertEnergyDisclosures(page, { label, width }) {
  const originalScroll = await page.evaluate(() => ({ x: window.scrollX, y: window.scrollY }));
  for (const expected of disclosures) {
    const root = page.locator(expected.selector);
    if ((await root.count()) !== 1) {
      fail(label, `${expected.label}: genau eine stabile Offenlegung ${expected.selector} fehlt`);
    }
    await root.scrollIntoViewIfNeeded();
    const initial = await disclosureState(root);
    if (!['SUMMARY', 'BUTTON'].includes(initial.triggerTag) || !initial.triggerName) {
      fail(label, `${expected.label}: semantischer, benannter Auslöser fehlt`, initial);
    }
    if (!initial.isDetails && (!initial.controls || initial.controls !== initial.panelID)) {
      fail(label, `${expected.label}: Button und Inhalt sind nicht mit aria-controls verbunden`, initial);
    }
    if (initial.open || initial.panelVisible) {
      fail(label, `${expected.label}: Erklärung muss in der reduzierten Ausgangsansicht geschlossen sein`, initial);
    }
    if (width <= 390 && (!initial.triggerRect || initial.triggerRect.height < 43.5 ||
        initial.triggerRect.left < -1 || initial.triggerRect.right > width + 1)) {
      fail(label, `${expected.label}: mobiles Tast-/Touch-Ziel ist kleiner als 44px oder abgeschnitten`, initial);
    }

    const trigger = root.locator(':scope > summary, :scope > button').first();
    await trigger.focus();
    if (!(await trigger.evaluate((element) => document.activeElement === element))) {
      fail(label, `${expected.label}: Auslöser nimmt keinen Tastaturfokus an`);
    }
    await trigger.press('Enter');
    await waitForDisclosure(root, true);
    const opened = await disclosureState(root);
    if (!opened.open || !opened.panelVisible || opened.panelText.length < 16 ||
        !expected.patterns.some((pattern) => pattern.test(opened.panelText)) ||
        opened.panelOutliers.length || opened.documentWidth > opened.viewportWidth + 1 ||
        !opened.panelRect || opened.panelRect.left < -1 || opened.panelRect.right > width + 1) {
      fail(label, `${expected.label}: Enter enthüllt den erhaltenen Inhalt nicht sauber`, opened);
    }

    await trigger.press('Space');
    await waitForDisclosure(root, false);
    const closed = await disclosureState(root);
    if (closed.open || closed.panelVisible || !(await trigger.evaluate((element) => document.activeElement === element))) {
      fail(label, `${expected.label}: Leertaste schließt nicht mit erhaltenem Fokus`, closed);
    }
  }
  await page.evaluate(({ x, y }) => window.scrollTo(x, y), originalScroll);
}

export async function assertEnergyMetricDisclosures(page, { label, width }) {
  const roots = page.locator('details.energy-metric-info[data-energy-help]');
  const actualIDs = await roots.evaluateAll((elements) => elements.map((element) => element.getAttribute('data-energy-help')));
  const hasEstimate = await page.locator('.energy-billed').isVisible().catch(() => false);
  const requiredIDs = hasEstimate ? metricDisclosures.map((item) => item.id) : ['tariff'];
  const missing = requiredIDs.filter((id) => !actualIDs.includes(id));
  const unexpected = actualIDs.filter((id) => !requiredIDs.includes(id));
  const duplicated = actualIDs.filter((id, index) => actualIDs.indexOf(id) !== index);
  if (missing.length || unexpected.length || duplicated.length || actualIDs.length !== requiredIDs.length) {
    fail(label, 'Kennzahl-Infos sind nicht auf Tarif und Jahreswert reduziert', {
      requiredIDs,
      actualIDs,
      missing,
      unexpected,
      duplicated,
    });
  }

  const originalScroll = await page.evaluate(() => ({ x: window.scrollX, y: window.scrollY }));
  for (const expected of metricDisclosures.filter((item) => actualIDs.includes(item.id))) {
    const root = page.locator(`details.energy-metric-info[data-energy-help="${expected.id}"]`);
    if ((await root.count()) !== 1 || (await root.getAttribute('name')) !== 'energy-metric-help') {
      fail(label, `Info „${expected.id}“ ist nicht eindeutig in der gegenseitig ausschließenden Gruppe`);
    }
    await root.scrollIntoViewIfNeeded();
    const initial = await disclosureState(root);
    if (initial.open || initial.panelVisible || initial.triggerTag !== 'SUMMARY' ||
        !initial.triggerName || initial.controls !== initial.panelID ||
        (width <= 900 && (!initial.triggerRect || initial.triggerRect.width < 43.5 || initial.triggerRect.height < 43.5))) {
      fail(label, `Info „${expected.id}“ hat keinen geschlossenen, benannten 44px-Tastaturauslöser`, initial);
    }

    const trigger = root.locator(':scope > summary');
    await trigger.focus();
    await trigger.press('Enter');
    await waitForDisclosure(root, true);
    const opened = await disclosureState(root);
    if (!opened.open || !opened.panelVisible || !opened.panelRect ||
        opened.panelRect.left < -1 || opened.panelRect.right > width + 1 ||
        !expected.patterns.every((pattern) => pattern.test(opened.panelText))) {
      fail(label, `Info „${expected.id}“ öffnet den erhaltenen Inhalt nicht mit Enter`, opened);
    }
    await trigger.press('Space');
    await waitForDisclosure(root, false);
    if ((await disclosureState(root)).open || !(await trigger.evaluate((element) => document.activeElement === element))) {
      fail(label, `Info „${expected.id}“ schließt nicht mit Leertaste und erhaltenem Fokus`);
    }
  }

  if (actualIDs.length > 1) {
    const first = page.locator(`details.energy-metric-info[data-energy-help="${actualIDs[0]}"]`);
    const second = page.locator(`details.energy-metric-info[data-energy-help="${actualIDs[1]}"]`);
    await first.scrollIntoViewIfNeeded();
    await first.locator(':scope > summary').press('Enter');
    await waitForDisclosure(first, true);
    await second.scrollIntoViewIfNeeded();
    await second.locator(':scope > summary').press('Enter');
    await waitForDisclosure(second, true);
    if ((await disclosureState(first)).open) {
      fail(label, 'benannte Kennzahl-Infos bleiben gleichzeitig offen statt sich gegenseitig zu schließen', actualIDs.slice(0, 2));
    }
    await second.locator(':scope > summary').press('Space');
    await waitForDisclosure(second, false);
  }
  await page.evaluate(({ x, y }) => window.scrollTo(x, y), originalScroll);
}

export async function assertEnergyFlowGeometry(page, { label, width }) {
  // Side-by-side flow + rail only exists from 1180px; below that the rail
  // stacks under the flow and the generic overlap checks cover it.
  if (width < 1180) return;
  const result = await page.locator('[data-energy-flow-diagram]').evaluate((root) => {
    const rect = (element) => {
      const box = element?.getBoundingClientRect();
      return box ? {
        top: box.top, right: box.right, bottom: box.bottom, left: box.left,
        width: box.width, height: box.height,
      } : null;
    };
    const area = root.querySelector('.energy-flow-area');
    const areaRect = rect(area);
    const tiles = [...(area?.querySelectorAll('[data-edge], .energy-flow-big') || [])]
      .map((element) => ({
        id: element.getAttribute('data-edge') || 'rail-' + [...element.parentElement.children].indexOf(element),
        rect: rect(element),
      }));
    const overlaps = [];
    for (let left = 0; left < tiles.length; left += 1) {
      for (let right = left + 1; right < tiles.length; right += 1) {
        const a = tiles[left];
        const b = tiles[right];
        const overlapX = Math.min(a.rect.right, b.rect.right) - Math.max(a.rect.left, b.rect.left);
        const overlapY = Math.min(a.rect.bottom, b.rect.bottom) - Math.max(a.rect.top, b.rect.top);
        if (overlapX > 1 && overlapY > 1) overlaps.push(`${a.id}/${b.id}:${overlapX.toFixed(1)}x${overlapY.toFixed(1)}`);
      }
    }
    const byEdge = Object.fromEntries(tiles.filter((tile) => tile.id).map((tile) => [tile.id, tile.rect]));
    const rail = rect(area?.querySelector('.energy-flow-rail'));
    const svg = area?.querySelector('svg.energy-flow-ribbons');
    return {
      areaRect,
      overlaps,
      ribbonCount: svg ? svg.querySelectorAll('path').length : 0,
      // The approved geometry: PV above the hub, grid below it, storage to its
      // left, the consumer rail fully right of the hub column.
      pvAboveHub: Boolean(byEdge['producer-0'] && byEdge.hub && byEdge['producer-0'].bottom <= byEdge.hub.top + 1),
      hubAboveGrid: Boolean(byEdge.hub && byEdge.grid && byEdge.hub.bottom <= byEdge.grid.top + 1),
      storageLeftOfHub: !byEdge.storage || Boolean(byEdge.hub && byEdge.storage.right <= byEdge.hub.left + 1),
      railRightOfHub: !rail || Boolean(byEdge.hub && rail.left >= byEdge.hub.right - 1),
      outsideArea: tiles.filter(({ rect: box }) => !box || !areaRect ||
        box.left < areaRect.left - 1 || box.right > areaRect.right + 1 ||
        box.top < areaRect.top - 1 || box.bottom > areaRect.bottom + 1).map(({ id }) => id),
    };
  });

  if (!result.areaRect || result.overlaps.length || result.outsideArea.length ||
      result.ribbonCount < 2 || !result.pvAboveHub || !result.hubAboveGrid ||
      !result.storageLeftOfHub || !result.railRightOfHub) {
    fail(label, 'Desktop-Energiefluss verletzt die freigegebene Geometrie (Slots, Rail, Ribbons)', result);
  }
}

export async function assertEnergyTopGeometry(page, { label, width }) {
  const result = await page.evaluate(() => {
    const rect = (element) => {
      const box = element?.getBoundingClientRect();
      return box ? { top: box.top, right: box.right, bottom: box.bottom, left: box.left, width: box.width, height: box.height } : null;
    };
    const visible = (element) => {
      const closedDetails = element.closest('details:not([open])');
      if (closedDetails) {
        const summary = closedDetails.querySelector(':scope > summary');
        if (element !== summary && !summary?.contains(element)) return false;
      }
      const style = getComputedStyle(element);
      const box = element.getBoundingClientRect();
      return style.display !== 'none' && style.visibility !== 'hidden' && box.width > 0 && box.height > 0;
    };
    const overlap = (left, right) => {
      const x = Math.min(left.right, right.right) - Math.max(left.left, right.left);
      const y = Math.min(left.bottom, right.bottom) - Math.max(left.top, right.top);
      return x > 1 && y > 1 ? { width: x, height: y } : null;
    };
    const groups = ['.energy-heading', '.energy-health', '.energy-lead-side', '.energy-nextstep', '.energy-tariff', '.energy-billed', '.energy-flow-area'];
    const siblingOverlaps = [];
    for (const selector of groups) {
      const parent = document.querySelector(selector);
      if (!parent) continue;
      const children = [...parent.children].filter(visible).filter((element) => {
        const position = getComputedStyle(element).position;
        return position !== 'absolute' && position !== 'fixed';
      });
      for (let left = 0; left < children.length; left += 1) {
        for (let right = left + 1; right < children.length; right += 1) {
          const amount = overlap(rect(children[left]), rect(children[right]));
          if (amount) siblingOverlaps.push({
            parent: selector,
            left: children[left].className || children[left].tagName,
            right: children[right].className || children[right].tagName,
            amount,
          });
        }
      }
    }

    const top = document.querySelector('.energy-health');
    const clippedControls = [...(top?.querySelectorAll('a[href], button, summary, input, select') || [])]
      .filter(visible)
      .map((element) => ({ element, box: rect(element) }))
      .filter(({ box }) => box.left < -1 || box.right > window.innerWidth + 1)
      .map(({ element, box }) => ({
        element: `${element.tagName.toLowerCase()}.${String(element.className || '').trim().replace(/\s+/g, '.')}`,
        text: (element.getAttribute('aria-label') || element.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 80),
        box,
      }));
    const checked = ['[data-energy-flow-diagram]', '.energy-info-disclosure', '.energy-tariff-disclosure', '.energy-next-why']
      .map((selector) => ({ selector, element: document.querySelector(selector) }))
      .filter(({ element }) => element);
    const componentOverflow = checked
      .filter(({ element }) => element.scrollWidth > element.clientWidth + 1)
      .map(({ selector, element }) => ({ selector, client: element.clientWidth, scroll: element.scrollWidth }));
    const diagram = document.querySelector('[data-energy-flow-diagram] .energy-flow-area');
    const diagramRect = rect(diagram);
    const live = document.querySelector('[data-energy-flow-diagram]');
    const liveRect = rect(live);
    const tariff = document.querySelector('.energy-tariff');
    const tariffEmpty = tariff?.querySelector('.energy-tariff-empty');
    const nextOverflow = document.querySelector('.energy-next-overflow');
    const flowOutliers = [...(diagram?.querySelectorAll('[data-edge]') || [])]
      .filter(visible)
      .map((element) => ({
        element: element.getAttribute('data-edge'),
        box: rect(element),
      }))
      .filter(({ box }) => !diagramRect || box.left < diagramRect.left - 1 || box.right > diagramRect.right + 1 ||
        box.top < diagramRect.top - 1 || box.bottom > diagramRect.bottom + 1);
    const liveOutliers = [...(live?.querySelectorAll('*') || [])]
      .filter(visible)
      .map((element) => ({
        element: `${element.tagName.toLowerCase()}.${String(element.className || '').trim().replace(/\s+/g, '.')}`,
        text: (element.getAttribute('aria-label') || element.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 60),
        box: rect(element),
      }))
      .filter(({ box }) => !liveRect || box.left < liveRect.left - 1 || box.right > liveRect.right + 1);
    const overflowingDescendants = [...(live?.querySelectorAll('*') || [])]
      .filter(visible)
      .filter((element) => element.scrollWidth > element.clientWidth + 1)
      .slice(0, 12)
      .map((element) => ({
        element: `${element.tagName.toLowerCase()}.${String(element.className || '').trim().replace(/\s+/g, '.')}`,
        client: element.clientWidth,
        scroll: element.scrollWidth,
      }));
    const flowNodeRects = [...(diagram?.querySelectorAll('[data-edge]') || [])]
      .filter(visible)
      .map((element) => ({
        id: element.getAttribute('data-edge'),
        box: rect(element),
      }));
    const flowNodeOverlaps = [];
    for (let left = 0; left < flowNodeRects.length; left += 1) {
      for (let right = left + 1; right < flowNodeRects.length; right += 1) {
        const amount = overlap(flowNodeRects[left].box, flowNodeRects[right].box);
        if (amount) flowNodeOverlaps.push({
          left: flowNodeRects[left].id,
          right: flowNodeRects[right].id,
          amount,
        });
      }
    }
    const liveCard = document.querySelector('.energy-live');
    const headingCopy = document.querySelector('.energy-heading-copy');
    const headingAction = document.querySelector('.energy-heading-action');
    const modeState = document.querySelector('.energy-mode-state');
    const modeAction = document.querySelector('.energy-mode-action');
    const next = document.querySelector('.energy-nextstep');
    return {
      viewport: window.innerWidth,
      documentWidth: document.documentElement.scrollWidth,
      bodyWidth: document.body.scrollWidth,
      siblingOverlaps,
      clippedControls,
      componentOverflow,
      diagramRect,
      flowOutliers,
      flowNodeOverlaps,
      liveOutliers,
      overflowingDescendants,
      layout: {
        headingCopy: rect(headingCopy),
        headingAction: rect(headingAction),
        headingControlOverlap: headingCopy && headingAction && visible(headingAction)
          ? overlap(rect(headingCopy), rect(headingAction)) : null,
        modeControlOverlap: modeState && modeAction && visible(modeAction)
          ? overlap(rect(modeState), rect(modeAction)) : null,
        live: rect(liveCard),
        tariff: rect(tariff),
        next: rect(next),
      },
      emptyTariffHeight: tariffEmpty && visible(tariffEmpty) ? rect(tariff)?.height || 0 : 0,
      nextOverflow: nextOverflow && visible(nextOverflow) ? {
        box: rect(nextOverflow),
        name: nextOverflow.getAttribute('aria-label') || '',
      } : null,
    };
  });

  const cards = result.layout;
  const missingCards = !cards.live || !cards.tariff || !cards.next;
  // One shared column at every width since 0.69.0: live first, next step after
  // it, the tariff detail card below the chart.
  const stackedWrong = !missingCards && (
    Math.abs(cards.live.left - cards.tariff.left) > 1 ||
    cards.live.bottom > cards.next.top + 1 ||
    cards.next.bottom > cards.tariff.top + 1);
  const columnsWrong = false;

  if (result.documentWidth > width + 1 || result.bodyWidth > width + 1 ||
      result.siblingOverlaps.length || result.clippedControls.length || result.componentOverflow.length ||
      !result.diagramRect || result.diagramRect.left < -1 || result.diagramRect.right > width + 1 ||
      result.flowOutliers.length || result.flowNodeOverlaps.length || result.layout.headingControlOverlap ||
      result.layout.modeControlOverlap || missingCards || stackedWrong || columnsWrong ||
      (width <= 1024 && result.emptyTariffHeight > 400) ||
      (width <= 390 && result.nextOverflow &&
        (result.nextOverflow.name !== 'Weitere Optionen' ||
         result.nextOverflow.box.width < 43.5 || result.nextOverflow.box.width > 54 ||
         result.nextOverflow.box.height < 43.5 || result.nextOverflow.box.height > 54))) {
    fail(label, 'Top-Redesign läuft über, überlappt oder schneidet Bedienelemente ab', result);
  }
}

export async function assertExistingEnergyChartBoundary(page, label) {
  const result = await page.evaluate(() => {
    const top = document.querySelector('[data-energy-flow-diagram]');
    const chart = document.querySelector('#energieverlauf.energy-chart');
    return {
      topBeforeChart: Boolean(top && chart && (top.compareDocumentPosition(chart) & Node.DOCUMENT_POSITION_FOLLOWING)),
      chartCount: document.querySelectorAll('#energieverlauf.energy-chart').length,
      headingCount: document.querySelectorAll('#energy-chart-title').length,
      rangeLinks: document.querySelectorAll('#energieverlauf .energy-chart-range a').length,
      interactiveCharts: document.querySelectorAll('[data-energy-chart-interactive]').length,
      zoomButtons: document.querySelectorAll('#energieverlauf [data-dialog="energy-chart-dialog"]').length,
      dialogCount: document.querySelectorAll('#energy-chart-dialog').length,
      fullscreenSurfaces: document.querySelectorAll('[data-energy-fullscreen-surface]').length,
      roadmapAfterChart: Boolean(chart && document.querySelector('#fahrplan') &&
        (chart.compareDocumentPosition(document.querySelector('#fahrplan')) & Node.DOCUMENT_POSITION_FOLLOWING)),
    };
  });
  if (!result.topBeforeChart || result.chartCount !== 1 || result.headingCount !== 1 ||
      result.rangeLinks !== 2 || result.interactiveCharts < 2 || result.zoomButtons !== 2 ||
      result.dialogCount !== 1 || result.fullscreenSurfaces !== 1 || !result.roadmapAfterChart) {
    fail(label, 'Top-Redesign hat die bestehende Energieverlauf-Grenze oder ihre Interaktionshaken verändert', result);
  }
}

export async function assertEnergyRedesignViewport(page, { label, width }) {
  await assertOwnerEnergySidebarOrder(page, label);
  await assertEnergyTopContent(page, label);
  await assertExistingEnergyChartBoundary(page, label);
  await assertEnergyTopGeometry(page, { label, width });
  await assertEnergyFlowGeometry(page, { label, width });
  await assertEnergyDisclosures(page, { label, width });
  await assertEnergyMetricDisclosures(page, { label, width });
  if (width === 390 || width === 1440) await assertEnergyConsumerManagement(page, { label, width });
}
