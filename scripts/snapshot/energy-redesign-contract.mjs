// Focused browser contract for the reduced energy-cockpit lead.
//
// Keep this module limited to the content above #energieverlauf. The chart and
// every section after it retain their established checks in qa-main-flows.mjs.

export const energyRedesignWidths = new Set([320, 390, 768, 1024, 1440]);

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

const flowNodes = [
  { id: 'home', patterns: [/Zuhause/i, /Hausverbrauch/i, /1,8\s*kW/i] },
  { id: 'pv', patterns: [/PV(?:-Leistung|-Erzeugung)?/i, /3,1\s*kW/i] },
  { id: 'grid', patterns: [/Netz(?:bezug)?/i, /2,4\s*kW/i] },
  { id: 'battery', patterns: [/Speicher/i, /78\s*%/i, /lädt/i] },
];

const flowConnections = [
  { left: 'pv', right: 'home' },
  { left: 'grid', right: 'home' },
  { left: 'battery', right: 'home' },
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
  { id: 'consumption', patterns: [/Hausverbrauch/i, /Home Assistant|gelesen/i] },
  { id: 'peak', patterns: [/Viertelstunde/i, /Kalendermonat/i] },
  { id: 'billed', patterns: [/Verrechnete Leistung/i, /2-kW-Sockel/i] },
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
    const node = diagram.locator(`[data-energy-node="${expected.id}"]`);
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

  const connectors = diagram.locator('[data-energy-connector]');
  const connectorData = await connectors.evaluateAll((nodes) => nodes.map((node) => ({
    from: node.getAttribute('data-from') || '',
    to: node.getAttribute('data-to') || '',
  })));
  const connectionKey = (left, right) => [left, right].sort().join('|');
  if (connectorData.length !== flowConnections.length ||
      new Set(connectorData.map((item) => connectionKey(item.from, item.to))).size !== connectorData.length) {
    fail(label, 'Energiefluss braucht genau drei eindeutig zuordenbare Verbindungen', connectorData);
  }
  for (const expected of flowConnections) {
    if (!connectorData.some((item) => connectionKey(item.from, item.to) === connectionKey(expected.left, expected.right))) {
      fail(label, `semantische Verbindung ${expected.left} ↔ ${expected.right} fehlt`, connectorData);
    }
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
  const requiredIDs = hasEstimate ? metricDisclosures.map((item) => item.id) : ['consumption'];
  const missing = requiredIDs.filter((id) => !actualIDs.includes(id));
  if (missing.length) {
    fail(label, 'erklärende Info-Knöpfe für Energiekennzahlen fehlen', { requiredIDs, actualIDs, missing });
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
  if (width < 1024) return;
  const result = await page.locator('[data-energy-flow-diagram]').evaluate((root, expectedPairs) => {
    const rect = (element) => {
      const box = element?.getBoundingClientRect();
      return box ? {
        top: box.top, right: box.right, bottom: box.bottom, left: box.left,
        width: box.width, height: box.height,
        centerX: box.left + box.width / 2,
        centerY: box.top + box.height / 2,
      } : null;
    };
    const diagram = root.querySelector('.energy-flow-stage');
    const diagramRect = rect(diagram);
    const nodeEntries = [...(diagram?.querySelectorAll('[data-energy-node]') || [])].map((element) => ({
      id: element.getAttribute('data-energy-node'),
      rect: rect(element),
    }));
    const nodes = Object.fromEntries(nodeEntries.map((entry) => [entry.id, entry.rect]));
    const overlaps = [];
    for (let left = 0; left < nodeEntries.length; left += 1) {
      for (let right = left + 1; right < nodeEntries.length; right += 1) {
        const a = nodeEntries[left];
        const b = nodeEntries[right];
        const overlapX = Math.min(a.rect.right, b.rect.right) - Math.max(a.rect.left, b.rect.left);
        const overlapY = Math.min(a.rect.bottom, b.rect.bottom) - Math.max(a.rect.top, b.rect.top);
        if (overlapX > 1 && overlapY > 1) overlaps.push(`${a.id}/${b.id}:${overlapX.toFixed(1)}x${overlapY.toFixed(1)}`);
      }
    }

    const connectors = [...(diagram?.querySelectorAll('[data-energy-connector]') || [])].map((element) => {
      const box = rect(element);
      const from = element.getAttribute('data-from');
      const to = element.getAttribute('data-to');
      const source = nodes[from];
      const target = nodes[to];
      if (!box || !source || !target) {
        return {
          id: element.getAttribute('data-energy-connector'),
          tag: element.tagName,
          from,
          to,
          box,
          horizontal: null,
          length: 0,
          crossSize: null,
          sourceGap: null,
          targetGap: null,
          sourceAxisDelta: null,
          targetAxisDelta: null,
          sourceTolerance: 0,
          targetTolerance: 0,
        };
      }
      const horizontal = box.width >= box.height;
      const crossSize = horizontal ? box.height : box.width;
      const length = horizontal ? box.width : box.height;
      const sourceBeforeTarget = horizontal ? source?.centerX <= target?.centerX : source?.centerY <= target?.centerY;
      const sourceEdge = horizontal
        ? (sourceBeforeTarget ? source?.right : source?.left)
        : (sourceBeforeTarget ? source?.bottom : source?.top);
      const targetEdge = horizontal
        ? (sourceBeforeTarget ? target?.left : target?.right)
        : (sourceBeforeTarget ? target?.top : target?.bottom);
      const connectorStart = horizontal
        ? (sourceBeforeTarget ? box.left : box.right)
        : (sourceBeforeTarget ? box.top : box.bottom);
      const connectorEnd = horizontal
        ? (sourceBeforeTarget ? box.right : box.left)
        : (sourceBeforeTarget ? box.bottom : box.top);
      const lineAxis = horizontal ? box.centerY : box.centerX;
      const sourceAxis = horizontal ? source?.centerY : source?.centerX;
      const targetAxis = horizontal ? target?.centerY : target?.centerX;
      const sourceTolerance = source ? Math.max(6, (horizontal ? source.height : source.width) * 0.22) : 0;
      const targetTolerance = target ? Math.max(6, (horizontal ? target.height : target.width) * 0.22) : 0;
      return {
        id: element.getAttribute('data-energy-connector'),
        tag: element.tagName,
        from,
        to,
        box,
        horizontal,
        length,
        crossSize,
        sourceGap: sourceEdge === undefined ? null : Math.abs(connectorStart - sourceEdge),
        targetGap: targetEdge === undefined ? null : Math.abs(connectorEnd - targetEdge),
        sourceAxisDelta: sourceAxis === undefined ? null : Math.abs(lineAxis - sourceAxis),
        targetAxisDelta: targetAxis === undefined ? null : Math.abs(lineAxis - targetAxis),
        sourceTolerance,
        targetTolerance,
      };
    });

    return {
      diagramRect,
      nodeEntries,
      overlaps,
      connectors,
      missingPairs: expectedPairs.filter((expected) => !connectors.some((item) =>
        [item.from, item.to].sort().join('|') === [expected.left, expected.right].sort().join('|'))),
      outsideDiagram: nodeEntries.filter(({ rect: box }) => !box || !diagramRect ||
        box.left < diagramRect.left - 1 || box.right > diagramRect.right + 1 ||
        box.top < diagramRect.top - 1 || box.bottom > diagramRect.bottom + 1).map(({ id }) => id),
    };
  }, flowConnections);

  const badConnector = result.connectors.find((item) =>
    !item.box || item.length < 16 || item.crossSize > 4 ||
    item.sourceGap === null || item.targetGap === null ||
    item.sourceGap > 24 || item.targetGap > 24 ||
    item.sourceAxisDelta > item.sourceTolerance || item.targetAxisDelta > item.targetTolerance);
  if (!result.diagramRect || result.nodeEntries.length !== flowNodes.length || result.overlaps.length ||
      result.missingPairs.length || result.outsideDiagram.length || badConnector) {
    fail(label, 'Desktop-Energiefluss hat überlappende Knoten oder keine geraden, sauber ausgerichteten Verbindungen', {
      ...result,
      badConnector,
    });
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
    const groups = ['.energy-heading', '.energy-health', '.energy-lead-side', '.energy-nextstep', '.energy-tariff'];
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
    const diagram = document.querySelector('[data-energy-flow-diagram] .energy-flow-stage');
    const diagramRect = rect(diagram);
    const live = document.querySelector('[data-energy-flow-diagram]');
    const liveRect = rect(live);
    const tariff = document.querySelector('.energy-tariff');
    const tariffEmpty = tariff?.querySelector('.energy-tariff-empty');
    const nextOverflow = document.querySelector('.energy-next-overflow');
    const flowOutliers = [...(diagram?.querySelectorAll('[data-energy-node], [data-energy-connector]') || [])]
      .filter(visible)
      .map((element) => ({
        element: element.getAttribute('data-energy-node') || element.getAttribute('data-energy-connector'),
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
    return {
      viewport: window.innerWidth,
      documentWidth: document.documentElement.scrollWidth,
      bodyWidth: document.body.scrollWidth,
      siblingOverlaps,
      clippedControls,
      componentOverflow,
      diagramRect,
      flowOutliers,
      liveOutliers,
      overflowingDescendants,
      emptyTariffHeight: tariffEmpty && visible(tariffEmpty) ? rect(tariff)?.height || 0 : 0,
      nextOverflow: nextOverflow && visible(nextOverflow) ? {
        box: rect(nextOverflow),
        name: nextOverflow.getAttribute('aria-label') || '',
      } : null,
    };
  });

  if (result.documentWidth > width + 1 || result.bodyWidth > width + 1 ||
      result.siblingOverlaps.length || result.clippedControls.length || result.componentOverflow.length ||
      !result.diagramRect || result.diagramRect.left < -1 || result.diagramRect.right > width + 1 ||
      result.flowOutliers.length || (width <= 1024 && result.emptyTariffHeight > 400) ||
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
}
