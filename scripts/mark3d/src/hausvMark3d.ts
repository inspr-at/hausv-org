// hausv.org — rotating 3D brand mark.
//
// Wraps the Canvas UI AsciiObject / GlassObject cores (create*Object) without
// React. The mark sits statically above the hero eyebrow — there is no scroll
// animation any more. The only remaining interaction is drag-to-rotate with a
// gentle return to the base tilt. Revealing the flat white nav mark (and the
// top veil) once the hero mark scrolls away is landing.js's job via an
// IntersectionObserver; this module never touches scroll state.

import { createAsciiObject } from "./asciiObjectCore";
import { createGlassObject } from "./glassObjectCore";

type Mode = "solid" | "glass";
type Instance = { destroy(): void; setOptions?(o: Record<string, unknown>): void };

const FADE_MS = 800;
// After a drag, sit still this long, then ease the view back to its base tilt.
const RETURN_DELAY_MS = 2000;
const RETURN_DURATION_MS = 900;

const lerp = (a: number, b: number, t: number) => a + (b - a) * t;
const ease = (t: number) => 1 - Math.pow(1 - t, 3);

function num(el: HTMLElement, key: string, fallback: number) {
  const raw = el.dataset[key];
  if (raw === undefined) return fallback;
  const parsed = Number(raw);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function str(el: HTMLElement, key: string, fallback: string) {
  return el.dataset[key] || fallback;
}

function hasWebgl() {
  try {
    const probe = document.createElement("canvas");
    return !!(probe.getContext("webgl2") || probe.getContext("webgl"));
  } catch {
    return false;
  }
}

// `bevel` is the detail killer and has almost no usable range. The core does
// bevelAmount = bevel * (depth * size2d) * 0.5, then rounds the traced contours
// by bevelAmount * 1.25. The mark's strokes are 2.2 units of a 72-unit box
// (~0.031 normalised), so the upstream default of bevel:1 rounds by more than a
// whole stroke width and melts the three houses into one lump. Anything above
// ~0.12 is already mush. `depth` is the glassiness lever — glass needs volume
// for light to travel through, so depth stays high and bevel stays low.
// bevel/depth only bite on the traced 2D fallback; the GLB carries its own
// geometry. The material applies either way: over the dark hero the mark stays
// near-clear and blue-lit.
const GLASS = {
  bevel: 0.02, depth: 0.175, thickness: 1.6, dispersion: 1.2, roughness: 0.03,
  tint: "#e7c574", tintDensity: 0.7, highlight: "#066aff",
};

function makeCanvas() {
  const canvas = document.createElement("canvas");
  canvas.style.cssText =
    "position:absolute;inset:0;width:100%;height:100%;display:block;" +
    `touch-action:none;opacity:0;transition:opacity ${FADE_MS}ms ease`;
  return canvas;
}

function createSolid(host: HTMLElement, canvas: HTMLCanvasElement, onReady: () => void) {
  return createAsciiObject(
    { canvas },
    {
      src: host.dataset.src!,
      cellSize: num(host, "cellSize", 13),
      cellAspect: num(host, "cellAspect", 0.9),
      contrast: num(host, "contrast", 2.5),
      edgeContrast: num(host, "edgeContrast", 3.3),
      environmentIntensity: num(host, "environmentIntensity", 1.6),
      roughness: num(host, "roughness", 0.35),
      scale: num(host, "scale", 5),
      floatIntensity: num(host, "floatIntensity", 0.3),
      rotationIntensity: num(host, "rotationIntensity", 0.12),
      floatSpeed: num(host, "floatSpeed", 0.7),
      fov: num(host, "fov", 28),
      cameraDistance: num(host, "cameraDistance", 9.2),
      autoRotateSpeed: num(host, "autoRotateSpeed", 0.5),
      ascii: false,
      colored: true,
      autoRotate: true,
      orbit: false,
      zoom: false,
      highlight: str(host, "highlight", "#e7c574"),
      onLoad: onReady,
      onError: () => host.setAttribute("data-mark3d-state", "failed"),
    },
  );
}

function createGlass(host: HTMLElement, canvas: HTMLCanvasElement, onReady: () => void) {
  return createGlassObject(
    { canvas },
    {
      // A raster, deliberately — see scripts/mark3d/make-raster.mjs. The SVG
      // path in this core treats our stroke-only artwork as filled polygons.
      src: host.dataset.glassSrc || host.dataset.src!,
      scale: num(host, "glassScale", 4.2),
      fov: num(host, "glassFov", 50),
      cameraDistance: num(host, "glassCameraDistance", 5),
      floatSpeed: num(host, "glassFloatSpeed", 0.1),
      floatIntensity: num(host, "glassFloatIntensity", 0.4),
      rotationIntensity: num(host, "glassRotationIntensity", 0.15),
      autoRotate: true,
      autoRotateSpeed: num(host, "glassAutoRotateSpeed", 0.5),
      ior: num(host, "glassIor", 1.6),
      clearcoat: num(host, "glassClearcoat", 0.5),
      environmentIntensity: num(host, "glassEnvironmentIntensity", 1),
      tint: str(host, "glassTint", GLASS.tint),
      tintDensity: num(host, "glassTintDensity", GLASS.tintDensity),
      highlight: str(host, "glassHighlight", GLASS.highlight),
      // The only supported way to put anything refractable in the scene: a
      // camera-parented plane the glass samples. Empty = nothing to refract.
      backgroundImage: str(host, "glassBackgroundImage", ""),
      // Drag-to-rotate, as in the Canvas UI playground.
      orbit: host.dataset.orbit !== "false",
      zoom: false,
      // Data attributes win over the preset so the material can be tuned from
      // the page without rebuilding the bundle.
      bevel: num(host, "glassBevel", GLASS.bevel),
      depth: num(host, "glassDepth", GLASS.depth),
      thickness: num(host, "glassThickness", GLASS.thickness),
      dispersion: num(host, "glassDispersion", GLASS.dispersion),
      roughness: num(host, "glassRoughness", GLASS.roughness),
      onLoad: onReady,
      onError: () => host.setAttribute("data-mark3d-state", "failed"),
    },
  );
}

class MarkStage {
  private instance: Instance | null = null;
  private canvas: HTMLCanvasElement;
  private idleTimer = 0 as unknown as ReturnType<typeof setTimeout>;
  private returnRaf = 0;
  private easeToBase: (() => void) | null = null;

  constructor(private host: HTMLElement) {
    const mode = (host.dataset.mark3dMode as Mode) || "glass";
    this.canvas = makeCanvas();
    host.appendChild(this.canvas);

    const create = mode === "glass" ? createGlass : createSolid;
    // Reduced motion needs no handling here: the cores gate autoRotate, float
    // and drift on the live prefers-reduced-motion query themselves.
    this.instance = create(host, this.canvas, () => {
      requestAnimationFrame(() =>
        requestAnimationFrame(() => {
          this.canvas.style.opacity = "1";
          host.setAttribute("data-mark3d-state", "ready");
          // Capture the base angle before anyone can drag it.
          this.wireOrbitReturn();
        }),
      );
    });
  }

  // Dragging tips the camera off its base polar angle and it stays there.
  // After the mark has been left alone, ease that tilt back — azimuth is left
  // to autoRotate, which owns the continuous turn.
  private wireOrbitReturn() {
    const inst = this.instance as unknown as {
      camera?: { position: { x: number; y: number; z: number; set(x: number, y: number, z: number): void } };
      controls?: {
        target: { x: number; y: number; z: number };
        update(): void;
        addEventListener(type: string, fn: () => void): void;
      };
    };
    const camera = inst?.camera;
    const controls = inst?.controls;
    if (!camera || !controls) return;

    const spherical = () => {
      const t = controls.target;
      const dx = camera.position.x - t.x;
      const dy = camera.position.y - t.y;
      const dz = camera.position.z - t.z;
      const r = Math.hypot(dx, dy, dz) || 1e-6;
      return { r, phi: Math.acos(Math.min(1, Math.max(-1, dy / r))), theta: Math.atan2(dx, dz) };
    };
    const write = (r: number, phi: number, theta: number) => {
      const t = controls.target;
      const horizontal = r * Math.sin(phi);
      camera.position.set(
        t.x + horizontal * Math.sin(theta),
        t.y + r * Math.cos(phi),
        t.z + horizontal * Math.cos(theta),
      );
      controls.update();
    };

    const base = spherical();

    // Eases only the tilt back to base and leaves the heading to autoRotate.
    this.easeToBase = () => {
      cancelAnimationFrame(this.returnRaf);
      const from = spherical();
      const started = performance.now();
      const step = () => {
        const k = Math.min(1, (performance.now() - started) / RETURN_DURATION_MS);
        const now = spherical();
        write(now.r, lerp(from.phi, base.phi, ease(k)), now.theta);
        if (k < 1) this.returnRaf = requestAnimationFrame(step);
      };
      step();
    };

    controls.addEventListener("start", () => {
      clearTimeout(this.idleTimer);
      cancelAnimationFrame(this.returnRaf);
    });

    controls.addEventListener("end", () => {
      clearTimeout(this.idleTimer);
      this.idleTimer = setTimeout(() => this.easeToBase?.(), RETURN_DELAY_MS);
    });
  }
}

export function initHausvMark3d() {
  const host = document.querySelector<HTMLElement>("[data-hausv-mark-3d]");
  if (!host || !host.dataset.src) return;

  const min = Number(host.dataset.minWidth ?? 0);
  if (min && !matchMedia(`(min-width: ${min}px)`).matches) return;
  if (!hasWebgl()) return;

  new MarkStage(host);
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initHausvMark3d, { once: true });
} else {
  initHausvMark3d();
}
