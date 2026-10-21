import { Search } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { Button, EmptyState, Input, PageHeader } from '@/components/ui';
import { isEthAddress, isEthTxHash } from '@/lib/validate';
import { useNav } from '@/state/navStore';
import { AddressPanel } from './AddressPanel';
import { TransactionPanel } from './TransactionPanel';

export function ExplorerView() {
  const explorerQuery = useNav((s) => s.explorerQuery);
  const [input, setInput] = useState(explorerQuery ?? '');
  const [query, setQuery] = useState((explorerQuery ?? '').trim());
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (explorerQuery) { setInput(explorerQuery); setQuery(explorerQuery.trim()); setFormError(null); }
  }, [explorerQuery]);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const q = input.trim();
    if (!isEthAddress(q) && !isEthTxHash(q)) { setFormError('Enter a 42-character address (0x…) or a 66-character transaction hash.'); return; }
    setFormError(null);
    setQuery(q);
  };

  return (
    <>
      <PageHeader title="On-chain Explorer" subtitle="Ethereum mainnet. Every value shown is retrieved live and carries its source and payload hash." />
      <form onSubmit={submit} className="mb-5 flex gap-3">
        <div className="flex-1">
          <Input aria-label="Address or transaction hash" value={input} onChange={(e) => setInput(e.target.value)} placeholder="0x… address or transaction hash" spellCheck={false} />
          {formError && <p role="alert" className="mt-1.5 text-[11px]">▲ {formError}</p>}
        </div>
        <Button variant="primary" icon={<Search size={14} />} onClick={submit as unknown as () => void} disabled={!input.trim()}>Search</Button>
        <button type="submit" className="hidden" />
      </form>
      {!query ? (
        <EmptyState title="Search the chain" hint="Look up an address for balance, labels and activity, or a transaction hash for status, fees and token transfers. Attach anything you find to a case with its source snapshot." />
      ) : isEthAddress(query) ? <AddressPanel key={query.toLowerCase()} address={query.toLowerCase()} /> : <TransactionPanel key={query.toLowerCase()} hash={query.toLowerCase()} />}
    </>
  );
}
