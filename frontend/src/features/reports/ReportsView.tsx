import { Download, FileText } from 'lucide-react';
import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { ReportBundle } from '@/api/types';
import { Button, ErrorBanner, HashText, Label, PageHeader, Panel } from '@/components/ui';
import { downloadText } from '@/lib/download';
import { formatUTC } from '@/lib/format';
import { CaseSelect } from '../cases/CaseSelect';

export function ReportsView() {
  const [caseId, setCaseId] = useState('');
  const [bundle, setBundle] = useState<ReportBundle | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const generate = async () => {
    setBusy(true);
    setError(null);
    try { setBundle(await api.reports.export(caseId)); }
    catch (e) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setBusy(false); }
  };

  const base = `xolograph-report-${caseId.replace(/^case_/, '').slice(0, 8)}`;

  return (
    <>
      <PageHeader title="Reports" subtitle="Export a case with its evidence digests, custody chains, findings and labels. Each export is hashed and recorded in the audit log." />
      <Panel title="Generate report">
        <div className="flex flex-wrap items-end gap-4">
          <div className="w-80"><Label htmlFor="rep-case">Case</Label><CaseSelect id="rep-case" openOnly={false} value={caseId} onChange={(id) => { setCaseId(id); setBundle(null); }} /></div>
          <Button variant="primary" icon={<FileText size={14} />} loading={busy} disabled={!caseId} onClick={() => void generate()}>Generate</Button>
        </div>
      </Panel>
      {error && <div className="mt-4"><ErrorBanner message={error} /></div>}
      {bundle && (
        <div className="mt-5 space-y-5">
          <Panel
            title="Export"
            actions={
              <>
                <Button size="sm" icon={<Download size={12} />} onClick={() => downloadText(`${base}.md`, bundle.markdown, 'text/markdown')}>Markdown</Button>
                <Button size="sm" icon={<Download size={12} />} onClick={() => downloadText(`${base}.json`, bundle.json, 'application/json')}>JSON</Button>
              </>
            }
          >
            <dl className="space-y-2 text-xs">
              <div className="flex flex-wrap items-center justify-between gap-3"><dt className="text-fg-muted">Generated</dt><dd>{formatUTC(bundle.generatedAt)}</dd></div>
              <div className="flex flex-wrap items-center justify-between gap-3"><dt className="text-fg-muted">Markdown SHA-256</dt><dd><HashText value={bundle.markdownSha256} head={16} tail={8} /></dd></div>
              <div className="flex flex-wrap items-center justify-between gap-3"><dt className="text-fg-muted">JSON SHA-256</dt><dd><HashText value={bundle.jsonSha256} head={16} tail={8} /></dd></div>
            </dl>
            <p className="mt-3 text-[11px] text-fg-muted">Verify a downloaded file with <code>sha256sum</code> / <code>shasum -a 256</code>; it must equal the digest above and the one in the audit log.</p>
          </Panel>
          <Panel title="Preview (Markdown)">
            <pre className="max-h-[28rem] overflow-auto whitespace-pre-wrap break-words text-[11px] leading-relaxed">{bundle.markdown}</pre>
          </Panel>
        </div>
      )}
    </>
  );
}
