#!/usr/bin/env python3
"""Generate internal/web/assets/hausv-mark.glb — the dedicated 3D brand mark.

Why a hand-built model instead of the traced SVG:

GlassObject's 2D path rasterises the artwork, walks the contours, and rounds
every corner by `bevel`. Our mark is stroke art drawn with round caps and round
joins, so each stroke traced out as a capsule and the whole thing read as
balloon sculpture rather than architecture.

A GLB skips that path completely: the core mounts the geometry as-is and only
swaps the material, so no tracing and no corner rounding ever runs.

The outer silhouette is taken vertex-for-vertex from hausv-mark.svg so the logo
is unchanged in outline. What improves is the interior: mitred corners, flat
butt ends, and real rectangular window and door openings instead of the slots
the stroke art implied.
"""

import json
import math
import pathlib
import struct

OUT = pathlib.Path(__file__).resolve().parents[2] / "internal/web/assets/hausv-mark.glb"

STROKE = 2.2   # matches the SVG stroke-width, so the silhouette is identical
FRAME = 1.5    # slimmer bars for the window frames
DEPTH = 4.0    # extrusion along z, in SVG units
ROUND = 0.30   # edge break, ~14% of a stroke: catches a highlight, stays crisp
ROUND_SEGMENTS = 2

# Centre of the artwork's bounding box (x 9..63, y 9..35) in SVG coordinates.
CX, CY = 36.0, 22.0

# --- elements -------------------------------------------------------------
# (points, width, closed). Coordinates are SVG user units, y pointing down;
# they are flipped to y-up at emit time.
#
# The five structural bands are copied straight from the SVG path data. The
# windows are the one deliberate change: the SVG draws them as single
# horizontal strokes, which extrude into floating bars, so here they become
# closed rectangular frames centred on exactly those strokes.
ELEMENTS = [
    # ground line: M9 35h54
    ([(9, 35), (63, 35)], STROKE, False),
    # Left and right houses, minus their inner wall.
    #
    # The SVG draws the left house down to (27,35) and the centre house down to
    # (25,35). Two parallel walls 2 units apart, each 2.2 wide, span x 25.9-28.1
    # and 23.9-26.1 — they intersect. Flat art hides that; as solids they fuse
    # into one bar twice the proper thickness, poking through the centre house.
    #
    # Terraced houses share a party wall, so the centre house's wall serves both
    # and the outer roofs simply land on it. Only interior detail changes; the
    # outer silhouette is untouched.
    # The roofs are cut back to the party wall for the same reason. The SVG runs
    # them to (27,23) and (45,23), both past the centre house's wall faces at
    # x=23.9 and x=48.1, so each roof drove a wedge through that wall.
    #
    # A butt end is square to its own direction, so on a 36.87-degree roof the
    # cap's leading corner sits half_width * 0.6 = 0.66 ahead of the centreline.
    # Ending at 23.9 - 0.66 = 23.24 therefore lands the corner exactly on the
    # wall face: touching, not crossing. y = 17 + (23.24-19) * 0.75 = 20.18.
    # left house: M11 35V23l8-6 8 6v12, inner wall dropped, roof cut to the wall
    ([(11, 35), (11, 23), (19, 17), (23.24, 20.18)], STROKE, False),
    # right house: mirrored about x=36
    ([(48.76, 20.18), (53, 17), (61, 23), (61, 35)], STROKE, False),
    # centre house: M25 35V17.5L36 9l11 8.5V35
    ([(25, 35), (25, 17.5), (36, 9), (47, 17.5), (47, 35)], STROKE, False),
    # door: M31.5 35v-9h9v9
    ([(31.5, 35), (31.5, 26), (40.5, 26), (40.5, 35)], STROKE, False),
    # windows, centred on M15.5 27h5 / M51.5 27h5 / M31 21h10
    ([(15.5, 24.5), (20.5, 24.5), (20.5, 29.5), (15.5, 29.5)], FRAME, True),
    ([(51.5, 24.5), (56.5, 24.5), (56.5, 29.5), (51.5, 29.5)], FRAME, True),
    ([(31.0, 18.5), (41.0, 18.5), (41.0, 23.5), (31.0, 23.5)], FRAME, True),
]


