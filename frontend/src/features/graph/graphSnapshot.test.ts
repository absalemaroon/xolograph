import Graph from 'graphology';
import { describe, expect, it } from 'vitest';
import { graphToSnapshot, snapshotToGraph } from './graphSnapshot';
import type { EdgeAttrs, NodeAttrs, XGraph } from './graphStore';

const A = '0x' + 'a'.repeat(40);
const B = '0x' + 'b'.repeat(40);

function sample(): XGraph {
  const g: XGraph = new Graph<NodeAttrs, EdgeAttrs>({ type: 'directed', multi: false });
  g.addNode(A, { label: 'a', x: 1, y: 2, size: 9, expanded: true });
  g.addNode(B, { label: 'b', x: -1, y: 0.5, size: 5, expanded: false });
  g.addDirectedEdgeWithKey(`${A}>${B}`, A, B, { count: 3, nativeWei: '1000000000000000000', tokenCount: 1, size: 2 });
  return g;
}

describe('graph snapshots', () => {
  it('round-trips positions and aggregates', () => {
    const snap = graphToSnapshot(sample(), { seeds: [A], truncated: false, provenance: [], savedAt: '2026-01-01T00:00:00Z' });
    const back = snapshotToGraph(JSON.parse(JSON.stringify(snap)));
    expect(back.graph.order).toBe(2);
    expect(back.graph.getNodeAttribute(B, 'x')).toBe(-1);
    expect(back.graph.getEdgeAttribute(`${A}>${B}`, 'nativeWei')).toBe('1000000000000000000');
    expect(back.seeds).toEqual([A]);
  });
  it('rejects malformed or hostile data', () => {
    const good = JSON.parse(JSON.stringify(graphToSnapshot(sample(), { seeds: [A], truncated: false, provenance: [], savedAt: '' })));
    expect(() => snapshotToGraph(null)).toThrow();
    expect(() => snapshotToGraph({ ...good, format: 'other' })).toThrow();
    expect(() => snapshotToGraph({ ...good, nodes: [{ id: 'not-an-address', x: 0, y: 0, size: 1, expanded: true }] })).toThrow();
    expect(() => snapshotToGraph({ ...good, edges: [{ source: A, target: '0x' + 'c'.repeat(40), count: 1, nativeWei: '1', tokenCount: 0 }] })).toThrow();
    expect(() => snapshotToGraph({ ...good, edges: [{ source: A, target: B, count: 1, nativeWei: '-1', tokenCount: 0 }] })).toThrow();
    expect(() => snapshotToGraph({ ...good, seeds: ['0x' + 'd'.repeat(40)] })).toThrow();
  });
});
