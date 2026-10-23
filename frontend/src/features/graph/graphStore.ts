import Graph from 'graphology';
import forceAtlas2 from 'graphology-layout-forceatlas2';
import { create } from 'zustand';
import { api } from '@/api/endpoints';
import type { AddressAttribution, Provenance, Transfer } from '@/api/types';
import { shortAddr } from '@/lib/units';
import { isEthAddress } from '@/lib/validate';
import { snapshotToGraph } from './graphSnapshot';

export const MAX_NODES = 300;
export const MAX_SEEDS = 10;

export interface NodeAttrs { label: string; x: number; y: number; size: number; expanded: boolean }
export interface EdgeAttrs { count: number; nativeWei: string; tokenCount: number; size: number }
export type XGraph = Graph<NodeAttrs, EdgeAttrs>;

interface GraphState {
  graph: XGraph;
  version: number;
  seed: string | null;
  seeds: string[];
  /** True for a restored snapshot: values are a saved point in time and the graph cannot be expanded. */
  readOnly: boolean;
  restoredAt: string | null;
  attributions: Record<string, AddressAttribution[]>;
  attributionError: string | null;
  loading: boolean;
  error: string | null;
  truncated: boolean;
  provenance: Provenance[];
  build: (address: string) => Promise<void>;
  buildMulti: (addresses: string[]) => Promise<void>;
  restore: (data: unknown) => void;
  loadAttributions: () => Promise<void>;
  expand: (address: string) => Promise<void>;
  clear: () => void;
}

const newGraph = (): XGraph => new Graph<NodeAttrs, EdgeAttrs>({ type: 'directed', multi: false });
let seen = new Set<string>(); // transfers already merged (avoid double counting on re-expansion)

/** Merge value movements into the graph. Failed transactions moved no value and are skipped. */
function merge(g: XGraph, origin: string, transfers: Transfer[]): boolean {
  let truncated = false;
  const o = g.getNodeAttributes(origin);
  const ensure = (a: string): boolean => {
    if (g.hasNode(a)) return true;
    if (g.order >= MAX_NODES) { truncated = true; return false; }
    g.addNode(a, { label: shortAddr(a), x: o.x + (Math.random() - 0.5), y: o.y + (Math.random() - 0.5), size: 5, expanded: false });
    return true;
  };
  for (const t of transfers) {
    if (t.status === 'FAILED' || !t.from || !t.to || t.from === t.to) continue;
    const id = `${t.txHash}:${t.logIndex}`;
    if (seen.has(id)) continue;
    if (!ensure(t.from) || !ensure(t.to)) continue;
    seen.add(id);
    const key = `${t.from}>${t.to}`;
    if (!g.hasEdge(key)) g.addDirectedEdgeWithKey(key, t.from, t.to, { count: 0, nativeWei: '0', tokenCount: 0, size: 1 });
    g.updateEdgeAttributes(key, (a) => {
      const count = a.count + 1;
      const isNative = t.kind === 'NATIVE' && /^\d+$/.test(t.value);
      return {
        count,
        nativeWei: isNative ? (BigInt(a.nativeWei) + BigInt(t.value)).toString() : a.nativeWei,
        tokenCount: isNative ? a.tokenCount : a.tokenCount + 1,
        size: 1 + Math.log2(count + 1) * 0.8,
      };
    });
  }
  return truncated;
}

function layout(g: XGraph): void {
  if (g.order < 2) return;
  forceAtlas2.assign(g, { iterations: 60, settings: { ...forceAtlas2.inferSettings(g), gravity: 1, slowDown: 2 } });
}

async function fetchNeighbours(address: string): Promise<{ items: Transfer[]; provenance: Provenance[]; partial: string | null }> {
  const [n, t] = await Promise.allSettled([api.chain.activity(address, 'native'), api.chain.activity(address, 'token')]);
  if (n.status === 'rejected' && t.status === 'rejected') {
    throw n.reason instanceof Error ? n.reason : new Error(String(n.reason));
  }
  const items: Transfer[] = [];
  const provenance: Provenance[] = [];
  for (const r of [n, t]) if (r.status === 'fulfilled') { items.push(...r.value.items); provenance.push(r.value.provenance); }
  const partial = n.status === 'rejected' ? 'ETH transactions unavailable' : t.status === 'rejected' ? 'Token transfers unavailable' : null;
  return { items, provenance, partial };
}

