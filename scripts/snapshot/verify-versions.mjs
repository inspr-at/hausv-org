// Verify the versions the browser QA actually launched.
//
// Pinning an input in a config file says what *should* run. This says what
// *did*: the Node interpreter executing the harness, the Playwright package
// resolved from the lockfile, and the Chromium that actually starts.
//
// Expected values are read from the files that already declare them —
// .github/workflows/ci.yml for Node, package.json for Playwright — so there is
// no second place to keep in sync.
//
//   node verify-versions.mjs           report only (local development)
//   node verify-versions.mjs --strict  fail on any mismatch (CI)

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";
import { chromium } from "playwright";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "..", "..");
const strict = process.argv.includes("--strict");

function expectedNodeVersion() {
  const workflow = readFileSync(path.join(repo, ".github/workflows/ci.yml"), "utf8");
  const match = workflow.match(/node-version:\s*([^\s#]+)/);
  if (!match) throw new Error("no node-version found in .github/workflows/ci.yml");
  return match[1];
}

function expectedPlaywrightVersion() {
  const pkg = JSON.parse(readFileSync(path.join(here, "package.json"), "utf8"));
  const version = pkg.devDependencies?.playwright;
  if (!version) throw new Error("no playwright version in scripts/snapshot/package.json");
  return version;
}

function installedPlaywrightVersion() {
  const pkg = JSON.parse(
    readFileSync(path.join(here, "node_modules/playwright/package.json"), "utf8"),
  );
  return pkg.version;
}

const findings = [];
const report = [];

const wantNode = expectedNodeVersion();
const gotNode = process.versions.node;
report.push(`node        expected ${wantNode}  actual ${gotNode}`);
if (wantNode !== gotNode) {
  findings.push(`node ${gotNode} does not match the pinned ${wantNode}`);
}

const wantPlaywright = expectedPlaywrightVersion();
const gotPlaywright = installedPlaywrightVersion();
report.push(`playwright  expected ${wantPlaywright}  actual ${gotPlaywright}`);
if (wantPlaywright !== gotPlaywright) {
  findings.push(`playwright ${gotPlaywright} does not match the pinned ${wantPlaywright}`);
}

// Chromium has no separate pin: it is whatever this Playwright ships, which the
// lockfile fixes. Recording the launched build makes a silent swap visible.
let browser;
try {
  browser = await chromium.launch();
  report.push(`chromium    launched ${browser.version()} (bundled with playwright ${gotPlaywright})`);
} catch (error) {
  findings.push(`chromium failed to launch: ${error.message}`);
} finally {
  await browser?.close();
}

console.log(report.join("\n"));

if (findings.length === 0) {
  console.log("versions: ok");
  process.exit(0);
}

for (const finding of findings) console.error(`versions: ${finding}`);
if (!strict) {
  console.error("versions: reporting only (pass --strict to fail the run)");
  process.exit(0);
}
console.error("versions: FAILED — see docs/supply-chain-pins.md to update a pin deliberately");
process.exit(1);
