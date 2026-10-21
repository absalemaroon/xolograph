import type { Provenance } from '@/api/types';
import { HashText } from '@/components/ui';
import { formatUTC } from '@/lib/format';

export function ProvenanceList({ items }: { items: Provenance[] }) {
  if (items.length === 0) return null;
  return (
    <details className="border border-line bg-raised">
      <summary className="cursor-pointer px-4 py-3 text-[11px] font-medium uppercase tracking-[0.14em] text-fg-soft">
        Data provenance · {items.length} source call{items.length > 1 ? 's' : ''}
      </summary>
      <ul className="divide-y divide-[var(--border-subtle)] border-t border-line">
        {items.map((p, i) => (
          <li key={i} className="space-y-1 px-4 py-3 text-[11px]">
            <div><span className="text-fg-muted">Source </span>{p.source} · {p.endpoint}</div>
            <div className="break-all"><span className="text-fg-muted">Query </span><code>{p.query}</code></div>
            <div><span className="text-fg-muted">Retrieved </span>{formatUTC(p.retrievedAt)}</div>
            <div className="flex items-center gap-2"><span className="text-fg-muted">Raw payload SHA-256 </span><HashText value={p.payloadSha256} head={12} tail={8} /></div>
            <div className="text-fg-muted">{p.coverage}</div>
          </li>
        ))}
      </ul>
    </details>
  );
}
