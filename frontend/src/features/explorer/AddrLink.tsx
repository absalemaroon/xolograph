import { shortAddr } from '@/lib/units';
import { useNav } from '@/state/navStore';

export function AddrLink({ address }: { address: string }) {
  const openExplorer = useNav((s) => s.openExplorer);
  if (!address) return <span className="text-fg-muted">—</span>;
  return (
    <button type="button" title={address} onClick={() => openExplorer(address)} className="font-mono text-[11px] underline-offset-2 hover:underline">
      {shortAddr(address)}
    </button>
  );
}
