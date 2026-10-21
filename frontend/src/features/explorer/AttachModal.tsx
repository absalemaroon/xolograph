import { useState } from 'react';
import { api } from '@/api/endpoints';
import { CaseSelect } from '@/features/cases/CaseSelect';
import { Button, ErrorBanner, Input, Label, Modal, Textarea } from '@/components/ui';

export interface AttachSpec {
  kind: string;
  refType: string;
  refId: string;
  defaultTitle: string;
  snapshot: unknown; // preserved verbatim as the finding's source snapshot
}

export function AttachModal({ spec, onClose }: { spec: AttachSpec; onClose: () => void }) {
  const [caseId, setCaseId] = useState('');
  const [title, setTitle] = useState(spec.defaultTitle);
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.findings.add(caseId, { kind: spec.kind, title, note, refType: spec.refType, refId: spec.refId, sourceJson: JSON.stringify(spec.snapshot) });
      setDone(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      title="Attach to case"
      onClose={onClose}
      footer={done ? <Button variant="primary" onClick={onClose}>Done</Button> : (
        <><Button onClick={onClose}>Cancel</Button><Button variant="primary" loading={busy} disabled={!caseId || !title.trim()} onClick={() => void save()}>Record finding</Button></>
      )}
    >
      {done ? <p className="text-xs">Finding recorded with a snapshot of the source data.</p> : (
        <div className="space-y-4">
          {error && <ErrorBanner message={error} />}
          <div><Label htmlFor="att-case">Case</Label><CaseSelect id="att-case" value={caseId} onChange={setCaseId} /></div>
          <div><Label htmlFor="att-title">Title</Label><Input id="att-title" value={title} onChange={(e) => setTitle(e.target.value)} /></div>
          <div><Label hint={`${spec.kind} · ${spec.refId.slice(0, 18)}${spec.refId.length > 18 ? '…' : ''}`}>Reference</Label></div>
          <div><Label htmlFor="att-note" hint="optional">Note</Label><Textarea id="att-note" value={note} onChange={(e) => setNote(e.target.value)} /></div>
        </div>
      )}
    </Modal>
  );
}
