import { ShieldCheck, Upload } from 'lucide-react';
import { useState } from 'react';
import type { FormEvent } from 'react';
import { api } from '@/api/endpoints';
import { Button, EmptyState, ErrorBanner, HashText, Input, Label, Panel, Spinner, StatusBadge } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { isDesktop, pickEvidenceFile } from '@/lib/desktop';
import { formatBytes, formatUTC } from '@/lib/format';
import { CustodyModal } from './CustodyModal';

export function EvidenceTab({ caseId, onChanged }: { caseId: string; onChanged: () => void }) {
  const { data, error, loading, reload } = useAsync(() => api.evidence.list(caseId), [caseId]);
  const [path, setPath] = useState('');
  const [busy, setBusy] = useState<string | null>(null); // 'import' | evidence id
  const [actionError, setActionError] = useState<string | null>(null);
  const [custodyFor, setCustodyFor] = useState<string | null>(null);

  const run = async (key: string, fn: () => Promise<unknown>) => {
    setBusy(key);
    setActionError(null);
    try { await fn(); reload(); onChanged(); }
    catch (e) { setActionError(e instanceof Error ? e.message : String(e)); }
    finally { setBusy(null); }
  };

  const onImport = (e: FormEvent) => {
    e.preventDefault();
    if (!path.trim()) return;
    void run('import', async () => { await api.evidence.import(caseId, path.trim()); setPath(''); });
  };

  const failed = (data ?? []).filter((e) => e.status === 'FAILED');

  return (
    <div className="space-y-5">
      <Panel title="Import evidence">
        <form onSubmit={onImport} className="flex items-end gap-3">
          <div className="flex-1">
            <Label htmlFor="ev-path" hint="absolute path of a local file; it is copied into the vault and hashed (SHA-256)">File path</Label>
            <Input id="ev-path" value={path} onChange={(e) => setPath(e.target.value)} placeholder="/path/to/evidence.bin" spellCheck={false} />
          </div>
          {isDesktop() && (
            <Button onClick={() => void pickEvidenceFile().then((p) => { if (p) setPath(p); }).catch((e: unknown) => setActionError(e instanceof Error ? e.message : String(e)))}>Browse…</Button>
          )}
          <Button variant="primary" icon={<Upload size={14} />} loading={busy === 'import'} disabled={!path.trim()} onClick={onImport as unknown as () => void}>Import</Button>
          <button type="submit" className="hidden" />
        </form>
      </Panel>

      {actionError && <ErrorBanner message={actionError} />}
      {failed.length > 0 && (
        <ErrorBanner message={`INTEGRITY FAILURE: ${failed.map((f) => f.name).join(', ')} no longer matches its recorded SHA-256. Review the custody log before relying on this item.`} />
      )}
      {error && <ErrorBanner message={error} onRetry={reload} />}
      {loading && !data ? <Spinner /> : !error && (data ?? []).length === 0 ? (
        <EmptyState title="Vault is empty" hint="Imported files are stored read-only with a recorded digest and a chain-of-custody log." />
      ) : (
        !error && (
          <Panel flush>
            <table className="x-table">
              <thead><tr><th>File</th><th className="text-right">Size</th><th>SHA-256</th><th>Status</th><th>Last verified</th><th className="text-right">Actions</th></tr></thead>
              <tbody>
                {(data ?? []).map((ev) => (
                  <tr key={ev.id}>
                    <td className="max-w-[16rem]"><div className="truncate font-medium" title={ev.name}>{ev.name}</div><div className="font-mono text-[10px] text-fg-muted">{ev.id}</div></td>
                    <td className="text-right tabular-nums">{formatBytes(ev.sizeBytes)}</td>
                    <td><HashText value={ev.sha256} /></td>
                    <td><StatusBadge status={ev.status} /></td>
                    <td className="whitespace-nowrap text-fg-muted">{formatUTC(ev.lastVerifiedAt)}</td>
                    <td className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button size="sm" icon={<ShieldCheck size={12} />} loading={busy === ev.id} onClick={() => void run(ev.id, () => api.evidence.verify(ev.id))}>Verify</Button>
                        <Button size="sm" variant="ghost" onClick={() => setCustodyFor(ev.id)}>Custody</Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Panel>
        )
      )}
      {custodyFor && <CustodyModal evidenceId={custodyFor} onClose={() => setCustodyFor(null)} />}
    </div>
  );
}
