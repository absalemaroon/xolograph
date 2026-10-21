import { create } from 'zustand';
import { api } from '@/api/endpoints';

interface EngineState {
  status: 'checking' | 'online' | 'offline';
  check: () => Promise<void>;
}

export const useEngine = create<EngineState>((set) => ({
  status: 'checking',
  check: async () => {
    try {
      await api.health();
      set({ status: 'online' });
    } catch {
      set({ status: 'offline' });
    }
  },
}));
