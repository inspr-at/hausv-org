// Renders the mark to a high-res transparent PNG for the glass renderer.
//
// Why a raster: GlassObject parses SVG as vectors (shapesFromSvg -> SVGLoader
// -> createShapes). Our mark is open, fill:none stroke polylines, so
// createShapes closes each path into a filled polygon and the three houses come
// out as one solid slab with every window and door filled in.
//
// Handing it a PNG routes it through the contour tracer instead — the same path
// AsciiObject uses — which walks the rendered strokes and keeps the interior
// openings as real holes.

import { chromium } from "../snapshot/node_modules/playwright/index.mjs";
import { readFileSync, writeFileSync } from "node:fs";

const SRC = new URL("../../internal/web/assets/hausv-mark.svg", import.meta.url);
const OUT = new URL("../../internal/web/assets/hausv-mark.png", import.meta.url);
const WIDTH = 1440; // 72:42 viewBox -> 1440x840, well above the 512 trace grid

const svg = readFileSync(SRC, "utf8");
const browser = await chromium.launch();
const page = await browser.newPage({
  viewport: { width: WIDTH, height: Math.round((WIDTH * 42) / 72) },
});
await page.setContent(
  `<style>html,body{margin:0;background:transparent}svg{display:block;width:100vw;height:100vh}</style>${svg}`,
);
const png = await page.screenshot({ omitBackground: true });
writeFileSync(OUT, png);
await browser.close();
console.log(`wrote hausv-mark.png (${WIDTH}px wide, ${png.length} bytes)`);
