import { ShieldCheck } from 'lucide-react';
import { api } from '@/api/endpoints';
import { Badge, Button, EmptyState, ErrorBanner, HashText, PageHeader, Panel, Spinner } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';

export function AuditView() {
  const log = useAsync(() => api.audit.list(200), []);
  const chain = useAsync(() => api.audit.verify(), []);
  const refresh = () => { log.reload(); chain.reload(); };

  return (
    <>
      <PageHeader
        title="Audit Log"
        subtitle="Append-only, hash-chained record of every state change. Showing the latest 200 entries."
        actions={
          <>
            {chain.data && (chain.data.valid
              ? <Badge tone="outline">● Chain valid · {chain.data.count} entries</Badge>
              : <Badge tone="alert">▲ Chain broken at #{chain.data.brokenAt}</Badge>)}
            <Button variant="primary" icon={<ShieldCheck size={14} />} loading={chain.loading} onClick={refresh}>Re-verify</Button>
          </>
        }
      />
      {(log.error || chain.error) && <ErrorBanner message={(log.error ?? chain.error) as string} onRetry={refresh} />}
      {log.loading && !log.data ? <Spinner /> : !log.error && (log.data ?? []).length === 0 ? <EmptyState title="No audit entries" /> : (
        !log.error && (
          <Panel flush>
            <table className="x-table">
              <thead><tr><th>#</th><th>Time</th><th>Actor</th><th>Action</th><th>Entity</th><th>Detail</th><th>Hash</th></tr></thead>
              <tbody>
                {(log.data ?? []).map((a) => (
                  <tr key={a.id}>
                    <td className="tabular-nums text-fg-muted">{a.id}</td>
                    <td className="whitespace-nowrap">{formatUTC(a.timestamp)}</td>
                    <td className="text-fg-soft">{a.actor}</td>
                    <td><Badge tone={a.action.includes('FAILURE') ? 'alert' : 'neutral'}>{a.action}</Badge></td>
                    <td className="font-mono text-[10px] text-fg-muted">{a.entityType}:{a.entityId}</td>
                    <td className="max-w-xs truncate text-fg-soft" title={a.detail}>{a.detail}</td>
                    <td><HashText value={a.entryHash} head={8} tail={4} /></td>
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
