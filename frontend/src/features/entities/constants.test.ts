import { describe, expect, it } from 'vitest';
import { explorerSupported, graphSupported, isValidAddress, needsSource } from './constants';

describe('entity constants', () => {
  it('validates addresses per network', () => {
    expect(isValidAddress('ethereum-mainnet', '0x' + 'a'.repeat(40))).toBe(true);
    expect(isValidAddress('base', '0x123')).toBe(false);
    expect(isValidAddress('solana', '9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM')).toBe(true);
    expect(isValidAddress('solana', '0x' + 'a'.repeat(40))).toBe(false);
    expect(isValidAddress('bitcoin', 'bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq')).toBe(true);
    expect(isValidAddress('bitcoin', '0x' + 'a'.repeat(40))).toBe(false);
  });
  it('requires a source only for strong claims', () => {
    expect(needsSource('VERIFIED')).toBe(true);
    expect(needsSource('PUBLICLY_REPORTED')).toBe(true);
    expect(needsSource('SUSPECTED')).toBe(false);
  });
  it('knows which networks have an explorer adapter and which have a graph', () => {
    expect(explorerSupported('ethereum-mainnet')).toBe(true);
    expect(explorerSupported('solana')).toBe(true);
    expect(explorerSupported('dogecoin')).toBe(false);
    expect(graphSupported('ethereum-mainnet')).toBe(true);
    expect(graphSupported('solana')).toBe(false);
  });
});
