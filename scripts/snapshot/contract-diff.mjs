#!/usr/bin/env node
// Compare the interactive contract of two snapshot runs of the SAME code, one
// rendered by the legacy templates and one by templ.
//
//   node contract-diff.mjs <legacy-dir> <templ-dir>
//
// The rule is one-directional on purpose: the templ rendering must offer at least
// what the legacy rendering offered. Additions are the redesign doing its job and
// are reported as information, never as failures. Only losses fail.
//
// This exists because Phase 3 was verified by two oracles that could not see this
// class of defect: a byte diff with the switch OFF (which never renders templ at
// all) and an href-reachability check with the switch ON (scripts, body
// attributes, POST targets and confirmations are not hrefs). It found nothing
// wrong. Four regressions had already shipped.

import { readFile, readdir } from 'node:fs/promises';
import path from 'node:path';

const [legacyDir, templDir] = process.argv.slice(2);
if (!legacyDir || !templDir) {
  console.error('usage: contract-diff.mjs <legacy-dir> <templ-dir>');
  process.exit(2);
}

// Losses that are the 1.x redesign doing its job rather than a regression. Each
// one is a judgement someone made and had to write down; an entry with no `why`
// is a silent hole in the check, so `why` is required.
//
// `routes` scopes an entry to the pages the judgement was actually made about.
// Omitting it waives the value EVERYWHERE, which is almost never what you mean:
// the same attribute name usually carries different weight on different pages.
// `data-label` is the worked example — a stacked-table artefact on the users
// page, but a behavioural hook that app.js reads on the energy chart and the
// home-type selector. Waiving it globally would have let a real loss through
// there, silently, forever.
//
// Accepted losses are printed rather than swallowed, and reported as stale once
// they stop matching, so the list cannot outlive the judgement behind it.
const ACCEPTED_LOSSES = [
  {
    aspect: 'hooks',
    values: ['data-mobile-menu-toggle'],
    why: 'the JS-driven mobile menu became a native <details>, which also works with JS off',
  },
  {
    aspect: 'hooks',
    values: ['data-dialog', 'data-close-dialog'],
    why: 'the shared release-history dialog became a native <details>; page-level dialogs still carry these',
  },
  {
    aspect: 'hooks',
    values: ['data-home-identity', 'data-home-display-name'],
    why: 'styling hooks only — no script reads them; the templ sidebar styles its own classes',
  },
  {
    aspect: 'hooks',
    values: ['data-label'],
    routes: ['settings-users'],
    why: 'the users table became a card list, so the stacked-table label pattern no longer applies — what mattered (role and status staying visible on a phone) is CSS, which this tool cannot see, and is covered by TestUserCardsKeepRoleAndStatusOnNarrowScreens instead. Scoped: app.js also reads data-label on the energy chart and the home-type selector, where losing it IS a regression',
  },
];

// Worth stating plainly, because the check reads more complete than it is: this
// compares RENDERED MARKUP. An element that is present and then removed by a
// media query looks identical here to one that is visible. That is not a corner
// case — it is how HAUSV-545 hid the role and status pills on a phone while
// every markup-level check stayed green. CSS-visibility contracts need a test
// that reads the stylesheet, and those live in internal/web.

const keyOf = (aspect, value) => `${aspect} :: ${value}`;

// key -> { why, routes }. routes === null means the entry applies everywhere.
const accepted = new Map();
for (const { aspect, values, routes, why } of ACCEPTED_LOSSES) {
  if (!why) throw new Error(`ACCEPTED_LOSSES entry for ${aspect} has no reason`);
  for (const value of values) {
    accepted.set(keyOf(aspect, value), { why, routes: routes ? new Set(routes) : null });
  }
}

// Returns the waiver that covers this loss on this route, or null. Route scope
// is checked here rather than at lookup sites so a caller cannot forget it.
function waiverFor(aspect, value, route) {
  const entry = accepted.get(keyOf(aspect, value));
  if (!entry) return null;
  if (entry.routes && !entry.routes.has(route)) return null;
  return entry;
}

