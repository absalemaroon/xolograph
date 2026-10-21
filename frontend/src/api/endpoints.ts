import { useSettings } from '@/state/settingsStore';
import { request } from './client';
import type { AuditEntry, Case, CaseStatus, ChainResult, CustodyEvent, Evidence, Finding, Note } from './types';

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
    add: (caseId: string, f: { kind: string; title: string; note: string; refType: string; refId: string }) =>
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
  audit: {
    list: (limit = 200) => request<AuditEntry[]>('GET', `/api/audit?limit=${limit}`),
    verify: () => request<ChainResult>('GET', '/api/audit/verify'),
  },
};
