import { Link2, Maximize2, Pencil, Plus, Trash2 } from 'lucide-react';
import { useMemo, useRef, useState } from 'react';
import type { PointerEvent as RPointerEvent } from 'react';
import { api } from '@/api/endpoints';
import type { AddressAttribution, BoardKind, BoardLink, BoardNode } from '@/api/types';
import { Badge, Button, EmptyState, ErrorBanner, HashText, Panel, Select, Spinner } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';
import { useNav } from '@/state/navStore';
import { AttributionBadge } from '../entities/AttributionBadge';
import { ConfirmModal } from '../entities/ConfirmModal';
import { LinkModal } from './LinkModal';
import { NodeModal } from './NodeModal';
import { NODE_H, NODE_W, linkEndpoints, pairKey } from './geometry';
import type { Pt } from './geometry';

interface Drag { kind: 'node' | 'pan'; id?: string; sx: number; sy: number; ox: number; oy: number; moved: boolean }
type Sel = { type: 'node' | 'link'; id: string } | null;
const KIND_TAG: Record<BoardKind, string> = { ENTITY: 'ENTITY', WALLET: 'WALLET', EVIDENCE: 'EVIDENCE', NOTE: 'NOTE', EVENT: 'EVENT' };
const clip = (s: string, n: number) => (s.length > n ? `${s.slice(0, n - 1)}…` : s);