// Ordered worst-first: a reader who stops after the first block should have read
// the thing most likely to hurt someone.
const ASPECTS = [
  ['scripts', 'script no longer loaded — every enhancement it carries is dead'],
  ['bodyAttrs', 'body attribute dropped — app.js keys bfcache invalidation on one of these'],
  ['confirms', 'destructive action no longer asks before it runs'],
  ['forms', 'form target no longer offered — the action is gone'],
  ['required', 'field is no longer required — validation moved to the server alone'],
  ['fields', 'form field no longer submitted'],
  ['hooks', 'DOM hook dropped — a script that looks for it is loaded but inert'],
];

async function load(dir, persona, route) {
  try {
    return JSON.parse(await readFile(path.join(dir, persona, `${route}.contract.json`), 'utf8'));
  } catch (err) {
    if (err.code === 'ENOENT') return null;
    throw err;
  }
}

const personas = (await readdir(legacyDir, { withFileTypes: true }))
  .filter((e) => e.isDirectory())
  .map((e) => e.name)
  .sort();

let losses = 0;
let additions = 0;
let compared = 0;
const missing = [];
const acceptedSeen = new Map();

for (const persona of personas) {
  const routes = (await readdir(path.join(legacyDir, persona)))
    .filter((f) => f.endsWith('.contract.json'))
    .map((f) => f.replace(/\.contract\.json$/, ''))
    .sort();

  for (const route of routes) {
    const legacy = await load(legacyDir, persona, route);
    const templ = await load(templDir, persona, route);
    if (!legacy || !templ) {
      missing.push(`${persona}/${route}`);
      continue;
    }
    compared++;

    const blocks = [];
    for (const [aspect, why] of ASPECTS) {
      const before = new Set(legacy[aspect] || []);
      const after = new Set(templ[aspect] || []);
      const dropped = [...before].filter((v) => !after.has(v));
      const lost = dropped.filter((v) => !waiverFor(aspect, v, route));
      const waived = dropped.filter((v) => waiverFor(aspect, v, route));
      const gained = [...after].filter((v) => !before.has(v));
      for (const v of waived) {
        const k = keyOf(aspect, v);
        acceptedSeen.set(k, (acceptedSeen.get(k) || 0) + 1);
      }
      if (!lost.length && !gained.length) continue;
      losses += lost.length;
      additions += gained.length;
      blocks.push({ aspect, why, lost, gained });
    }

    if (!blocks.length) continue;
    console.log(`\n${persona}/${route}`);
    for (const { aspect, why, lost, gained } of blocks) {
      if (lost.length) {
        console.log(`  LOST  ${aspect} — ${why}`);
        for (const v of lost) console.log(`        - ${v}`);
      }
      if (gained.length) {
        console.log(`  added ${aspect}`);
        for (const v of gained) console.log(`        + ${v}`);
      }
    }
  }
}

console.log(`\ncompared ${compared} page pairs across ${personas.length} personas`);

if (acceptedSeen.size) {
  console.log('\nwaived by ACCEPTED_LOSSES — judged redesign, not regression:');
  for (const [k, count] of [...acceptedSeen].sort()) {
    const entry = accepted.get(k);
    const scope = entry.routes ? ` [only on ${[...entry.routes].sort().join(', ')}]` : '';
    console.log(`  ${k} (${count} pairs)${scope} — ${entry.why}`);
  }
}
// An entry nobody hit is an entry describing a world that no longer exists. Say
// so, so the list cannot quietly outlive its reasons.
const stale = [...accepted.keys()].filter((k) => !acceptedSeen.has(k));
if (stale.length) console.log(`\nSTALE ACCEPTED_LOSSES (never matched): ${stale.join(', ')}`);

if (missing.length) {
  // A page present in one run and absent from the other is not a pass. It is the
  // comparison failing to happen, which reads identically to "no differences".
  console.log(`UNPAIRED (${missing.length}): ${missing.join(', ')}`);
}
console.log(`losses: ${losses}  additions: ${additions}`);

if (losses || missing.length) {
  console.log('\nFAIL — the templ rendering offers less than the legacy rendering.');
  process.exit(1);
}
console.log('\nPASS — nothing the legacy rendering offered was dropped.');
