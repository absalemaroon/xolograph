import { describe, expect, it } from 'vitest';
import { formatBytes, formatUTC, shortHash } from './format';
import { isEthAddress, isEthTxHash } from './validate';

describe('format', () => {
  it('formats bytes', () => {
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(1536)).toBe('1.5 KB');
    expect(formatBytes(-1)).toBe('—');
  });
  it('formats UTC timestamps', () => {
    expect(formatUTC('2026-10-21T01:02:03.456Z')).toBe('2026-10-21 01:02:03 UTC');
    expect(formatUTC(undefined)).toBe('—');
    expect(formatUTC('garbage')).toBe('—');
  });
  it('shortens hashes', () => {
    expect(shortHash('abcdef')).toBe('abcdef');
    expect(shortHash('0123456789abcdef0123456789abcdef')).toBe('01234567…abcdef');
  });
});

describe('validate', () => {
  it('validates ethereum identifiers', () => {
    expect(isEthAddress('0x' + 'a'.repeat(40))).toBe(true);
    expect(isEthAddress('0x' + 'a'.repeat(39))).toBe(false);
    expect(isEthTxHash('0x' + 'f'.repeat(64))).toBe(true);
    expect(isEthTxHash('0x123')).toBe(false);
  });
});
