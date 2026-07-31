// hausv.org — page bend.
//
// Wraps the Canvas UI Bend core without React, reproducing the wrapper's
// support gate exactly: the effect needs the experimental html-in-canvas API
// (ctx.drawElementImage + canvas.requestPaint + <canvas layoutsubtree>), and
// children of a <canvas> are fallback content that a browser without it will
// not render. So the content is SERVED in a normal div and only moved into the
// canvas once the API is confirmed present. Anywhere else the page is untouched.
//
// The fixed chrome — sticky header, veil, 3D mark — deliberately lives outside
// the frame, so the fold never touches it.

import { createBend, supportsHtmlInCanvas } from "./bendCore";

// Exactly the playground configuration.
const OPTIONS = {
  zone: 240,
  angle: 80,
  rounding: 150,
  perspective: 700,
  ease: 300,
  smoothing: 0.5,
  tumble: 0.5,
  tilt: 0,
  direction: "in" as const,
  top: false,
  bottom: false,
};

export function initBend() {
  const frame = document.querySelector<HTMLElement>("[data-bend]");
  const source = frame?.querySelector<HTMLCanvasElement>(".bend-source");
  const content = frame?.querySelector<HTMLElement>(".bend-content");
  const output = frame?.querySelector<HTMLCanvasElement>(".bend-output");
  if (!frame || !source || !content || !output) return;

  // Unsupported: leave the DOM exactly as served. Creating the instance anyway
  // would only burn a WebGL context on an effect that draws nothing.
  if (!supportsHtmlInCanvas()) return;

  const scrollTop = content.scrollTop;
  source.hidden = false;
  source.appendChild(content);

  const instance = createBend({ source, content, output }, OPTIONS);
  if (!instance) {
    // WebGL2 refused. Put the content back where it was rather than leaving it
    // stranded inside a canvas that will never paint it.
    source.hidden = true;
    frame.insertBefore(content, output);
  }
  content.scrollTop = scrollTop;
  return instance;
}
