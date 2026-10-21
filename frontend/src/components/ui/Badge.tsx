import type { ReactNode } from 'react';

type Tone = 'neutral' | 'outline' | 'solid' | 'alert';
const tones: Record<Tone, string> = {
  neutral: 'border-line text-fg-soft',
  outline: 'border-fg text-fg',
  solid: 'border-fg bg-fg text-fg-inverse',
  alert: 'border-fg bg-fg text-fg-inverse font-bold underline decoration-dotted underline-offset-2',
};

export function Badge({ tone = 'neutral', children }: { tone?: Tone; children: ReactNode }) {
  return (
    <span className={`inline-flex items-center gap-1 whitespace-nowrap border px-1.5 py-0.5 text-[10px] uppercase tracking-wider ${tones[tone]}`}>
      {children}
    </span>
  );
}

/** Status is conveyed by symbol + text + weight, never by hue (monochrome UI). */
export function StatusBadge({ status }: { status: string }) {
  switch (status) {
    case 'VERIFIED': return <Badge tone="outline">● Verified</Badge>;
    case 'IMPORTED': return <Badge>○ Imported</Badge>;
    case 'FAILED': return <Badge tone="alert">▲ Failed</Badge>;
    case 'OPEN': return <Badge tone="outline">● Open</Badge>;
    case 'CLOSED': return <Badge>■ Closed</Badge>;
    case 'ARCHIVED': return <Badge>– Archived</Badge>;
    default: return <Badge>{status}</Badge>;
  }
}
