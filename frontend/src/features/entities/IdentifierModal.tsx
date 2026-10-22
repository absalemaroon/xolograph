import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { Entity } from '@/api/types';
import { Button, ErrorBanner, Input, Label, Modal, Select, Textarea } from '@/components/ui';
import { IDENT_STATUSES, PLATFORMS, pretty } from './constants';

export function IdentifierModal({ entityId, onClose, onSaved }: { entityId: string; onClose: () => void; onSaved: (e: Entity) => void }) {
  const [platform, setPlatform] = useState<string>('X');
  const [value, setValue] = useState('');
  const [status, setStatus] = useState('UNVERIFIED');
  const [sourceRef, setSourceRef] = useState('');
  const [confidence, setConfidence] = useState('LOW');
  const [notes, setNotes] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const strong = status === 'VERIFIED' || status === 'PUBLICLY_LISTED';
  const valid = value.trim() !== '' && (!strong || sourceRef.trim() !== '');

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      onSaved(await api.entities.addIdentifier(entityId, { platform, value, status, sourceRef, confidence, notes }));
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <Modal title="Add public identifier" onClose={onClose}
      footer={<><Button onClick={onClose}>Cancel</Button><Button variant="primary" loading={busy} disabled={!valid} onClick={() => void save()}>Save</Button></>}>
      <div className="space-y-4">
        {error && <ErrorBanner message={error} />}
        <p className="text-[11px] text-fg-muted">Public information only. A matching name, avatar or username is not proof of identity, so identifiers start as unverified.</p>
        <div className="grid grid-cols-3 gap-4">
          <div><Label htmlFor="id-pf">Platform</Label><Select id="id-pf" value={platform} onChange={(e) => setPlatform(e.target.value)}>{PLATFORMS.map((p) => <option key={p} value={p}>{pretty(p)}</option>)}</Select></div>
          <div className="col-span-2"><Label htmlFor="id-val">Handle, URL or address</Label><Input id="id-val" value={value} maxLength={500} onChange={(e) => setValue(e.target.value)} /></div>
          <div className="col-span-2"><Label htmlFor="id-st">Status</Label><Select id="id-st" value={status} onChange={(e) => setStatus(e.target.value)}>{IDENT_STATUSES.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}</Select></div>
          <div><Label htmlFor="id-cf">Confidence</Label><Select id="id-cf" value={confidence} onChange={(e) => setConfidence(e.target.value)}><option>LOW</option><option>MEDIUM</option><option>HIGH</option></Select></div>
        </div>
        <div><Label htmlFor="id-src" hint={strong ? 'required' : 'optional'}>Source</Label><Input id="id-src" value={sourceRef} onChange={(e) => setSourceRef(e.target.value)} placeholder="Where this was found" /></div>
        <div><Label htmlFor="id-notes" hint="optional">Notes</Label><Textarea id="id-notes" value={notes} onChange={(e) => setNotes(e.target.value)} /></div>
      </div>
    </Modal>
  );
}