def normalise(x, y):
    length = math.hypot(x, y) or 1e-9
    return x / length, y / length


def offsets(points, width, closed):
    """Left/right offset polylines with mitred joins and butt ends."""
    half = width / 2.0
    count = len(points)
    left, right = [], []
    for i in range(count):
        px, py = points[i]
        prev_dir = next_dir = None
        if i > 0 or closed:
            ax, ay = points[(i - 1) % count]
            prev_dir = normalise(px - ax, py - ay)
        if i < count - 1 or closed:
            bx, by = points[(i + 1) % count]
            next_dir = normalise(bx - px, by - py)

        if prev_dir is None:
            dx, dy = next_dir
            nx, ny = -dy, dx
            ox, oy = nx * half, ny * half
        elif next_dir is None:
            dx, dy = prev_dir
            nx, ny = -dy, dx
            ox, oy = nx * half, ny * half
        else:
            n1 = (-prev_dir[1], prev_dir[0])
            n2 = (-next_dir[1], next_dir[0])
            mx, my = normalise(n1[0] + n2[0], n1[1] + n2[1])
            # Mitre length grows as the turn sharpens; clamp so a near-reversal
            # cannot fire a spike off into space.
            denom = mx * n1[0] + my * n1[1]
            scale = 1.0 / denom if abs(denom) > 0.25 else 4.0
            ox, oy = mx * half * scale, my * half * scale
        left.append((px + ox, py + oy))
        right.append((px - ox, py - oy))
    return left, right


def to3d(pt, z):
    """SVG (y-down) -> model space (y-up), centred on the artwork."""
    return (pt[0] - CX, CY - pt[1], z)


def cross(a, b):
    return (a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0])


def sub(a, b):
    return (a[0] - b[0], a[1] - b[1], a[2] - b[2])


class Mesh:
    def __init__(self):
        self.positions = []
        self.normals = []

    def triangle(self, a, b, c, outward):
        """Emit a triangle, flipping winding so its normal faces `outward`."""
        n = cross(sub(b, a), sub(c, a))
        if n[0] * outward[0] + n[1] * outward[1] + n[2] * outward[2] < 0:
            b, c = c, b
            n = cross(sub(b, a), sub(c, a))
        length = math.sqrt(n[0] ** 2 + n[1] ** 2 + n[2] ** 2) or 1e-9
        unit = (n[0] / length, n[1] / length, n[2] / length)
        for vertex in (a, b, c):
            self.positions.append(vertex)
            self.normals.append(unit)

    def quad(self, a, b, c, d, outward):
        self.triangle(a, b, c, outward)
        self.triangle(a, c, d, outward)


def profile():
    """Cross-section rings from front face to back face.

    A single sharp extrusion reads as laser-cut card. Breaking the four long
    edges with a quarter-round gives them a highlight to catch without
    softening the silhouette — each ring is the same path inset a little and
    pushed toward the face.
    """
    straight = DEPTH / 2.0 - ROUND
    rings = []
    for k in range(ROUND_SEGMENTS, -1, -1):          # front rounding, outermost first
        a = (k / ROUND_SEGMENTS) * (math.pi / 2)
        rings.append((ROUND * (1 - math.cos(a)), straight + ROUND * math.sin(a)))
    for k in range(0, ROUND_SEGMENTS + 1):           # back rounding
        a = (k / ROUND_SEGMENTS) * (math.pi / 2)
        rings.append((ROUND * (1 - math.cos(a)), -(straight + ROUND * math.sin(a))))
    return rings


