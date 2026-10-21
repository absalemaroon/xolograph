import { api } from '@/api/endpoints';
import { Button, EmptyState, ErrorBanner, PageHeader, Panel, Spinner, Stat, StatusBadge, Badge } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';
import { useNav } from '@/state/navStore';

export function OverviewView() {
  const { go, openCase } = useNav();
  const cases = useAsync(() => api.cases.list(), []);
  const audit = useAsync(() => api.audit.list(8), []);
  const chain = useAsync(() => api.audit.verify(), []);

  const list = cases.data ?? [];
  const open = list.filter((c) => c.status === 'OPEN').length;
  const evidence = list.reduce((n, c) => n + c.evidenceCount, 0);
  const findings = list.reduce((n, c) => n + c.findingCount, 0);

  return (
    <>
      <PageHeader
        title="Overview"
        subtitle="Local investigation workspace. All records are stored on this machine."
        actions={<Button variant="primary" onClick={() => go('cases')}>Open cases</Button>}
      />
      {cases.error && <div className="mb-4"><ErrorBanner message={cases.error} onRetry={cases.reload} /></div>}
      <div className="mb-5 grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Stat label="Open cases" value={cases.data ? open : '—'} hint={cases.data ? `${list.length} total` : undefined} />
        <Stat label="Evidence items" value={cases.data ? evidence : '—'} />
        <Stat label="Findings" value={cases.data ? findings : '—'} />
        <Stat
          label="Audit chain"
          value={chain.data ? (chain.data.valid ? '● Valid' : '▲ Broken') : '—'}
          hint={chain.data ? `${chain.data.count} entries verified` : undefined}
        />
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        <Panel title="Recent cases" flush>
          {cases.loading && !cases.data ? <Spinner /> : list.length === 0 ? (
            <div className="p-4"><EmptyState title="No cases yet" hint="Create a case to start collecting evidence and findings." action={<Button variant="primary" onClick={() => go('cases')}>Create case</Button>} /></div>
          ) : (
            <table className="x-table">
              <thead><tr><th>Case</th><th>Status</th><th>Updated</th></tr></thead>
              <tbody>
                {list.slice(0, 6).map((c) => (
                  <tr key={c.id}>
                    <td><button type="button" className="text-left font-medium underline-offset-2 hover:underline" onClick={() => openCase(c.id)}>{c.title}</button></td>
                    <td><StatusBadge status={c.status} /></td>
                    <td className="whitespace-nowrap text-fg-muted">{formatUTC(c.updatedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Panel>
        <Panel title="Recent activity" actions={<Button size="sm" variant="ghost" onClick={() => go('audit')}>Full log</Button>} flush>
          {audit.loading && !audit.data ? <Spinner /> : (audit.data ?? []).length === 0 ? (
            <div className="p-4"><EmptyState title="No activity recorded" /></div>
          ) : (
            <table className="x-table">
              <thead><tr><th>Action</th><th>Actor</th><th>Time</th></tr></thead>
              <tbody>
                {(audit.data ?? []).map((a) => (
                  <tr key={a.id}>
                    <td><Badge>{a.action}</Badge></td>
                    <td className="text-fg-soft">{a.actor}</td>
                    <td className="whitespace-nowrap text-fg-muted">{formatUTC(a.timestamp)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Panel>
      </div>
    </>
  );
}
