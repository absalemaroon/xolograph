import { NetworkID } from './blockchain';

export type NodeType = 'ADDRESS' | 'ENTITY' | 'TRANSACTION';
export type EdgeType = 'NATIVE_TRANSFER' | 'TOKEN_TRANSFER' | 'ASSOCIATION';

export interface GraphNodeAttributes {
  id: string;
  label: string;
  nodeType: NodeType;
  size: number;
  color: string;
  isInvestigated?: boolean;
}

export interface GraphEdgeAttributes {
  id: string;
  source: string;
  target: string;
  edgeType: EdgeType;
  label: string;
  valueFormatted: string;
}

export interface GraphDataPayload {
  nodes: GraphNodeAttributes[];
  edges: GraphEdgeAttributes[];
  network: NetworkID;
}