def build():
    mesh = Mesh()
    rings = profile()

    for points, width, closed in ELEMENTS:
        count = len(points)
        span = count if closed else count - 1
        # One offset pair per ring: narrower rings sit closer to the faces.
        bands = [offsets(points, width - 2 * inset, closed) for inset, _ in rings]
        zs = [z for _, z in rings]

        for i in range(span):
            j = (i + 1) % count

            # Flat front and back faces, from the outermost rings.
            for ring, outward in ((0, (0, 0, 1)), (len(rings) - 1, (0, 0, -1))):
                left, right = bands[ring]
                z = zs[ring]
                mesh.quad(to3d(left[i], z), to3d(right[i], z),
                          to3d(right[j], z), to3d(left[j], z), outward)

            # Everything between: rounded edges, then the flat side walls.
            for r in range(len(rings) - 1):
                (la, ra), (lb, rb) = bands[r], bands[r + 1]
                za, zb = zs[r], zs[r + 1]

                edge = normalise(la[j][0] - la[i][0], la[j][1] - la[i][1])
                out = (-edge[1], -edge[0], 0)
                mesh.quad(to3d(la[i], za), to3d(la[j], za),
                          to3d(lb[j], zb), to3d(lb[i], zb), out)

                edge = normalise(ra[j][0] - ra[i][0], ra[j][1] - ra[i][1])
                out = (edge[1], edge[0], 0)
                mesh.quad(to3d(ra[i], za), to3d(ra[j], za),
                          to3d(rb[j], zb), to3d(rb[i], zb), out)

        if not closed:
            # Flat butt ends — the SVG's round caps are what made them sausages.
            # The end is closed by strips across the same cross-section.
            for idx in (0, count - 1):
                other = points[1] if idx == 0 else points[-2]
                dx, dy = normalise(points[idx][0] - other[0], points[idx][1] - other[1])
                out = (dx, -dy, 0)
                for r in range(len(rings) - 1):
                    (la, ra), (lb, rb) = bands[r], bands[r + 1]
                    za, zb = zs[r], zs[r + 1]
                    mesh.quad(to3d(la[idx], za), to3d(ra[idx], za),
                              to3d(rb[idx], zb), to3d(lb[idx], zb), out)
    return mesh


def write_glb(mesh, path):
    flat = []
    for p in mesh.positions:
        flat.extend(p)
    for n in mesh.normals:
        flat.extend(n)
    blob = struct.pack(f"<{len(flat)}f", *flat)
    while len(blob) % 4:
        blob += b"\x00"

    count = len(mesh.positions)
    pos_bytes = count * 12
    xs = [p[0] for p in mesh.positions]
    ys = [p[1] for p in mesh.positions]
    zs = [p[2] for p in mesh.positions]

    gltf = {
        "asset": {"version": "2.0", "generator": "hausv-org scripts/mark3d/make-model.py"},
        "scene": 0,
        "scenes": [{"nodes": [0]}],
        "nodes": [{"mesh": 0, "name": "hausv-mark"}],
        "meshes": [{"name": "hausv-mark", "primitives": [{"attributes": {"POSITION": 0, "NORMAL": 1}, "mode": 4}]}],
        "buffers": [{"byteLength": len(blob)}],
        "bufferViews": [
            {"buffer": 0, "byteOffset": 0, "byteLength": pos_bytes, "target": 34962},
            {"buffer": 0, "byteOffset": pos_bytes, "byteLength": pos_bytes, "target": 34962},
        ],
        "accessors": [
            {"bufferView": 0, "componentType": 5126, "count": count, "type": "VEC3",
             "min": [min(xs), min(ys), min(zs)], "max": [max(xs), max(ys), max(zs)]},
            {"bufferView": 1, "componentType": 5126, "count": count, "type": "VEC3"},
        ],
    }

    json_chunk = json.dumps(gltf, separators=(",", ":")).encode()
    while len(json_chunk) % 4:
        json_chunk += b" "

    total = 12 + 8 + len(json_chunk) + 8 + len(blob)
    with open(path, "wb") as fh:
        fh.write(struct.pack("<III", 0x46546C67, 2, total))
        fh.write(struct.pack("<II", len(json_chunk), 0x4E4F534A))
        fh.write(json_chunk)
        fh.write(struct.pack("<II", len(blob), 0x004E4942))
        fh.write(blob)
    return count, total


if __name__ == "__main__":
    mesh = build()
    verts, size = write_glb(mesh, OUT)
    print(f"wrote {OUT.name}: {verts} vertices, {verts // 3} triangles, {size} bytes")
