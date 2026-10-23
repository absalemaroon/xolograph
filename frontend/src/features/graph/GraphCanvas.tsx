import Sigma from 'sigma';
import { useEffect, useRef } from 'react';
import type Graph from 'graphology';
import type { AddressAttribution } from '@/api/types';
import type { ThemeMode } from '@/state/settingsStore';
import { displayAttribution, edgePasses, nodeLabel } from './graphFilters';
import type { GraphFilters } from './graphFilters';
import type { XGraph } from './graphStore';

interface Props {
  graph: XGraph;
  version: number;
  selected: string | null;
  seeds: string[];
  filters: GraphFilters;
  attributions: Record<string, AddressAttribution[]>;
  pathNodes: string[];
  pathEdgeKeys: string[];
  mode: ThemeMode;
  onSelect: (id: string | null) => void;
  onExpand: (id: string) => void;
}

interface Live {
  selected: string | null; seeds: Set<string>; pathNodes: Set<string>; pathEdges: Set<string>;
  filters: GraphFilters; attributions: Record<string, AddressAttribution[]>;
}

export function GraphCanvas({ graph, version, selected, seeds, filters, attributions, pathNodes, pathEdgeKeys, mode, onSelect, onExpand }: Props) {
  const container = useRef<HTMLDivElement>(null);
  const sigmaRef = useRef<Sigma | null>(null);
  const mk = (): Live => ({ selected, seeds: new Set(seeds), pathNodes: new Set(pathNodes), pathEdges: new Set(pathEdgeKeys), filters, attributions });
  const live = useRef<Live>(mk());
  live.current = mk();
  const handlers = useRef({ onSelect, onExpand });
  handlers.current = { onSelect, onExpand };

  useEffect(() => {
    const el = container.current;
    if (!el) return;
    const css = getComputedStyle(document.documentElement);
    const c = (v: string) => css.getPropertyValue(v).trim();
    const col = { fg: c('--text-primary'), soft: c('--text-secondary'), muted: c('--text-muted'), line: c('--border-strong') };

    const sigma = new Sigma(graph as unknown as Graph, el, {
      renderEdgeLabels: false,
      defaultEdgeType: 'arrow',
      labelColor: { color: col.fg },
      labelFont: 'JetBrains Mono, ui-monospace, monospace',
      labelSize: 11,
      labelRenderedSizeThreshold: 6,
      zIndex: true,
      nodeReducer: (node, data) => {
        const s = live.current;
        const pathActive = s.pathNodes.size > 0;
        const isSeed = s.seeds.has(node);
        // Nodes whose every edge is filtered out disappear, except seeds and the selection.
        if (!isSeed && node !== s.selected && !graph.someEdge(node, (_k, a) => edgePasses(a, s.filters))) return { ...data, hidden: true };
        const expanded = graph.getNodeAttribute(node, 'expanded');
        const ent = displayAttribution(s.attributions[node]);
        let color = isSeed ? col.fg : expanded ? col.soft : col.muted;
        let size = data.size;
        let label = data.label;
        let forceLabel = false;
        let zIndex = 0;
        if (ent) {
          label = nodeLabel(s.attributions[node], data.label);
          if (s.filters.highlightEntities) {
            // Stronger claims are drawn larger and always labelled; weak ones stay modest.
            const strong = ent.status === 'VERIFIED' || ent.status === 'PUBLICLY_REPORTED';
            color = col.fg; size += strong ? 4 : 2; forceLabel = strong; zIndex = 1;
          }
        } else if (s.filters.highlightEntities && Object.keys(s.attributions).length > 0 && !isSeed) {
          color = col.muted;
        }
        if (pathActive) {
          if (s.pathNodes.has(node)) { color = col.fg; forceLabel = true; zIndex = 2; size += 2; } else { color = col.line; label = ''; }
        }
        if (node === s.selected) { color = col.fg; size += 3; forceLabel = true; zIndex = 3; }
        return { ...data, color, size, label, forceLabel, zIndex };
      },
      edgeReducer: (edge, data) => {
        const s = live.current;
        if (!edgePasses(graph.getEdgeAttributes(edge), s.filters)) return { ...data, hidden: true };
        if (s.pathNodes.size > 0) {
          return s.pathEdges.has(edge) ? { ...data, color: col.fg, size: 3 } : { ...data, color: col.line };
        }
        return { ...data, color: col.line };
      },
    });
    sigma.on('clickNode', ({ node }) => handlers.current.onSelect(node));
    sigma.on('clickStage', () => handlers.current.onSelect(null));
    sigma.on('doubleClickNode', (e) => { e.preventSigmaDefault(); handlers.current.onExpand(e.node); });
    sigmaRef.current = sigma;
    return () => { sigma.kill(); sigmaRef.current = null; };
  }, [graph, mode]);

  useEffect(() => { sigmaRef.current?.refresh(); }, [version, selected, pathNodes, pathEdgeKeys, seeds, filters, attributions]);

  return <div ref={container} className="h-full w-full" aria-label="Transaction graph" role="img" />;
}
