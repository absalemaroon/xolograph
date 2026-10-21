import React, { useState, useEffect } from 'react';
import { Transaction } from '../types/blockchain';
import { TransactionFilterInput, TransferDirection, TokenTypeFilter } from '../types/transaction_explorer';
import { transactionExplorerService } from '../services/transactionExplorerService';
import { Button } from '../components/common/Button';
import { Badge } from '../components/common/Badge';

export const TransactionExplorerView: React.FC = () => {
  const [address, setAddress] = useState('0x71c7656ec7ab88b098defb751b7401b5f6d8976f');
  const [direction, setDirection] = useState<TransferDirection>('ALL');
  const [minEthValue, setMinEthValue] = useState<number>(0);
  const [tokenType, setTokenType] = useState<TokenTypeFilter>('ALL');
  const [page, setPage] = useState(1);
  const pageSize = 5;

  const [txData, setTxData] = useState<Transaction[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const loadTransactions = async () => {
    if (!address.trim()) return;
    setIsLoading(true);
    try {
      const filter: TransactionFilterInput = {
        address, network: 'ethereum-mainnet', direction, minEthValue, tokenType, page, pageSize,
      };
      const res = await transactionExplorerService.fetchPaginatedTransactions(filter);
      setTxData(res.transactions);
      setTotalCount(res.totalCount);
      setHasMore(res.hasMore);
    } catch (err: any) {
      alert('Transaction Query Error: ' + err.message);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => { loadTransactions(); }, [address, direction, minEthValue, tokenType, page]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between border-b border-[var(--border-subtle)] pb-4">
        <div>
          <h1 className="text-xl font-bold uppercase tracking-wider text-[var(--text-primary)]">
            TRANSACTION EXPLORER & ACTIVITY
          </h1>
        </div>
      </div>
      <div className="p-4 border border-[var(--border-default)] bg-[var(--bg-secondary)] space-y-4 font-mono-val text-xs">
        <div>
          <label className="text-[var(--text-muted)] block mb-1">TARGET INVESTIGATED ADDRESS</label>
          <input
            type="text" value={address} onChange={(e) => { setAddress(e.target.value); setPage(1); }}
            className="w-full bg-[var(--bg-primary)] border border-[var(--border-strong)] p-2 text-xs font-mono-val text-[var(--text-primary)] focus:outline-none"
          />
        </div>
      </div>
      <div className="border border-[var(--border-default)] bg-[var(--bg-secondary)] overflow-hidden font-mono-val text-xs">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b border-[var(--border-default)] text-[var(--text-muted)]">
              <th className="p-3">TX HASH</th>
              <th className="p-3">DIRECTION</th>
              <th className="p-3">VALUE / ASSET</th>
            </tr>
          </thead>
          <tbody>
            {txData.map((tx) => (
              <tr key={tx.hash} className="border-b border-[var(--border-subtle)]">
                <td className="p-3 font-bold text-[var(--text-primary)]">{tx.hash.substring(0, 16)}...</td>
                <td className="p-3">
                  <Badge label={tx.toAddress.toLowerCase() === address.toLowerCase() ? 'INCOMING' : 'OUTGOING'} variant="default" />
                </td>
                <td className="p-3 font-bold text-[var(--text-primary)]">{tx.valueFormatted}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};