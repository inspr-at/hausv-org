// hausv.org — scroll-driven 3D brand mark.
//
// Wraps the Canvas UI AsciiObject / GlassObject cores (create*Object) without
// React. At the top of the page the mark is a large glass object; scrolling
// shrinks it into the nav logo slot, refines its material, fades it back, and
// leaves it as a rotating back-to-top control.
//
// The stage is a fixed child of <body> rather than living in the nav, because
// .landing-hero sets overflow:hidden and would clip it once it had to survive
// past the hero.

import { createAsciiObject } from "./asciiObjectCore";
import { createGlassObject } from "./glassObjectCore";

type Mode = "solid" | "glass";
type Instance = { destroy(): void; setOptions?(o: Record<string, unknown>): void };

const FADE_MS = 800;
const START_SCALE = 14; // stage is drawn at 14x the logo slot and scaled down

const lerp = (a: number, b: number, t: number) => a + (b - a) * t;
const clamp01 = (v: number) => (v < 0 ? 0 : v > 1 ? 1 : v);
// easeOutCubic: most of the shrink happens early, so the mark settles into the
// logo slot well before the hero has fully scrolled away.
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

// Material at the two ends of the scroll.
//
// `bevel` is the detail killer and has almost no usable range. The core does
// bevelAmount = bevel * (depth * size2d) * 0.5, then rounds the traced contours
// by bevelAmount * 1.25. The mark's strokes are 2.2 units of a 72-unit box
// (~0.031 normalised), so the upstream default of bevel:1 rounds by more than a
// whole stroke width and melts the three houses into one lump. Anything above
// ~0.12 is already mush; the difference between the two ends below is material
// (dispersion, roughness, depth), not silhouette.
// `depth` is the glassiness lever and `bevel`/RASTER_SIZE are the detail
// levers — they are independent, and trading depth away for crisper contours
// (as an earlier pass here did) just yields a flat wafer. Glass needs volume
// for light to travel through, so depth stays high and bevel stays low.
// bevel/depth only bite on the traced 2D fallback; the GLB carries its own
// geometry. The rest is material and applies either way.
//
// Big, over the dark hero, the mark can stay near-clear and blue-lit. Small it
// sits on cream, where clear glass on near-white is invisible — so it picks up
// the heading's gold as an actual body tint and swaps its ring light to warm.
const GLASS_COARSE = {
  bevel: 0.02, depth: 0.175, thickness: 1.6, dispersion: 1.2, roughness: 0.03,
  tint: "#e7c574", tintDensity: 0.7, highlight: "#066aff",
};
const GLASS_FINE = {
  bevel: 0.015, depth: 0.09, thickness: 1, dispersion: 0.5, roughness: 0.05,
  tint: "#e7c574", tintDensity: 3.2, highlight: "#e7c574",
};

// After a drag, sit still this long, then ease the view back to its base angle.
const RETURN_DELAY_MS = 2000;
const RETURN_DURATION_MS = 900;

function makeCanvas() {
  const canvas = document.createElement("canvas");
  canvas.style.cssText =
    "position:absolute;inset:0;width:100%;height:100%;display:block;" +
    `pointer-events:none;touch-action:none;opacity:0;transition:opacity ${FADE_MS}ms ease`;
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
      tint: str(host, "glassTint", GLASS_COARSE.tint),
      tintDensity: num(host, "glassTintDensity", GLASS_COARSE.tintDensity),
      highlight: str(host, "glassHighlight", GLASS_COARSE.highlight),
      // The only supported way to put anything refractable in the scene: a
      // camera-parented plane the glass samples. Empty = nothing to refract.
      backgroundImage: str(host, "glassBackgroundImage", ""),
      // Drag-to-rotate, as in the Canvas UI playground. Only reachable while
      // the mark is big; see the pointer-events switch in apply().
      orbit: host.dataset.orbit !== "false",
      zoom: false,
      // Data attributes win over the coarse preset so the material can be
      // tuned from the page without rebuilding the bundle.
      bevel: num(host, "glassBevel", GLASS_COARSE.bevel),
      depth: num(host, "glassDepth", GLASS_COARSE.depth),
      thickness: num(host, "glassThickness", GLASS_COARSE.thickness),
      dispersion: num(host, "glassDispersion", GLASS_COARSE.dispersion),
      roughness: num(host, "glassRoughness", GLASS_COARSE.roughness),
      onLoad: onReady,
      onError: () => host.setAttribute("data-mark3d-state", "failed"),
    },
  );
}

