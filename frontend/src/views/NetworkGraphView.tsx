import React, { useState, useEffect } from 'react';
import { GraphDataPayload } from '../types/graph';
import { graphService } from '../services/graphService';
import { transactionExplorerService } from '../services/transactionExplorerService';
import { Button } from '../components/common/Button';

export const NetworkGraphView: React.FC = () => {
  const [seedAddress, setSeedAddress] = useState('0x71c7656ec7ab88b098defb751b7401b5f6d8976f');
  const [graphData, setGraphData] = useState<GraphDataPayload | null>(null);

  const handleBuildGraph = async () => {
    const res = await transactionExplorerService.fetchPaginatedTransactions({
      address: seedAddress, network: 'ethereum-mainnet', direction: 'ALL', minEthValue: 0, tokenType: 'ALL', page: 1, pageSize: 15,
    });
    setGraphData(graphService.buildGraphFromTransactions(seedAddress, res.transactions));
  };

  useEffect(() => { handleBuildGraph(); }, []);

  return (
    <div className="flex h-full flex-col space-y-4">
      <div className="flex items-center justify-between border-b border-[var(--border-subtle)] pb-3 font-mono-val text-xs">
        <h1 className="text-xl font-bold uppercase tracking-wider text-[var(--text-primary)]">INTERACTIVE TRANSACTION NETWORK GRAPH</h1>
        <Button variant="primary" size="sm" onClick={handleBuildGraph}>GENERATE GRAPH</Button>
      </div>
      <div className="flex-1 border border-[var(--border-default)] bg-[var(--bg-secondary)] relative flex items-center justify-center p-6 bg-grid-pattern overflow-auto">
        {!graphData ? <span>INITIALIZING...</span> : <span>NODES: {graphData.nodes.length} | EDGES: {graphData.edges.length}</span>}
      </div>
    </div>
  );
};