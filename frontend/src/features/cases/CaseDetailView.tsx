import { ArrowLeft } from 'lucide-react';
import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { CaseStatus } from '@/api/types';
import { Button, ErrorBanner, PageHeader, Select, Spinner, StatusBadge } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';
import { useNav } from '@/state/navStore';
import { EvidenceTab } from './EvidenceTab';
import { FindingsTab } from './FindingsTab';
import { BoardTab } from '../board/BoardTab';
import { NotesTab } from './NotesTab';

type Tab = 'evidence' | 'notes' | 'findings' | 'board';

export function CaseDetailView({ caseId }: { caseId: string }) {
  const { closeCase } = useNav();
  const { data, error, loading, reload } = useAsync(() => api.cases.get(caseId), [caseId]);
  const [tab, setTab] = useState<Tab>('evidence');
  const [actionError, setActionError] = useState<string | null>(null);

  const changeStatus = async (status: CaseStatus) => {
    setActionError(null);
    try { await api.cases.setStatus(caseId, status); reload(); }
    catch (e) { setActionError(e instanceof Error ? e.message : String(e)); }
  };

  if (loading && !data) return <Spinner />;
  if (error || !data) return <ErrorBanner message={error ?? 'Case not found'} onRetry={reload} />;

  const tabs: { id: Tab; label: string; count?: number }[] = [
    { id: 'evidence', label: 'Evidence', count: data.evidenceCount },
    { id: 'notes', label: 'Notes' },
    { id: 'findings', label: 'Findings', count: data.findingCount },
    { id: 'board', label: 'Board' },
  ];

  return (
    <>
      <div className="mb-3"><Button variant="ghost" size="sm" icon={<ArrowLeft size={13} />} onClick={closeCase}>All cases</Button></div>
      <PageHeader
        title={data.title}
        subtitle={`${data.id} · created ${formatUTC(data.createdAt)} · updated ${formatUTC(data.updatedAt)}`}
        actions={
          <>
            <StatusBadge status={data.status} />
            <div className="w-36">
              <Select aria-label="Case status" value={data.status} onChange={(e) => void changeStatus(e.target.value as CaseStatus)}>
                <option value="OPEN">Open</option>
                <option value="CLOSED">Closed</option>
                <option value="ARCHIVED">Archived</option>
              </Select>
            </div>
          </>
        }
      />
      {data.description && <p className="mb-5 max-w-3xl whitespace-pre-wrap text-xs text-fg-soft">{data.description}</p>}
      {actionError && <div className="mb-4"><ErrorBanner message={actionError} /></div>}
      <div role="tablist" aria-label="Case sections" className="mb-5 flex border-b border-line">
        {tabs.map((t) => (
          <button
            key={t.id}
            role="tab"
            type="button"
            aria-selected={tab === t.id}
            onClick={() => setTab(t.id)}
            className={`-mb-px border-b-2 px-4 py-2.5 text-xs uppercase tracking-[0.14em] ${tab === t.id ? 'border-fg font-semibold text-fg' : 'border-transparent text-fg-muted hover:text-fg'}`}
          >
            {t.label}{t.count !== undefined && <span className="ml-2 text-fg-muted">{t.count}</span>}
          </button>
        ))}
      </div>
      {tab === 'evidence' && <EvidenceTab caseId={caseId} onChanged={reload} />}
      {tab === 'notes' && <NotesTab caseId={caseId} onChanged={reload} />}
      {tab === 'findings' && <FindingsTab caseId={caseId} onChanged={reload} />}
      {tab === 'board' && <BoardTab caseId={caseId} onChanged={reload} />}
    </>
  );
}
