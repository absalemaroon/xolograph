import type Graph from 'graphology';

/* eslint-disable @typescript-eslint/no-explicit-any */
type AnyGraph = Graph<any, any>;

/** Breadth-first shortest path (fewest hops) within the loaded graph. */
export function shortestPath(g: AnyGraph, from: string, to: string, directed: boolean): string[] | null {
  if (!g.hasNode(from) || !g.hasNode(to)) return null;
  if (from === to) return [from];
  const prev = new Map<string, string>();
  const seen = new Set<string>([from]);
  const queue: string[] = [from];
  while (queue.length > 0) {
    const n = queue.shift() as string;
    for (const m of directed ? g.outNeighbors(n) : g.neighbors(n)) {
      if (seen.has(m)) continue;
      seen.add(m);
      prev.set(m, n);
      if (m === to) {
        const path = [to];
        let c = to;
        while (prev.has(c)) { c = prev.get(c) as string; path.push(c); }
        return path.reverse();
      }
      queue.push(m);
    }
  }
  return null;
}

/** Edge keys (directed `a>b`) traversed by a node path, whichever direction exists. */
export function pathEdges(g: AnyGraph, path: string[]): string[] {
  const out: string[] = [];
  for (let i = 0; i < path.length - 1; i++) {
    const f = `${path[i]}>${path[i + 1]}`;
    const r = `${path[i + 1]}>${path[i]}`;
    if (g.hasEdge(f)) out.push(f);
    else if (g.hasEdge(r)) out.push(r);
  }
  return out;
}

export function graphToJSON(g: AnyGraph, meta: Record<string, unknown>): object {
  return {
    format: 'xolograph-graph-v1',
    ...meta,
    nodes: g.mapNodes((id, a) => ({ id, expanded: a.expanded })),
    edges: g.mapEdges((key, a, source, target) => ({ key, source, target, count: a.count, nativeWei: a.nativeWei, tokenTransfers: a.tokenCount })),
  };
}
