import { Plus } from 'lucide-react';
import { useState } from 'react';
import { api } from '@/api/endpoints';
import { Badge, Button, Panel } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { useNav } from '@/state/navStore';
import { AttributionBadge } from './AttributionBadge';
import { EntityFormModal } from './EntityFormModal';

/** Explorer panel: which entity records claim this address, and how strongly. Rejected claims stay visible. */
export function EntityAttributions({ address }: { address: string }) {
  const openEntity = useNav((s) => s.openEntity);
  const { data, error, reload } = useAsync(() => api.entities.forAddress(address), [address]);
  const [creating, setCreating] = useState(false);
  const rows = data ?? [];
  return (
    <Panel title="Entity attributions" actions={<Button size="sm" icon={<Plus size={12} />} onClick={() => setCreating(true)}>New entity</Button>}>
      {error ? <p className="text-xs">Could not load entity records: {error}</p> : rows.length === 0 ? (
        <p className="text-xs text-fg-muted">No entity record claims this address. Absence of a record is not evidence that it has no owner.</p>
      ) : (
        <ul className="space-y-2">
          {rows.map((e) => (
            <li key={e.id} className="flex flex-wrap items-center gap-2 text-xs">
              <button type="button" className="font-medium underline-offset-2 hover:underline" onClick={() => openEntity(e.id)}>{e.name}</button>
              {(e.addresses ?? []).map((a) => <span key={a.id} className="inline-flex items-center gap-1.5"><AttributionBadge status={a.status} /><Badge>Confidence {a.confidence}</Badge></span>)}
            </li>
          ))}
        </ul>
      )}
      {creating && <EntityFormModal defaultAddress={address} onClose={() => setCreating(false)} onSaved={(e) => { reload(); openEntity(e.id); }} />}
    </Panel>
  );
}
