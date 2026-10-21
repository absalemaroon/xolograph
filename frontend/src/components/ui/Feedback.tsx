import { AlertTriangle } from 'lucide-react';
import type { ReactNode } from 'react';
import { Button } from './Button';

export function Spinner({ label = 'Loading' }: { label?: string }) {
  return <div role="status" className="animate-pulse py-8 text-center text-[11px] uppercase tracking-[0.2em] text-fg-muted">{label}…</div>;
}

export function ErrorBanner({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div role="alert" className="flex items-start gap-3 border border-dashed border-fg bg-raised p-3 text-xs">
      <AlertTriangle size={16} className="mt-0.5 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1 break-words">{message}</div>
      {onRetry && <Button size="sm" onClick={onRetry}>Retry</Button>}
    </div>
  );
}

export function EmptyState({ title, hint, action }: { title: string; hint?: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 border border-dashed border-line px-6 py-12 text-center">
      <div className="text-xs font-medium uppercase tracking-[0.16em]">{title}</div>
      {hint && <p className="max-w-md text-xs text-fg-muted">{hint}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}
