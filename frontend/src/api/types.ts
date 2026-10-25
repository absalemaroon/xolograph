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
export interface NetworkInfo {
  id: string; name: string; family: 'EVM' | 'SOLANA' | 'BITCOIN'; nativeSymbol: string; nativeDecimals: number;
  hasHistory: boolean; hasLabels: boolean; endpoint: string;
}
export interface SearchHit { kind: 'ENTITY' | 'CASE' | 'EVIDENCE' | 'LABEL'; id: string; title: string; subtitle: string; caseId?: string }
export interface TxIO { address: string; value: string; type?: string }
export interface AddressSummary {
  address: string; network: string; nativeSymbol: string; nativeDecimals: number; balanceWei: string; nonce: number; isContract: boolean;
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
  hash: string; network: string; nativeSymbol: string; nativeDecimals: number; inputs?: TxIO[]; outputs?: TxIO[]; status: string; blockNumber: number; blockHash?: string; timestamp?: string;
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

// ---- entities (mirrors backend/pkg/store/entities.go)
export type AttributionStatus =
  | 'VERIFIED' | 'PUBLICLY_REPORTED' | 'INVESTIGATOR_ATTRIBUTED' | 'SUSPECTED' | 'DISPUTED' | 'REJECTED' | 'UNKNOWN';
export type Confidence = 'LOW' | 'MEDIUM' | 'HIGH';
export interface AddressHistory { timestamp: string; actor: string; fromStatus: string; toStatus: string; note: string }
export interface EntityAddress {
  id: string; entityId: string; network: string; address: string; label: string; role: string;
  status: AttributionStatus; confidence: Confidence; sourceType: string; sourceRef: string; basis: string; notes: string;
  createdAt: string; updatedAt: string; history?: AddressHistory[];
}
export interface EntityIdentifier {
  id: string; entityId: string; platform: string; value: string; status: string; sourceRef: string;
  confidence: Confidence; notes: string; createdAt: string;
}
export interface EntityEvidence {
  id: string; entityId: string; evidenceId: string; caseId: string; caseTitle: string; name: string;
  sha256: string; sizeBytes: number; addressId: string; description: string; createdAt: string;
}
export interface Entity {
  id: string; name: string; type: string; description: string; country: string; tags: string[];
  status: AttributionStatus; confidence: Confidence; notes: string; createdAt: string; updatedAt: string;
  addressCount: number; addresses?: EntityAddress[]; identifiers?: EntityIdentifier[]; evidence?: EntityEvidence[];
}
export interface AddressInput {
  network: string; address: string; label?: string; role?: string; status?: string; confidence?: string;
  sourceType?: string; sourceRef?: string; basis?: string; notes?: string;
}
export interface EntityInput {
  name: string; type: string; description: string; country: string; tags: string[]; status: string; confidence: string;
  notes: string; addresses?: AddressInput[];
}

export interface AddressAttribution { entityId: string; name: string; type: string; status: AttributionStatus; confidence: Confidence }
export interface GraphSnapshotSummary { id: string; name: string; seeds: string[]; nodeCount: number; edgeCount: number; createdAt: string }
export interface GraphSnapshot extends GraphSnapshotSummary { data: unknown }

// ---- case relationship board (mirrors backend/pkg/store/board.go)
export type BoardKind = 'ENTITY' | 'WALLET' | 'EVIDENCE' | 'NOTE' | 'EVENT';
export type BoardClaim = 'FACT' | 'HYPOTHESIS';
export interface BoardNode {
  id: string; caseId: string; kind: BoardKind; refId: string; network: string; title: string; body: string; x: number; y: number;
  createdAt: string; updatedAt: string; liveName?: string; entityStatus?: string; evidenceHash?: string; evidenceState?: string;
}
export interface BoardLinkEvidence { id: string; linkId: string; evidenceId: string; name: string; sha256: string; status: string; description: string; createdAt: string }
export interface BoardLink {
  id: string; caseId: string; sourceId: string; targetId: string; relation: string; claim: BoardClaim; confidence: Confidence;
  sourceRef: string; note: string; evidence: BoardLinkEvidence[]; createdAt: string; updatedAt: string;
}
export interface Board { nodes: BoardNode[]; links: BoardLink[] }

// ---- portfolio and flows (mirrors backend/pkg/portfolio)
export interface PortfolioAsset {
  kind: string; symbol: string; name: string; tokenAddress?: string; balance: string; decimals: number;
  priced: boolean; priceUsd?: number; valueUsd?: number; priceAt?: string; priceSource?: string;
}
export interface WalletPortfolio {
  network: string; address: string; assets: PortfolioAsset[]; pricedValueUsd: number; pricedCount: number; unpricedCount: number;
  asOf: string; provenance: Provenance[]; notes: string[];
}
export interface EntityWalletValue {
  addressId: string; network: string; address: string; label: string; status: AttributionStatus; confidence: Confidence;
  portfolio?: WalletPortfolio; error?: string;
}
export interface EntityPortfolio {
  entityId: string; name: string; asOf: string; wallets: EntityWalletValue[]; strongUsd: number; allUsd: number;
  byStatusUsd: Record<string, number>; skipped: string[]; notes: string[];
}
export interface Flows {
  network: string; address: string; symbol: string; decimals: number; pagesLoaded: number; historyExhausted: boolean;
  transferCount: number; failedExcluded: number; from?: string; to?: string; inflow: string; outflow: string; net: string;
  inflowUsd: number; outflowUsd: number; netUsd: number; valuedCount: number; unvaluedCount: number; qualification: string[]; provenance: Provenance[];
}
