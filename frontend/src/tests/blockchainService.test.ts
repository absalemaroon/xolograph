import { blockchainService } from '../services/blockchainService';

describe('Blockchain Data Layer & Normalization', () => {
  test('validates and fetches address profile', async () => {
    const address = '0x71c7656ec7ab88b098defb751b7401b5f6d8976f';
    const profile = await blockchainService.lookupAddress(address);

    expect(profile.address).toBe(address);
    expect(profile.network).toBe('ethereum-mainnet');
    expect(profile.provenance.dataSource).toBeDefined();
    expect(profile.provenance.rawPayloadHash.length).toBe(64);
  });

  test('converts wei to ETH formatted string accurately', () => {
    const weiVal = '1000000000000000000'; // 1 ETH
    const formatted = blockchainService.formatWeiToETH(weiVal);
    expect(formatted).toBe('1.000000 ETH');
  });

  test('validates and parses transaction hash query', async () => {
    const txHash = '0x3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b';
    const tx = await blockchainService.lookupTransaction(txHash);

    expect(tx.hash).toBe(txHash);
    expect(tx.status).toBe('CONFIRMED');
    expect(tx.tokenTransfers?.length).toBeGreaterThan(0);
    expect(tx.tokenTransfers?.[0].tokenSymbol).toBe('USDC');
  });
});