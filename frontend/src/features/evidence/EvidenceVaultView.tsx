import { ShieldCheck } from 'lucide-react';
import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { Evidence } from '@/api/types';
import { Button, EmptyState, ErrorBanner, HashText, PageHeader, Panel, Spinner, Stat, StatusBadge } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatBytes, formatUTC } from '@/lib/format';
import { useNav } from '@/state/navStore';

type Row = Evidence & { caseTitle: string };

export function EvidenceVaultView() {
  const { openCase } = useNav();
  const { data, error, loading, reload } = useAsync<Row[]>(async () => {
    const cases = await api.cases.list();
    const lists = await Promise.all(cases.map(async (c) => (await api.evidence.list(c.id)).map((e) => ({ ...e, caseTitle: c.title }))));
    return lists.flat();
  }, []);
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(null);
  const [runError, setRunError] = useState<string | null>(null);

  const rows = data ?? [];
  const verifyAll = async () => {
    setRunError(null);
    setProgress({ done: 0, total: rows.length });
    try {
      for (let i = 0; i < rows.length; i++) {
        await api.evidence.verify(rows[i].id);
        setProgress({ done: i + 1, total: rows.length });
      }
    } catch (e) { setRunError(e instanceof Error ? e.message : String(e)); }
    setProgress(null);
    reload();
  };

  const failed = rows.filter((r) => r.status === 'FAILED').length;
  const totalSize = rows.reduce((n, r) => n + r.sizeBytes, 0);

  return (
    <>
      <PageHeader
        title="Evidence Vault"
        subtitle="Every imported item across all cases, with its recorded SHA-256 digest."
        actions={<Button variant="primary" icon={<ShieldCheck size={14} />} loading={progress !== null} disabled={rows.length === 0} onClick={() => void verifyAll()}>{progress ? `Verifying ${progress.done}/${progress.total}` : 'Verify all'}</Button>}
      />
      <div className="mb-5 grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Stat label="Items" value={data ? rows.length : '—'} />
        <Stat label="Total size" value={data ? formatBytes(totalSize) : '—'} />
        <Stat label="Verified" value={data ? rows.filter((r) => r.status === 'VERIFIED').length : '—'} />
        <Stat label="Integrity failures" value={data ? (failed > 0 ? `▲ ${failed}` : '0') : '—'} />
      </div>
      {runError && <div className="mb-4"><ErrorBanner message={runError} /></div>}
      {error && <ErrorBanner message={error} onRetry={reload} />}
      {loading && !data ? <Spinner /> : !error && rows.length === 0 ? (
        <EmptyState title="No evidence imported" hint="Open a case and import a file to start building the vault." />
      ) : (
        !error && (
          <Panel flush>
            <table className="x-table">
              <thead><tr><th>File</th><th>Case</th><th className="text-right">Size</th><th>SHA-256</th><th>Status</th><th>Last verified</th></tr></thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className="max-w-[14rem] truncate font-medium" title={r.name}>{r.name}</td>
                    <td><button type="button" className="underline-offset-2 hover:underline" onClick={() => openCase(r.caseId)}>{r.caseTitle}</button></td>
                    <td className="text-right tabular-nums">{formatBytes(r.sizeBytes)}</td>
                    <td><HashText value={r.sha256} /></td>
                    <td><StatusBadge status={r.status} /></td>
                    <td className="whitespace-nowrap text-fg-muted">{formatUTC(r.lastVerifiedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Panel>
        )
      )}
    </>
  );
}
