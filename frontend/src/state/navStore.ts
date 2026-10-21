import { create } from 'zustand';

export type ViewId =
  | 'overview' | 'cases' | 'evidence' | 'audit'
  | 'explorer' | 'graph' | 'entities' | 'reports' | 'settings';

interface NavState {
  view: ViewId;
  caseId: string | null;
  go: (view: ViewId) => void;
  openCase: (id: string) => void;
  closeCase: () => void;
}

export const useNav = create<NavState>((set) => ({
  view: 'overview',
  caseId: null,
  go: (view) => set({ view, caseId: null }),
  openCase: (caseId) => set({ view: 'cases', caseId }),
  closeCase: () => set({ caseId: null }),
}));
