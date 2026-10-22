import { describe, expect, it } from 'vitest';
import { explorerSupported, isValidAddress, needsSource } from './constants';

describe('entity constants', () => {
  it('validates addresses per network', () => {
    expect(isValidAddress('ethereum-mainnet', '0x' + 'a'.repeat(40))).toBe(true);
    expect(isValidAddress('base', '0x123')).toBe(false);
    expect(isValidAddress('solana', '9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM')).toBe(true);
    expect(isValidAddress('solana', '0x' + 'a'.repeat(40))).toBe(false);
  });
  it('requires a source only for strong claims', () => {
    expect(needsSource('VERIFIED')).toBe(true);
    expect(needsSource('PUBLICLY_REPORTED')).toBe(true);
    expect(needsSource('SUSPECTED')).toBe(false);
  });
  it('only claims an explorer adapter for Ethereum mainnet', () => {
    expect(explorerSupported('ethereum-mainnet')).toBe(true);
    expect(explorerSupported('solana')).toBe(false);
  });
});
