import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { Entity, EntityAddress } from '@/api/types';
import { Button, ErrorBanner, Input, Label, Modal } from '@/components/ui';
import { AddressFields, addrError, emptyAddr } from './AddressFields';
import type { AddrDraft } from './AddressFields';

/** Add a wallet association, or edit the claim on an existing one (status changes are recorded in its history). */
export function AddressModal({ entityId, existing, onClose, onSaved }: {
  entityId: string; existing?: EntityAddress; onClose: () => void; onSaved: (e: Entity) => void;
}) {
  const [d, setD] = useState<AddrDraft>(existing
    ? { network: existing.network, address: existing.address, label: existing.label, role: existing.role, status: existing.status, confidence: existing.confidence,
        sourceType: existing.sourceType, sourceRef: existing.sourceRef, basis: existing.basis, notes: existing.notes }
    : emptyAddr());
  const [changeNote, setChangeNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const err = addrError(d, !existing);
  const statusChanged = !!existing && existing.status !== d.status;

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const saved = existing
        ? await api.entities.updateAddress(existing.id, { label: d.label, role: d.role, status: d.status, confidence: d.confidence, sourceType: d.sourceType, sourceRef: d.sourceRef, basis: d.basis, notes: d.notes, changeNote })
        : await api.entities.addAddress(entityId, d);
      onSaved(saved);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <Modal wide title={existing ? 'Edit wallet claim' : 'Add wallet'} onClose={onClose}
      footer={<><Button onClick={onClose}>Cancel</Button><Button variant="primary" loading={busy} disabled={!!err} onClick={() => void save()}>Save</Button></>}>
      <div className="space-y-4">
        {error && <ErrorBanner message={error} />}
        <AddressFields idp="ad" value={d} lockAddress={!!existing} onChange={(p) => setD((x) => ({ ...x, ...p }))} />
        {statusChanged && <div><Label htmlFor="ad-note" hint="stored in the attribution history">Reason for status change</Label><Input id="ad-note" value={changeNote} onChange={(e) => setChangeNote(e.target.value)} /></div>}
        {err && <p className="text-[11px]">⚠ {err}</p>}
      </div>
    </Modal>
  );
}
