import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { BoardClaim, BoardLink, BoardNode } from '@/api/types';
import { Button, ErrorBanner, Input, Label, Modal, Select, Spinner, Textarea } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';

/** Draw (or edit) a link. The claim type is explicit: a FACT needs evidence or a source, otherwise it is a hypothesis. */
export function LinkModal({ caseId, source, target, existing, onClose, onSaved }: {
  caseId: string; source: BoardNode; target: BoardNode; existing?: BoardLink; onClose: () => void; onSaved: () => void;
}) {
  const [relation, setRelation] = useState(existing?.relation ?? '');
  const [claim, setClaim] = useState<BoardClaim>(existing?.claim ?? 'HYPOTHESIS');
  const [confidence, setConfidence] = useState<string>(existing?.confidence ?? 'LOW');
  const [sourceRef, setSourceRef] = useState(existing?.sourceRef ?? '');
  const [note, setNote] = useState(existing?.note ?? '');
  const [picked, setPicked] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const evidence = useAsync(() => (existing ? Promise.resolve([]) : api.evidence.list(caseId)), [caseId, existing?.id]);

  const supported = sourceRef.trim() !== '' || picked.length > 0 || (existing?.evidence.length ?? 0) > 0;
  const factBlocked = claim === 'FACT' && !supported;
  const valid = relation.trim() !== '' && !factBlocked;

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      if (existing) await api.board.updateLink(existing.id, { relation, claim, confidence, sourceRef, note });
      else await api.board.addLink(caseId, { sourceId: source.id, targetId: target.id, relation, claim, confidence, sourceRef, note, evidenceIds: picked });
      onSaved();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <Modal wide title={existing ? 'Edit link' : 'New link'} onClose={onClose}
      footer={<><Button onClick={onClose}>Cancel</Button><Button variant="primary" loading={busy} disabled={!valid} onClick={() => void save()}>Save link</Button></>}>
      <div className="space-y-4">
        {error && <ErrorBanner message={error} />}
        <p className="text-xs"><span className="font-medium">{source.liveName || source.title}</span> → <span className="font-medium">{target.liveName || target.title}</span></p>
        <div><Label htmlFor="bl-rel" hint="e.g. funded, sent to, same operator">Relationship</Label><Input id="bl-rel" value={relation} maxLength={80} onChange={(e) => setRelation(e.target.value)} /></div>
        <div className="grid grid-cols-2 gap-4">
          <div><Label htmlFor="bl-claim">Claim</Label>
            <Select id="bl-claim" value={claim} onChange={(e) => setClaim(e.target.value as BoardClaim)}><option value="HYPOTHESIS">Hypothesis (unconfirmed)</option><option value="FACT">Fact (supported)</option></Select></div>
          <div><Label htmlFor="bl-conf">Confidence</Label><Select id="bl-conf" value={confidence} onChange={(e) => setConfidence(e.target.value)}><option>LOW</option><option>MEDIUM</option><option>HIGH</option></Select></div>
        </div>
        <div><Label htmlFor="bl-src" hint={claim === 'FACT' ? 'URL or citation, or attach evidence below' : 'optional'}>Source</Label><Input id="bl-src" value={sourceRef} onChange={(e) => setSourceRef(e.target.value)} /></div>
        {!existing && (
          <div>
            <Label>Supporting evidence from this case</Label>
            {evidence.loading && !evidence.data ? <Spinner /> : (evidence.data ?? []).length === 0 ? <p className="text-xs text-fg-muted">No evidence in this case yet.</p> : (
              <ul className="max-h-36 space-y-1 overflow-auto border border-line p-2">
                {(evidence.data ?? []).map((x) => (
                  <li key={x.id}><label className="flex items-center gap-2 text-xs"><input type="checkbox" checked={picked.includes(x.id)} onChange={(e) => setPicked((p) => (e.target.checked ? [...p, x.id] : p.filter((i) => i !== x.id)))} />{x.name}</label></li>
                ))}
              </ul>
            )}
          </div>
        )}
        <div><Label htmlFor="bl-note" hint="what supports this, and what would disprove it">Reasoning</Label><Textarea id="bl-note" value={note} onChange={(e) => setNote(e.target.value)} /></div>
        {factBlocked && <p className="text-[11px]">⚠ A fact needs at least one evidence item or a stated source. Otherwise save it as a hypothesis.</p>}
      </div>
    </Modal>
  );
}
