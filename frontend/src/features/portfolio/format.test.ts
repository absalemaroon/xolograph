import { describe, expect, it } from 'vitest';
import { formatUnits } from '@/lib/units';
import { formatUSD, signedUSD, signedUnits } from './format';

describe('portfolio formatting', () => {
  it('formats USD and never invents a number', () => {
    expect(formatUSD(1234.5)).toBe('$1,234.50');
    expect(formatUSD(undefined)).toBe('—');
    expect(formatUSD(Number.NaN)).toBe('—');
    expect(formatUSD(0.00123)).toBe('$0.00123');
  });
  it('signs flows', () => {
    expect(signedUSD(10)).toBe('+$10.00');
    expect(signedUSD(-10)).toBe('-$10.00');
    expect(signedUnits('2500000000000000000', 18, formatUnits)).toBe('+2.5');
    expect(signedUnits('-500000000000000000', 18, formatUnits)).toBe('-0.5');
    expect(signedUnits('0', 18, formatUnits)).toBe('0');
  });
});
