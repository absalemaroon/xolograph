// Mirrors backend/pkg/store JSON shapes.
export type CaseStatus = 'OPEN' | 'CLOSED' | 'ARCHIVED';
export type EvidenceStatus = 'IMPORTED' | 'VERIFIED' | 'FAILED';

export interface Case {
  id: string; title: string; description: string; status: CaseStatus;
  createdAt: string; updatedAt: string; evidenceCount: number; findingCount: number;
}
export interface Note { id: string; caseId: string; body: string; createdAt: string }
export interface Finding {
  id: string; caseId: string; kind: string; refType: string; refId: string;
  title: string; note: string; sourceJson: string; createdAt: string;
}
export interface Evidence {
  id: string; caseId: string; name: string; sizeBytes: number; sha256: string;
  status: EvidenceStatus; importedAt: string; lastVerifiedAt?: string;
}
export interface CustodyEvent {
  id: number; evidenceId: string; caseId: string; timestamp: string; actor: string;
  action: string; detail: string; sha256: string; eventHash: string;
}
export interface ChainResult { valid: boolean; count: number; brokenAt?: number }
export interface AuditEntry {
  id: number; timestamp: string; actor: string; action: string;
  entityType: string; entityId: string; detail: string; entryHash: string;
}

export interface Provenance {
  source: string; endpoint: string; query: string; retrievedAt: string; payloadSha256: string; coverage: string;
}
export interface ThirdPartyLabel { name: string; kind: string; source: string }
export interface AddressSummary {
  address: string; network: string; balanceWei: string; nonce: number; isContract: boolean;
  thirdPartyLabels: ThirdPartyLabel[]; notes: string[]; provenance: Provenance[];
}
export interface Transfer {
  txHash: string; logIndex: number; kind: string; from: string; to: string; value: string;
  decimals: number; symbol: string; tokenAddress?: string; tokenId?: string;
  timestamp: string; blockNumber: number; status: string;
}
export interface ActivityPage {
  address: string; kind: 'native' | 'token'; items: Transfer[]; nextCursor: string; provenance: Provenance;
}
export interface TokenTransfer {
  logIndex: number; standard: string; token: string; symbol: string; decimals: number;
  from: string; to: string; value: string; tokenId?: string;
}
export interface ChainTx {
  hash: string; network: string; status: string; blockNumber: number; blockHash?: string; timestamp?: string;
  from: string; to: string; contractCreated?: string; valueWei: string; gasUsed: number;
  gasPriceWei: string; feeWei: string; tokenTransfers: TokenTransfer[]; notes: string[]; provenance: Provenance[];
}
export interface Label {
  id: string; caseId: string; caseTitle: string; address: string; network: string; name: string;
  category: string; source: string; confidence: string; basis: string; createdBy: string; createdAt: string;
}
export interface ReportBundle {
  generatedAt: string; markdown: string; markdownSha256: string; json: string; jsonSha256: string;
}

export interface BackupResult {
  path: string; sha256: string; sizeBytes: number; files: number; includesEvidence: boolean; createdAt: string;
}
export interface StageResult {
  archiveSha256: string; archiveCreatedAt: string; stagedAt: string; files: number; bytes: number;
  includesEvidence: boolean; cases: number; auditEntries: number; auditChainValid: boolean; auditBrokenAt?: number;
}
export interface RestoreResult {
  applied: boolean; appliedAt: string; archiveSha256: string; archiveCreatedAt: string; previousData?: string; error?: string;
}
export interface RestoreStatus { pending: StageResult | null; last: RestoreResult | null }
