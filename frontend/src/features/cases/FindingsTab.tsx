import { useState } from 'react';
import { api } from '@/api/endpoints';
import { Badge, Button, EmptyState, ErrorBanner, Input, Label, Panel, Select, Spinner, Textarea } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';

const KINDS = ['OBSERVATION', 'ADDRESS', 'TRANSACTION', 'ENTITY', 'PATH'];

export function FindingsTab({ caseId, onChanged }: { caseId: string; onChanged: () => void }) {
  const { data, error, loading, reload } = useAsync(() => api.findings.list(caseId), [caseId]);
  const [kind, setKind] = useState('OBSERVATION');
  const [title, setTitle] = useState('');
  const [refId, setRefId] = useState('');
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const add = async () => {
    setBusy(true);
    setActionError(null);
    try {
      await api.findings.add(caseId, { kind, title, note, refType: kind === 'OBSERVATION' ? '' : kind.toLowerCase(), refId: refId.trim() });
      setTitle(''); setRefId(''); setNote(''); reload(); onChanged();
    } catch (e) { setActionError(e instanceof Error ? e.message : String(e)); }
    finally { setBusy(false); }
  };

  return (
    <div className="space-y-5">
      <Panel title="Record finding">
        <div className="grid gap-4 md:grid-cols-[10rem_1fr_1fr]">
          <div><Label htmlFor="f-kind">Type</Label><Select id="f-kind" value={kind} onChange={(e) => setKind(e.target.value)}>{KINDS.map((k) => <option key={k}>{k}</option>)}</Select></div>
          <div><Label htmlFor="f-title">Title</Label><Input id="f-title" value={title} onChange={(e) => setTitle(e.target.value)} /></div>
          <div><Label htmlFor="f-ref" hint="address, tx hash…">Reference</Label><Input id="f-ref" value={refId} onChange={(e) => setRefId(e.target.value)} spellCheck={false} /></div>
        </div>
        <div className="mt-4"><Label htmlFor="f-note">Note</Label><Textarea id="f-note" value={note} onChange={(e) => setNote(e.target.value)} /></div>
        <div className="mt-3 flex justify-end"><Button variant="primary" loading={busy} disabled={!title.trim()} onClick={() => void add()}>Record finding</Button></div>
      </Panel>
      {actionError && <ErrorBanner message={actionError} />}
      {error && <ErrorBanner message={error} onRetry={reload} />}
      {loading && !data ? <Spinner /> : !error && (data ?? []).length === 0 ? (
        <EmptyState title="No findings" hint="Findings link analysis conclusions to their source records." />
      ) : (
        <ul className="space-y-3">
          {(data ?? []).map((f) => (
            <li key={f.id} className="border border-line bg-raised p-4">
              <div className="mb-2 flex flex-wrap items-center gap-3"><Badge tone="outline">{f.kind}</Badge><span className="text-xs font-semibold">{f.title}</span><span className="ml-auto text-[10px] uppercase tracking-wider text-fg-muted">{formatUTC(f.createdAt)}</span></div>
              {f.refId && <div className="mb-1 break-all font-mono text-[11px] text-fg-soft">{f.refId}</div>}
              {f.note && <p className="whitespace-pre-wrap break-words text-xs text-fg-soft">{f.note}</p>}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
