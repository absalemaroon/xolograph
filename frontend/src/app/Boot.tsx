import { useCallback, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import logo from '@/assets/xolograph.svg';
import { Button } from '@/components/ui';
import { bootEngine, isDesktop } from '@/lib/desktop';

type Phase = { name: 'starting' } | { name: 'ready' } | { name: 'failed'; message: string };

/** Holds the UI until the desktop engine is up (instant in the browser). */
export function Boot({ children }: { children: ReactNode }) {
  const [phase, setPhase] = useState<Phase>(isDesktop() ? { name: 'starting' } : { name: 'ready' });

  const start = useCallback(() => {
    setPhase({ name: 'starting' });
    bootEngine()
      .then(() => setPhase({ name: 'ready' }))
      .catch((e: unknown) => setPhase({ name: 'failed', message: e instanceof Error ? e.message : String(e) }));
  }, []);

  useEffect(() => { if (isDesktop()) start(); }, [start]);

  if (phase.name === 'ready') return <>{children}</>;
  return (
    <div className="bg-grid flex h-full flex-col items-center justify-center gap-5 bg-base text-fg">
      <span className="flex h-14 w-14 items-center justify-center bg-black p-2"><img src={logo} alt="" className="h-full w-full" /></span>
      <div className="text-sm font-bold uppercase tracking-[0.3em]">Xolograph</div>
      {phase.name === 'starting' ? (
        <div role="status" className="animate-pulse text-[11px] uppercase tracking-[0.2em] text-fg-muted">Starting engine…</div>
      ) : (
        <div role="alert" className="max-w-md space-y-3 text-center">
          <p className="text-xs">▲ {phase.message}</p>
          <Button variant="primary" onClick={start}>Retry</Button>
        </div>
      )}
    </div>
  );
}
