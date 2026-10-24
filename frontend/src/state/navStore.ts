import { create } from 'zustand';

export type ViewId =
  | 'overview' | 'cases' | 'evidence' | 'audit'
  | 'explorer' | 'graph' | 'entities' | 'reports' | 'settings';

interface NavState {
  view: ViewId;
  caseId: string | null;
  entityId: string | null;
  explorerQuery: string | null;
  explorerNetwork: string | null;
  graphSeed: string | null;
  graphSeeds: string[] | null;
  go: (view: ViewId) => void;
  openCase: (id: string) => void;
  openEntity: (id: string) => void;
  closeEntity: () => void;
  closeCase: () => void;
  openExplorer: (query: string, network?: string) => void;
  openGraph: (address: string) => void;
  openGraphMulti: (addresses: string[]) => void;
}

export const useNav = create<NavState>((set) => ({
  view: 'overview',
  caseId: null,
  entityId: null,
  explorerQuery: null,
  explorerNetwork: null,
  graphSeed: null,
  graphSeeds: null,
  go: (view) => set({ view, caseId: null, entityId: null }),
  openCase: (caseId) => set({ view: 'cases', caseId }),
  closeCase: () => set({ caseId: null }),
  openEntity: (entityId) => set({ view: 'entities', caseId: null, entityId }),
  closeEntity: () => set({ entityId: null }),
  openExplorer: (explorerQuery, network) => set({ view: 'explorer', caseId: null, entityId: null, explorerQuery, explorerNetwork: network ?? null }),
  openGraph: (graphSeed) => set({ view: 'graph', caseId: null, entityId: null, graphSeed, graphSeeds: null }),
  openGraphMulti: (graphSeeds) => set({ view: 'graph', caseId: null, entityId: null, graphSeed: null, graphSeeds }),
}));
