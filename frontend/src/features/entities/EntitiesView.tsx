import { Plus } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '@/api/endpoints';
import { Badge, Button, EmptyState, ErrorBanner, HashText, Input, PageHeader, Panel, Segmented, Spinner } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';
import { useNav } from '@/state/navStore';
import { LabelModal } from '../explorer/LabelModal';
import { AttributionBadge } from './AttributionBadge';
import { EntityDetailView } from './EntityDetailView';
import { EntityFormModal } from './EntityFormModal';
import { pretty } from './constants';

export function EntitiesView() {
  const { entityId, openEntity } = useNav();
  if (entityId) return <EntityDetailView id={entityId} />;
  return <EntityList onOpen={openEntity} />;
}

function EntityList({ onOpen }: { onOpen: (id: string) => void }) {
  const [mode, setMode] = useState<'entities' | 'labels'>('entities');
  const [q, setQ] = useState('');
  const [debounced, setDebounced] = useState('');
  const [creating, setCreating] = useState(false);
  useEffect(() => { const t = window.setTimeout(() => setDebounced(q), 250); return () => window.clearTimeout(t); }, [q]);

  return (
    <>
      <PageHeader
        title="Entities"
        subtitle="Investigative records. Each wallet association carries its own attribution status, confidence and source; none is proof of ownership."
        actions={mode === 'entities' && <Button variant="primary" icon={<Plus size={14} />} onClick={() => setCreating(true)}>New entity</Button>}
      />
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <Segmented label="Entities or case labels" value={mode} onChange={setMode} options={[{ value: 'entities', label: 'Entities' }, { value: 'labels', label: 'Case labels' }]} />
        <div className="w-80"><Input aria-label="Filter" placeholder={mode === 'entities' ? 'Filter by name, tag, address, wallet label…' : 'Filter by address, name, category, case…'} value={q} onChange={(e) => setQ(e.target.value)} /></div>
      </div>
      {mode === 'entities' ? <EntityTable q={debounced} onOpen={onOpen} onCreate={() => setCreating(true)} /> : <LabelsTable q={q} />}
      {creating && <EntityFormModal onClose={() => setCreating(false)} onSaved={(e) => onOpen(e.id)} />}
    </>
  );
}

function EntityTable({ q, onOpen, onCreate }: { q: string; onOpen: (id: string) => void; onCreate: () => void }) {
  const { data, error, loading, reload } = useAsync(() => api.entities.list(q), [q]);
  if (error) return <ErrorBanner message={error} onRetry={reload} />;
  if (loading && !data) return <Spinner />;
  const rows = data ?? [];
  if (rows.length === 0) {
    return <EmptyState title={q ? 'No matching entities' : 'No entities yet'} hint={q ? 'Try a different name, tag or address.' : 'Create an entity, then attach wallets, public identifiers and evidence to it.'} action={!q && <Button variant="primary" onClick={onCreate}>New entity</Button>} />;
  }
  return (
    <Panel flush>
      <table className="x-table">
        <thead><tr><th>Name</th><th>Type</th><th>Attribution</th><th>Confidence</th><th className="text-right">Wallets</th><th>Tags</th><th>Updated</th></tr></thead>
        <tbody>
          {rows.map((e) => (
            <tr key={e.id}>
              <td><button type="button" className="font-medium underline-offset-2 hover:underline" onClick={() => onOpen(e.id)}>{e.name}</button></td>
              <td>{pretty(e.type)}</td>
              <td><AttributionBadge status={e.status} /></td>
              <td><Badge tone={e.confidence === 'HIGH' ? 'outline' : 'neutral'}>{e.confidence}</Badge></td>
              <td className="text-right tabular-nums">{e.addressCount}</td>
              <td className="max-w-[14rem] truncate text-fg-soft">{e.tags.join(', ') || '—'}</td>
              <td className="whitespace-nowrap text-fg-muted">{formatUTC(e.updatedAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Panel>
  );
}

/** v1 case-scoped labels, kept as-is so existing analyst work stays visible. */
function LabelsTable({ q }: { q: string }) {
  const { openCase, openExplorer } = useNav();
  const { data, error, loading, reload } = useAsync(() => api.labels.list(), []);
  const [adding, setAdding] = useState(false);
  const s = q.trim().toLowerCase();
  const rows = (data ?? []).filter((l) => !s || [l.address, l.name, l.category, l.caseTitle].some((v) => v.toLowerCase().includes(s)));
  return (
    <>
      <div className="mb-3"><Button icon={<Plus size={14} />} onClick={() => setAdding(true)}>Add case label</Button></div>
      {error && <ErrorBanner message={error} onRetry={reload} />}
      {loading && !data ? <Spinner /> : !error && rows.length === 0 ? (
        <EmptyState title={(data ?? []).length === 0 ? 'No case labels yet' : 'No matching labels'} hint="Case labels name a single address inside one case. Use an entity to group several wallets." />
      ) : !error && (
        <Panel flush>
          <table className="x-table">
            <thead><tr><th>Address</th><th>Name</th><th>Source</th><th>Confidence</th><th>Basis</th><th>Case</th><th>Added</th></tr></thead>
            <tbody>
              {rows.map((l) => (
                <tr key={l.id}>
                  <td><div className="flex items-center gap-2"><HashText value={l.address} head={8} tail={6} /><button type="button" className="text-[10px] uppercase underline-offset-2 hover:underline" onClick={() => openExplorer(l.address)}>View</button></div></td>
                  <td className="font-medium">{l.name}<div className="text-[10px] text-fg-muted">{l.category}</div></td>
                  <td><Badge>{l.source}</Badge></td>
                  <td><Badge tone={l.confidence === 'HIGH' ? 'outline' : 'neutral'}>{l.confidence}</Badge></td>
                  <td className="max-w-xs text-fg-soft">{l.basis}</td>
                  <td><button type="button" className="underline-offset-2 hover:underline" onClick={() => openCase(l.caseId)}>{l.caseTitle}</button></td>
                  <td className="whitespace-nowrap text-fg-muted">{formatUTC(l.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Panel>
      )}
      {adding && <LabelModal onClose={() => setAdding(false)} onSaved={reload} />}
    </>
  );
}
