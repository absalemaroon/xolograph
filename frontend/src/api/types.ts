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
