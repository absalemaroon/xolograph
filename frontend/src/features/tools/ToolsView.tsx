import { FileSearch } from 'lucide-react';
import { useState } from 'react';
import type { FormEvent } from 'react';
import { api } from '@/api/endpoints';
import type { InspectionReport } from '@/api/types';
import { Button, EmptyState, ErrorBanner, Input, Label, PageHeader, Panel, Segmented, Select } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { isDesktop, pickEvidenceFile } from '@/lib/desktop';
import { formatBytes } from '@/lib/format';
import { CaseSelect } from '../cases/CaseSelect';
import { InspectionReportView } from './InspectionReportView';

export function ToolsView() {
  const [mode, setMode] = useState<'file' | 'vault'>('file');
  const [path, setPath] = useState('');
  const [expected, setExpected] = useState('');
  const [caseId, setCaseId] = useState('');
  const [evidenceId, setEvidenceId] = useState('');
  const [report, setReport] = useState<InspectionReport | null>(null);
  const [baseline, setBaseline] = useState<InspectionReport | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const evidence = useAsync(() => (mode === 'vault' && caseId ? api.evidence.list(caseId) : Promise.resolve([])), [mode, caseId]);

  const run = async (fn: () => Promise<InspectionReport>) => {
    setBusy(true);
    setError(null);
    try { setReport(await fn()); } catch (e) { setError(e instanceof Error ? e.message : String(e)); setReport(null); } finally { setBusy(false); }
  };
  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (mode === 'file') { if (path.trim()) void run(() => api.tools.inspectFile(path.trim(), expected.trim())); }
    else if (evidenceId) void run(() => api.tools.inspectEvidence(evidenceId, expected.trim()));
  };
  const ready = mode === 'file' ? path.trim() !== '' : evidenceId !== '';

  return (
    <>
      <PageHeader title="File & Metadata Inspector" subtitle="Identify a file by its content, compute hashes and read embedded metadata. Everything is processed locally and read-only; nothing is uploaded and nothing is executed." />
      <Panel title="Choose a file">
        <form onSubmit={submit} className="space-y-4">
          <Segmented label="Source" value={mode} onChange={(m) => { setMode(m); setReport(null); setError(null); }} options={[{ value: 'file', label: 'Local file' }, { value: 'vault', label: 'Evidence vault item' }]} />
          {mode === 'file' ? (
            <div className="flex items-end gap-3">
              <div className="flex-1"><Label htmlFor="in-path" hint="absolute path of a local file">File path</Label><Input id="in-path" value={path} spellCheck={false} onChange={(e) => setPath(e.target.value)} placeholder="/path/to/file" /></div>
              {isDesktop() && <Button onClick={() => void pickEvidenceFile().then((p) => { if (p) setPath(p); }).catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))}>Browse…</Button>}
            </div>
          ) : (
            <div className="grid gap-4 lg:grid-cols-2">
              <div><Label htmlFor="in-case">Case</Label><CaseSelect id="in-case" openOnly={false} value={caseId} onChange={(v) => { setCaseId(v); setEvidenceId(''); }} /></div>
              <div><Label htmlFor="in-ev">Evidence item</Label>
                <Select id="in-ev" value={evidenceId} onChange={(e) => setEvidenceId(e.target.value)} disabled={!caseId}>
                  <option value="">{caseId ? 'Select…' : 'Choose a case first'}</option>
                  {(evidence.data ?? []).map((x) => <option key={x.id} value={x.id}>{x.name} ({formatBytes(x.sizeBytes)})</option>)}
                </Select></div>
              <p className="text-[11px] text-fg-muted lg:col-span-2">The item is compared with the SHA-256 recorded at import unless you enter a different expected hash. Inspecting does not change its verification status or custody log.</p>
            </div>
          )}
          <div><Label htmlFor="in-exp" hint="optional · MD5, SHA-1, SHA-256 or SHA-512 hex">Expected hash to compare</Label><Input id="in-exp" value={expected} spellCheck={false} onChange={(e) => setExpected(e.target.value)} placeholder="e.g. sha256:2cf24dba…" /></div>
          <Button variant="primary" icon={<FileSearch size={14} />} loading={busy} disabled={!ready} onClick={submit as unknown as () => void}>Inspect</Button>
          <button type="submit" className="hidden" />
        </form>
      </Panel>

      <div className="mt-5 space-y-5">
        {error && <ErrorBanner message={error} />}
        {report ? <InspectionReportView report={report} baseline={baseline} onSetBaseline={setBaseline} /> : !error && (
          <EmptyState title="Nothing inspected yet" hint="Supported: images (EXIF, GPS, PNG text), PDFs, Office and OpenDocument files, archives (listed, not extracted), text, JSON, CSV and more. Unsupported or damaged files still get a type guess and hashes." />
        )}
      </div>
    </>
  );
}
