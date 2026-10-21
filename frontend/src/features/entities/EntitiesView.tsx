import { Plus } from 'lucide-react';
import { useMemo, useState } from 'react';
import { api } from '@/api/endpoints';
import { Badge, Button, EmptyState, ErrorBanner, HashText, Input, PageHeader, Panel, Spinner } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';
import { useNav } from '@/state/navStore';
import { LabelModal } from '../explorer/LabelModal';

export function EntitiesView() {
  const { openCase, openExplorer } = useNav();
  const { data, error, loading, reload } = useAsync(() => api.labels.list(), []);
  const [q, setQ] = useState('');
  const [adding, setAdding] = useState(false);

  const rows = useMemo(() => {
    const s = q.trim().toLowerCase();
    return (data ?? []).filter((l) => !s || [l.address, l.name, l.category, l.caseTitle].some((v) => v.toLowerCase().includes(s)));
  }, [data, q]);

  return (
    <>
      <PageHeader
        title="Entities"
        subtitle="Address labels recorded in your cases. Each carries its source, confidence and basis; none is proof of ownership."
        actions={<Button variant="primary" icon={<Plus size={14} />} onClick={() => setAdding(true)}>Add label</Button>}
      />
      <div className="mb-4 w-80"><Input aria-label="Filter labels" placeholder="Filter by address, name, category, case…" value={q} onChange={(e) => setQ(e.target.value)} /></div>
      {error && <ErrorBanner message={error} onRetry={reload} />}
      {loading && !data ? <Spinner /> : !error && rows.length === 0 ? (
        <EmptyState title={(data ?? []).length === 0 ? 'No labels yet' : 'No matching labels'} hint="Add a label from here, from an address in the Explorer, or from a node in the Graph." />
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
