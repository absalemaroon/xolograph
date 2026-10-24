import { describe, expect, it } from 'vitest';
import { detect, pickTarget, resolveTargets } from './detect';

const NETS = [
  { id: 'ethereum-mainnet', family: 'EVM' }, { id: 'base', family: 'EVM' }, { id: 'solana', family: 'SOLANA' }, { id: 'bitcoin', family: 'BITCOIN' },
];

describe('universal search detection', () => {
  it('recognises EVM addresses and hashes', () => {
    expect(detect('0x' + 'A'.repeat(40))).toEqual([{ kind: 'address', family: 'EVM', value: '0x' + 'a'.repeat(40) }]);
    expect(detect('0x' + 'a'.repeat(64))[0]).toMatchObject({ kind: 'tx', family: 'EVM' });
  });
  it('recognises Bitcoin and Solana forms', () => {
    expect(detect('bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq')[0]).toMatchObject({ kind: 'address', family: 'BITCOIN' });
    expect(detect('ab'.repeat(32))[0]).toMatchObject({ kind: 'tx', family: 'BITCOIN' });
    expect(detect('So11111111111111111111111111111111111111112')).toEqual([{ kind: 'address', family: 'SOLANA', value: 'So11111111111111111111111111111111111111112' }]);
    expect(detect('2'.repeat(88))[0]).toMatchObject({ kind: 'tx', family: 'SOLANA' });
  });
  it('reports ambiguity instead of guessing', () => {
    const both = detect('1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2');
    expect(both.map((d) => d.family)).toEqual(['BITCOIN']); // 34 chars: valid Bitcoin; too short to be a Solana key
    const ambiguous = detect('11' + 'B'.repeat(32));
    expect(ambiguous.map((d) => d.family).sort()).toEqual(['BITCOIN', 'SOLANA']);
  });
  it('rejects junk', () => {
    expect(detect('hello world')).toEqual([]);
    expect(detect('0x123')).toEqual([]);
    expect(detect('')).toEqual([]);
  });
  it('prefers the requested network, then Ethereum', () => {
    const t = resolveTargets('0x' + 'a'.repeat(40), NETS);
    expect(t.map((x) => x.network)).toEqual(['ethereum-mainnet', 'base']);
    expect(pickTarget(t, 'base')?.network).toBe('base');
    expect(pickTarget(t, 'solana')?.network).toBe('ethereum-mainnet');
    expect(pickTarget([], null)).toBeNull();
  });
});
