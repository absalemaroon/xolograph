import { Investigation, EvidenceItem, AuditEvent } from '../types/investigation';
import { Entity } from './entity';

export interface CaseReport {
  generatedAt: string;
  investigation: Investigation;
  evidenceItems: EvidenceItem[];
  auditEvents: AuditEvent[];
  entities: Entity[];
  totalVaultVolume: number;
  integrityStatus: string;
  reportChecksum: string;
}