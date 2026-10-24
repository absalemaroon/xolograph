import { Search } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { api } from '@/api/endpoints';
import { Badge, Button, EmptyState, ErrorBanner, Input, Label, PageHeader, Panel, Select } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { pickTarget, resolveTargets } from '@/lib/detect';
import type { Target } from '@/lib/detect';
import { useNav } from '@/state/navStore';
import { AddressPanel } from './AddressPanel';
import { TransactionPanel } from './TransactionPanel';

const FAMILY_LABEL = { EVM: 'EVM', SOLANA: 'Solana', BITCOIN: 'Bitcoin' } as const;

export function ExplorerView() {
  const { explorerQuery, explorerNetwork, openEntity, openCase } = useNav();
  const networks = useAsync(() => api.chain.networks(), []);
  const [input, setInput] = useState(explorerQuery ?? '');
  const [debounced, setDebounced] = useState('');
  const [target, setTarget] = useState<Target | null>(null);
  const [ran, setRan] = useState(''); // the text the current result was produced from
  const [formError, setFormError] = useState<string | null>(null);

  const list = networks.data ?? [];
  const opts = useMemo(() => resolveTargets(ran, list), [ran, list]);
  const infoOf = (id: string) => list.find((n) => n.id === id);

  const run = (q: string, preferred: string | null) => {
    const ts = resolveTargets(q, list);
    setRan(q);
    if (ts.length === 0) {
      setTarget(null);
      setFormError(list.length === 0 ? 'No networks are available yet.' : 'Not recognised. Paste an EVM address or hash (0x…), a Solana address or signature, or a Bitcoin address or transaction id.');
      return;
    }
    setFormError(null);
    setTarget(pickTarget(ts, preferred));
  };

  // Hand-over from other pages (entity wallets, graph, board): run once the network list is known.
  useEffect(() => {
    if (!explorerQuery || list.length === 0) return;
    setInput(explorerQuery);
    run(explorerQuery.trim(), explorerNetwork);
    useNav.setState({ explorerQuery: null, explorerNetwork: null });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [explorerQuery, list.length]);

  useEffect(() => { const t = window.setTimeout(() => setDebounced(input.trim()), 250); return () => window.clearTimeout(t); }, [input]);
  const local = useAsync(() => (debounced.length >= 2 ? api.search(debounced) : Promise.resolve([])), [debounced]);

  const submit = (e: FormEvent) => { e.preventDefault(); run(input.trim(), target?.network ?? null); };
  const onChain = target?.det.value ? target : null;
  const multiFamily = new Set(opts.map((x) => x.det.family)).size > 1;

  return (
    <>
      <PageHeader title="Explorer" subtitle="Search any supported network, plus your own cases, entities and evidence. Every on-chain value is retrieved live and carries its source and payload hash." />
      <form onSubmit={submit} className="mb-5 flex gap-3">
        <div className="flex-1">
          <Input aria-label="Search" value={input} onChange={(e) => setInput(e.target.value)} placeholder="Address or transaction (EVM, Solana, Bitcoin), or a name, case or file hash" spellCheck={false} />
          {formError && <p role="alert" className="mt-1.5 text-[11px]">▲ {formError}</p>}
        </div>
        <Button variant="primary" icon={<Search size={14} />} onClick={submit as unknown as () => void} disabled={!input.trim()}>Search</Button>
        <button type="submit" className="hidden" />
      </form>
      {networks.error && <div className="mb-4"><ErrorBanner message={networks.error} onRetry={networks.reload} /></div>}

      {onChain && (
        <div className="mb-5 flex flex-wrap items-end gap-4">
          <div className="w-72">
            <Label htmlFor="ex-net" hint={opts.length > 1 ? 'pick the network' : undefined}>Network</Label>
            <Select id="ex-net" value={`${onChain.det.family}:${onChain.det.kind}:${onChain.network}`}
              onChange={(e) => { const t = opts.find((x) => `${x.det.family}:${x.det.kind}:${x.network}` === e.target.value); if (t) setTarget(t); }}>
              {opts.map((t) => <option key={`${t.det.family}:${t.det.kind}:${t.network}`} value={`${t.det.family}:${t.det.kind}:${t.network}`}>{infoOf(t.network)?.name ?? t.network}{multiFamily ? ` (${FAMILY_LABEL[t.det.family]})` : ''}</option>)}
            </Select>
          </div>
          <div className="pb-2 text-[11px] text-fg-muted">
            {onChain.det.family === 'EVM' ? 'The same EVM address exists on every EVM chain; check you are on the right one.' : onChain.det.kind === 'address' && multiFamily ? 'This string is valid on more than one network family; pick the one you mean.' : ''}
          </div>
        </div>
      )}

      {(local.data ?? []).length > 0 && (
        <div className="mb-5">
          <Panel title="Your records" flush>
            <ul className="divide-y divide-[var(--border-subtle)]">
              {(local.data ?? []).map((h) => (
                <li key={`${h.kind}:${h.id}`} className="flex items-center justify-between gap-3 px-4 py-2.5 text-xs">
                  <div className="flex min-w-0 items-center gap-3"><Badge>{h.kind}</Badge><span className="truncate font-medium">{h.title}</span><span className="truncate text-fg-muted">{h.subtitle}</span></div>
                  <Button size="sm" onClick={() => (h.kind === 'ENTITY' ? openEntity(h.id) : openCase(h.caseId ?? h.id))}>Open</Button>
                </li>
              ))}
            </ul>
          </Panel>
        </div>
      )}
      {local.error && <p className="mb-4 text-[11px]">Your records could not be searched: {local.error}</p>}

      {!onChain ? (
        <EmptyState title="Search the chain" hint="Paste an address or transaction from Ethereum or another EVM chain, Solana, or Bitcoin. Look up balance, activity and entity records, then attach what you find to a case with its source snapshot." />
      ) : onChain.det.kind === 'address' ? (
        <AddressPanel key={`${onChain.network}:${onChain.det.value}`} address={onChain.det.value} network={onChain.network} info={infoOf(onChain.network)} />
      ) : (
        <TransactionPanel key={`${onChain.network}:${onChain.det.value}`} hash={onChain.det.value} network={onChain.network} info={infoOf(onChain.network)} />
      )}
    </>
  );
}