class MarkStage {
  private instance: Instance | null = null;
  private canvas: HTMLCanvasElement;
  private mode: Mode;
  private progress = -1;
  private appliedStep = -1;
  private hovered = false;
  private idleTimer = 0 as unknown as ReturnType<typeof setTimeout>;
  private returnRaf = 0;
  private hideTimer = 0 as unknown as ReturnType<typeof setTimeout>;
  private frozen = false;
  private easeToBase: ((squareUp: boolean) => void) | null = null;
  private anchor = { left: 0, top: 0 };
  private startScale = 1;
  private startTop = 90;
  private startLeft = 24;
  private travel = 600;
  private ticking = false;
  private reduced: MediaQueryList;

  constructor(
    private host: HTMLElement,
    private button: HTMLElement | null,
    private navMark: HTMLElement | null,
    private veil: HTMLElement | null,
  ) {
    this.mode = (host.dataset.mark3dMode as Mode) || "glass";
    this.reduced = window.matchMedia("(prefers-reduced-motion: reduce)");
    this.canvas = makeCanvas();
    host.appendChild(this.canvas);

    this.measure();
    this.apply(this.reduced.matches ? 1 : this.scrollProgress());

    const create = this.mode === "glass" ? createGlass : createSolid;
    this.instance = create(host, this.canvas, () => {
      requestAnimationFrame(() =>
        requestAnimationFrame(() => {
          this.canvas.style.opacity = "1";
          host.setAttribute("data-mark3d-state", "ready");
          // The nav keeps its own flat mark: the 3D one now parks in the left
          // margin instead of the logo slot, so hiding it would leave a hole.
          // Capture the base angle before anyone can drag it.
          this.wireOrbitReturn();
        }),
      );
    });

    addEventListener("scroll", this.onScroll, { passive: true });
    addEventListener("resize", this.onResize, { passive: true });

    if (this.button) {
      this.button.addEventListener("click", () => this.scrollToTop());
      this.button.addEventListener("mouseenter", () => this.setHover(true));
      this.button.addEventListener("mouseleave", () => this.setHover(false));
      this.button.addEventListener("focus", () => this.setHover(true));
      this.button.addEventListener("blur", () => this.setHover(false));
    }
  }

  // Hand-rolled rather than scrollTo({behavior:"smooth"}): native smooth
  // scrolling inside a container is distance-proportional and took ~4.5s from
  // the foot of this page, which is far too slow for a back-to-top.
  private scrollToTop() {
    const from = scrollY;
    if (from <= 0) return;
    if (this.reduced.matches) {
      scrollTo(0, 0);
      return;
    }
    const started = performance.now();
    const duration = clamp01(from / 4000) * 400 + 420; // 420-820ms by distance
    const step = () => {
      const k = Math.min(1, (performance.now() - started) / duration);
      const y = from * (1 - ease(k));
      scrollTo(0, y);
      if (k < 1) requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  }

  private setHover(on: boolean) {
    this.hovered = on;
    this.paintOpacity();
  }

  // Dragging tips the camera off its base polar angle and it stays there.
  // After the mark has been left alone, ease that tilt back — azimuth is left
  // to autoRotate, which owns the continuous turn.
  // Once parked, the mark stops turning and squares up to the camera: a
  // restless logo in the corner competes with the content it sits over.
  private setFrozen(frozen: boolean) {
    // autoRotate alone leaves the idle float and drift running, which still
    // reads as movement — all three have to stop for it to actually sit still.
    this.instance?.setOptions?.(
      frozen
        ? { autoRotate: false, floatIntensity: 0, rotationIntensity: 0 }
        : {
            autoRotate: true,
            floatIntensity: num(this.host, "glassFloatIntensity", 0.4),
            rotationIntensity: num(this.host, "glassRotationIntensity", 0.15),
          },
    );
    this.host.setAttribute("data-mark3d-frozen", String(frozen));
    // Set inline, because the reveal already wrote an inline opacity onto the
    // canvas and a stylesheet rule cannot win against that.
    this.canvas.style.opacity = frozen ? "0" : "1";

    // Parked, the flat SVG is what you see, but the renderer would happily keep
    // drawing the full 1008x616 stage behind it — 2016x1232 with transmission
    // at DPR 2, for something invisible. display:none drops the canvas to
    // clientWidth 0, the core's ResizeObserver follows it down, and the work
    // goes away. Deferred past the cross-fade so the swap stays smooth.
    clearTimeout(this.hideTimer);
    if (frozen) {
      this.hideTimer = setTimeout(() => { this.canvas.style.display = "none"; }, FADE_MS + 60);
    } else {
      this.canvas.style.display = "";
    }

    if (frozen) this.easeToBase?.(true);
  }

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

    // squareUp=false eases only the tilt and leaves heading to autoRotate;
    // true also brings the heading home, so the mark faces front and stops.
    this.easeToBase = (squareUp: boolean) => {
      cancelAnimationFrame(this.returnRaf);
      const from = spherical();
      // Shortest way round, so a nearly-complete turn does not unwind
      // backwards through everything it just rotated past.
      let deltaTheta = base.theta - from.theta;
      while (deltaTheta > Math.PI) deltaTheta -= 2 * Math.PI;
      while (deltaTheta < -Math.PI) deltaTheta += 2 * Math.PI;

      const started = performance.now();
      const step = () => {
        const k = Math.min(1, (performance.now() - started) / RETURN_DURATION_MS);
        const eased = ease(k);
        const now = spherical();
        write(
          now.r,
          lerp(from.phi, base.phi, eased),
          squareUp ? from.theta + deltaTheta * eased : now.theta,
        );
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
      this.idleTimer = setTimeout(
        () => this.easeToBase?.(this.frozen),
        RETURN_DELAY_MS,
      );
    });
  }

