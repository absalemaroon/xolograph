import { Plus } from 'lucide-react';
import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { EntityPortfolio } from '@/api/types';
import { Badge, Button, EmptyState, ErrorBanner, HashText, Panel, Stat } from '@/components/ui';
import { formatUTC } from '@/lib/format';
import { AttachModal } from '../explorer/AttachModal';
import { ProvenanceList } from '../explorer/Provenance';
import { AttributionBadge } from '../entities/AttributionBadge';
import { networkLabel, statusLabel } from '../entities/constants';
import { AssetTable } from './HoldingsPanel';
import { formatUSD } from './format';

/** Values every non-rejected wallet separately and reports totals by attribution strength. */
export function EntityPortfolioTab({ entityId, walletCount }: { entityId: string; walletCount: number }) {
  const [data, setData] = useState<EntityPortfolio | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [attach, setAttach] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(null);
    try { setData(await api.portfolio.entity(entityId)); } catch (e) { setError(e instanceof Error ? e.message : String(e)); } finally { setLoading(false); }
  };

  if (walletCount === 0) return <EmptyState title="No wallets to value" hint="Add wallets to this entity first. Each claim is valued on its own." />;

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs text-fg-muted">Loads balances and reference prices for each wallet, one after another. This can take a while for many wallets.</p>
        <div className="flex gap-2">
          {data && <Button size="sm" icon={<Plus size={12} />} onClick={() => setAttach(true)}>Attach to case</Button>}
          <Button variant={data ? 'secondary' : 'primary'} loading={loading} onClick={() => void load()}>{data ? 'Refresh' : 'Load portfolio'}</Button>
        </div>
      </div>
      {error && <ErrorBanner message={error} />}
      {data && (
        <>
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
            <Stat label="Verified + publicly reported" value={`≈ ${formatUSD(data.strongUsd)}`} hint="Wallets with a sourced ownership claim" />
            <Stat label="All non-rejected claims" value={`≈ ${formatUSD(data.allUsd)}`} hint="Includes suspected and investigator-attributed" />
            <Stat label="As of" value={<span className="text-sm">{formatUTC(data.asOf)}</span>} />
          </div>
          <Panel title="Totals by attribution status">
            <ul className="space-y-1.5 text-xs">
              {Object.entries(data.byStatusUsd).map(([st, v]) => (
                <li key={st} className="flex items-center justify-between"><AttributionBadge status={st} /><span className="tabular-nums">≈ {formatUSD(v)}</span></li>
              ))}
              {Object.keys(data.byStatusUsd).length === 0 && <li className="text-fg-muted">Nothing could be valued.</li>}
            </ul>
          </Panel>
          {data.wallets.map((w) => (
            <Panel key={w.addressId} title={`${networkLabel(w.network)} · ${w.label || 'wallet'}`} flush actions={<div className="flex items-center gap-2"><AttributionBadge status={w.status} /><Badge>{w.confidence}</Badge></div>}>
              <div className="border-b border-line px-4 py-2"><HashText value={w.address} head={10} tail={8} /></div>
              {w.error ? <div className="p-4"><ErrorBanner message={w.error} /></div> : w.portfolio && (
                <>
                  <AssetTable assets={w.portfolio.assets} />
                  <div className="flex justify-between border-t border-line px-4 py-2 text-xs">
                    <span className="text-fg-muted">{w.portfolio.pricedCount} of {w.portfolio.assets.length} assets priced</span>
                    <span className="font-semibold tabular-nums">≈ {formatUSD(w.portfolio.pricedValueUsd)}</span>
                  </div>
                </>
              )}
            </Panel>
          ))}
          {data.skipped.length > 0 && <Panel title="Not valued"><ul className="space-y-1 text-[11px]">{data.skipped.map((s) => <li key={s}>{s}</li>)}</ul></Panel>}
          <ul className="space-y-1.5 border border-line bg-raised p-3 text-[11px]">
            {data.notes.map((n) => <li key={n}>▲ {n}</li>)}
            <li>▲ Prices are third-party market references (not execution prices). {statusLabel('VERIFIED')} means a sourced claim of ownership, not a guarantee.</li>
          </ul>
          <ProvenanceList items={data.wallets.flatMap((w) => w.portfolio?.provenance ?? [])} />
        </>
      )}
      {attach && data && <AttachModal spec={{ kind: 'PORTFOLIO', refType: 'entity', refId: data.entityId, defaultTitle: `Portfolio of ${data.name}`, snapshot: data }} onClose={() => setAttach(false)} />}
    </div>
  );
}
