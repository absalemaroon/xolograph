import { ArrowLeft, ChevronDown, ChevronRight, Network, Pencil, Plus, Search, Trash2 } from 'lucide-react';
import { Fragment, useState } from 'react';
import { api } from '@/api/endpoints';
import type { Entity, EntityAddress, Provenance } from '@/api/types';
import { Badge, Button, EmptyState, ErrorBanner, HashText, Panel, Select, Spinner, Stat } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatBytes, formatUTC } from '@/lib/format';
import { useNav } from '@/state/navStore';
import { ActivityTable } from '../explorer/ActivityTable';
import { AttachModal } from '../explorer/AttachModal';
import type { AttachSpec } from '../explorer/AttachModal';
import { ProvenanceList } from '../explorer/Provenance';
import { AddressModal } from './AddressModal';
import { AttributionBadge } from './AttributionBadge';
import { ConfirmModal } from './ConfirmModal';
import { EntityFormModal } from './EntityFormModal';
import { IdentifierModal } from './IdentifierModal';
import { LinkEvidenceModal } from './LinkEvidenceModal';
import { STATUSES, explorerSupported, graphSupported, networkLabel, pretty } from './constants';

type Tab = 'overview' | 'wallets' | 'identifiers' | 'evidence' | 'activity';
const TABS: { id: Tab; label: string }[] = [
  { id: 'overview', label: 'Overview' }, { id: 'wallets', label: 'Wallets' }, { id: 'identifiers', label: 'Identifiers' },
  { id: 'evidence', label: 'Evidence' }, { id: 'activity', label: 'Activity' },
];