  // The stage is fixed, so it must be told where the nav logo actually is;
  // that x shifts with the hero's clamp()-based padding at every width.
  // The mark is drawn at 504x308 and scaled; how big it may START is decided
  // by the gap between the fixed header and the hero copy, not by a constant.
  // A fixed 7x overlapped the eyebrow the moment the headline grew a line, and
  // would do so again on any short viewport.
  private fit() {
    // Anchored to the text column, not to the viewport centre. The old model
    // was `innerWidth/2 - width/2 - 400`, so the mark slid horizontally at half
    // the rate the window resized while its scale changed with height — two
    // unrelated drifts at once, which is what read as "weird" on resize.
    //
    // Now: same left edge as the headline, width never exceeding it, and
    // vertically centred in whatever band is left above the copy. Resizing
    // rescales it in place instead of moving it.
    const eyebrow = document.querySelector(".landing-eyebrow")?.getBoundingClientRect();
    const heading = document.querySelector(".landing-copy h1")?.getBoundingClientRect();
    const copy = document.querySelector(".landing-copy")?.getBoundingClientRect();

    // Rise behind the header: the bar is empty on the left and its links sit at
    // a higher z-index, so that band is free height.
    const top = num(this.host, "startTop", 8);
    const band = Math.max(96, (eyebrow?.top ?? innerHeight * 0.6) - top - 12);
    const column = heading?.width ?? copy?.width ?? innerWidth * 0.5;

    this.startScale = Math.min(1, band / 616, column / 1008);
    this.startTop = top + (band - 616 * this.startScale) / 2;
    this.startLeft = eyebrow?.left ?? heading?.left ?? 24;
  }

  private measure() {
    this.fit();
    const slot = this.navMark?.getBoundingClientRect();
    if (slot) {
      // Offset from the nav logo slot so the mark parks in the left margin,
      // clear of the content column, rather than on top of the brand link.
      // Hard against the window edge with a small inset, independent of the
      // content column; only the vertical position still tracks the nav.
      // The navbar is fixed, so its rect is already viewport-relative and must
      // not have the scroll offset added back in.
      this.anchor = { left: num(this.host, "endInsetX", 70), top: slot.top };
    }
    const hero = document.querySelector(".landing-hero");
    const heroHeight = hero ? hero.getBoundingClientRect().height : innerHeight;
    this.travel = Math.max(240, Math.min(620, heroHeight * 0.65));
    this.host.style.left = `${this.anchor.left}px`;
    this.host.style.top = `${this.anchor.top}px`;
    if (this.button) {
      this.button.style.left = `${this.anchor.left}px`;
      this.button.style.top = `${this.anchor.top}px`;
    }
  }

