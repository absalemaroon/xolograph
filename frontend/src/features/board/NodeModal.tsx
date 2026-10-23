import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { BoardKind, BoardNode } from '@/api/types';
import { Button, ErrorBanner, Input, Label, Modal, Select, Spinner, Textarea } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { NETWORKS, isValidAddress } from '../entities/constants';

const KINDS: { value: BoardKind; label: string }[] = [
  { value: 'ENTITY', label: 'Entity' }, { value: 'WALLET', label: 'Wallet' }, { value: 'EVIDENCE', label: 'Evidence item' },
  { value: 'NOTE', label: 'Note' }, { value: 'EVENT', label: 'Event' },
];

/** Add an item to the board (or, with `existing`, edit a node's title and text). Referenced records are never copied. */
export function NodeModal({ caseId, at, existing, initialKind, onClose, onSaved }: {
  caseId: string; at: { x: number; y: number }; existing?: BoardNode; initialKind?: BoardKind; onClose: () => void; onSaved: () => void;
}) {
  const [kind, setKind] = useState<BoardKind>(existing?.kind ?? initialKind ?? 'ENTITY');
  const [refId, setRefId] = useState(existing?.refId ?? '');
  const [network, setNetwork] = useState('ethereum-mainnet');
  const [title, setTitle] = useState(existing?.title ?? '');
  const [body, setBody] = useState(existing?.body ?? '');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const entities = useAsync(() => (kind === 'ENTITY' && !existing ? api.entities.list('') : Promise.resolve([])), [kind, existing?.id]);
  const evidence = useAsync(() => (kind === 'EVIDENCE' && !existing ? api.evidence.list(caseId) : Promise.resolve([])), [kind, caseId, existing?.id]);

  const textKind = kind === 'NOTE' || kind === 'EVENT';
  const valid = existing ? title.trim() !== '' : textKind ? title.trim() !== '' : kind === 'WALLET' ? isValidAddress(network, refId) : refId !== '';

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      if (existing) await api.board.updateNode(existing.id, title, body);
      else await api.board.addNode(caseId, { kind, refId: textKind ? '' : refId, network: kind === 'WALLET' ? network : '', title, body, x: at.x, y: at.y });
      onSaved();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <Modal title={existing ? 'Edit item' : 'Add to board'} onClose={onClose}
      footer={<><Button onClick={onClose}>Cancel</Button><Button variant="primary" loading={busy} disabled={!valid} onClick={() => void save()}>{existing ? 'Save' : 'Add'}</Button></>}>
      <div className="space-y-4">
        {error && <ErrorBanner message={error} />}
        {!existing && (
          <div><Label htmlFor="bn-kind">Type</Label>
            <Select id="bn-kind" value={kind} onChange={(e) => { setKind(e.target.value as BoardKind); setRefId(''); }}>{KINDS.map((k) => <option key={k.value} value={k.value}>{k.label}</option>)}</Select></div>
        )}
        {!existing && kind === 'ENTITY' && (
          <div><Label htmlFor="bn-ent">Entity</Label>
            {entities.loading && !entities.data ? <Spinner /> : (entities.data ?? []).length === 0 ? <p className="text-xs text-fg-muted">No entities yet. Create one on the Entities page first.</p> : (
              <Select id="bn-ent" value={refId} onChange={(e) => setRefId(e.target.value)}><option value="">Select…</option>{(entities.data ?? []).map((x) => <option key={x.id} value={x.id}>{x.name}</option>)}</Select>)}</div>
        )}
        {!existing && kind === 'WALLET' && (
          <div className="grid grid-cols-3 gap-4">
            <div><Label htmlFor="bn-net">Network</Label><Select id="bn-net" value={network} onChange={(e) => setNetwork(e.target.value)}>{NETWORKS.map((n) => <option key={n.value} value={n.value}>{n.label}</option>)}</Select></div>
            <div className="col-span-2"><Label htmlFor="bn-addr">Address</Label><Input id="bn-addr" value={refId} spellCheck={false} onChange={(e) => setRefId(e.target.value.trim())} /></div>
          </div>
        )}
        {!existing && kind === 'EVIDENCE' && (
          <div><Label htmlFor="bn-ev">Evidence in this case</Label>
            {evidence.loading && !evidence.data ? <Spinner /> : (evidence.data ?? []).length === 0 ? <p className="text-xs text-fg-muted">This case has no evidence yet. Import files in the Evidence tab.</p> : (
              <Select id="bn-ev" value={refId} onChange={(e) => setRefId(e.target.value)}><option value="">Select…</option>{(evidence.data ?? []).map((x) => <option key={x.id} value={x.id}>{x.name}</option>)}</Select>)}</div>
        )}
        <div><Label htmlFor="bn-title" hint={textKind || existing ? 'required' : 'optional, defaults to the record name'}>Title</Label><Input id="bn-title" value={title} maxLength={120} onChange={(e) => setTitle(e.target.value)} /></div>
        <div><Label htmlFor="bn-body" hint="optional">Details</Label><Textarea id="bn-body" value={body} onChange={(e) => setBody(e.target.value)} /></div>
      </div>
    </Modal>
  );
}
