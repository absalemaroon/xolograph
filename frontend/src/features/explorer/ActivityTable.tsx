import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '@/api/endpoints';
import type { Provenance, Transfer } from '@/api/types';
import { Badge, Button, EmptyState, ErrorBanner, HashText, Panel, Spinner } from '@/components/ui';
import { formatUTC } from '@/lib/format';
import { formatUnits } from '@/lib/units';
import { AddrLink } from './AddrLink';
import type { AttachSpec } from './AttachModal';

export function ActivityTable({ address, network = 'ethereum-mainnet', kind, onAttach, onProvenance }: {
  address: string; network?: string; kind: 'native' | 'token'; onAttach: (s: AttachSpec) => void; onProvenance: (p: Provenance) => void;
}) {
  const [items, setItems] = useState<Transfer[]>([]);
  const [cursor, setCursor] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const gen = useRef(0);

  const load = useCallback(async (next: string, reset: boolean) => {
    const my = ++gen.current;
    setLoading(true);
    setError(null);
    try {
      const page = await api.chain.activity(address, kind, next || undefined, network);
      if (my !== gen.current) return;
      setItems((cur) => (reset ? page.items : [...cur, ...page.items]));
      setCursor(page.nextCursor);
      onProvenance(page.provenance);
    } catch (e) {
      if (my === gen.current) setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (my === gen.current) setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [address, kind, network]);

  useEffect(() => { setItems([]); setCursor(''); void load('', true); }, [load]);

  const dir = (t: Transfer) => (t.from === address && t.to === address ? 'SELF' : t.from === address ? 'OUT' : 'IN');

  return (
    <Panel flush>
      {error && <div className="p-4"><ErrorBanner message={error} onRetry={() => void load(cursor, items.length === 0)} /></div>}
      {items.length === 0 && loading ? <Spinner /> : items.length === 0 && !error ? (
        <div className="p-4"><EmptyState title="No activity returned" hint="The history source returned nothing for this address. This does not prove the address is inactive." /></div>
      ) : items.length > 0 && (
        <div className="overflow-x-auto">
          <table className="x-table">
            <thead><tr><th>Time</th><th>Tx</th><th>Dir</th><th>From</th><th>To</th><th className="text-right">Amount</th><th>Status</th><th /></tr></thead>
            <tbody>
              {items.map((t) => (
                <tr key={`${t.txHash}:${t.logIndex}`}>
                  <td className="whitespace-nowrap text-fg-muted">{formatUTC(t.timestamp)}</td>
                  <td><HashText value={t.txHash} head={8} tail={4} /></td>
                  <td><Badge tone={dir(t) === 'OUT' ? 'outline' : 'neutral'}>{dir(t)}</Badge></td>
                  <td><AddrLink address={t.from} network={network} /></td>
                  <td><AddrLink address={t.to} network={network} /></td>
                  <td className="whitespace-nowrap text-right tabular-nums">
                    {t.kind === 'ERC721' || t.kind === 'ERC1155' ? `#${t.tokenId ?? '?'}` : t.decimals < 0 ? `${t.value} (raw)` : formatUnits(t.value, t.decimals)} <span className="text-fg-muted">{t.symbol || (t.kind === 'SPL' ? 'SPL token' : t.decimals < 0 ? 'raw units' : '')}</span>
                  </td>
                  <td>{t.status === 'FAILED' ? <Badge tone="alert">▲ Failed</Badge> : <span className="text-[10px] uppercase text-fg-muted">{t.status}</span>}</td>
                  <td className="text-right"><Button size="sm" variant="ghost" onClick={() => onAttach({ kind: 'TRANSACTION', refType: 'transaction', refId: t.txHash, defaultTitle: `${kind === 'native' ? 'Transfer' : t.symbol || 'Token transfer'} in ${t.txHash.slice(0, 10)}…`, snapshot: { ...t, network } })}>Attach</Button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {cursor && (
        <div className="flex justify-center border-t border-line p-3"><Button loading={loading} onClick={() => void load(cursor, false)}>Load more</Button></div>
      )}
    </Panel>
  );
}
