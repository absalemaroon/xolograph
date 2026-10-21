import { useState } from 'react';
import { api } from '@/api/endpoints';
import { Button, EmptyState, ErrorBanner, Panel, Spinner, Textarea } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';

export function NotesTab({ caseId, onChanged }: { caseId: string; onChanged: () => void }) {
  const { data, error, loading, reload } = useAsync(() => api.notes.list(caseId), [caseId]);
  const [body, setBody] = useState('');
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const add = async () => {
    setBusy(true);
    setActionError(null);
    try { await api.notes.add(caseId, body); setBody(''); reload(); onChanged(); }
    catch (e) { setActionError(e instanceof Error ? e.message : String(e)); }
    finally { setBusy(false); }
  };

  return (
    <div className="space-y-5">
      <Panel title="Add note">
        <Textarea aria-label="Note" value={body} onChange={(e) => setBody(e.target.value)} placeholder="Observation, hypothesis, next step…" />
        <div className="mt-3 flex justify-end"><Button variant="primary" loading={busy} disabled={!body.trim()} onClick={() => void add()}>Add note</Button></div>
      </Panel>
      {actionError && <ErrorBanner message={actionError} />}
      {error && <ErrorBanner message={error} onRetry={reload} />}
      {loading && !data ? <Spinner /> : !error && (data ?? []).length === 0 ? <EmptyState title="No notes" hint="Notes are timestamped and recorded in the audit log." /> : (
        <ul className="space-y-3">
          {(data ?? []).map((n) => (
            <li key={n.id} className="border border-line bg-raised p-4">
              <div className="mb-2 text-[10px] uppercase tracking-wider text-fg-muted">{formatUTC(n.createdAt)}</div>
              <p className="whitespace-pre-wrap break-words text-xs leading-relaxed">{n.body}</p>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
