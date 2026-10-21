import { transactionExplorerService } from '../services/transactionExplorerService';

describe('Transaction Explorer Pagination & Filtering Engine', () => {
  const targetAddress = '0x71c7656ec7ab88b098defb751b7401b5f6d8976f';

  test('paginates transactions correctly across pages', async () => {
    const page1 = await transactionExplorerService.fetchPaginatedTransactions({
      address: targetAddress, network: 'ethereum-mainnet', direction: 'ALL', minEthValue: 0, tokenType: 'ALL', page: 1, pageSize: 5,
    });
    expect(page1.transactions.length).toBe(5);
  });
});