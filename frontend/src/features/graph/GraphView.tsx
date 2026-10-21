import { Download, Network, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { Badge, Button, ErrorBanner, HashText, Input, PageHeader, Panel } from '@/components/ui';
import { downloadText } from '@/lib/download';
import { formatUnits, shortAddr } from '@/lib/units';
import { useNav } from '@/state/navStore';
import { useSettings } from '@/state/settingsStore';
import { AttachModal } from '../explorer/AttachModal';
import type { AttachSpec } from '../explorer/AttachModal';
import { LabelModal } from '../explorer/LabelModal';
import { GraphCanvas } from './GraphCanvas';
import { graphToJSON, pathEdges, shortestPath } from './graphAlgo';
import { MAX_NODES, useGraph } from './graphStore';

export function GraphView() {
  const { graph, version, seed, loading, error, truncated, provenance, build, expand, clear } = useGraph();
  const { graphSeed, openExplorer } = useNav();
  const mode = useSettings((s) => s.mode);
  const [input, setInput] = useState(seed ?? '');
  const [selected, setSelected] = useState<string | null>(null);
  const [pathA, setPathA] = useState<string | null>(null);
  const [pathB, setPathB] = useState<string | null>(null);
  const [directed, setDirected] = useState(false);
  const [attach, setAttach] = useState<AttachSpec | null>(null);
  const [labelFor, setLabelFor] = useState<string | null>(null);

  useEffect(() => {
    if (graphSeed && graphSeed !== seed) { setInput(graphSeed); void build(graphSeed); }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [graphSeed]);

  const path = useMemo(() => (pathA && pathB ? shortestPath(graph, pathA, pathB, directed) : null), [graph, version, pathA, pathB, directed]); // eslint-disable-line react-hooks/exhaustive-deps
  const pEdges = useMemo(() => (path ? pathEdges(graph, path) : []), [graph, path]);

  const submit = (e: FormEvent) => { e.preventDefault(); setSelected(null); setPathA(null); setPathB(null); void build(input); };
  const onClear = () => { clear(); setSelected(null); setPathA(null); setPathB(null); };

  const stats = useMemo(() => {
    if (!selected || !graph.hasNode(selected)) return null;
    let outWei = 0n, inWei = 0n, outTx = 0, inTx = 0;
    graph.forEachOutEdge(selected, (_k, a) => { outWei += BigInt(a.nativeWei); outTx += a.count; });
    graph.forEachInEdge(selected, (_k, a) => { inWei += BigInt(a.nativeWei); inTx += a.count; });
    return { outWei: outWei.toString(), inWei: inWei.toString(), outTx, inTx, expanded: graph.getNodeAttribute(selected, 'expanded') };
  }, [graph, selected, version]); // eslint-disable-line react-hooks/exhaustive-deps

  const snapshot = () => graphToJSON(graph, { seed, exportedAt: new Date().toISOString(), maxNodes: MAX_NODES, truncated, coverage: 'Built from the most recent page of ETH and token activity per expanded node; not complete history.', provenance });
  const exportJson = () => downloadText(`xolograph-graph-${seed ? seed.slice(0, 10) : 'empty'}.json`, JSON.stringify(snapshot(), null, 2), 'application/json');

  return (
    <>
      <PageHeader title="Network Graph" subtitle={`Directed value-flow graph from real activity. Click a node to inspect, double-click to expand. Max ${MAX_NODES} nodes.`} />
      <form onSubmit={submit} className="mb-4 flex flex-wrap gap-3">
        <div className="min-w-72 flex-1"><Input aria-label="Seed address" value={input} onChange={(e) => setInput(e.target.value)} placeholder="Seed address 0x…" spellCheck={false} /></div>
        <Button variant="primary" icon={<Network size={14} />} loading={loading} disabled={!input.trim()} onClick={submit as unknown as () => void}>Build graph</Button>
        <Button icon={<Download size={14} />} disabled={graph.order === 0} onClick={exportJson}>Export JSON</Button>
        <Button disabled={graph.order === 0} onClick={() => setAttach({ kind: 'GRAPH', refType: 'graph', refId: seed ?? '', defaultTitle: `Graph from ${seed ? shortAddr(seed) : ''}`, snapshot: snapshot() })}>Attach graph</Button>
        <Button variant="ghost" icon={<Trash2 size={14} />} disabled={graph.order === 0} onClick={onClear}>Clear</Button>
        <button type="submit" className="hidden" />
      </form>
      {error && <div className="mb-4"><ErrorBanner message={error} /></div>}
      {truncated && <div className="mb-4"><ErrorBanner message={`Node limit (${MAX_NODES}) reached; some counterparties were not added.`} /></div>}

      <div className="grid gap-4 lg:grid-cols-[1fr_20rem]">
        <div className="relative h-[34rem] border border-line bg-base">
          {graph.order === 0 ? (
            <div className="flex h-full items-center justify-center p-6 text-center text-xs text-fg-muted">Enter a seed address and build the graph. Nothing is drawn until real data is retrieved.</div>
          ) : (
            <GraphCanvas graph={graph} version={version} selected={selected} seed={seed} pathNodes={path ?? []} pathEdgeKeys={pEdges} mode={mode} onSelect={setSelected} onExpand={(id) => void expand(id)} />
          )}
          <div className="pointer-events-none absolute bottom-2 left-3 text-[10px] uppercase tracking-wider text-fg-muted">{graph.order} nodes · {graph.size} edges{loading ? ' · loading…' : ''}</div>
        </div>

        <div className="space-y-4">
          <Panel title="Selected node">
            {!selected || !stats ? <p className="text-xs text-fg-muted">Select a node.</p> : (
              <div className="space-y-3 text-xs">
                <HashText value={selected} head={12} tail={8} />
                <div className="flex gap-2">{selected === seed && <Badge tone="solid">Seed</Badge>}{stats.expanded ? <Badge>Expanded</Badge> : <Badge>Not expanded</Badge>}</div>
                <dl className="space-y-1.5">
                  <div className="flex justify-between"><dt className="text-fg-muted">Sent</dt><dd>{stats.outTx} transfers · {formatUnits(stats.outWei, 18, 4)} ETH</dd></div>
                  <div className="flex justify-between"><dt className="text-fg-muted">Received</dt><dd>{stats.inTx} transfers · {formatUnits(stats.inWei, 18, 4)} ETH</dd></div>
                </dl>
                <p className="text-[10px] text-fg-muted">Totals cover loaded transfers only.</p>
                <div className="flex flex-wrap gap-2">
                  <Button size="sm" variant="primary" disabled={stats.expanded || loading} onClick={() => void expand(selected)}>Expand</Button>
                  <Button size="sm" onClick={() => openExplorer(selected)}>Explorer</Button>
                  <Button size="sm" onClick={() => setPathA(selected)}>Path from</Button>
                  <Button size="sm" onClick={() => setPathB(selected)}>Path to</Button>
                  <Button size="sm" onClick={() => setLabelFor(selected)}>Label</Button>
                  <Button size="sm" onClick={() => setAttach({ kind: 'ADDRESS', refType: 'address', refId: selected, defaultTitle: `Address ${shortAddr(selected)}`, snapshot: { address: selected, loadedStats: stats, seed, provenance } })}>Attach</Button>
                </div>
              </div>
            )}
          </Panel>

          <Panel title="Path" actions={(pathA || pathB) ? <Button size="sm" variant="ghost" onClick={() => { setPathA(null); setPathB(null); }}>Reset</Button> : undefined}>
            <div className="space-y-2 text-xs">
              <div className="flex justify-between"><span className="text-fg-muted">From</span><span className="font-mono">{pathA ? shortAddr(pathA) : '—'}</span></div>
              <div className="flex justify-between"><span className="text-fg-muted">To</span><span className="font-mono">{pathB ? shortAddr(pathB) : '—'}</span></div>
              <label className="flex items-center gap-2 text-[11px]"><input type="checkbox" checked={directed} onChange={(e) => setDirected(e.target.checked)} className="accent-[var(--text-primary)]" />Follow fund direction only</label>
              {pathA && pathB && (path ? (
                <>
                  <p>● {path.length - 1} hop{path.length - 1 === 1 ? '' : 's'}: <span className="break-all font-mono text-[11px]">{path.map(shortAddr).join(' → ')}</span></p>
                  <Button size="sm" onClick={() => setAttach({ kind: 'PATH', refType: 'path', refId: `${pathA}->${pathB}`, defaultTitle: `Path ${shortAddr(pathA)} → ${shortAddr(pathB)}`, snapshot: { from: pathA, to: pathB, directed, hops: path, edges: pEdges.map((k) => ({ key: k, ...graph.getEdgeAttributes(k) })), coverage: 'Shortest path within the loaded graph only; other paths may exist.', provenance } })}>Attach path</Button>
                </>
              ) : <p>No path within the loaded graph. Expand more nodes to search further.</p>)}
            </div>
          </Panel>
        </div>
      </div>
      {attach && <AttachModal spec={attach} onClose={() => setAttach(null)} />}
      {labelFor && <LabelModal defaultAddress={labelFor} onClose={() => setLabelFor(null)} />}
    </>
  );
}
