import { Input, Label, Select, Textarea } from '@/components/ui';
import { NETWORKS, STATUSES, isValidAddress, needsSource } from './constants';

export interface AddrDraft {
  network: string; address: string; label: string; role: string; status: string; confidence: string;
  sourceType: string; sourceRef: string; basis: string; notes: string;
}
export const emptyAddr = (address = ''): AddrDraft => ({
  network: 'ethereum-mainnet', address, label: '', role: '', status: 'UNKNOWN', confidence: 'LOW', sourceType: '', sourceRef: '', basis: '', notes: '',
});

/** Mirrors the engine's rules for fast feedback; the engine remains authoritative. */
export function addrError(d: AddrDraft, checkAddress = true): string | null {
  if (checkAddress && !isValidAddress(d.network, d.address)) return 'Enter a valid address for the selected network.';
  if (needsSource(d.status) && !d.sourceRef.trim()) return `${d.status.replace('_', ' ').toLowerCase()} attribution needs a source (URL or citation).`;
  if (d.status === 'INVESTIGATOR_ATTRIBUTED' && !d.sourceRef.trim() && d.basis.trim().length < 3) return 'Investigator attribution needs a basis or a source.';
  return null;
}

export function AddressFields({ value, onChange, lockAddress, idp }: { value: AddrDraft; onChange: (p: Partial<AddrDraft>) => void; lockAddress?: boolean; idp: string }) {
  const hint = STATUSES.find((s) => s.value === value.status)?.hint;
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-4">
        <div><Label htmlFor={`${idp}-net`}>Network</Label>
          <Select id={`${idp}-net`} value={value.network} disabled={lockAddress} onChange={(e) => onChange({ network: e.target.value })}>{NETWORKS.map((n) => <option key={n.value} value={n.value}>{n.label}</option>)}</Select></div>
        <div className="col-span-2"><Label htmlFor={`${idp}-addr`}>Address</Label>
          <Input id={`${idp}-addr`} value={value.address} disabled={lockAddress} spellCheck={false} onChange={(e) => onChange({ address: e.target.value })} placeholder={value.network === 'solana' ? 'Base58 address' : '0x…'} /></div>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div><Label htmlFor={`${idp}-lbl`} hint="optional">Label</Label><Input id={`${idp}-lbl`} value={value.label} maxLength={120} onChange={(e) => onChange({ label: e.target.value })} placeholder="e.g. Hot wallet 3" /></div>
        <div><Label htmlFor={`${idp}-role`} hint="optional">Role</Label><Input id={`${idp}-role`} value={value.role} maxLength={80} onChange={(e) => onChange({ role: e.target.value })} placeholder="e.g. treasury, deposit" /></div>
        <div><Label htmlFor={`${idp}-st`}>Attribution status</Label>
          <Select id={`${idp}-st`} value={value.status} onChange={(e) => onChange({ status: e.target.value })}>{STATUSES.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}</Select></div>
        <div><Label htmlFor={`${idp}-cf`}>Confidence</Label>
          <Select id={`${idp}-cf`} value={value.confidence} onChange={(e) => onChange({ confidence: e.target.value })}><option>LOW</option><option>MEDIUM</option><option>HIGH</option></Select></div>
      </div>
      {hint && <p className="text-[11px] text-fg-muted">{hint}</p>}
      <div><Label htmlFor={`${idp}-src`} hint={needsSource(value.status) ? 'required' : 'URL or citation'}>Source</Label>
        <Input id={`${idp}-src`} value={value.sourceRef} onChange={(e) => onChange({ sourceRef: e.target.value })} placeholder="https://… or document reference" /></div>
      <div><Label htmlFor={`${idp}-basis`} hint={value.status === 'INVESTIGATOR_ATTRIBUTED' ? 'required unless a source is given' : 'optional'}>Basis / reasoning</Label>
        <Textarea id={`${idp}-basis`} value={value.basis} onChange={(e) => onChange({ basis: e.target.value })} placeholder="What supports this claim? A transfer between two wallets is not, by itself, evidence of common ownership." /></div>
    </div>
  );
}
