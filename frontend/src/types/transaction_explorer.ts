import { Transaction, DataProvenance, NetworkID } from './blockchain';

export type TransferDirection = 'ALL' | 'INCOMING' | 'OUTGOING';
export type TokenTypeFilter = 'ALL' | 'NATIVE' | 'ERC20';

export interface TransactionFilterInput {
  address: string;
  network: NetworkID;
  direction: TransferDirection;
  minEthValue: number;
  tokenType: TokenTypeFilter;
  page: number;
  pageSize: number;
}

export interface PaginatedTransactionsResponse {
  address: string;
  totalCount: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
  transactions: Transaction[];
  provenance: DataProvenance;
  coverageNote: string;
}