import { shortAddr } from '@/lib/units';
import { useNav } from '@/state/navStore';

export function AddrLink({ address, network }: { address: string; network?: string }) {
  const openExplorer = useNav((s) => s.openExplorer);
  if (!address) return <span className="text-fg-muted">—</span>;
  if (address === 'coinbase') return <span className="text-[11px] text-fg-muted">coinbase</span>;
  return (
    <button type="button" title={address} onClick={() => openExplorer(address, network)} className="font-mono text-[11px] underline-offset-2 hover:underline">
      {shortAddr(address)}
    </button>
  );
}
