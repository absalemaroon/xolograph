import { Network, Plus, Tag } from 'lucide-react';
import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { NetworkInfo, Provenance } from '@/api/types';
import { Badge, Button, ErrorBanner, Panel, Spinner, Stat } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';
import { formatUnits } from '@/lib/units';
import { useNav } from '@/state/navStore';
import { ActivityTable } from './ActivityTable';
import { AttachModal } from './AttachModal';
import type { AttachSpec } from './AttachModal';
import { FlowsPanel } from '../portfolio/FlowsPanel';
import { HoldingsPanel } from '../portfolio/HoldingsPanel';
import { EntityAttributions } from '../entities/EntityAttributions';
import { LabelModal } from './LabelModal';
import { ProvenanceList } from './Provenance';

export function AddressPanel({ address, network, info }: { address: string; network: string; info?: NetworkInfo }) {
  const openGraph = useNav((s) => s.openGraph);
  const summary = useAsync(() => api.chain.address(address, network), [address, network]);
  const isEth = network === 'ethereum-mainnet';
  const labels = useAsync(() => (isEth ? api.labels.list({ address }) : Promise.resolve([])), [address, isEth]);
  const [tab, setTab] = useState<'native' | 'token'>('native');
  const [attach, setAttach] = useState<AttachSpec | null>(null);
  const [labeling, setLabeling] = useState(false);
  const [pages, setPages] = useState<Provenance[]>([]);
  const addProv = (p: Provenance) => setPages((cur) => (cur.some((c) => c.payloadSha256 === p.payloadSha256 && c.query === p.query) ? cur : [...cur, p]));

  if (summary.loading && !summary.data) return <Spinner label="Querying chain" />;
  if (summary.error || !summary.data) return <ErrorBanner message={summary.error ?? 'No data'} onRetry={summary.reload} />;
  const s = summary.data;

  return (
    <div className="space-y-5">
      <Panel
        title="Address"
        actions={
          <>
            {isEth && <Button size="sm" icon={<Network size={12} />} onClick={() => openGraph(s.address)}>Graph</Button>}
            {isEth && <Button size="sm" icon={<Tag size={12} />} onClick={() => setLabeling(true)}>Case label</Button>}
            <Button size="sm" variant="primary" icon={<Plus size={12} />} onClick={() => setAttach({ kind: 'ADDRESS', refType: 'address', refId: s.address, defaultTitle: `Address ${s.address.slice(0, 10)}…`, snapshot: { summary: s, network, retrievedBy: 'xolograph-explorer' } })}>Attach to case</Button>
          </>
        }
      >
        <div className="flex flex-wrap items-center gap-3">
          <code className="break-all text-sm font-semibold">{s.address}</code>
          <Badge tone="outline">{info?.name ?? s.network}</Badge>
          <Badge tone="outline">{s.isContract ? (info?.family === 'SOLANA' ? 'Program' : 'Contract') : info?.family === 'EVM' ? 'EOA' : 'Account'}</Badge>
        </div>
        <div className="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-3">
          <Stat label={`Balance (${s.nativeSymbol})`} value={formatUnits(s.balanceWei, s.nativeDecimals, 8)} hint={info?.family === 'BITCOIN' ? 'Confirmed only' : 'Latest state'} />
          {(info?.family ?? 'EVM') === 'EVM' && <Stat label="Nonce" value={s.nonce} hint={s.isContract ? 'Contract nonce' : 'Outgoing transactions'} />}
          <Stat label="Third-party labels" value={s.thirdPartyLabels.length} />
        </div>
        {s.thirdPartyLabels.length > 0 && (
          <div className="mt-4 flex flex-wrap items-center gap-2 text-[11px]">
            <span className="text-fg-muted">Labels from {s.thirdPartyLabels[0].source} (unverified third party):</span>
            {s.thirdPartyLabels.map((l, i) => <Badge key={i}>{l.name}</Badge>)}
          </div>
        )}
        {s.notes.map((n, i) => <p key={i} className="mt-3 text-[11px] text-fg-muted">{n}</p>)}
      </Panel>

      <EntityAttributions address={s.address} network={network} />

      <HoldingsPanel key={`h-${network}-${s.address}`} address={s.address} network={network} />
      {(!info || info.hasHistory) && <FlowsPanel key={`f-${network}-${s.address}`} address={s.address} network={network} />}

      {(labels.data ?? []).length > 0 && (
        <Panel title="Analyst labels (your cases)" flush>
          <table className="x-table">
            <thead><tr><th>Name</th><th>Source</th><th>Confidence</th><th>Basis</th><th>Case</th><th>Added</th></tr></thead>
            <tbody>
              {(labels.data ?? []).map((l) => (
                <tr key={l.id}>
                  <td className="font-medium">{l.name}<div className="text-[10px] text-fg-muted">{l.category}</div></td>
                  <td><Badge>{l.source}</Badge></td>
                  <td><Badge tone={l.confidence === 'HIGH' ? 'outline' : 'neutral'}>{l.confidence}</Badge></td>
                  <td className="max-w-xs text-fg-soft">{l.basis}</td>
                  <td className="text-fg-soft">{l.caseTitle}</td>
                  <td className="whitespace-nowrap text-fg-muted">{formatUTC(l.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Panel>
      )}

      <div>
        <div role="tablist" aria-label="Activity" className="mb-3 flex border-b border-line">
          {(info?.family === 'BITCOIN' ? ([['native', `${s.nativeSymbol} transactions`]] as const) : ([['native', `${s.nativeSymbol} transfers`], ['token', info?.family === 'SOLANA' ? 'SPL token transfers' : 'Token transfers']] as const)).map(([k, label]) => (
            <button key={k} role="tab" type="button" aria-selected={tab === k} onClick={() => setTab(k)}
              className={`-mb-px border-b-2 px-4 py-2.5 text-xs uppercase tracking-[0.14em] ${tab === k ? 'border-fg font-semibold' : 'border-transparent text-fg-muted hover:text-fg'}`}>{label}</button>
          ))}
        </div>
        {info && !info.hasHistory ? (
          <p className="border border-line bg-raised p-4 text-xs text-fg-muted">No address-history source is configured for {info.name}. Balance and transaction lookups still work. Set XOLOGRAPH_HISTORY_{network.toUpperCase().replace(/-/g, '_')} to a Blockscout v2 URL to enable history.</p>
        ) : (
          <>
            <ActivityTable key={`${network}-${address}-${tab}`} address={s.address} network={network} kind={tab} onAttach={setAttach} onProvenance={addProv} />
            <p className="mt-2 text-[11px] text-fg-muted">History comes from third-party indexes or public nodes and may be incomplete or lag the chain head. Absence of a transfer is not proof it did not occur.</p>
          </>
        )}
      </div>

      <ProvenanceList items={[...s.provenance, ...pages]} />
      {attach && <AttachModal spec={attach} onClose={() => setAttach(null)} />}
      {labeling && <LabelModal defaultAddress={s.address} onClose={() => setLabeling(false)} onSaved={labels.reload} />}
    </div>
  );
}
