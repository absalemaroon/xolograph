import { describe, expect, it } from 'vitest';
import type { AddressAttribution } from '@/api/types';
import { DEFAULT_FILTERS, displayAttribution, edgePasses, nodeLabel } from './graphFilters';

const at = (status: AddressAttribution['status'], name = 'E', entityId = 'e1'): AddressAttribution => ({ entityId, name, type: 'OTHER', status, confidence: 'LOW' });

describe('graph filters', () => {
  it('filters edges by count and asset', () => {
    const e = { count: 2, nativeWei: '0', tokenCount: 2, size: 1 };
    expect(edgePasses(e, DEFAULT_FILTERS)).toBe(true);
    expect(edgePasses(e, { ...DEFAULT_FILTERS, minTransfers: 3 })).toBe(false);
    expect(edgePasses(e, { ...DEFAULT_FILTERS, asset: 'native' })).toBe(false);
    expect(edgePasses(e, { ...DEFAULT_FILTERS, asset: 'token' })).toBe(true);
  });
  it('labels nodes with the strongest non-rejected claim only', () => {
    expect(displayAttribution([at('REJECTED')])).toBeNull();
    expect(displayAttribution([at('SUSPECTED', 'A', 'a'), at('VERIFIED', 'B', 'b')])?.name).toBe('B');
    expect(nodeLabel([at('REJECTED')], '0xab…cd')).toBe('0xab…cd');
    expect(nodeLabel([at('VERIFIED', 'B', 'b'), at('SUSPECTED', 'A', 'a')], 'x')).toBe('B +1');
  });
});
