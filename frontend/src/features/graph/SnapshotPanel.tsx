import { Trash2 } from 'lucide-react';
import { useState } from 'react';
import { api } from '@/api/endpoints';
import { Button, ErrorBanner, Input, Panel } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';
import { shortAddr } from '@/lib/units';

interface Props {
  canSave: boolean;
  /** Builds the payload at click time so it always reflects the current graph. */
  getPayload: () => { seeds: string[]; nodeCount: number; edgeCount: number; data: unknown };
  onRestore: (data: unknown) => void;
}

export function SnapshotPanel({ canSave, getPayload, onRestore }: Props) {
  const { data, error, reload } = useAsync(() => api.graphs.list(), []);
  const [name, setName] = useState('');
  const [busy, setBusy] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  const run = async (key: string, fn: () => Promise<void>) => {
    setBusy(key);
    setMsg(null);
    try { await fn(); } catch (e) { setMsg(e instanceof Error ? e.message : String(e)); } finally { setBusy(null); }
  };

  const save = () => run('save', async () => {
    const p = getPayload();
    await api.graphs.save({ name: name.trim(), ...p });
    setName('');
    reload();
  });

  return (
    <Panel title="Saved views">
      <div className="space-y-3">
        <div className="flex gap-2">
          <Input aria-label="Snapshot name" placeholder="Name this view" value={name} maxLength={120} onChange={(e) => setName(e.target.value)} />
          <Button size="sm" variant="primary" loading={busy === 'save'} disabled={!canSave || name.trim() === ''} onClick={() => void save()}>Save</Button>
        </div>
        <p className="text-[10px] text-fg-muted">A saved view keeps node positions and the transfers loaded at save time. Loading it does not re-query the chain, and it cannot be expanded.</p>
        {error && <ErrorBanner message={error} onRetry={reload} />}
        {msg && <p className="text-[11px]">⚠ {msg}</p>}
        <ul className="divide-y divide-[var(--border-subtle)]">
          {(data ?? []).map((g) => (
            <li key={g.id} className="flex items-center justify-between gap-2 py-2 text-xs">
              <div className="min-w-0">
                <div className="truncate font-medium">{g.name}</div>
                <div className="text-[10px] text-fg-muted">{g.nodeCount} nodes · {g.edgeCount} edges · {g.seeds.map(shortAddr).join(', ')} · {formatUTC(g.createdAt)}</div>
              </div>
              <div className="flex shrink-0 gap-1">
                <Button size="sm" loading={busy === g.id} onClick={() => void run(g.id, async () => { onRestore((await api.graphs.get(g.id)).data); })}>Load</Button>
                <Button size="sm" variant="ghost" icon={<Trash2 size={12} />} aria-label={`Delete ${g.name}`} onClick={() => void run(`d${g.id}`, async () => { await api.graphs.remove(g.id); reload(); })} />
              </div>
            </li>
          ))}
        </ul>
        {data && data.length === 0 && <p className="text-xs text-fg-muted">No saved views yet.</p>}
      </div>
    </Panel>
  );
}
