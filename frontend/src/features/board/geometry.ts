export const NODE_W = 176;
export const NODE_H = 52;
export interface Pt { x: number; y: number }

/** Point where a ray from the centre of a node toward `toward` leaves the node's rectangle. */
export function clipToRect(c: Pt, toward: Pt, w = NODE_W, h = NODE_H): Pt {
  const dx = toward.x - c.x;
  const dy = toward.y - c.y;
  if (dx === 0 && dy === 0) return c;
  const tx = dx !== 0 ? w / 2 / Math.abs(dx) : Infinity;
  const ty = dy !== 0 ? h / 2 / Math.abs(dy) : Infinity;
  const t = Math.min(tx, ty);
  return { x: c.x + dx * t, y: c.y + dy * t };
}

/** Line between two node centres, trimmed to the node borders and shifted sideways so parallel links do not overlap. */
export function linkEndpoints(a: Pt, b: Pt, offset: number): { x1: number; y1: number; x2: number; y2: number; mx: number; my: number } {
  const p1 = clipToRect(a, b);
  const p2 = clipToRect(b, a);
  const len = Math.hypot(b.x - a.x, b.y - a.y) || 1;
  const ox = (-(b.y - a.y) / len) * offset;
  const oy = ((b.x - a.x) / len) * offset;
  const x1 = p1.x + ox, y1 = p1.y + oy, x2 = p2.x + ox, y2 = p2.y + oy;
  return { x1, y1, x2, y2, mx: (x1 + x2) / 2, my: (y1 + y2) / 2 };
}

export function pairKey(a: string, b: string): string { return a < b ? `${a}|${b}` : `${b}|${a}`; }
