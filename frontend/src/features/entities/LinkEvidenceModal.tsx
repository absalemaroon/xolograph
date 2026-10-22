import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { Entity, Evidence } from '@/api/types';
import { Button, EmptyState, ErrorBanner, Input, Label, Modal, Select, Spinner } from '@/components/ui';
import { CaseSelect } from '@/features/cases/CaseSelect';
import { useAsync } from '@/hooks/useAsync';
import { formatBytes } from '@/lib/format';
import { networkLabel } from './constants';

/** Links an item that already lives in a case's evidence vault. Nothing is copied or re-hashed. */
export function LinkEvidenceModal({ entity, onClose, onSaved }: { entity: Entity; onClose: () => void; onSaved: (e: Entity) => void }) {
  const [caseId, setCaseId] = useState('');
  const [evidenceId, setEvidenceId] = useState('');
  const [addressId, setAddressId] = useState('');
  const [description, setDescription] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const items = useAsync(() => (caseId ? api.evidence.list(caseId) : Promise.resolve([] as Evidence[])), [caseId]);
  const linked = new Set((entity.evidence ?? []).map((x) => x.evidenceId));
  const available = (items.data ?? []).filter((x) => !linked.has(x.id));

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      onSaved(await api.entities.linkEvidence(entity.id, evidenceId, addressId, description));
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <Modal title="Link evidence" onClose={onClose}
      footer={<><Button onClick={onClose}>Cancel</Button><Button variant="primary" loading={busy} disabled={!evidenceId} onClick={() => void save()}>Link</Button></>}>
      <div className="space-y-4">
        {error && <ErrorBanner message={error} />}
        <div><Label htmlFor="le-case">Case</Label><CaseSelect id="le-case" openOnly={false} value={caseId} onChange={(v) => { setCaseId(v); setEvidenceId(''); }} /></div>
        <div>
          <Label htmlFor="le-ev">Evidence item</Label>
          {items.loading && !items.data ? <Spinner /> : items.error ? <ErrorBanner message={items.error} onRetry={items.reload} /> : available.length === 0 ? (
            <EmptyState title="Nothing to link" hint="This case has no evidence in its vault, or all of it is already linked. Import files from the case's Evidence tab." />
          ) : (
            <Select id="le-ev" value={evidenceId} onChange={(e) => setEvidenceId(e.target.value)}>
              <option value="">Select…</option>
              {available.map((x) => <option key={x.id} value={x.id}>{x.name} ({formatBytes(x.sizeBytes)})</option>)}
            </Select>
          )}
        </div>
        {(entity.addresses ?? []).length > 0 && (
          <div><Label htmlFor="le-addr" hint="optional">Applies to wallet</Label>
            <Select id="le-addr" value={addressId} onChange={(e) => setAddressId(e.target.value)}>
              <option value="">Whole entity</option>
              {(entity.addresses ?? []).map((a) => <option key={a.id} value={a.id}>{networkLabel(a.network)} {a.address.slice(0, 10)}…</option>)}
            </Select></div>
        )}
        <div><Label htmlFor="le-desc" hint="optional">What does it show?</Label><Input id="le-desc" value={description} onChange={(e) => setDescription(e.target.value)} /></div>
      </div>
    </Modal>
  );
}
