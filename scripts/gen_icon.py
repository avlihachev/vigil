#!/usr/bin/env python3
"""Generate Vigil app icon: "CYCLOPEAN" — a single terminal eye on near-black."""
from PIL import Image, ImageDraw, ImageFilter
import math, os

# ── palette ──────────────────────────────────────────────────────────────────
BG      = (9,   9,  11)
IRIS_BG = (12,  12,  15)
LINE    = (221, 213, 196)   # warm off-white
WHITE   = (255, 255, 255)

# ── helpers ───────────────────────────────────────────────────────────────────

def rounded_rect_mask(size, radius):
    m = Image.new("L", (size, size), 0)
    ImageDraw.Draw(m).rounded_rectangle([0, 0, size-1, size-1], radius=radius, fill=255)
    return m

def quadratic_bezier_pts(p0, p1, p2, n=200):
    """Return n+1 points along a quadratic bezier (p0→p1 control→p2)."""
    pts = []
    for i in range(n + 1):
        t = i / n
        x = (1-t)**2 * p0[0] + 2*t*(1-t)*p1[0] + t**2 * p2[0]
        y = (1-t)**2 * p0[1] + 2*t*(1-t)*p1[1] + t**2 * p2[1]
        pts.append((x, y))
    return pts

def eye_polygon(cx, cy, hw, ctrl_top_y, ctrl_bot_y, n=200):
    """Almond eye: two quadratic bezier arcs meeting at left/right tips."""
    left  = (cx - hw, cy)
    right = (cx + hw, cy)
    top_ctrl = (cx, ctrl_top_y)
    bot_ctrl = (cx, ctrl_bot_y)
    top_arc = quadratic_bezier_pts(left,  top_ctrl, right, n)
    bot_arc = quadratic_bezier_pts(right, bot_ctrl, left,  n)
    return top_arc + bot_arc

def composite(base, draw_fn, blur=0):
    layer = Image.new("RGBA", base.size, (0, 0, 0, 0))
    draw_fn(layer)
    if blur:
        layer = layer.filter(ImageFilter.GaussianBlur(blur))
    return Image.alpha_composite(base, layer)

# ── icon builder ──────────────────────────────────────────────────────────────

def make_icon(size):
    s = size / 1024

    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    bg  = Image.new("RGBA", (size, size), BG + (255,))
    img.paste(bg, mask=rounded_rect_mask(size, size // 6))

    cx, cy = size // 2, size // 2

    # eye geometry (symmetric arcs ±155px so iris fits naturally)
    hw          = int(350 * s)   # half-width of eye
    ctrl_top_y  = int(202 * s)   # control point → top peak at cy-155
    ctrl_bot_y  = int(822 * s)   # control point → bot peak at cy+155
    iris_r      = int(142 * s)   # iris radius (fits with ~13px clearance)
    ring_radii  = [int(r * s) for r in (134, 102, 68)]
    rl_inner    = int(36  * s)   # radial line inner radius
    rl_outer    = int(128 * s)   # radial line outer radius (< 155 ✓)
    pupil_r     = int(46  * s)
    cursor_w    = max(2, int(5  * s))
    cursor_h    = int(42  * s)

    eye_pts = eye_polygon(cx, cy, hw, ctrl_top_y, ctrl_bot_y)

    # ── iris fill ─────────────────────────────────────────────────────────────
    draw = ImageDraw.Draw(img)
    draw.ellipse([cx - iris_r, cy - iris_r, cx + iris_r, cy + iris_r],
                 fill=IRIS_BG + (255,))

    # ── iris rings ────────────────────────────────────────────────────────────
    lw = max(1, int(2 * s))
    for r, alpha in zip(ring_radii, (36, 28, 20)):
        def _ring(layer, r=r, alpha=alpha):
            ImageDraw.Draw(layer).ellipse(
                [cx-r, cy-r, cx+r, cy+r],
                outline=LINE + (alpha,), width=lw)
        img = composite(img, _ring)

    # ── iris radial lines (24 × 15°) ─────────────────────────────────────────
    def _radials(layer):
        d = ImageDraw.Draw(layer)
        for deg in range(0, 360, 15):
            angle = math.radians(deg)
            x1 = cx + rl_inner * math.cos(angle)
            y1 = cy + rl_inner * math.sin(angle)
            x2 = cx + rl_outer * math.cos(angle)
            y2 = cy + rl_outer * math.sin(angle)
            d.line([(x1, y1), (x2, y2)], fill=LINE + (17,),
                   width=max(1, int(1.5 * s)))
    img = composite(img, _radials)

    # ── pupil ─────────────────────────────────────────────────────────────────
    draw = ImageDraw.Draw(img)
    draw.ellipse([cx - pupil_r, cy - pupil_r, cx + pupil_r, cy + pupil_r],
                 fill=BG + (255,))

    # ── cursor (blinking caret) ───────────────────────────────────────────────
    draw.rectangle([cx - cursor_w//2, cy - cursor_h//2,
                    cx + cursor_w//2, cy + cursor_h//2],
                   fill=WHITE + (230,))

    # ── eye outline ───────────────────────────────────────────────────────────
    line_w = max(2, int(4 * s))
    draw.line(eye_pts, fill=LINE + (230,), width=line_w, joint="curve")

    # ── specular highlight ────────────────────────────────────────────────────
    hx = cx + int(68 * s)
    hy = cy - int(62 * s)
    hr = max(2, int(7 * s))
    draw.ellipse([hx - hr, hy - hr, hx + hr, hy + hr],
                 fill=WHITE + (107,))

    return img


# ── generate all sizes ────────────────────────────────────────────────────────
sizes = [16, 32, 64, 128, 256, 512, 1024]
os.makedirs("iconset.iconset", exist_ok=True)

for sz in sizes:
    icon = make_icon(sz)
    icon.save(f"iconset.iconset/icon_{sz}x{sz}.png")
    if sz <= 512:
        make_icon(sz * 2).save(f"iconset.iconset/icon_{sz}x{sz}@2x.png")

make_icon(1024).save("build/appicon.png")
os.system("iconutil -c icns iconset.iconset -o build/darwin/vigil.icns")
print("Done: build/appicon.png + build/darwin/vigil.icns")
