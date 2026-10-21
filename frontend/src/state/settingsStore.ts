import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type ThemeMode = 'dark' | 'light';
export type Density = 'compact' | 'comfortable';

interface SettingsState {
  mode: ThemeMode;
  fontSize: number; // px, 12-16
  density: Density;
  analyst: string; // recorded as actor in custody + audit entries
  setMode: (m: ThemeMode) => void;
  toggleMode: () => void;
  setFontSize: (n: number) => void;
  setDensity: (d: Density) => void;
  setAnalyst: (a: string) => void;
}

export const useSettings = create<SettingsState>()(
  persist(
    (set) => ({
      mode: 'dark',
      fontSize: 14,
      density: 'comfortable',
      analyst: '',
      setMode: (mode) => set({ mode }),
      toggleMode: () => set((s) => ({ mode: s.mode === 'dark' ? 'light' : 'dark' })),
      setFontSize: (n) => set({ fontSize: Math.min(16, Math.max(12, Math.round(n))) }),
      setDensity: (density) => set({ density }),
      setAnalyst: (analyst) => set({ analyst: analyst.slice(0, 64) }),
    }),
    { name: 'xolograph.settings' },
  ),
);

export function applySettings(s: Pick<SettingsState, 'mode' | 'fontSize' | 'density'>): void {
  const root = document.documentElement;
  root.setAttribute('data-theme', s.mode);
  root.setAttribute('data-density', s.density);
  root.style.fontSize = `${s.fontSize}px`;
}
