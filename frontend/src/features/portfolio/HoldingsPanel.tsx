import { Plus } from 'lucide-react';
import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { PortfolioAsset, WalletPortfolio } from '@/api/types';
import { Badge, Button, ErrorBanner, HashText, Panel } from '@/components/ui';
import { formatUTC } from '@/lib/format';
import { formatUnits } from '@/lib/units';
import { AttachModal } from '../explorer/AttachModal';
import { ProvenanceList } from '../explorer/Provenance';
import { formatUSD } from './format';

export const assetLabel = (a: PortfolioAsset): string => a.symbol || (a.kind === 'SPL' ? 'SPL token' : a.name || 'Unknown token');

export function AssetTable({ assets }: { assets: PortfolioAsset[] }) {
  return (
    <table className="x-table">
      <thead><tr><th>Asset</th><th className="text-right">Balance</th><th className="text-right">Reference price</th><th className="text-right">Value</th></tr></thead>
      <tbody>
        {assets.map((a, i) => (
          <tr key={`${a.kind}-${a.tokenAddress ?? ''}-${i}`}>
            <td><div className="font-medium">{assetLabel(a)}</div>{a.tokenAddress && <HashText value={a.tokenAddress} head={8} tail={6} />}</td>
            <td className="text-right tabular-nums">{a.decimals < 0 ? `${a.balance} (raw)` : formatUnits(a.balance, a.decimals, 6)}</td>
            <td className="text-right tabular-nums">{a.priced ? <>{formatUSD(a.priceUsd)}<div className="text-[10px] text-fg-muted">{a.priceAt ? formatUTC(a.priceAt) : ''}</div></> : <Badge>Unpriced</Badge>}</td>
            <td className="text-right tabular-nums">{a.priced ? formatUSD(a.valueUsd) : '—'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

/** Holdings of one wallet. Loaded on request so browsing does not spend rate-limited price and index calls. */
export function HoldingsPanel({ address, network }: { address: string; network: string }) {
  const [data, setData] = useState<WalletPortfolio | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [attach, setAttach] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(null);
    try { setData(await api.portfolio.wallet(address, network)); } catch (e) { setError(e instanceof Error ? e.message : String(e)); } finally { setLoading(false); }
  };

  return (
    <Panel title="Holdings and value" flush={!!data} actions={<div className="flex gap-2">
      {data && <Button size="sm" icon={<Plus size={12} />} onClick={() => setAttach(true)}>Attach to case</Button>}
      <Button size="sm" variant={data ? 'secondary' : 'primary'} loading={loading} onClick={() => void load()}>{data ? 'Refresh' : 'Load holdings'}</Button>
    </div>}>
      {error && <ErrorBanner message={error} />}
      {!data && !error && <p className="text-xs text-fg-muted">Lists native and token balances and values the ones that have a market reference price. Loaded on request.</p>}
      {data && (
        <>
          <AssetTable assets={data.assets} />
          <div className="space-y-2 border-t border-line p-4 text-xs">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <span className="text-fg-muted">Priced total ({data.pricedCount} of {data.assets.length} assets)</span>
              <span className="text-base font-semibold tabular-nums">≈ {formatUSD(data.pricedValueUsd)}</span>
            </div>
            {data.unpricedCount > 0 && <p>▲ {data.unpricedCount} asset(s) have no reference price and are not in this total. They are not worth zero; their value is unknown.</p>}
            <ul className="space-y-1 text-[11px] text-fg-muted">{data.notes.map((n) => <li key={n}>{n}</li>)}</ul>
            <p className="text-[11px] text-fg-muted">As of {formatUTC(data.asOf)}. Prices are third-party market references, not execution prices.</p>
          </div>
          <div className="p-4 pt-0"><ProvenanceList items={data.provenance} /></div>
        </>
      )}
      {attach && data && <AttachModal spec={{ kind: 'PORTFOLIO', refType: 'address', refId: data.address, defaultTitle: `Holdings of ${data.address.slice(0, 10)}…`, snapshot: data }} onClose={() => setAttach(false)} />}
    </Panel>
  );
}
