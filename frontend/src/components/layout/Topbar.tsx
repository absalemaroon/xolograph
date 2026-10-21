import { Moon, Sun } from 'lucide-react';
import { viewLabel } from '@/app/navigation';
import { Badge } from '@/components/ui';
import { useEngine } from '@/state/engineStore';
import { useNav } from '@/state/navStore';
import { useSettings } from '@/state/settingsStore';

export function Topbar() {
  const { view, caseId } = useNav();
  const status = useEngine((s) => s.status);
  const { mode, toggleMode, analyst } = useSettings();
  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b border-line bg-raised px-6">
      <div className="flex min-w-0 items-center gap-2 text-xs uppercase tracking-[0.14em]">
        <span className="text-fg-muted">{viewLabel(view)}</span>
        {caseId && (<><span className="text-fg-muted">/</span><span className="truncate font-semibold">{caseId}</span></>)}
      </div>
      <div className="flex items-center gap-4">
        <span className="hidden text-[11px] text-fg-muted sm:inline">ANALYST: <span className="text-fg">{analyst || 'local-analyst'}</span></span>
        {status === 'online' && <Badge tone="outline">● Engine online</Badge>}
        {status === 'checking' && <Badge>… Connecting</Badge>}
        {status === 'offline' && <Badge tone="alert">▲ Engine offline</Badge>}
        <button
          type="button"
          onClick={toggleMode}
          aria-label={`Switch to ${mode === 'dark' ? 'light' : 'dark'} theme`}
          className="flex h-8 w-8 items-center justify-center border border-line-strong text-fg-soft hover:bg-hover hover:text-fg"
        >
          {mode === 'dark' ? <Sun size={14} /> : <Moon size={14} />}
        </button>
      </div>
    </header>
  );
}