export const useGraph = create<GraphState>((set, get) => ({
  graph: newGraph(),
  version: 0,
  seed: null,
  seeds: [],
  readOnly: false,
  restoredAt: null,
  attributions: {},
  attributionError: null,
  loading: false,
  error: null,
  truncated: false,
  provenance: [],

  clear: () => { seen = new Set(); set({ graph: newGraph(), seed: null, seeds: [], readOnly: false, restoredAt: null, attributions: {}, attributionError: null, error: null, truncated: false, provenance: [], version: get().version + 1 }); },

  build: async (input) => get().buildMulti([input]),

  buildMulti: async (inputs) => {
    const addrs = [...new Set(inputs.map((a) => a.trim().toLowerCase()).filter(Boolean))];
    if (addrs.length === 0 || addrs.some((a) => !isEthAddress(a))) {
      set({ error: addrs.length > 1 ? 'Every seed must be a valid 42-character Ethereum address.' : 'Enter a valid 42-character Ethereum address.' });
      return;
    }
    if (addrs.length > MAX_SEEDS) { set({ error: `At most ${MAX_SEEDS} seed addresses at once.` }); return; }
    seen = new Set();
    const g = newGraph();
    addrs.forEach((a, i) => {
      const ang = (2 * Math.PI * i) / addrs.length;
      g.addNode(a, { label: shortAddr(a), x: addrs.length > 1 ? 3 * Math.cos(ang) : 0, y: addrs.length > 1 ? 3 * Math.sin(ang) : 0, size: 9, expanded: true });
    });
    set({ graph: g, seed: addrs[0], seeds: addrs, readOnly: false, restoredAt: null, attributions: {}, attributionError: null, loading: true, error: null, truncated: false, provenance: [] });
    let truncated = false;
    const provenance: Provenance[] = [];
    const notes: string[] = [];
    let firstError: unknown = null;
    let okCount = 0;
    // Sequential on purpose: the public index is rate limited.
    for (const a of addrs) {
      try {
        const r = await fetchNeighbours(a);
        truncated = merge(g, a, r.items) || truncated;
        provenance.push(...r.provenance);
        okCount++;
        if (r.partial) notes.push(`${shortAddr(a)}: ${r.partial}`);
      } catch (e) {
        firstError = firstError ?? e;
        notes.push(`${shortAddr(a)}: ${e instanceof Error ? e.message : String(e)}`);
      }
    }
    if (okCount === 0) {
      set({ loading: false, error: firstError instanceof Error ? firstError.message : String(firstError), version: get().version + 1 });
      void get().loadAttributions();
      return;
    }
    layout(g);
    set({ truncated, provenance, error: notes.length > 0 ? notes.join(' · ') : null, loading: false, version: get().version + 1 });
    void get().loadAttributions();
  },

  restore: (data) => {
    try {
      const r = snapshotToGraph(data);
      seen = new Set();
      set({ graph: r.graph, seed: r.seeds[0], seeds: r.seeds, readOnly: true, restoredAt: r.savedAt || null, attributions: {}, attributionError: null,
        truncated: r.truncated, provenance: r.provenance, error: null, loading: false, version: get().version + 1 });
      void get().loadAttributions();
    } catch (e) {
      set({ error: e instanceof Error ? e.message : String(e) });
    }
  },

  /** Entity claims are read live from the local database, so a restored snapshot shows current attributions. */
  loadAttributions: async () => {
    const g = get().graph;
    if (g.order === 0) return;
    try {
      const attributions = await api.entities.lookup(g.nodes());
      if (get().graph !== g) return; // graph was replaced while the request was in flight
      set({ attributions, attributionError: null, version: get().version + 1 });
    } catch (e) {
      if (get().graph === g) set({ attributionError: e instanceof Error ? e.message : String(e) });
    }
  },

  expand: async (address) => {
    const { graph: g, loading, readOnly } = get();
    if (readOnly || loading || !g.hasNode(address) || g.getNodeAttribute(address, 'expanded')) return;
    set({ loading: true, error: null });
    try {
      const r = await fetchNeighbours(address);
      const truncated = merge(g, address, r.items);
      g.setNodeAttribute(address, 'expanded', true);
      g.setNodeAttribute(address, 'size', 7);
      layout(g);
      set((s) => ({ truncated: s.truncated || truncated, provenance: [...s.provenance, ...r.provenance], error: r.partial, loading: false, version: s.version + 1 }));
      void get().loadAttributions();
    } catch (e) {
      set({ loading: false, error: e instanceof Error ? e.message : String(e) });
    }
  },
}));
