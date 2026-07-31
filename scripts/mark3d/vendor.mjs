// Refetches the Canvas UI components and strips their React wrappers, leaving
// the framework-agnostic create*() cores in src/.
//
// Each upstream file is a React distribution: a "use client" pragma, a react
// import, and a wrapper bolted onto a renderer that is otherwise plain DOM or
// three.js. hausv.org has no React, so we keep only the cores.

import { writeFileSync } from "node:fs";

// Post-fetch source patches, applied here so they survive re-vendoring.
// Each asserts its target still exists, so an upstream change fails loudly
// instead of silently dropping the behaviour.
const PATCHES = {
  "glass-object-react": [
    {
      why: "Trace the mark at 768px, not 256px. The contour tracer runs on this "
        + "downsampled raster, and at 256 the thin strokes come out chunky.",
      from: "const RASTER_SIZE = 256;",
      to: "const RASTER_SIZE = 768;",
    },
    {
      why: "Expose camera + controls so the page can ease the view back to its "
        + "base angle after a drag; the stock instance only returns "
        + "setOptions/resize/destroy.",
      from: "  return {\n    setOptions(next: GlassObjectOptions) {",
      to: "  return {\n    camera,\n    controls,\n    setOptions(next: GlassObjectOptions) {",
    },
  ],
};

const COMPONENTS = [
  { registry: "ascii-object-react", boundary: "export interface AsciiObjectProps", out: "asciiObjectCore.ts" },
  { registry: "glass-object-react", boundary: "export interface GlassObjectProps", out: "glassObjectCore.ts" },
];

// Matches both the single-line react imports and Bend's multi-line one.
// A line filter would drop only the `} from "react";` line and leave the
// opening `import {` dangling.
const REACT_IMPORT = /import\s+(?:\{[\s\S]*?\}|[\w*\s,]+)\s+from\s+["']react["'];?\n?/g;

for (const { registry, boundary, out } of COMPONENTS) {
  const url = `https://canvasui.dev/r/${registry}.json`;
  const response = await fetch(url);
  if (!response.ok) throw new Error(`registry fetch failed for ${registry}: ${response.status}`);

  const source = (await response.json()).files?.[0]?.content;
  if (!source) throw new Error(`registry payload for ${registry} has no component source`);

  const cut = source.indexOf(boundary);
  if (cut < 0) throw new Error(`could not locate the React wrapper boundary in ${registry}`);

  let body = source
    .slice(0, cut)
    .replace(/^"use client";\n*/, "")
    .replace(REACT_IMPORT, "");

  const leftovers = body
    .split("\n")
    .filter((l) => /\bReact\b|\buseRef\b|\buseEffect\b|\buseState\b|\buseSyncExternalStore\b|ReactNode/.test(l));
  if (leftovers.length) {
    throw new Error(`React references survived stripping ${registry}:\n${leftovers.join("\n")}`);
  }

  for (const patch of PATCHES[registry] ?? []) {
    if (!body.includes(patch.from)) {
      throw new Error(
        `patch target missing in ${registry}: ${JSON.stringify(patch.from)}\n` +
          `reason for the patch: ${patch.why}`,
      );
    }
    body = body.replace(patch.from, patch.to);
  }

  const header = `// Vendored from Canvas UI (React distribution), with the React wrapper
// stripped so the renderer can run from plain JS: hausv.org ships no React,
// and the create*() core never needed it.
//
// Upstream: https://canvasui.dev/docs/components/${registry.replace("-react", "")}
// Registry:  ${url}
// Re-vendor by re-running scripts/mark3d/build.fish --vendor; do not hand-edit.

`;
  writeFileSync(new URL(`./src/${out}`, import.meta.url), header + body);
  console.log(`vendored ${body.split("\n").length} lines -> src/${out}`);
}