  private scrollProgress() {
    return clamp01(scrollY / this.travel);
  }

  private onScroll = () => {
    if (this.ticking || this.reduced.matches) return;
    this.ticking = true;
    requestAnimationFrame(() => {
      this.ticking = false;
      this.apply(this.scrollProgress());
    });
  };

  private onResize = () => {
    this.measure();
    // apply() bails when the scroll progress is unchanged, which on resize it
    // always is — so the freshly measured size and position would be computed
    // and then never written. Invalidate first to force the re-layout.
    this.progress = -1;
    this.apply(this.reduced.matches ? 1 : this.scrollProgress());
  };

  private paintOpacity() {
    const t = ease(Math.max(this.progress, 0));
    // Spec: full at the top, 50% once scrolled, 90% while hovered.
    this.host.style.opacity = String(this.hovered ? 0.9 : lerp(1, 0.5, t));
  }

  private apply(raw: number) {
    if (Math.abs(raw - this.progress) < 0.001) return;
    this.progress = raw;
    const t = ease(raw);

    const scale = lerp(this.startScale, 1 / START_SCALE, t);
    // Travel is expressed as "where it starts" minus "where it parks", so both
    // ends are exact: the start sits on the text column, the end on the inset.
    const dx = (this.startLeft - this.anchor.left) * (1 - t);
    const dy = (this.startTop - this.anchor.top) * (1 - t);
    this.host.style.transform = `translate(${dx}px, ${dy}px) scale(${scale})`;
    this.paintOpacity();

    // drop-shadow follows the canvas alpha, so it traces the mark's actual
    // outline: a tight pass draws a contour (clear glass has no edge of its own
    // against cream) and a soft offset pass reads as a contact shadow.
    //
    // Both are divided by `scale` because the filter is applied before the
    // stage's transform shrinks it — otherwise the shadow all but vanishes at
    // logo size, which is exactly where it is needed.
    // On the host, not the canvas, so the flat SVG that replaces it when parked
    // carries the same contour and contact shadow.
    const px = (visual: number) => (visual / scale).toFixed(1);
    const contour = lerp(0.06, 0.5, t).toFixed(3);
    const contact = lerp(0.1, 0.36, t).toFixed(3);
    this.host.style.filter =
      `drop-shadow(0 0 ${px(0.8)}px rgba(24,28,22,${contour})) ` +
      `drop-shadow(0 ${px(lerp(3, 4, t))}px ${px(lerp(9, 7, t))}px rgba(24,28,22,${contact}))`;

    // One switch, not a sweep: the core's setOptions() calls loadAsset(), which
    // refetches and re-traces the SVG. Interpolating across it re-traced the
    // asset on every scroll frame and never settled.
    if (this.mode === "glass" && this.instance?.setOptions) {
      const step = t >= 0.45 ? 1 : 0;
      if (step !== this.appliedStep) {
        this.appliedStep = step;
        this.instance.setOptions(step ? GLASS_FINE : GLASS_COARSE);
      }
    }

    if (this.veil) this.veil.style.opacity = String(t);

    // Parked: square up and stop turning. Guarded so it only fires on the
    // crossing, not on every scroll frame.
    const frozen = t > 0.9;
    if (frozen !== this.frozen) {
      this.frozen = frozen;
      this.setFrozen(frozen);
    }

    // While big the canvas takes the pointer so it can be dragged to rotate;
    // once it is logo-sized the back-to-top button takes over instead.
    const big = t <= 0.6;
    this.host.style.pointerEvents = big ? "auto" : "none";
    this.canvas.style.pointerEvents = big ? "auto" : "none";
    if (this.button) this.button.hidden = big;
  }
}

export function initHausvMark3d() {
  const host = document.querySelector<HTMLElement>("[data-hausv-mark-3d]");
  if (!host || !host.dataset.src) return;

  const min = Number(host.dataset.minWidth ?? 0);
  if (min && !matchMedia(`(min-width: ${min}px)`).matches) return;
  if (!hasWebgl()) return;

  new MarkStage(
    host,
    document.querySelector<HTMLElement>("[data-mark3d-top]"),
    document.querySelector<HTMLElement>(".landing-mark"),
    document.querySelector<HTMLElement>(".mark3d-veil"),
  );
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initHausvMark3d, { once: true });
} else {
  initHausvMark3d();
}