export function EntityDetailView({ id }: { id: string }) {
  const { closeEntity } = useNav();
  const { data, error, loading, reload } = useAsync(() => api.entities.get(id), [id]);
  const [tab, setTab] = useState<Tab>('overview');
  const [editing, setEditing] = useState(false);
  const [deleting, setDeleting] = useState(false);

  if (loading && !data) return <Spinner />;
  if (error || !data) return <><Button icon={<ArrowLeft size={14} />} onClick={closeEntity}>Entities</Button><div className="mt-4"><ErrorBanner message={error ?? 'Not found'} onRetry={reload} /></div></>;
  const e = data;

  return (
    <div className="space-y-5">
      <Button variant="ghost" size="sm" icon={<ArrowLeft size={12} />} onClick={closeEntity}>All entities</Button>
      <div className="flex flex-wrap items-end justify-between gap-3 border-b border-line pb-4">
        <div className="min-w-0">
          <h1 className="truncate text-lg font-semibold uppercase tracking-[0.12em]">{e.name}</h1>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <Badge tone="outline">{pretty(e.type)}</Badge>
            <AttributionBadge status={e.status} />
            <Badge>Confidence {e.confidence}</Badge>
            {e.country && <Badge>{e.country}</Badge>}
            {e.tags.map((t) => <Badge key={t}>#{t}</Badge>)}
          </div>
        </div>
        <div className="flex gap-2">
          <Button icon={<Pencil size={14} />} onClick={() => setEditing(true)}>Edit</Button>
          <Button variant="danger" icon={<Trash2 size={14} />} onClick={() => setDeleting(true)}>Delete</Button>
        </div>
      </div>

      <div role="tablist" aria-label="Entity sections" className="flex border-b border-line">
        {TABS.map((t) => (
          <button key={t.id} role="tab" type="button" aria-selected={tab === t.id} onClick={() => setTab(t.id)}
            className={`-mb-px border-b-2 px-4 py-2.5 text-xs uppercase tracking-[0.14em] ${tab === t.id ? 'border-fg font-semibold' : 'border-transparent text-fg-muted hover:text-fg'}`}>{t.label}</button>
        ))}
      </div>

      {tab === 'overview' && <Overview e={e} />}
      {tab === 'wallets' && <Wallets e={e} onChange={reload} />}
      {tab === 'identifiers' && <Identifiers e={e} onChange={reload} />}
      {tab === 'evidence' && <EvidenceTab e={e} onChange={reload} />}
      {tab === 'activity' && <Activity e={e} />}

      {editing && <EntityFormModal entity={e} onClose={() => setEditing(false)} onSaved={reload} />}
      {deleting && (
        <ConfirmModal title="Delete entity" confirmLabel="Delete entity"
          message={`Delete “${e.name}” with its ${e.addressCount} wallet association(s), identifiers and evidence links? Evidence files, cases and case labels are not affected. This is recorded in the audit log.`}
          onClose={() => setDeleting(false)} onConfirm={async () => { await api.entities.remove(e.id); closeEntity(); }} />
      )}
    </div>
  );
}

function Overview({ e }: { e: Entity }) {
  const addrs = e.addresses ?? [];
  const counts = STATUSES.map((s) => ({ s, n: addrs.filter((a) => a.status === s.value).length })).filter((x) => x.n > 0);
  const unsupported = addrs.filter((a) => !graphSupported(a.network)).length;
  const warnings: string[] = [];
  if (addrs.length === 0) warnings.push('No wallets are associated with this entity, so there is no on-chain activity to show.');
  if (unsupported > 0) warnings.push(`${unsupported} wallet(s) are on networks the graph does not cover yet (graphs use Ethereum mainnet activity). They are still browsable in the Explorer.`);
  if (addrs.some((a) => a.status === 'DISPUTED')) warnings.push('Some wallet attributions are disputed.');
  if (e.status === 'VERIFIED' && addrs.length > 0 && !addrs.some((a) => a.status === 'VERIFIED')) warnings.push('The entity is marked verified but none of its wallets are. Entity identity and wallet ownership are separate claims.');
  return (
    <div className="space-y-5">
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Stat label="Wallets" value={addrs.length} />
        <Stat label="Identifiers" value={(e.identifiers ?? []).length} />
        <Stat label="Evidence" value={(e.evidence ?? []).length} />
        <Stat label="Updated" value={<span className="text-sm">{formatUTC(e.updatedAt)}</span>} />
      </div>
      {warnings.length > 0 && <Panel title="Data gaps and cautions"><ul className="space-y-1.5 text-xs">{warnings.map((w) => <li key={w}>⚠ {w}</li>)}</ul></Panel>}
      <Panel title="Profile">
        <p className="whitespace-pre-wrap text-xs">{e.description || <span className="text-fg-muted">No description.</span>}</p>
        {counts.length > 0 && <div className="mt-4 flex flex-wrap gap-2">{counts.map((c) => <span key={c.s.value} className="inline-flex items-center gap-1.5 text-[11px]"><AttributionBadge status={c.s.value} /> × {c.n}</span>)}</div>}
      </Panel>
      <Panel title="Investigator notes"><p className="whitespace-pre-wrap text-xs">{e.notes || <span className="text-fg-muted">No notes.</span>}</p></Panel>
    </div>
  );
}

function Wallets({ e, onChange }: { e: Entity; onChange: () => void }) {
  const { openExplorer, openGraph, openGraphMulti } = useNav();
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<EntityAddress | null>(null);
  const [removing, setRemoving] = useState<EntityAddress | null>(null);
  const [open, setOpen] = useState<string | null>(null);
  const addrs = e.addresses ?? [];
  const graphable = [...new Set(addrs.filter((a) => graphSupported(a.network) && a.status !== 'REJECTED').map((a) => a.address))].slice(0, 10);
  return (
    <Panel title="Associated wallets" flush actions={<div className="flex gap-2">
      {graphable.length > 0 && <Button size="sm" icon={<Network size={12} />} onClick={() => openGraphMulti(graphable)}>Graph wallets ({graphable.length})</Button>}
      <Button size="sm" variant="primary" icon={<Plus size={12} />} onClick={() => setAdding(true)}>Add wallet</Button>
    </div>}>
      {addrs.length === 0 ? <div className="p-4"><EmptyState title="No wallets" hint="Add a wallet only with a stated basis. Transfers between wallets do not prove common ownership." /></div> : (
        <div className="overflow-x-auto">
          <table className="x-table">
            <thead><tr><th /><th>Chain</th><th>Address</th><th>Label / role</th><th>Attribution</th><th>Confidence</th><th>Source</th><th /></tr></thead>
            <tbody>
              {addrs.map((a) => (
                <Fragment key={a.id}>
                  <tr>
                    <td><button type="button" aria-label="Toggle history" aria-expanded={open === a.id} onClick={() => setOpen(open === a.id ? null : a.id)} className="text-fg-muted hover:text-fg">{open === a.id ? <ChevronDown size={14} /> : <ChevronRight size={14} />}</button></td>
                    <td>{networkLabel(a.network)}</td>
                    <td><HashText value={a.address} head={8} tail={6} /></td>
                    <td>{a.label || '—'}<div className="text-[10px] text-fg-muted">{a.role}</div></td>
                    <td><AttributionBadge status={a.status} /></td>
                    <td><Badge tone={a.confidence === 'HIGH' ? 'outline' : 'neutral'}>{a.confidence}</Badge></td>
                    <td className="max-w-[14rem] break-words text-fg-soft">{a.sourceRef || <span className="text-fg-muted">none stated</span>}</td>
                    <td>
                      <div className="flex justify-end gap-1">
                        {explorerSupported(a.network) ? (
                          <>
                            <Button size="sm" variant="ghost" icon={<Search size={12} />} onClick={() => openExplorer(a.address, a.network)}>Explorer</Button>
                            {graphSupported(a.network) && <Button size="sm" variant="ghost" icon={<Network size={12} />} onClick={() => openGraph(a.address)}>Graph</Button>}
                          </>
                        ) : <span className="self-center text-[10px] uppercase text-fg-muted">No adapter</span>}
                        <Button size="sm" variant="ghost" icon={<Pencil size={12} />} aria-label="Edit claim" onClick={() => setEditing(a)} />
                        <Button size="sm" variant="ghost" icon={<Trash2 size={12} />} aria-label="Remove association" onClick={() => setRemoving(a)} />
                      </div>
                    </td>
                  </tr>
                  {open === a.id && (
                    <tr><td /><td colSpan={7} className="bg-base">
                      {a.basis && <p className="mb-2 text-[11px]"><span className="text-fg-muted">Basis: </span>{a.basis}</p>}
                      <div className="text-[10px] uppercase tracking-wider text-fg-muted">Attribution history</div>
                      <ul className="mt-1 space-y-1 text-[11px]">
                        {(a.history ?? []).map((h, i) => <li key={i}>{formatUTC(h.timestamp)} · {h.actor} · {h.fromStatus ? `${pretty(h.fromStatus)} → ` : ''}{pretty(h.toStatus)}{h.note ? ` — ${h.note}` : ''}</li>)}
                      </ul>
                    </td></tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {adding && <AddressModal entityId={e.id} onClose={() => setAdding(false)} onSaved={onChange} />}
      {editing && <AddressModal entityId={e.id} existing={editing} onClose={() => setEditing(null)} onSaved={onChange} />}
      {removing && (
        <ConfirmModal title="Remove wallet association" confirmLabel="Remove"
          message={`Remove ${removing.address} from “${e.name}”? Only the association and its status history are deleted. Blockchain data, cases and evidence are untouched.`}
          onClose={() => setRemoving(null)} onConfirm={async () => { await api.entities.removeAddress(removing.id); onChange(); }} />
      )}
    </Panel>
  );
}

function Identifiers({ e, onChange }: { e: Entity; onChange: () => void }) {
  const [adding, setAdding] = useState(false);
  const [removeId, setRemoveId] = useState<string | null>(null);
  const items = e.identifiers ?? [];
  return (
    <Panel title="Public identifiers" flush actions={<Button size="sm" variant="primary" icon={<Plus size={12} />} onClick={() => setAdding(true)}>Add identifier</Button>}>
      {items.length === 0 ? <div className="p-4"><EmptyState title="No identifiers" hint="Record public profiles and sites, each with where you found it. They are not proof of identity." /></div> : (
        <table className="x-table">
          <thead><tr><th>Platform</th><th>Value</th><th>Status</th><th>Confidence</th><th>Source</th><th /></tr></thead>
          <tbody>
            {items.map((i) => (
              <tr key={i.id}>
                <td>{pretty(i.platform)}</td>
                <td className="break-all font-mono text-[11px]">{i.value}</td>
                <td><Badge tone={i.status === 'VERIFIED' ? 'solid' : i.status === 'DISPUTED' ? 'alert' : 'neutral'}>{pretty(i.status)}</Badge></td>
                <td>{i.confidence}</td>
                <td className="max-w-[14rem] break-words text-fg-soft">{i.sourceRef || <span className="text-fg-muted">none stated</span>}</td>
                <td className="text-right"><Button size="sm" variant="ghost" icon={<Trash2 size={12} />} aria-label="Remove identifier" onClick={() => setRemoveId(i.id)} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {adding && <IdentifierModal entityId={e.id} onClose={() => setAdding(false)} onSaved={onChange} />}
      {removeId && <ConfirmModal title="Remove identifier" confirmLabel="Remove" message="Remove this identifier from the entity?" onClose={() => setRemoveId(null)} onConfirm={async () => { await api.entities.removeIdentifier(removeId); onChange(); }} />}
    </Panel>
  );
}

function EvidenceTab({ e, onChange }: { e: Entity; onChange: () => void }) {
  const { openCase } = useNav();
  const [linking, setLinking] = useState(false);
  const [unlinkId, setUnlinkId] = useState<string | null>(null);
  const items = e.evidence ?? [];
  return (
    <Panel title="Linked evidence" flush actions={<Button size="sm" variant="primary" icon={<Plus size={12} />} onClick={() => setLinking(true)}>Link evidence</Button>}>
      {items.length === 0 ? <div className="p-4"><EmptyState title="No evidence linked" hint="Link files from a case's evidence vault. The original stays in its case with its hash and chain of custody." /></div> : (
        <table className="x-table">
          <thead><tr><th>File</th><th>SHA-256</th><th>Case</th><th>Shows</th><th /></tr></thead>
          <tbody>
            {items.map((x) => (
              <tr key={x.id}>
                <td>{x.name}<div className="text-[10px] text-fg-muted">{formatBytes(x.sizeBytes)}</div></td>
                <td><HashText value={x.sha256} head={10} tail={6} /></td>
                <td><button type="button" className="underline-offset-2 hover:underline" onClick={() => openCase(x.caseId)}>{x.caseTitle}</button></td>
                <td className="text-fg-soft">{x.description || '—'}{x.addressId && <div className="text-[10px] text-fg-muted">linked to a wallet</div>}</td>
                <td className="text-right"><Button size="sm" variant="ghost" icon={<Trash2 size={12} />} aria-label="Unlink evidence" onClick={() => setUnlinkId(x.id)} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {linking && <LinkEvidenceModal entity={e} onClose={() => setLinking(false)} onSaved={onChange} />}
      {unlinkId && <ConfirmModal title="Unlink evidence" confirmLabel="Unlink" message="Remove the link? The file stays in its case vault." onClose={() => setUnlinkId(null)} onConfirm={async () => { await api.entities.unlinkEvidence(unlinkId); onChange(); }} />}
    </Panel>
  );
}

function Activity({ e }: { e: Entity }) {
  const wallets = (e.addresses ?? []).filter((a) => explorerSupported(a.network) && a.status !== 'REJECTED');
  const [sel, setSel] = useState(wallets[0]?.id ?? '');
  const [kind, setKind] = useState<'native' | 'token'>('native');
  const [attach, setAttach] = useState<AttachSpec | null>(null);
  const [pages, setPages] = useState<Provenance[]>([]);
  const addProv = (p: Provenance) => setPages((cur) => (cur.some((c) => c.payloadSha256 === p.payloadSha256 && c.query === p.query) ? cur : [...cur, p]));
  const networks = useAsync(() => api.chain.networks(), []);
  if (wallets.length === 0) return <EmptyState title="No queryable wallets" hint="Activity needs at least one non-rejected wallet on a supported network." />;
  const cur = wallets.find((w) => w.id === sel) ?? wallets[0];
  const info = (networks.data ?? []).find((n) => n.id === cur.network);
  const kinds = info?.family === 'BITCOIN' ? ([['native', `${info.nativeSymbol} transactions`]] as const) : ([['native', `${info?.nativeSymbol ?? 'Native'} transfers`], ['token', 'Token transfers']] as const);
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <Select aria-label="Wallet" className="max-w-md" value={cur.id} onChange={(ev) => { setSel(ev.target.value); setKind('native'); }}>
          {wallets.map((w) => <option key={w.id} value={w.id}>{networkLabel(w.network)} · {w.label ? `${w.label} · ` : ''}{w.address}</option>)}
        </Select>
        {kinds.map(([k, label]) => (
          <Button key={k} variant={kind === k ? 'primary' : 'secondary'} size="sm" onClick={() => setKind(k)}>{label}</Button>
        ))}
      </div>
      <p className="text-[11px] text-fg-muted">Activity is shown for one wallet at a time. An entity-wide aggregate is not implemented, and history comes from third-party indexes or public nodes and may be incomplete.</p>
      <ActivityTable key={`${cur.id}-${kind}`} address={cur.address} network={cur.network} kind={kind} onAttach={setAttach} onProvenance={addProv} />
      <ProvenanceList items={pages} />
      {attach && <AttachModal spec={attach} onClose={() => setAttach(null)} />}
    </div>
  );
}
