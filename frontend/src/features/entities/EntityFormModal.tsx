import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { Entity } from '@/api/types';
import { Button, ErrorBanner, Input, Label, Modal, Select, Textarea } from '@/components/ui';
import { AddressFields, addrError, emptyAddr } from './AddressFields';
import type { AddrDraft } from './AddressFields';
import { ENTITY_TYPES, STATUSES, pretty } from './constants';

/** Create (no `entity`) or edit the profile. Wallets of an existing entity are managed on its Wallets tab. */
export function EntityFormModal({ entity, defaultAddress, defaultNetwork, onClose, onSaved }: {
  entity?: Entity; defaultAddress?: string; defaultNetwork?: string; onClose: () => void; onSaved: (e: Entity) => void;
}) {
  const [name, setName] = useState(entity?.name ?? '');
  const [type, setType] = useState(entity?.type ?? 'OTHER');
  const [description, setDescription] = useState(entity?.description ?? '');
  const [country, setCountry] = useState(entity?.country ?? '');
  const [tags, setTags] = useState((entity?.tags ?? []).join(', '));
  const [status, setStatus] = useState<string>(entity?.status ?? 'UNKNOWN');
  const [confidence, setConfidence] = useState<string>(entity?.confidence ?? 'LOW');
  const [notes, setNotes] = useState(entity?.notes ?? '');
  const [withWallet, setWithWallet] = useState(!!defaultAddress);
  const [wallet, setWallet] = useState<AddrDraft>(emptyAddr(defaultAddress, defaultNetwork));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const walletErr = !entity && withWallet ? addrError(wallet) : null;
  const valid = name.trim().length > 0 && !walletErr;

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const body = {
        name, type, description, country, notes, status, confidence,
        tags: tags.split(',').map((t) => t.trim()).filter(Boolean),
      };
      const saved = entity
        ? await api.entities.update(entity.id, body)
        : await api.entities.create({ ...body, addresses: withWallet ? [wallet] : [] });
      onSaved(saved);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <Modal wide title={entity ? 'Edit entity' : 'New entity'} onClose={onClose}
      footer={<><Button onClick={onClose}>Cancel</Button><Button variant="primary" loading={busy} disabled={!valid} onClick={() => void save()}>{entity ? 'Save changes' : 'Create entity'}</Button></>}>
      <div className="space-y-4">
        {error && <ErrorBanner message={error} />}
        <p className="text-[11px] text-fg-muted">An entity is an investigative record. It does not need a wallet, and it never merges wallets automatically.</p>
        <div className="grid grid-cols-3 gap-4">
          <div className="col-span-2"><Label htmlFor="en-name">Name</Label><Input id="en-name" value={name} maxLength={120} onChange={(e) => setName(e.target.value)} /></div>
          <div><Label htmlFor="en-type">Type</Label><Select id="en-type" value={type} onChange={(e) => setType(e.target.value)}>{ENTITY_TYPES.map((t) => <option key={t} value={t}>{pretty(t)}</option>)}</Select></div>
          <div><Label htmlFor="en-status" hint="identity claim">Attribution status</Label>
            <Select id="en-status" value={status} onChange={(e) => setStatus(e.target.value)}>{STATUSES.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}</Select></div>
          <div><Label htmlFor="en-conf">Confidence</Label><Select id="en-conf" value={confidence} onChange={(e) => setConfidence(e.target.value)}><option>LOW</option><option>MEDIUM</option><option>HIGH</option></Select></div>
          <div><Label htmlFor="en-country" hint="optional, sourced">Country</Label><Input id="en-country" value={country} maxLength={80} onChange={(e) => setCountry(e.target.value)} /></div>
        </div>
        <div><Label htmlFor="en-tags" hint="comma separated">Tags</Label><Input id="en-tags" value={tags} onChange={(e) => setTags(e.target.value)} placeholder="otc, market-maker" /></div>
        <div><Label htmlFor="en-desc">Description</Label><Textarea id="en-desc" value={description} onChange={(e) => setDescription(e.target.value)} /></div>
        <div><Label htmlFor="en-notes">Investigator notes</Label><Textarea id="en-notes" value={notes} onChange={(e) => setNotes(e.target.value)} /></div>
        {!entity && (
          <div className="border-t border-line pt-4">
            <label className="flex items-center gap-2 text-xs"><input type="checkbox" checked={withWallet} onChange={(e) => setWithWallet(e.target.checked)} /> Add a first wallet now</label>
            {withWallet && <div className="mt-4"><AddressFields idp="en-w" value={wallet} onChange={(p) => setWallet((w) => ({ ...w, ...p }))} />{walletErr && <p className="mt-3 text-[11px]">⚠ {walletErr}</p>}</div>}
          </div>
        )}
      </div>
    </Modal>
  );
}
