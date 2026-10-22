import { Badge } from '@/components/ui';
import { statusLabel } from './constants';

/** Conveyed by symbol + text, never by hue alone. Weaker claims look visibly weaker. */
const SYMBOL: Record<string, string> = {
  VERIFIED: '●', PUBLICLY_REPORTED: '◐', INVESTIGATOR_ATTRIBUTED: '◑', SUSPECTED: '?', DISPUTED: '▲', REJECTED: '✕', UNKNOWN: '○',
};

export function AttributionBadge({ status }: { status: string }) {
  const tone = status === 'VERIFIED' ? 'solid' : status === 'DISPUTED' ? 'alert' : status === 'PUBLICLY_REPORTED' || status === 'INVESTIGATOR_ATTRIBUTED' ? 'outline' : 'neutral';
  return <Badge tone={tone}>{SYMBOL[status] ?? '○'} {statusLabel(status)}</Badge>;
}
