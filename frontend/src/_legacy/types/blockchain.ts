export type NetworkID = 'ethereum-mainnet' | 'ethereum-sepolia';

export interface DataProvenance {
  network: NetworkID;
  dataSource: string;
  endpoint: string;
  retrievedAt: string;
  queryParameters: string;
  normalizationVer: string;
  rawPayloadHash: string;
  coverageLimit: string;
}

export interface TokenTransfer {
  txHash: string;
  logIndex: number;
  contractAddress: string;
  tokenSymbol?: string;
  tokenDecimals?: number;
  fromAddress: string;
  toAddress: string;
  rawValue: string;
  formattedValue: string;
}

export interface Transaction {
  hash: string;
  network: NetworkID;
  blockNumber: number;
  blockHash: string;
  blockTimestamp: string;
  fromAddress: string;
  toAddress: string;
  status: 'CONFIRMED' | 'FAILED' | 'PENDING';
  valueWei: string;
  valueFormatted: string;
  gasUsed: number;
  gasPriceWei: string;
  txFeeFormatted: string;
  tokenTransfers?: TokenTransfer[];
  provenance: DataProvenance;
}

export interface AddressProfile {
  address: string;
  network: NetworkID;
  firstObservedAt?: string;
  lastObservedAt?: string;
  txCountTotal: number;
  incomingTxCount: number;
  outgoingTxCount: number;
  totalETHReceivedWei: string;
  totalETHSentWei: string;
  labels?: string[];
  provenance: DataProvenance;
}