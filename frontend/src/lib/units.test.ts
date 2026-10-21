import { describe, expect, it } from 'vitest';
import { formatUnits, shortAddr } from './units';

describe('formatUnits', () => {
  it('formats without float error', () => {
    expect(formatUnits('1500000', 6)).toBe('1.5');
    expect(formatUnits('1000000000000000000', 18)).toBe('1');
    expect(formatUnits('1234567000000000000000', 18)).toBe('1,234.567');
    expect(formatUnits('123456789012345678901234567890', 18)).toBe('123,456,789,012.345678');
  });
  it('flags dust and passes through unknown decimals', () => {
    expect(formatUnits('1', 18)).toBe('<0.000001');
    expect(formatUnits('0', 18)).toBe('0');
    expect(formatUnits('5', -1)).toBe('5');
    expect(formatUnits('abc', 18)).toBe('abc');
  });
  it('shortens addresses', () => {
    expect(shortAddr('0x' + 'a'.repeat(40))).toBe('0xaaaa…aaaa');
  });
});
