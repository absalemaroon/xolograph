import { useState } from 'react';
import { api } from '@/api/endpoints';
import { CaseSelect } from '@/features/cases/CaseSelect';
import { Button, ErrorBanner, Input, Label, Modal, Select, Textarea } from '@/components/ui';
import { isEthAddress } from '@/lib/validate';

const CATEGORIES = ['UNCATEGORIZED', 'EXCHANGE', 'MIXER', 'BRIDGE', 'DEFI', 'SERVICE', 'CONTRACT', 'INDIVIDUAL'];

export function LabelModal({ defaultAddress = '', onClose, onSaved }: { defaultAddress?: string; onClose: () => void; onSaved?: () => void }) {
  const [caseId, setCaseId] = useState('');
  const [address, setAddress] = useState(defaultAddress);
  const [name, setName] = useState('');
  const [category, setCategory] = useState('UNCATEGORIZED');
  const [source, setSource] = useState('ANALYST_ASSERTION');
  const [confidence, setConfidence] = useState('LOW');
  const [basis, setBasis] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const valid = isEthAddress(address) && name.trim() && basis.trim().length >= 3 && caseId;

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.labels.add(caseId, { address: address.trim(), name, category, source, confidence, basis });
      onSaved?.();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <Modal
      title="Add entity label"
      onClose={onClose}
      footer={<><Button onClick={onClose}>Cancel</Button><Button variant="primary" loading={busy} disabled={!valid} onClick={() => void save()}>Save label</Button></>}
    >
      <div className="space-y-4">
        <p className="text-[11px] text-fg-muted">A label is an attribution, not proof of ownership. It must state its source, confidence and basis.</p>
        {error && <ErrorBanner message={error} />}
        <div><Label htmlFor="lb-case">Case</Label><CaseSelect id="lb-case" value={caseId} onChange={setCaseId} /></div>
        <div><Label htmlFor="lb-addr">Address</Label><Input id="lb-addr" value={address} onChange={(e) => setAddress(e.target.value)} spellCheck={false} placeholder="0x…" /></div>
        <div className="grid grid-cols-2 gap-4">
          <div><Label htmlFor="lb-name">Name</Label><Input id="lb-name" value={name} maxLength={120} onChange={(e) => setName(e.target.value)} /></div>
          <div><Label htmlFor="lb-cat">Category</Label><Select id="lb-cat" value={category} onChange={(e) => setCategory(e.target.value)}>{CATEGORIES.map((c) => <option key={c}>{c}</option>)}</Select></div>
          <div><Label htmlFor="lb-src">Source</Label><Select id="lb-src" value={source} onChange={(e) => setSource(e.target.value)}><option value="ANALYST_ASSERTION">Analyst assertion</option><option value="THIRD_PARTY_PROVIDER">Third-party provider</option><option value="VERIFIED_SOURCE">Verified source</option></Select></div>
          <div><Label htmlFor="lb-conf">Confidence</Label><Select id="lb-conf" value={confidence} onChange={(e) => setConfidence(e.target.value)}><option>LOW</option><option>MEDIUM</option><option>HIGH</option></Select></div>
        </div>
        <div><Label htmlFor="lb-basis" hint="required">Basis / supporting evidence</Label><Textarea id="lb-basis" value={basis} onChange={(e) => setBasis(e.target.value)} placeholder="e.g. Funds traced from case wallet X; cluster matches published disclosure Y" /></div>
      </div>
    </Modal>
  );
}
