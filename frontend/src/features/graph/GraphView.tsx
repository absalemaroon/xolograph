import { Download, Network, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { Badge, Button, ErrorBanner, HashText, Input, PageHeader, Panel, Select } from '@/components/ui';
import { saveTextFile } from '@/lib/desktop';
import { formatUTC } from '@/lib/format';
import { formatUnits, shortAddr } from '@/lib/units';
import { useNav } from '@/state/navStore';
import { useSettings } from '@/state/settingsStore';
import { AttachModal } from '../explorer/AttachModal';
import type { AttachSpec } from '../explorer/AttachModal';
import { EntityFormModal } from '../entities/EntityFormModal';
import { AttributionBadge } from '../entities/AttributionBadge';
import { LabelModal } from '../explorer/LabelModal';
import { GraphCanvas } from './GraphCanvas';
import { graphToJSON, pathEdges, shortestPath } from './graphAlgo';
import { DEFAULT_FILTERS } from './graphFilters';
import type { GraphFilters } from './graphFilters';
import { graphToSnapshot } from './graphSnapshot';
import { MAX_NODES, MAX_SEEDS, useGraph } from './graphStore';
import { SnapshotPanel } from './SnapshotPanel';

export function GraphView() {
  const { graph, version, seed, seeds, readOnly, restoredAt, attributions, attributionError, loading, error, truncated, provenance, buildMulti, restore, expand, clear } = useGraph();
  const { graphSeed, graphSeeds, openExplorer, openEntity } = useNav();
  const mode = useSettings((s) => s.mode);
  const [input, setInput] = useState(seeds.join(', '));
  const [selected, setSelected] = useState<string | null>(null);
  const [pathA, setPathA] = useState<string | null>(null);
  const [pathB, setPathB] = useState<string | null>(null);
  const [directed, setDirected] = useState(false);
  const [attach, setAttach] = useState<AttachSpec | null>(null);
  const [labelFor, setLabelFor] = useState<string | null>(null);
  const [entityFor, setEntityFor] = useState<string | null>(null);
  const [filters, setFilters] = useState<GraphFilters>(DEFAULT_FILTERS);
  const [notice, setNotice] = useState<string | null>(null);

  const parse = (v: string): string[] => v.split(/[\s,;]+/).filter(Boolean);
  useEffect(() => {
    // Seeds handed over from Explorer / Entities are consumed once so returning to this page never rebuilds over later work.
    const handoff = graphSeeds ?? (graphSeed ? [graphSeed] : null);
    if (!handoff) return;
    useNav.setState({ graphSeed: null, graphSeeds: null });
    setInput(handoff.join(', '));
    setSelected(null); setPathA(null); setPathB(null);
    void buildMulti(handoff);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [graphSeed, graphSeeds]);

  const path = useMemo(() => (pathA && pathB ? shortestPath(graph, pathA, pathB, directed) : null), [graph, version, pathA, pathB, directed]); // eslint-disable-line react-hooks/exhaustive-deps
  const pEdges = useMemo(() => (path ? pathEdges(graph, path) : []), [graph, path]);

  const submit = (e: FormEvent) => { e.preventDefault(); setSelected(null); setPathA(null); setPathB(null); void buildMulti(parse(input)); };
  const onClear = () => { clear(); setSelected(null); setPathA(null); setPathB(null); };

  const stats = useMemo(() => {
    if (!selected || !graph.hasNode(selected)) return null;
    let outWei = 0n, inWei = 0n, outTx = 0, inTx = 0;
    graph.forEachOutEdge(selected, (_k, a) => { outWei += BigInt(a.nativeWei); outTx += a.count; });
    graph.forEachInEdge(selected, (_k, a) => { inWei += BigInt(a.nativeWei); inTx += a.count; });
    return { outWei: outWei.toString(), inWei: inWei.toString(), outTx, inTx, expanded: graph.getNodeAttribute(selected, 'expanded') };
  }, [graph, selected, version]); // eslint-disable-line react-hooks/exhaustive-deps

  const snapshot = () => graphToJSON(graph, { seed, seeds, exportedAt: new Date().toISOString(), maxNodes: MAX_NODES, truncated, coverage: 'Built from the most recent page of ETH and token activity per expanded node; not complete history.', provenance });
  const savePayload = () => ({
    seeds, nodeCount: graph.order, edgeCount: graph.size,
    data: graphToSnapshot(graph, { seeds, truncated, provenance, savedAt: new Date().toISOString() }),
  });
  const exportJson = () => {
    setNotice(null);
    saveTextFile(`xolograph-graph-${seed ? seed.slice(0, 10) : 'empty'}.json`, JSON.stringify(snapshot(), null, 2), 'application/json')
      .then((r) => { if (r.saved) setNotice(r.path ? `Saved to ${r.path}` : 'Download started'); })
      .catch((e: unknown) => setNotice(`Export failed: ${e instanceof Error ? e.message : String(e)}`));
  };

  return (
    <>
      <PageHeader title="Network Graph" subtitle={`Directed value-flow graph from real activity. Click a node to inspect, double-click to expand. Up to ${MAX_SEEDS} seeds, max ${MAX_NODES} nodes. Entity names come from your own records, not from the chain.`} />
      <form onSubmit={submit} className="mb-4 flex flex-wrap gap-3">
        <div className="min-w-72 flex-1"><Input aria-label="Seed addresses" value={input} onChange={(e) => setInput(e.target.value)} placeholder="Seed address 0x… (several allowed, separated by commas)" spellCheck={false} /></div>
        <Button variant="primary" icon={<Network size={14} />} loading={loading} disabled={!input.trim()} onClick={submit as unknown as () => void}>Build graph</Button>
        <Button icon={<Download size={14} />} disabled={graph.order === 0} onClick={exportJson}>Export JSON</Button>
        <Button disabled={graph.order === 0} onClick={() => setAttach({ kind: 'GRAPH', refType: 'graph', refId: seed ?? '', defaultTitle: `Graph from ${seed ? shortAddr(seed) : ''}`, snapshot: snapshot() })}>Attach graph</Button>
        <Button variant="ghost" icon={<Trash2 size={14} />} disabled={graph.order === 0} onClick={onClear}>Clear</Button>
        <button type="submit" className="hidden" />
      </form>
      {notice && <p className="mb-4 break-all text-[11px]">{notice}</p>}
      {readOnly && <div className="mb-4 border border-line bg-raised px-4 py-2 text-xs">◐ Restored snapshot{restoredAt ? ` saved ${formatUTC(restoredAt)}` : ''}. Values are from that moment and the graph cannot be expanded. Entity names are current. Build a new graph to refresh the data.</div>}
      {attributionError && <div className="mb-4"><ErrorBanner message={`Entity names unavailable: ${attributionError}`} /></div>}
      {error && <div className="mb-4"><ErrorBanner message={error} /></div>}
      {truncated && <div className="mb-4"><ErrorBanner message={`Node limit (${MAX_NODES}) reached; some counterparties were not added.`} /></div>}

      <div className="grid gap-4 lg:grid-cols-[1fr_20rem]">
        <div className="relative h-[34rem] border border-line bg-base">
          {graph.order === 0 ? (
            <div className="flex h-full items-center justify-center p-6 text-center text-xs text-fg-muted">Enter a seed address and build the graph. Nothing is drawn until real data is retrieved.</div>
          ) : (
            <GraphCanvas graph={graph} version={version} selected={selected} seeds={seeds} filters={filters} attributions={attributions} pathNodes={path ?? []} pathEdgeKeys={pEdges} mode={mode} onSelect={setSelected} onExpand={(id) => void expand(id)} />
          )}
          <div className="pointer-events-none absolute bottom-2 left-3 text-[10px] uppercase tracking-wider text-fg-muted">{graph.order} nodes · {graph.size} edges{loading ? ' · loading…' : ''}</div>
        </div>

        <div className="space-y-4">
          <Panel title="Selected node">
            {!selected || !stats ? <p className="text-xs text-fg-muted">Select a node.</p> : (
              <div className="space-y-3 text-xs">
                <HashText value={selected} head={12} tail={8} />
                <div className="flex gap-2">{seeds.includes(selected) && <Badge tone="solid">Seed</Badge>}{stats.expanded ? <Badge>Expanded</Badge> : <Badge>Not expanded</Badge>}</div>
                <dl className="space-y-1.5">
                  <div className="flex justify-between"><dt className="text-fg-muted">Sent</dt><dd>{stats.outTx} transfers · {formatUnits(stats.outWei, 18, 4)} ETH</dd></div>
                  <div className="flex justify-between"><dt className="text-fg-muted">Received</dt><dd>{stats.inTx} transfers · {formatUnits(stats.inWei, 18, 4)} ETH</dd></div>
                </dl>
                <p className="text-[10px] text-fg-muted">Totals cover loaded transfers only.</p>
                <div className="border-t border-line pt-3">
                  <div className="mb-1.5 text-[10px] uppercase tracking-wider text-fg-muted">Entity records</div>
                  {(attributions[selected] ?? []).length === 0 ? <p className="text-fg-muted">No entity record claims this address.</p> : (
                    <ul className="space-y-1.5">
                      {(attributions[selected] ?? []).map((a) => (
                        <li key={a.entityId} className="flex flex-wrap items-center gap-2">
                          <button type="button" className="font-medium underline-offset-2 hover:underline" onClick={() => openEntity(a.entityId)}>{a.name}</button>
                          <AttributionBadge status={a.status} />
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button size="sm" variant="primary" disabled={stats.expanded || loading || readOnly} onClick={() => void expand(selected)}>Expand</Button>
                  <Button size="sm" onClick={() => openExplorer(selected)}>Explorer</Button>
                  <Button size="sm" onClick={() => setPathA(selected)}>Path from</Button>
                  <Button size="sm" onClick={() => setPathB(selected)}>Path to</Button>
                  <Button size="sm" onClick={() => setEntityFor(selected)}>New entity</Button>
                  <Button size="sm" onClick={() => setLabelFor(selected)}>Case label</Button>
                  <Button size="sm" onClick={() => setAttach({ kind: 'ADDRESS', refType: 'address', refId: selected, defaultTitle: `Address ${shortAddr(selected)}`, snapshot: { address: selected, loadedStats: stats, seed, provenance } })}>Attach</Button>
                </div>
              </div>
            )}
          </Panel>

          <Panel title="Filters">
            <div className="space-y-3 text-xs">
              <label className="flex items-center justify-between gap-3">Minimum transfers per link
                <Input aria-label="Minimum transfers" type="number" min={1} max={1000} className="!w-20" value={filters.minTransfers}
                  onChange={(e) => setFilters((f) => ({ ...f, minTransfers: Math.max(1, Math.min(1000, Number(e.target.value) || 1)) }))} /></label>
              <label className="flex items-center justify-between gap-3">Asset
                <Select aria-label="Asset filter" className="!w-36" value={filters.asset} onChange={(e) => setFilters((f) => ({ ...f, asset: e.target.value as GraphFilters['asset'] }))}>
                  <option value="all">All</option><option value="native">ETH only</option><option value="token">Tokens only</option>
                </Select></label>
              <label className="flex items-center gap-2 text-[11px]"><input type="checkbox" checked={filters.highlightEntities} onChange={(e) => setFilters((f) => ({ ...f, highlightEntities: e.target.checked }))} className="accent-[var(--text-primary)]" />Highlight addresses with entity records</label>
              <p className="text-[10px] text-fg-muted">Larger, labelled nodes have an entity record; verified and publicly reported claims are drawn largest. Rejected claims never label a node. Filters hide data on screen only; counts in the inspector still cover all loaded transfers.</p>
            </div>
          </Panel>

          <SnapshotPanel canSave={graph.order > 0} getPayload={savePayload} onRestore={(d) => { setSelected(null); setPathA(null); setPathB(null); restore(d); }} />

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
      {entityFor && <EntityFormModal defaultAddress={entityFor} onClose={() => setEntityFor(null)} onSaved={(e) => { void useGraph.getState().loadAttributions(); openEntity(e.id); }} />}
    </>
  );
}
