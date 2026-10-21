import { GraphDataPayload, GraphNodeAttributes, GraphEdgeAttributes } from '../types/graph';
import { Transaction } from '../types/blockchain';

class GraphService {
  buildGraphFromTransactions(seedAddress: string, transactions: Transaction[]): GraphDataPayload {
    const nodesMap = new Map<string, GraphNodeAttributes>();
    const edgesList: GraphEdgeAttributes[] = [];

    const cleanSeed = seedAddress.toLowerCase();
    nodesMap.set(cleanSeed, {
      id: cleanSeed, label: cleanSeed.substring(0, 10), nodeType: 'ADDRESS', size: 14, color: 'var(--accent-mono)', isInvestigated: true,
    });

    transactions.forEach((tx, idx) => {
      const from = tx.fromAddress.toLowerCase();
      const to = tx.toAddress.toLowerCase();

      if (!nodesMap.has(from)) nodesMap.set(from, { id: from, label: from.substring(0, 10), nodeType: 'ADDRESS', size: 8, color: 'var(--text-secondary)' });
      if (!nodesMap.has(to)) nodesMap.set(to, { id: to, label: to.substring(0, 10), nodeType: 'ADDRESS', size: 8, color: 'var(--text-secondary)' });

      edgesList.push({
        id: `edge_${idx}`, source: from, target: to, edgeType: 'NATIVE_TRANSFER', label: tx.valueFormatted, valueFormatted: tx.valueFormatted
      });
    });

    return { nodes: Array.from(nodesMap.values()), edges: edgesList, network: 'ethereum-mainnet' };
  }
}

export const graphService = new GraphService();