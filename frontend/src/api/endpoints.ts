import { useSettings } from '@/state/settingsStore';
import { request } from './client';
import type {
  ActivityPage, AddressSummary, AuditEntry, Case, CaseStatus, ChainResult, ChainTx, CustodyEvent,
  AddressInput, BackupResult, Entity, EntityInput, Evidence, Finding, Label, Note, ReportBundle, RestoreStatus, StageResult,
} from './types';

const actor = (): string | undefined => useSettings.getState().analyst.trim() || undefined;
const enc = encodeURIComponent;

export const api = {
  health: () => request<{ status: string }>('GET', '/api/health'),
  cases: {
    list: () => request<Case[]>('GET', '/api/cases'),
    get: (id: string) => request<Case>('GET', `/api/cases/${enc(id)}`),
    create: (title: string, description: string) =>
      request<Case>('POST', '/api/cases', { title, description, actor: actor() }),
    setStatus: (id: string, status: CaseStatus) =>
      request<Case>('PATCH', `/api/cases/${enc(id)}`, { status, actor: actor() }),
  },
  notes: {
    list: (caseId: string) => request<Note[]>('GET', `/api/cases/${enc(caseId)}/notes`),
    add: (caseId: string, body: string) =>
      request<Note>('POST', `/api/cases/${enc(caseId)}/notes`, { body, actor: actor() }),
  },
  findings: {
    list: (caseId: string) => request<Finding[]>('GET', `/api/cases/${enc(caseId)}/findings`),
    add: (caseId: string, f: { kind: string; title: string; note: string; refType: string; refId: string; sourceJson?: string }) =>
      request<Finding>('POST', `/api/cases/${enc(caseId)}/findings`, { ...f, actor: actor() }),
  },
  evidence: {
    list: (caseId: string) => request<Evidence[]>('GET', `/api/cases/${enc(caseId)}/evidence`),
    import: (caseId: string, path: string) =>
      request<Evidence>('POST', `/api/cases/${enc(caseId)}/evidence`, { path, actor: actor() }),
    verify: (id: string) => request<Evidence>('POST', `/api/evidence/${enc(id)}/verify`, { actor: actor() }),
    custody: (id: string) =>
      request<{ events: CustodyEvent[]; chain: ChainResult }>('GET', `/api/evidence/${enc(id)}/custody`),
  },
  chain: {
    address: (a: string) => request<AddressSummary>('GET', `/api/chain/address/${enc(a)}`),
    activity: (a: string, kind: 'native' | 'token', cursor?: string) =>
      request<ActivityPage>('GET', `/api/chain/address/${enc(a)}/activity?kind=${kind}${cursor ? `&cursor=${enc(cursor)}` : ''}`),
    tx: (h: string) => request<ChainTx>('GET', `/api/chain/tx/${enc(h)}`),
  },
  labels: {
    list: (f: { address?: string; caseId?: string } = {}) => {
      const q = new URLSearchParams();
      if (f.address) q.set('address', f.address);
      if (f.caseId) q.set('caseId', f.caseId);
      return request<Label[]>('GET', `/api/labels?${q.toString()}`);
    },
    add: (caseId: string, l: { address: string; name: string; category: string; source: string; confidence: string; basis: string }) =>
      request<Label>('POST', `/api/cases/${enc(caseId)}/labels`, { ...l, actor: actor() }),
  },
  entities: {
    list: (q = '') => request<Entity[]>('GET', `/api/entities?q=${enc(q)}`),
    forAddress: (address: string, network = 'ethereum-mainnet') =>
      request<Entity[]>('GET', `/api/entities?address=${enc(address)}&network=${enc(network)}`),
    get: (id: string) => request<Entity>('GET', `/api/entities/${enc(id)}`),
    create: (e: EntityInput) => request<Entity>('POST', '/api/entities', { ...e, actor: actor() }),
    update: (id: string, e: EntityInput) => request<Entity>('PATCH', `/api/entities/${enc(id)}`, { ...e, actor: actor() }),
    remove: (id: string) => request<null>('DELETE', `/api/entities/${enc(id)}`, { actor: actor() }),
    addAddress: (id: string, a: AddressInput) => request<Entity>('POST', `/api/entities/${enc(id)}/addresses`, { ...a, actor: actor() }),
    updateAddress: (addrId: string, a: Omit<AddressInput, 'network' | 'address'> & { changeNote?: string }) =>
      request<Entity>('PATCH', `/api/entity-addresses/${enc(addrId)}`, { ...a, actor: actor() }),
    removeAddress: (addrId: string) => request<Entity>('DELETE', `/api/entity-addresses/${enc(addrId)}`, { actor: actor() }),
    addIdentifier: (id: string, i: { platform: string; value: string; status: string; sourceRef: string; confidence: string; notes: string }) =>
      request<Entity>('POST', `/api/entities/${enc(id)}/identifiers`, { ...i, actor: actor() }),
    removeIdentifier: (identId: string) => request<Entity>('DELETE', `/api/entity-identifiers/${enc(identId)}`, { actor: actor() }),
    linkEvidence: (id: string, evidenceId: string, addressId: string, description: string) =>
      request<Entity>('POST', `/api/entities/${enc(id)}/evidence`, { evidenceId, addressId, description, actor: actor() }),
    unlinkEvidence: (linkId: string) => request<Entity>('DELETE', `/api/entity-evidence/${enc(linkId)}`, { actor: actor() }),
  },
  reports: {
    export: (caseId: string) => request<ReportBundle>('POST', `/api/cases/${enc(caseId)}/report`, { actor: actor() }),
  },
  backup: {
    create: (path: string, includeEvidence: boolean) =>
      request<BackupResult>('POST', '/api/backup', { path, includeEvidence, actor: actor() }),
    status: () => request<RestoreStatus>('GET', '/api/restore'),
    stage: (path: string) => request<StageResult>('POST', '/api/restore/stage', { path, actor: actor() }),
    cancel: () => request<null>('DELETE', '/api/restore/stage', { actor: actor() }),
  },
  audit: {
    list: (limit = 200) => request<AuditEntry[]>('GET', `/api/audit?limit=${limit}`),
    verify: () => request<ChainResult>('GET', '/api/audit/verify'),
  },
};
