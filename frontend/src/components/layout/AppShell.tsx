import { useEffect } from 'react';
import type { ReactNode } from 'react';
import { NAV_FLAT } from '@/app/navigation';
import { Button } from '@/components/ui';
import { useEngine } from '@/state/engineStore';
import { useNav } from '@/state/navStore';
import { Sidebar } from './Sidebar';
import { Topbar } from './Topbar';

export function AppShell({ children }: { children: ReactNode }) {
  const { view, go } = useNav();
  const { status, check } = useEngine();

  useEffect(() => {
    void check();
    const t = window.setInterval(() => void check(), 10000);
    return () => window.clearInterval(t);
  }, [check]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!e.altKey || e.ctrlKey || e.metaKey) return;
      const idx = Number(e.key) - 1;
      if (Number.isInteger(idx) && idx >= 0 && idx < NAV_FLAT.length) { e.preventDefault(); go(NAV_FLAT[idx].id); }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [go]);

  return (
    <div className="flex h-full w-full overflow-hidden bg-base text-fg">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar />
        {status === 'offline' && (
          <div role="alert" className="flex items-center justify-between gap-3 border-b border-dashed border-fg bg-raised px-6 py-2 text-xs">
            <span>ENGINE OFFLINE — data cannot be loaded or saved. Start it with <code>go run ./cmd/xolographd --port 8787 --token dev</code></span>
            <Button size="sm" onClick={() => void check()}>Retry</Button>
          </div>
        )}
        <main className="bg-grid flex-1 overflow-y-auto">
          <div key={view} className="view-enter mx-auto max-w-7xl p-6">{children}</div>
        </main>
      </div>
    </div>
  );
}
