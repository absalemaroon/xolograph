import { describe, expect, it } from 'vitest';
import { NODE_H, NODE_W, clipToRect, linkEndpoints, pairKey } from './geometry';

describe('board geometry', () => {
  it('clips to the rectangle border', () => {
    expect(clipToRect({ x: 0, y: 0 }, { x: 1000, y: 0 })).toEqual({ x: NODE_W / 2, y: 0 });
    expect(clipToRect({ x: 0, y: 0 }, { x: 0, y: -500 })).toEqual({ x: 0, y: -NODE_H / 2 });
    expect(clipToRect({ x: 5, y: 5 }, { x: 5, y: 5 })).toEqual({ x: 5, y: 5 });
  });
  it('offsets parallel links to opposite sides', () => {
    const a = { x: 0, y: 0 }, b = { x: 400, y: 0 };
    expect(linkEndpoints(a, b, 10).y1).toBeCloseTo(10);
    expect(linkEndpoints(a, b, -10).y1).toBeCloseTo(-10);
  });
  it('keys unordered pairs identically', () => { expect(pairKey('a', 'b')).toBe(pairKey('b', 'a')); });
});
