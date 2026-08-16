#!/usr/bin/env python3
"""Is the hoist behaviour-preserving in EVERY state, not just the captured ones?

Screenshots cannot see a focused skip link, a closed <details>, a hover, a cursor,
or anything below the mobile breakpoint. But a CSS move is behaviour-preserving
if and only if, for each page, the browser ends up with the same rules and the
same relative order among rules that can fight each other.

So compare, per page:
  before = rules of the page's own <style> blocks, in order
  after  = rules of PortalBaseStyles followed by the page's remaining blocks

and assert
  1. identical multiset of (context, selector, declarations)
  2. for every selector appearing more than once, its subsequence is unchanged
     (this is what a hoist can break: moving a rule earlier lets a rule that
     used to lose start winning)
"""
import re, sys, subprocess, pathlib, collections

WT = pathlib.Path(sys.argv[1])
BASE_REF = sys.argv[2]


def parse(css):
    out, i, n, ctx, buf = [], 0, len(css), [], ""
    while i < n:
        c = css[i]
        if c == '@':
            m = re.match(r'@[^{]*\{', css[i:])
            if m:
                ctx.append(re.sub(r'\s+', ' ', m.group(0)[:-1]).strip())
                i += m.end(); buf = ""
                continue
        if c == '{':
            sel = re.sub(r'\s+', ' ', buf).strip(); buf = ""
            j, depth = i + 1, 1
            while j < n and depth:
                if css[j] == '{': depth += 1
                elif css[j] == '}': depth -= 1
                j += 1
            out.append((" | ".join(ctx), sel, re.sub(r'\s+', ' ', css[i + 1:j - 1]).strip()))
            i = j
            continue
        if c == '}':
            if ctx: ctx.pop()
            buf = ""; i += 1
            continue
        buf += c; i += 1
    return out


def blocks(src):
    return "\n".join(re.findall(r"<style>(.*?)</style>", src, re.S))


def at(ref, rel):
    r = subprocess.run(["git", "-C", str(WT), "show", f"{ref}:{rel}"],
                       capture_output=True, text=True)
    return r.stdout if r.returncode == 0 else ""


d = WT / "internal/web"
now_portal = (d / "portal.templ").read_text(encoding="utf8")
m = re.search(r"templ PortalBaseStyles\(\) \{\s*<style>(.*?)</style>", now_portal, re.S)
if not m:
    print("FAIL: PortalBaseStyles not found"); sys.exit(2)
base_rules = parse(m.group(1))

# PortalBaseStyles lives inside portal.templ, so strip it from portal's own blocks.
def page_rules_now(name):
    src = (d / name).read_text(encoding="utf8")
    if name == "portal.templ":
        src = src.replace(m.group(0), "")
    return parse(blocks(src))


bad = 0
for p in sorted(d.glob("*.templ")):
    if p.name == "templ_example.templ":
        continue
    before_src = at(BASE_REF, f"internal/web/{p.name}")
    if not before_src:
        continue
    before = parse(blocks(before_src))
    if not before:
        continue
    after = base_rules + page_rules_now(p.name)

    cb, ca = collections.Counter(before), collections.Counter(after)
    missing = cb - ca
    extra = ca - cb
    if missing or extra:
        bad += 1
        print(f"FAIL {p.name}: rule multiset changed")
        for r, k in list(missing.items())[:4]: print(f"   lost  x{k}: {r[1][:70]}")
        for r, k in list(extra.items())[:4]: print(f"   new   x{k}: {r[1][:70]}")
        continue

    # relative order among rules sharing a (context, selector)
    def seq(rules):
        by = collections.defaultdict(list)
        for idx, (ctx, sel, dd) in enumerate(rules):
            by[(ctx, sel)].append(dd)
        return {k: v for k, v in by.items() if len(v) > 1}

    sb, sa = seq(before), seq(after)
    if sb != sa:
        bad += 1
        print(f"FAIL {p.name}: order changed among same-selector rules")
        for k in set(sb) | set(sa):
            if sb.get(k) != sa.get(k):
                print(f"   {k[1][:60]}")
                print(f"     before: {sb.get(k)}")
                print(f"     after : {sa.get(k)}")
        continue
    print(f"ok   {p.name}: {len(before)} rules, multiset and conflict order preserved")

print()
print("INVARIANT HOLDS" if bad == 0 else f"{bad} PAGE(S) FAILED")
sys.exit(1 if bad else 0)
