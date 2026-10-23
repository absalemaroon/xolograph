import Graph from 'graphology';
import type { Provenance } from '@/api/types';
import type { EdgeAttrs, NodeAttrs, XGraph } from './graphStore';

export const SNAPSHOT_FORMAT = 'xolograph-graph-v2';
export const SNAPSHOT_MAX_NODES = 300;
export const SNAPSHOT_MAX_EDGES = 20000;

export interface SnapshotMeta { seeds: string[]; truncated: boolean; provenance: Provenance[]; savedAt: string }

/** Positions + the aggregates that were loaded. A point-in-time record: restoring it never re-queries the chain. */
export function graphToSnapshot(g: XGraph, meta: SnapshotMeta): Record<string, unknown> {
  return {
    format: SNAPSHOT_FORMAT,
    ...meta,
    coverage: 'Built from the most recent page of ETH and token activity per expanded node; not complete history.',
    nodes: g.mapNodes((id, a) => ({ id, x: a.x, y: a.y, size: a.size, expanded: a.expanded })),
    edges: g.mapEdges((_k, a, source, target) => ({ source, target, count: a.count, nativeWei: a.nativeWei, tokenCount: a.tokenCount })),
  };
}

const ADDR = /^0x[0-9a-f]{40}$/;
const isObj = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null && !Array.isArray(v);
const num = (v: unknown): v is number => typeof v === 'number' && Number.isFinite(v);

/** Strictly validates stored data before it is turned into a graph. Throws on anything unexpected. */
export function snapshotToGraph(data: unknown): { graph: XGraph; seeds: string[]; truncated: boolean; provenance: Provenance[]; savedAt: string } {
  if (!isObj(data) || data.format !== SNAPSHOT_FORMAT) throw new Error('Unsupported snapshot format.');
  const { nodes, edges, seeds } = data;
  if (!Array.isArray(nodes) || !Array.isArray(edges) || !Array.isArray(seeds)) throw new Error('Snapshot is malformed.');
  if (nodes.length > SNAPSHOT_MAX_NODES || edges.length > SNAPSHOT_MAX_EDGES) throw new Error('Snapshot is larger than the supported limits.');
  const g: XGraph = new Graph<NodeAttrs, EdgeAttrs>({ type: 'directed', multi: false });
  for (const n of nodes) {
    if (!isObj(n) || typeof n.id !== 'string' || !ADDR.test(n.id) || !num(n.x) || !num(n.y) || !num(n.size) || typeof n.expanded !== 'boolean' || g.hasNode(n.id)) throw new Error('Snapshot contains an invalid node.');
    g.addNode(n.id, { label: `${n.id.slice(0, 6)}…${n.id.slice(-4)}`, x: n.x, y: n.y, size: Math.min(Math.max(n.size, 2), 20), expanded: n.expanded });
  }
  for (const e of edges) {
    if (!isObj(e) || typeof e.source !== 'string' || typeof e.target !== 'string' || !g.hasNode(e.source) || !g.hasNode(e.target) || !num(e.count) || !num(e.tokenCount)
      || typeof e.nativeWei !== 'string' || !/^\d+$/.test(e.nativeWei)) throw new Error('Snapshot contains an invalid edge.');
    const key = `${e.source}>${e.target}`;
    if (g.hasEdge(key)) throw new Error('Snapshot contains a duplicate edge.');
    g.addDirectedEdgeWithKey(key, e.source, e.target, { count: e.count, nativeWei: e.nativeWei, tokenCount: e.tokenCount, size: 1 + Math.log2(e.count + 1) * 0.8 });
  }
  const ss = seeds.filter((s): s is string => typeof s === 'string' && g.hasNode(s));
  if (ss.length === 0) throw new Error('Snapshot has no valid seed.');
  return {
    graph: g, seeds: ss, truncated: data.truncated === true,
    provenance: Array.isArray(data.provenance) ? (data.provenance as Provenance[]) : [],
    savedAt: typeof data.savedAt === 'string' ? data.savedAt : '',
  };
}
