import { NetworkID } from './blockchain';

export type AttributionSource = 'VERIFIED_SOURCE' | 'THIRD_PARTY_PROVIDER' | 'ANALYST_ASSERTION' | 'ALGORITHMIC_INFERENCE' | 'UNKNOWN';
export type ConfidenceLevel = 'HIGH' | 'MEDIUM' | 'LOW';

export interface EntityAddress {
  entityId: string;
  address: string;
  network: NetworkID;
  dateAttached: string;
  attachedBy: string;
}

export interface Entity {
  id: string;
  investigationId: string;
  name: string;
  category: string;
  attributionSource: AttributionSource;
  confidence: ConfidenceLevel;
  supportingEvidence: string;
  analystNotes: string;
  createdAt: string;
  updatedAt: string;
  addresses: EntityAddress[];
}

export interface CreateEntityInput {
  investigationId: string;
  name: string;
  category: string;
  attributionSource: AttributionSource;
  confidence: ConfidenceLevel;
  supportingEvidence: string;
  analystNotes: string;
}