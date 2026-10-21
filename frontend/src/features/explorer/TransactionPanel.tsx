import { Plus } from 'lucide-react';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { api } from '@/api/endpoints';
import { Badge, Button, ErrorBanner, HashText, Panel, Spinner, StatusBadge } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';
import { formatUnits } from '@/lib/units';
import { AddrLink } from './AddrLink';
import { AttachModal } from './AttachModal';
import { ProvenanceList } from './Provenance';

export function TransactionPanel({ hash }: { hash: string }) {
  const { data, error, loading, reload } = useAsync(() => api.chain.tx(hash), [hash]);
  const [attach, setAttach] = useState(false);
  if (loading && !data) return <Spinner label="Querying chain" />;
  if (error || !data) return <ErrorBanner message={error ?? 'No data'} onRetry={reload} />;
  const t = data;
  const row = (k: string, v: ReactNode) => (<div className="flex flex-wrap justify-between gap-3 border-b border-[var(--border-subtle)] py-2 text-xs last:border-0"><dt className="text-fg-muted">{k}</dt><dd className="min-w-0 break-all text-right">{v}</dd></div>);

  return (
    <div className="space-y-5">
      <Panel title="Transaction" actions={<Button size="sm" variant="primary" icon={<Plus size={12} />} onClick={() => setAttach(true)}>Attach to case</Button>}>
        <div className="mb-3 flex flex-wrap items-center gap-3"><code className="break-all text-sm font-semibold">{t.hash}</code>{t.status === 'SUCCESS' ? <Badge tone="outline">● Success</Badge> : t.status === 'FAILED' ? <Badge tone="alert">▲ Failed</Badge> : <StatusBadge status={t.status} />}</div>
        <dl>
          {row('Block', t.blockNumber ? t.blockNumber.toLocaleString('en-US') : 'pending')}
          {row('Time', formatUTC(t.timestamp))}
          {row('From', <AddrLink address={t.from} />)}
          {row('To', t.to ? <AddrLink address={t.to} /> : t.contractCreated ? <span>Contract created: <AddrLink address={t.contractCreated} /></span> : '—')}
          {row('Value', `${formatUnits(t.valueWei, 18, 8)} ETH`)}
          {row('Fee', `${formatUnits(t.feeWei, 18, 8)} ETH`)}
          {row('Gas used', t.gasUsed.toLocaleString('en-US'))}
        </dl>
        {t.notes.map((n, i) => <p key={i} className="mt-3 text-[11px] text-fg-muted">{n}</p>)}
      </Panel>
      <Panel title={`Token transfers (${t.tokenTransfers.length})`} flush>
        {t.tokenTransfers.length === 0 ? <p className="p-4 text-xs text-fg-muted">No ERC-20/721 Transfer events in this transaction.</p> : (
          <table className="x-table">
            <thead><tr><th>Std</th><th>Token</th><th>From</th><th>To</th><th className="text-right">Amount</th></tr></thead>
            <tbody>
              {t.tokenTransfers.map((x) => (
                <tr key={x.logIndex}>
                  <td><Badge>{x.standard}</Badge></td>
                  <td><div>{x.symbol || '—'}</div><HashText value={x.token} head={8} tail={4} /></td>
                  <td><AddrLink address={x.from} /></td>
                  <td><AddrLink address={x.to} /></td>
                  <td className="text-right tabular-nums">{x.standard === 'ERC721' ? `#${x.tokenId}` : x.decimals < 0 ? `${x.value} (raw)` : formatUnits(x.value, x.decimals)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Panel>
      <ProvenanceList items={t.provenance} />
      {attach && <AttachModal spec={{ kind: 'TRANSACTION', refType: 'transaction', refId: t.hash, defaultTitle: `Transaction ${t.hash.slice(0, 10)}…`, snapshot: t }} onClose={() => setAttach(false)} />}
    </div>
  );
}
