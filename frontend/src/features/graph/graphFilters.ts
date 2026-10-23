import type { AddressAttribution } from '@/api/types';
import type { EdgeAttrs } from './graphStore';

export interface GraphFilters { minTransfers: number; asset: 'all' | 'native' | 'token'; highlightEntities: boolean }
export const DEFAULT_FILTERS: GraphFilters = { minTransfers: 1, asset: 'all', highlightEntities: true };

/** An edge is shown when it meets the minimum transfer count and the asset filter. */
export function edgePasses(a: EdgeAttrs, f: GraphFilters): boolean {
  if (a.count < f.minTransfers) return false;
  const hasNative = a.nativeWei !== '0';
  if (f.asset === 'native') return hasNative;
  if (f.asset === 'token') return a.tokenCount > 0;
  return true;
}

const STRENGTH = ['VERIFIED', 'PUBLICLY_REPORTED', 'INVESTIGATOR_ATTRIBUTED', 'SUSPECTED', 'DISPUTED', 'UNKNOWN'];

/** The claim to show on a node: the strongest one that has not been rejected. Rejected claims never label a node. */
export function displayAttribution(list: AddressAttribution[] | undefined): AddressAttribution | null {
  if (!list) return null;
  let best: AddressAttribution | null = null;
  for (const a of list) {
    const r = STRENGTH.indexOf(a.status);
    if (r < 0) continue;
    if (!best || r < STRENGTH.indexOf(best.status)) best = a;
  }
  return best;
}

export function nodeLabel(list: AddressAttribution[] | undefined, fallback: string): string {
  const d = displayAttribution(list);
  if (!d) return fallback;
  const others = (list ?? []).filter((x) => x.status !== 'REJECTED' && x.entityId !== d.entityId).length;
  return others > 0 ? `${d.name} +${others}` : d.name;
}
