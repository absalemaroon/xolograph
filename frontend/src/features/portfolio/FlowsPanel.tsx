import { Plus } from 'lucide-react';
import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { Flows } from '@/api/types';
import { Button, ErrorBanner, Panel, Select, Stat } from '@/components/ui';
import { formatUTC } from '@/lib/format';
import { formatUnits } from '@/lib/units';
import { AttachModal } from '../explorer/AttachModal';
import { ProvenanceList } from '../explorer/Provenance';
import { formatUSD, signedUSD, signedUnits } from './format';

/** Net native-asset movements over the loaded window. Deliberately not called profit/loss. */
export function FlowsPanel({ address, network }: { address: string; network: string }) {
  const [pages, setPages] = useState(2);
  const [data, setData] = useState<Flows | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [attach, setAttach] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(null);
    try { setData(await api.portfolio.flows(address, network, pages)); } catch (e) { setError(e instanceof Error ? e.message : String(e)); } finally { setLoading(false); }
  };

  return (
    <Panel title="Net flows (loaded window)" actions={<div className="flex items-center gap-2">
      <Select aria-label="Pages of activity" className="!w-32" value={pages} onChange={(e) => setPages(Number(e.target.value))}>
        {[1, 2, 3, 4].map((n) => <option key={n} value={n}>{n} page{n > 1 ? 's' : ''}</option>)}
      </Select>
      {data && <Button size="sm" icon={<Plus size={12} />} onClick={() => setAttach(true)}>Attach</Button>}
      <Button size="sm" variant={data ? 'secondary' : 'primary'} loading={loading} onClick={() => void load()}>{data ? 'Reload' : 'Summarise'}</Button>
    </div>}>
      {error && <ErrorBanner message={error} />}
      {!data && !error && <p className="text-xs text-fg-muted">Totals incoming and outgoing native-asset movements over the most recent activity, valued at the price nearest each movement. This is a movement summary, not profit or loss.</p>}
      {data && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <Stat label={`In (${data.symbol})`} value={formatUnits(data.inflow, data.decimals, 6)} hint={formatUSD(data.inflowUsd)} />
            <Stat label={`Out (${data.symbol})`} value={formatUnits(data.outflow, data.decimals, 6)} hint={formatUSD(data.outflowUsd)} />
            <Stat label="Net" value={signedUnits(data.net, data.decimals, formatUnits)} hint={signedUSD(data.netUsd)} />
            <Stat label="Movements" value={data.transferCount} hint={`${data.pagesLoaded} page(s)${data.historyExhausted ? ', full history' : ', partial'}`} />
          </div>
          {data.from && data.to && <p className="text-[11px] text-fg-muted">Window {formatUTC(data.from)} → {formatUTC(data.to)}{data.failedExcluded > 0 ? ` · ${data.failedExcluded} failed transaction(s) excluded` : ''}</p>}
          <ul className="space-y-1.5 border border-line bg-raised p-3 text-[11px]">{data.qualification.map((q) => <li key={q}>▲ {q}</li>)}</ul>
          <ProvenanceList items={data.provenance} />
        </div>
      )}
      {attach && data && <AttachModal spec={{ kind: 'FLOWS', refType: 'address', refId: data.address, defaultTitle: `Net flows of ${data.address.slice(0, 10)}…`, snapshot: data }} onClose={() => setAttach(false)} />}
    </Panel>
  );
}
