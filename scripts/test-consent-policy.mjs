#!/usr/bin/env node
// Exercises the DOM-free policy part of internal/web/assets/consent.js
// (HAUSV-742): what may load, when the bar shows, how signals and expiry
// behave. Runs under plain Node; CI calls it next to verify-versions.mjs.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import vm from "node:vm";
import assert from "node:assert/strict";

const here = dirname(fileURLToPath(import.meta.url));
const source = readFileSync(join(here, "..", "internal", "web", "assets", "consent.js"), "utf8");
const sandbox = {};
vm.createContext(sandbox);
vm.runInContext(source, sandbox, { filename: "consent.js" });
const policy = sandbox.hausvConsentPolicy;
assert.ok(policy, "policy must be exported without a DOM");
// Objects come from another vm context, so compare plain copies.
const decide = (input) => ({ ...policy.decide(input) });
const parse = (raw) => { const v = policy.parse(raw); return v && { ...v }; };

const now = 1_800_000_000;
const grant = { revision: policy.revision, marketing: true, at: now - 60 };
const refusal = { revision: policy.revision, marketing: false, at: now - 60 };

// Fresh visitor: nothing loads, the bar shows.
assert.deepEqual(decide({ stored: null, now, signal: false, bot: false }), { marketing: false, prompt: true, persist: "none" });
// Stored grant: loads, no bar. Stored refusal: nothing, no bar.
assert.deepEqual(decide({ stored: grant, now, signal: false, bot: false }), { marketing: true, prompt: false, persist: "none" });
assert.deepEqual(decide({ stored: refusal, now, signal: false, bot: false }), { marketing: false, prompt: false, persist: "none" });
// GPC / DNT: refusal even over an older grant, stored, no bar.
assert.deepEqual(decide({ stored: grant, now, signal: true, bot: false }), { marketing: false, prompt: false, persist: "refuse" });
// Bots: nothing loads, no bar, nothing stored.
assert.deepEqual(decide({ stored: null, now, signal: false, bot: true }), { marketing: false, prompt: false, persist: "none" });
// Expiry after the retention window re-asks; a refusal inside it stays.
assert.deepEqual(decide({ stored: { ...grant, at: now - policy.maxAge - 1 }, now, signal: false, bot: false }), { marketing: false, prompt: true, persist: "none" });
assert.deepEqual(decide({ stored: { ...refusal, at: now - policy.maxAge + 10 }, now, signal: false, bot: false }), { marketing: false, prompt: false, persist: "none" });
// A material purpose revision re-asks; the old grant does not carry over.
assert.deepEqual(decide({ stored: { ...grant, revision: policy.revision - 1 }, now, signal: false, bot: false }), { marketing: false, prompt: true, persist: "none" });

// Cookie value: no identifier, round-trips.
const value = policy.serialize(true, now);
assert.equal(value, `v1;r=${policy.revision};m=1;t=${now}`);
assert.deepEqual(parse(value), { revision: policy.revision, marketing: true, at: now });
assert.equal(parse("garbage"), null);
assert.equal(parse(""), null);
assert.equal(policy.cookieName, "hausv_consent");
assert.ok(policy.maxAge >= 180 * 24 * 3600, "refusal is remembered for at least six months");

// The browser part must never reference the Google host outside the gated loader.
const googleRefs = source.match(/googletagmanager\.com/g) || [];
assert.equal(googleRefs.length, 1, "exactly one gated reference to Google's host");
assert.ok(/gtag\("consent", "default", \{ ad_storage: "denied"/.test(source), "Consent Mode v2 defaults denied");
assert.ok(/gtag\("consent", "update", \{ ad_storage: "granted", ad_user_data: "granted", ad_personalization: "denied"/.test(source), "grant covers measurement only, personalisation stays denied");
assert.ok(/localStorage\.removeItem\(key\)/.test(source) && /"_gcl_ls"/.test(source), "withdrawal clears Google's local storage entry");
assert.ok(/if \(googleLoaded \|\| !tagId \|\| !authorized\(\)\) return;/.test(source), "the loader re-checks authorization on every call");
assert.ok(/if \(revoked \|\| sessionRevoked\(\)\) return false;/.test(source), "an in-page or session revocation outranks a surviving grant");
assert.ok(/if \(googleLoaded && persisted\) location\.reload\(\);/.test(source), "withdrawal reloads only once the refusal is persisted");
assert.ok(/try \{ raw = document\.cookie; \} catch/.test(source), "cookie reads are exception-safe");
assert.ok(/try \{ clearGoogleCookies\(\); \} catch/.test(source), "cookie cleanup has its own failure boundary");
assert.ok(/if \(decision\.persist === "refuse"\) withdraw\(\);/.test(source), "signal-driven refusal uses the verified withdrawal path");
assert.ok(!/<script/.test(source), "no inline script markup");

console.log("consent policy: ok");
