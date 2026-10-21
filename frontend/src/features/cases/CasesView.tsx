import { Plus } from 'lucide-react';
import { useMemo, useState } from 'react';
import { api } from '@/api/endpoints';
import { Button, EmptyState, ErrorBanner, Input, PageHeader, Panel, Select, Spinner, StatusBadge } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';
import { useNav } from '@/state/navStore';
import { CaseDetailView } from './CaseDetailView';
import { NewCaseModal } from './NewCaseModal';

export function CasesView() {
  const { caseId, openCase } = useNav();
  const { data, error, loading, reload } = useAsync(() => api.cases.list(), []);
  const [query, setQuery] = useState('');
  const [status, setStatus] = useState('ALL');
  const [creating, setCreating] = useState(false);

  const rows = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (data ?? []).filter(
      (c) => (status === 'ALL' || c.status === status) && (!q || c.title.toLowerCase().includes(q) || c.id.toLowerCase().includes(q)),
    );
  }, [data, query, status]);

  if (caseId) return <CaseDetailView caseId={caseId} />;

  return (
    <>
      <PageHeader
        title="Cases"
        subtitle="Each case holds its evidence, notes and findings, with a verifiable audit trail."
        actions={<Button variant="primary" icon={<Plus size={14} />} onClick={() => setCreating(true)}>New case</Button>}
      />
      <div className="mb-4 flex flex-wrap gap-3">
        <div className="w-72"><Input aria-label="Search cases" placeholder="Search title or ID…" value={query} onChange={(e) => setQuery(e.target.value)} /></div>
        <div className="w-40">
          <Select aria-label="Filter by status" value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="ALL">All statuses</option>
            <option value="OPEN">Open</option>
            <option value="CLOSED">Closed</option>
            <option value="ARCHIVED">Archived</option>
          </Select>
        </div>
      </div>
      {error && <ErrorBanner message={error} onRetry={reload} />}
      {loading && !data ? <Spinner /> : !error && (
        rows.length === 0 ? (
          <EmptyState
            title={(data ?? []).length === 0 ? 'No cases yet' : 'No matching cases'}
            hint={(data ?? []).length === 0 ? 'Create your first case to begin an investigation.' : 'Adjust the search or status filter.'}
            action={(data ?? []).length === 0 ? <Button variant="primary" onClick={() => setCreating(true)}>New case</Button> : undefined}
          />
        ) : (
          <Panel flush>
            <table className="x-table">
              <thead><tr><th>Case</th><th>Status</th><th className="text-right">Evidence</th><th className="text-right">Findings</th><th>Updated</th></tr></thead>
              <tbody>
                {rows.map((c) => (
                  <tr key={c.id}>
                    <td>
                      <button type="button" onClick={() => openCase(c.id)} className="block text-left font-medium underline-offset-2 hover:underline">{c.title}</button>
                      <div className="font-mono text-[10px] text-fg-muted">{c.id}</div>
                    </td>
                    <td><StatusBadge status={c.status} /></td>
                    <td className="text-right tabular-nums">{c.evidenceCount}</td>
                    <td className="text-right tabular-nums">{c.findingCount}</td>
                    <td className="whitespace-nowrap text-fg-muted">{formatUTC(c.updatedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Panel>
        )
      )}
      {creating && <NewCaseModal onClose={() => setCreating(false)} onCreated={(c) => { setCreating(false); openCase(c.id); }} />}
    </>
  );
}
