import React, { useState } from 'react';
import { Transaction, AddressProfile } from '../types/blockchain';
import { blockchainService } from '../services/blockchainService';
import { Button } from '../components/common/Button';
import { Badge } from '../components/common/Badge';

export const BlockchainExplorerView: React.FC = () => {
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearching, setIsSearching] = useState(false);
  const [addressResult, setAddressResult] = useState<AddressProfile | null>(null);
  const [txResult, setTxResult] = useState<Transaction | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchQuery.trim()) return;

    setIsSearching(true);
    setError(null);
    setAddressResult(null);
    setTxResult(null);

    const q = searchQuery.trim();

    try {
      if (q.length === 42 && q.startsWith('0x')) {
        const addr = await blockchainService.lookupAddress(q);
        setAddressResult(addr);
      } else if (q.length === 66 && q.startsWith('0x')) {
        const tx = await blockchainService.lookupTransaction(q);
        setTxResult(tx);
      } else {
        setError('Invalid input query. Enter a 42-character ETH address (0x...) or a 66-character Transaction Hash.');
      }
    } catch (err: any) {
      setError(err.message || 'Data retrieval failed.');
    } finally {
      setIsSearching(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between border-b border-[var(--border-subtle)] pb-4">
        <div>
          <h1 className="text-xl font-bold uppercase tracking-wider text-[var(--text-primary)]">
            BLOCKCHAIN EXPLORER & DATA ACQUISITION
          </h1>
          <p className="text-xs text-[var(--text-secondary)] mt-1 font-mono-val">
            NETWORK: ETHEREUM MAINNET // DOMAIN NORMALIZED RPC LAYER
          </p>
        </div>
      </div>

      <form onSubmit={handleSearch} className="flex gap-2">
        <input
          type="text"
          placeholder="Enter Ethereum Wallet Address (0x...) or Transaction Hash (0x...)"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="flex-1 bg-[var(--bg-secondary)] border border-[var(--border-default)] p-3 text-xs font-mono-val text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-mono)]"
        />
        <Button variant="primary" size="md" type="submit" disabled={isSearching}>
          {isSearching ? 'QUERYING RPC...' : 'INSPECT ON-CHAIN'}
        </Button>
      </form>

      {error && (
        <div className="p-4 border border-[var(--border-strong)] bg-red-950/20 text-xs font-mono-val text-red-400">
          QUERY ERROR: {error}
        </div>
      )}

      {addressResult && (
        <div className="space-y-4">
          <div className="p-4 border border-[var(--border-default)] bg-[var(--bg-secondary)] space-y-3">
            <div className="flex justify-between items-center border-b border-[var(--border-subtle)] pb-2">
              <span className="text-xs font-mono-val text-[var(--text-muted)]">ADDRESS PROFILE</span>
              <Badge label={addressResult.network} variant="default" />
            </div>
            <div className="text-base font-bold font-mono-val text-[var(--text-primary)] break-all">
              {addressResult.address}
            </div>
            <div className="flex gap-2">
              {addressResult.labels?.map((l) => (
                <Badge key={l} label={l} variant="outline" />
              ))}
            </div>
          </div>

          <div className="grid grid-cols-4 gap-4">
            <div className="p-4 border border-[var(--border-default)] bg-[var(--bg-secondary)] font-mono-val">
              <span className="text-xs text-[var(--text-muted)] block">TOTAL TRANSACTIONS</span>
              <span className="text-xl font-bold text-[var(--text-primary)]">{addressResult.txCountTotal}</span>
            </div>
            <div className="p-4 border border-[var(--border-default)] bg-[var(--bg-secondary)] font-mono-val">
              <span className="text-xs text-[var(--text-muted)] block">INCOMING TRANSFERS</span>
              <span className="text-xl font-bold text-[var(--text-primary)]">{addressResult.incomingTxCount}</span>
            </div>
            <div className="p-4 border border-[var(--border-default)] bg-[var(--bg-secondary)] font-mono-val">
              <span className="text-xs text-[var(--text-muted)] block">ETH RECEIVED</span>
              <span className="text-xl font-bold text-[var(--text-primary)]">
                {blockchainService.formatWeiToETH(addressResult.totalETHReceivedWei)}
              </span>
            </div>
            <div className="p-4 border border-[var(--border-default)] bg-[var(--bg-secondary)] font-mono-val">
              <span className="text-xs text-[var(--text-muted)] block">ETH SENT</span>
              <span className="text-xl font-bold text-[var(--text-primary)]">
                {blockchainService.formatWeiToETH(addressResult.totalETHSentWei)}
              </span>
            </div>
          </div>

          <div className="p-4 border border-[var(--border-subtle)] bg-[var(--bg-elevated)] text-xs font-mono-val space-y-1">
            <span className="text-[var(--text-muted)] block font-bold">DATA PROVENANCE & COVERAGE AUDIT</span>
            <p className="text-[var(--text-secondary)]">Source: {addressResult.provenance.dataSource} ({addressResult.provenance.endpoint})</p>
            <p className="text-[var(--text-secondary)]">Coverage Limitation: {addressResult.provenance.coverageLimit}</p>
            <p className="text-[var(--text-muted)] break-all">Payload Hash: {addressResult.provenance.rawPayloadHash}</p>
          </div>
        </div>
      )}

      {txResult && (
        <div className="space-y-4 font-mono-val text-xs">
          <div className="p-4 border border-[var(--border-default)] bg-[var(--bg-secondary)] space-y-3">
            <div className="flex justify-between items-center border-b border-[var(--border-subtle)] pb-2">
              <span className="text-xs text-[var(--text-muted)]">TRANSACTION RECORD</span>
              <Badge label={txResult.status} variant="default" />
            </div>
            <div className="text-sm font-bold text-[var(--text-primary)] break-all">{txResult.hash}</div>
            <div className="grid grid-cols-2 gap-4 text-xs pt-2">
              <div>
                <span className="text-[var(--text-muted)] block">SENDER (FROM):</span>
                <span className="text-[var(--text-primary)] font-bold">{txResult.fromAddress}</span>
              </div>
              <div>
                <span className="text-[var(--text-muted)] block">RECIPIENT (TO):</span>
                <span className="text-[var(--text-primary)] font-bold">{txResult.toAddress}</span>
              </div>
              <div>
                <span className="text-[var(--text-muted)] block">VALUE:</span>
                <span className="text-[var(--text-primary)] font-bold">{txResult.valueFormatted}</span>
              </div>
              <div>
                <span className="text-[var(--text-muted)] block">BLOCK NUMBER:</span>
                <span className="text-[var(--text-primary)]">#{txResult.blockNumber}</span>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};