export function BoardTab({ caseId, onChanged }: { caseId: string; onChanged: () => void }) {
  const { openEntity, openExplorer } = useNav();
  const { data, error, loading, reload } = useAsync(() => api.board.get(caseId), [caseId]);
  const [local, setLocal] = useState<Record<string, Pt>>({});
  const [view, setView] = useState({ x: 450, y: 260, k: 1 });
  const [sel, setSel] = useState<Sel>(null);
  const [linkFrom, setLinkFrom] = useState<string | null>(null);
  const [adding, setAdding] = useState<BoardKind | null>(null);
  const [editNode, setEditNode] = useState<BoardNode | null>(null);
  const [draft, setDraft] = useState<{ source: BoardNode; target: BoardNode } | null>(null);
  const [editLink, setEditLink] = useState<BoardLink | null>(null);
  const [confirm, setConfirm] = useState<{ title: string; message: string; label: string; run: () => Promise<void> } | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const svg = useRef<SVGSVGElement>(null);
  const drag = useRef<Drag | null>(null);

  const nodes = data?.nodes ?? [];
  const links = data?.links ?? [];
  const byId = useMemo(() => new Map(nodes.map((n) => [n.id, n])), [nodes]);
  const posOf = (n: BoardNode): Pt => local[n.id] ?? { x: n.x, y: n.y };
  const changed = () => { setLocal({}); reload(); onChanged(); };

  const wallets = nodes.filter((n) => n.kind === 'WALLET' && n.network === 'ethereum-mainnet').map((n) => n.refId);
  const attrib = useAsync(() => (wallets.length ? api.entities.lookup(wallets) : Promise.resolve({} as Record<string, AddressAttribution[]>)), [wallets.join(',')]);

  // parallel links between the same two nodes are fanned out
  const offsets = useMemo(() => {
    const groups = new Map<string, string[]>();
    for (const l of links) { const k = pairKey(l.sourceId, l.targetId); groups.set(k, [...(groups.get(k) ?? []), l.id]); }
    const m = new Map<string, number>();
    groups.forEach((ids) => ids.forEach((id, i) => m.set(id, (i - (ids.length - 1) / 2) * 18)));
    return m;
  }, [links]);

  const onNodeDown = (e: RPointerEvent, n: BoardNode) => {
    e.stopPropagation();
    svg.current?.setPointerCapture(e.pointerId);
    const p = posOf(n);
    drag.current = { kind: 'node', id: n.id, sx: e.clientX, sy: e.clientY, ox: p.x, oy: p.y, moved: false };
  };
  const onBgDown = (e: RPointerEvent) => {
    svg.current?.setPointerCapture(e.pointerId);
    drag.current = { kind: 'pan', sx: e.clientX, sy: e.clientY, ox: view.x, oy: view.y, moved: false };
  };
  const onMove = (e: RPointerEvent) => {
    const d = drag.current;
    if (!d) return;
    const dx = e.clientX - d.sx, dy = e.clientY - d.sy;
    if (Math.abs(dx) + Math.abs(dy) > 3) d.moved = true;
    if (!d.moved) return;
    if (d.kind === 'node' && d.id) { const id = d.id; setLocal((l) => ({ ...l, [id]: { x: d.ox + dx / view.k, y: d.oy + dy / view.k } })); }
    else setView((v) => ({ ...v, x: d.ox + dx, y: d.oy + dy }));
  };
  const onUp = (e: RPointerEvent) => {
    const d = drag.current;
    drag.current = null;
    if (!d) return;
    if (d.kind === 'pan') { if (!d.moved) { setSel(null); setLinkFrom(null); } return; }
    const n = d.id ? byId.get(d.id) : undefined;
    if (!n) return;
    if (d.moved) {
      const x = d.ox + (e.clientX - d.sx) / view.k, y = d.oy + (e.clientY - d.sy) / view.k;
      setLocal((l) => ({ ...l, [n.id]: { x, y } }));
      api.board.move(caseId, [{ id: n.id, x, y }]).catch((err: unknown) => { setActionError(err instanceof Error ? err.message : String(err)); reload(); });
      return;
    }
    if (linkFrom && linkFrom !== n.id) {
      const source = byId.get(linkFrom);
      if (source) setDraft({ source, target: n });
      setLinkFrom(null);
      return;
    }
    setSel({ type: 'node', id: n.id });
  };
  const onWheel = (e: React.WheelEvent) => {
    const r = svg.current?.getBoundingClientRect();
    if (!r) return;
    const mx = e.clientX - r.left, my = e.clientY - r.top;
    setView((v) => {
      const k = Math.min(2.5, Math.max(0.3, v.k * (e.deltaY < 0 ? 1.1 : 1 / 1.1)));
      const wx = (mx - v.x) / v.k, wy = (my - v.y) / v.k;
      return { k, x: mx - wx * k, y: my - wy * k };
    });
  };
  const fit = () => {
    const r = svg.current?.getBoundingClientRect();
    if (!r || nodes.length === 0) { setView({ x: (r?.width ?? 900) / 2, y: (r?.height ?? 520) / 2, k: 1 }); return; }
    const ps = nodes.map(posOf);
    const minX = Math.min(...ps.map((p) => p.x)) - NODE_W / 2, maxX = Math.max(...ps.map((p) => p.x)) + NODE_W / 2;
    const minY = Math.min(...ps.map((p) => p.y)) - NODE_H / 2, maxY = Math.max(...ps.map((p) => p.y)) + NODE_H / 2;
    const k = Math.min(1.5, Math.max(0.3, Math.min((r.width - 60) / (maxX - minX), (r.height - 60) / (maxY - minY))));
    setView({ k, x: r.width / 2 - ((minX + maxX) / 2) * k, y: r.height / 2 - ((minY + maxY) / 2) * k });
  };
  const nextSpot = (): { x: number; y: number } => ({ x: (nodes.length % 4) * 220 - 330, y: Math.floor(nodes.length / 4) * 110 - 100 });

  if (loading && !data) return <Spinner />;
  if (error || !data) return <ErrorBanner message={error ?? 'Board unavailable'} onRetry={reload} />;

  const facts = links.filter((l) => l.claim === 'FACT').length;
  const selNode = sel?.type === 'node' ? byId.get(sel.id) : undefined;
  const selLink = sel?.type === 'link' ? links.find((l) => l.id === sel.id) : undefined;

  return (
    <div className="space-y-4">
      {actionError && <ErrorBanner message={actionError} />}
      <div className="flex flex-wrap items-center gap-2">
        {(['ENTITY', 'WALLET', 'EVIDENCE', 'NOTE', 'EVENT'] as BoardKind[]).map((k) => (
          <Button key={k} size="sm" icon={<Plus size={12} />} onClick={() => setAdding(k)}>{k[0] + k.slice(1).toLowerCase()}</Button>
        ))}
        <Button size="sm" icon={<Maximize2 size={12} />} onClick={fit}>Fit</Button>
        <span className="ml-auto text-[11px] text-fg-muted">{nodes.length} items · {facts} facts · {links.length - facts} hypotheses</span>
      </div>
      {linkFrom && <div className="border border-line bg-raised px-4 py-2 text-xs">Choose the item to link <b>to</b>, or click the empty canvas to cancel.</div>}

      <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
        <div className="relative h-[560px] border border-line bg-base">
          {nodes.length === 0 && <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center p-6"><EmptyState title="Empty board" hint="Add entities, wallets, evidence, notes and events, then link them. Mark each link as a fact (with evidence or a source) or a hypothesis." /></div>}
          <svg ref={svg} role="application" aria-label="Case relationship board" className="h-full w-full touch-none select-none"
            onPointerDown={onBgDown} onPointerMove={onMove} onPointerUp={onUp} onWheel={onWheel}>
            <defs>
              <marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill="var(--text-primary)" /></marker>
            </defs>
            <g transform={`translate(${view.x} ${view.y}) scale(${view.k})`}>
              {links.map((l) => {
                const a = byId.get(l.sourceId), b = byId.get(l.targetId);
                if (!a || !b) return null;
                const g = linkEndpoints(posOf(a), posOf(b), offsets.get(l.id) ?? 0);
                const fact = l.claim === 'FACT';
                const on = sel?.type === 'link' && sel.id === l.id;
                return (
                  <g key={l.id} onPointerDown={(e) => { e.stopPropagation(); setSel({ type: 'link', id: l.id }); setLinkFrom(null); }} className="cursor-pointer">
                    <line x1={g.x1} y1={g.y1} x2={g.x2} y2={g.y2} stroke="transparent" strokeWidth={14} />
                    <line x1={g.x1} y1={g.y1} x2={g.x2} y2={g.y2} stroke="var(--text-primary)" strokeWidth={fact ? 2.2 : 1.2} strokeDasharray={fact ? undefined : '6 5'} opacity={on ? 1 : 0.7} markerEnd="url(#arrow)" />
                    <text x={g.mx} y={g.my - 4} textAnchor="middle" fontSize={10} fill="var(--text-primary)" stroke="var(--bg-primary)" strokeWidth={4} paintOrder="stroke" className="font-mono">
                      {clip(l.relation, 26)}{fact ? '' : ' (?)'}
                    </text>
                  </g>
                );
              })}
              {nodes.map((n) => {
                const p = posOf(n);
                const on = sel?.type === 'node' && sel.id === n.id;
                const src = linkFrom === n.id;
                const wa = n.kind === 'WALLET' ? attrib.data?.[n.refId]?.find((x) => x.status !== 'REJECTED') : undefined;
                const name = n.kind === 'ENTITY' ? (n.liveName || n.title) : n.title;
                return (
                  <g key={n.id} transform={`translate(${p.x - NODE_W / 2} ${p.y - NODE_H / 2})`} onPointerDown={(e) => onNodeDown(e, n)} className="cursor-grab active:cursor-grabbing">
                    <rect width={NODE_W} height={NODE_H} fill="var(--bg-secondary)" stroke="var(--text-primary)" strokeWidth={on || src ? 3 : 1} strokeDasharray={n.kind === 'NOTE' || n.kind === 'EVENT' ? '4 3' : undefined} />
                    <text x={8} y={14} fontSize={8} letterSpacing={1.2} fill="var(--text-muted)" className="font-mono">{KIND_TAG[n.kind]}{n.kind === 'EVIDENCE' && n.evidenceState === 'FAILED' ? ' · ⚠ FAILED VERIFY' : ''}</text>
                    <text x={8} y={31} fontSize={12} fontWeight={600} fill="var(--text-primary)">{clip(name, 24)}</text>
                    <text x={8} y={44} fontSize={9} fill="var(--text-secondary)" className="font-mono">
                      {n.kind === 'WALLET' ? (wa ? `◐ ${clip(wa.name, 20)}` : clip(n.refId, 18)) : n.kind === 'EVIDENCE' ? `${n.evidenceHash?.slice(0, 16) ?? ''}…` : n.kind === 'ENTITY' ? (n.entityStatus ?? '').toLowerCase().replace(/_/g, ' ') : clip(n.body, 28)}
                    </text>
                  </g>
                );
              })}
            </g>
          </svg>
          <div className="pointer-events-none absolute bottom-2 left-3 text-[10px] text-fg-muted">Drag items · scroll to zoom · drag canvas to pan · dashed line (?) = hypothesis, solid = fact</div>
        </div>

        <div className="space-y-4">
          {!selNode && !selLink && <Panel title="Inspector"><p className="text-xs text-fg-muted">Select an item or link. Links are always drawn by you; chain activity never creates one.</p></Panel>}
          {selNode && (
            <Panel title={KIND_TAG[selNode.kind]} actions={<div className="flex gap-1">
              <Button size="sm" variant="ghost" icon={<Pencil size={12} />} aria-label="Edit item" onClick={() => setEditNode(selNode)} />
              <Button size="sm" variant="ghost" icon={<Trash2 size={12} />} aria-label="Remove from board" onClick={() => setConfirm({ title: 'Remove from board', label: 'Remove', message: `Remove “${selNode.title}” and its links from the board? The underlying record is not deleted.`, run: async () => { await api.board.removeNode(selNode.id); setSel(null); changed(); } })} />
            </div>}>
              <div className="space-y-3 text-xs">
                <div className="text-sm font-semibold">{selNode.kind === 'ENTITY' ? (selNode.liveName || selNode.title) : selNode.title}</div>
                {selNode.body && <p className="whitespace-pre-wrap text-fg-soft">{selNode.body}</p>}
                {selNode.kind === 'ENTITY' && <div className="flex items-center gap-2"><AttributionBadge status={selNode.entityStatus || 'UNKNOWN'} /><Button size="sm" onClick={() => openEntity(selNode.refId)}>Open entity</Button></div>}
                {selNode.kind === 'WALLET' && (
                  <div className="space-y-2"><HashText value={selNode.refId} head={10} tail={8} />
                    {(attrib.data?.[selNode.refId] ?? []).map((a) => <div key={a.entityId} className="flex flex-wrap items-center gap-2"><button type="button" className="underline-offset-2 hover:underline" onClick={() => openEntity(a.entityId)}>{a.name}</button><AttributionBadge status={a.status} /></div>)}
                    {selNode.network === 'ethereum-mainnet' && <Button size="sm" onClick={() => openExplorer(selNode.refId)}>Open in Explorer</Button>}</div>
                )}
                {selNode.kind === 'EVIDENCE' && <div className="space-y-1"><HashText value={selNode.evidenceHash ?? ''} head={12} tail={8} /><Badge tone={selNode.evidenceState === 'FAILED' ? 'alert' : 'neutral'}>{selNode.evidenceState}</Badge></div>}
                <div className="flex gap-2 border-t border-line pt-3">
                  <Button size="sm" variant={linkFrom === selNode.id ? 'primary' : 'secondary'} icon={<Link2 size={12} />} onClick={() => setLinkFrom(linkFrom === selNode.id ? null : selNode.id)}>{linkFrom === selNode.id ? 'Pick target…' : 'Link to…'}</Button>
                </div>
              </div>
            </Panel>
          )}
          {selLink && <LinkInspector link={selLink} byId={byId} onEdit={() => setEditLink(selLink)} onChanged={changed} setError={setActionError}
            onDelete={() => setConfirm({ title: 'Delete link', label: 'Delete link', message: `Delete the “${selLink.relation}” link? Evidence and items stay.`, run: async () => { await api.board.removeLink(selLink.id); setSel(null); changed(); } })} />}
        </div>
      </div>

      {adding && <NodeModal caseId={caseId} at={nextSpot()} initialKind={adding} onClose={() => setAdding(null)} onSaved={changed} />}
      {editNode && <NodeModal caseId={caseId} at={{ x: 0, y: 0 }} existing={editNode} onClose={() => setEditNode(null)} onSaved={changed} />}
      {draft && <LinkModal caseId={caseId} source={draft.source} target={draft.target} onClose={() => setDraft(null)} onSaved={changed} />}
      {editLink && byId.get(editLink.sourceId) && byId.get(editLink.targetId) && <LinkModal caseId={caseId} source={byId.get(editLink.sourceId)!} target={byId.get(editLink.targetId)!} existing={editLink} onClose={() => setEditLink(null)} onSaved={changed} />}
      {confirm && <ConfirmModal title={confirm.title} message={confirm.message} confirmLabel={confirm.label} onConfirm={confirm.run} onClose={() => setConfirm(null)} />}
    </div>
  );
}

function LinkInspector({ link, byId, onEdit, onDelete, onChanged, setError }: {
  link: BoardLink; byId: Map<string, BoardNode>; onEdit: () => void; onDelete: () => void; onChanged: () => void; setError: (m: string | null) => void;
}) {
  const [picking, setPicking] = useState(false);
  const [evId, setEvId] = useState('');
  const evidence = useAsync(() => (picking ? api.evidence.list(link.caseId) : Promise.resolve([])), [picking, link.caseId]);
  const a = byId.get(link.sourceId), b = byId.get(link.targetId);
  const attachable = (evidence.data ?? []).filter((x) => !link.evidence.some((e) => e.evidenceId === x.id));
  const run = async (fn: () => Promise<unknown>) => { setError(null); try { await fn(); onChanged(); } catch (e) { setError(e instanceof Error ? e.message : String(e)); } };
  return (
    <Panel title={link.claim === 'FACT' ? 'Fact' : 'Hypothesis'} actions={<div className="flex gap-1">
      <Button size="sm" variant="ghost" icon={<Pencil size={12} />} aria-label="Edit link" onClick={onEdit} />
      <Button size="sm" variant="ghost" icon={<Trash2 size={12} />} aria-label="Delete link" onClick={onDelete} />
    </div>}>
      <div className="space-y-3 text-xs">
        <div><span className="font-medium">{a?.liveName || a?.title}</span> — <i>{link.relation}</i> → <span className="font-medium">{b?.liveName || b?.title}</span></div>
        <div className="flex flex-wrap gap-2"><Badge tone={link.claim === 'FACT' ? 'solid' : 'outline'}>{link.claim === 'FACT' ? '● Fact' : '? Hypothesis'}</Badge><Badge>Confidence {link.confidence}</Badge></div>
        {link.sourceRef && <p className="break-words"><span className="text-fg-muted">Source: </span>{link.sourceRef}</p>}
        {link.note && <p className="whitespace-pre-wrap text-fg-soft">{link.note}</p>}
        <div className="border-t border-line pt-3">
          <div className="mb-1.5 text-[10px] uppercase tracking-wider text-fg-muted">Supporting evidence</div>
          {link.evidence.length === 0 ? <p className="text-fg-muted">None attached.</p> : (
            <ul className="space-y-2">{link.evidence.map((e) => (
              <li key={e.id} className="flex items-start justify-between gap-2"><div className="min-w-0"><div className="truncate">{e.name}{e.status === 'FAILED' && ' ⚠ failed verification'}</div><HashText value={e.sha256} head={8} tail={6} /></div>
                <Button size="sm" variant="ghost" icon={<Trash2 size={12} />} aria-label="Detach evidence" onClick={() => void run(() => api.board.removeLinkEvidence(e.id))} /></li>
            ))}</ul>
          )}
          {picking ? (
            <div className="mt-2 flex gap-2">
              <Select aria-label="Evidence to attach" className="min-w-0 flex-1" value={evId} onChange={(e) => setEvId(e.target.value)}><option value="">Select…</option>{attachable.map((x) => <option key={x.id} value={x.id}>{x.name}</option>)}</Select>
              <Button size="sm" variant="primary" disabled={!evId} onClick={() => void run(async () => { await api.board.addLinkEvidence(link.id, evId, ''); setPicking(false); setEvId(''); })}>Attach</Button>
            </div>
          ) : <Button size="sm" className="mt-2" icon={<Plus size={12} />} onClick={() => setPicking(true)}>Attach evidence</Button>}
        </div>
        <p className="text-[10px] text-fg-muted">Updated {formatUTC(link.updatedAt)}</p>
      </div>
    </Panel>
  );
}
