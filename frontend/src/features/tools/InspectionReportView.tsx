import { Check, Copy, Download, FileText, Pin, Plus } from 'lucide-react';
import { useState } from 'react';
import type { InspectionReport, InspectField } from '@/api/types';
import { Badge, Button, Panel } from '@/components/ui';
import { saveTextFile } from '@/lib/desktop';
import { formatBytes, formatUTC } from '@/lib/format';
import { AttachModal } from '../explorer/AttachModal';
import { reportToText } from './reportText';

const LEVEL_ORDER = ['sensitive', 'caution', 'info'] as const;
const LEVEL_LABEL = { sensitive: '◆ Sensitive', caution: '▲ Caution', info: '● Note' } as const;

/** Full digest, wrapped, with copy: a forensic hash should never be shown truncated. */
function FullHash({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try { await navigator.clipboard.writeText(value); setCopied(true); window.setTimeout(() => setCopied(false), 1200); } catch { /* clipboard unavailable */ }
  };
  return (
    <span className="inline-flex items-start gap-1.5">
      <code className="break-all font-mono text-[11px]">{value}</code>
      <button type="button" onClick={() => void copy()} aria-label="Copy hash" className="mt-0.5 shrink-0 text-fg-muted hover:text-fg">{copied ? <Check size={12} /> : <Copy size={12} />}</button>
    </span>
  );
}

function groups(fields: InspectField[]): [string, InspectField[]][] {
  const m = new Map<string, InspectField[]>();
  for (const f of fields) m.set(f.group, [...(m.get(f.group) ?? []), f]);
  return [...m.entries()];
}

export function InspectionReportView({ report: r, baseline, onSetBaseline }: { report: InspectionReport; baseline: InspectionReport | null; onSetBaseline: (r: InspectionReport) => void }) {
  const [attach, setAttach] = useState(false);
  const [saved, setSaved] = useState<string | null>(null);
  const stem = r.name.replace(/[^\w.-]+/g, '_');
  const save = async (name: string, text: string, mime: string) => {
    const res = await saveTextFile(name, text, mime);
    setSaved(res.saved ? (res.path ? `Saved to ${res.path}` : 'Saved') : null);
  };
  const sensitive = r.warnings.filter((w) => w.level === 'sensitive');

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-line pb-4">
        <div className="min-w-0">
          <h2 className="truncate text-base font-semibold" title={r.name}>{r.name}</h2>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <Badge tone="outline">{r.type.label}</Badge>
            <Badge>{r.type.mime}</Badge>
            <Badge>{formatBytes(r.sizeBytes)}</Badge>
            <Badge tone={r.type.category === 'executable' ? 'alert' : 'neutral'}>{r.type.category}</Badge>
          </div>
          <p className="mt-1.5 text-[11px] text-fg-muted">Type decided by: {r.type.basis}. Inspected {formatUTC(r.inspectedAt)}.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" icon={<Pin size={12} />} onClick={() => onSetBaseline(r)}>Use as comparison baseline</Button>
          <Button size="sm" icon={<Download size={12} />} onClick={() => void save(`inspection-${stem}.json`, JSON.stringify(r, null, 2), 'application/json')}>Export JSON</Button>
          <Button size="sm" icon={<FileText size={12} />} onClick={() => void save(`inspection-${stem}.txt`, reportToText(r), 'text/plain')}>Export report</Button>
          <Button size="sm" variant="primary" icon={<Plus size={12} />} onClick={() => setAttach(true)}>Attach to case</Button>
        </div>
      </div>
      {saved && <p className="text-[11px]">✓ {saved}</p>}

      {sensitive.length > 0 && (
        <div role="alert" className="border-2 border-line-strong bg-raised p-4 text-xs">
          <div className="mb-1 text-[11px] font-semibold uppercase tracking-[0.14em]">◆ Sensitive information inside this file</div>
          <ul className="space-y-1">{sensitive.map((w) => <li key={w.code}>{w.message}</li>)}</ul>
          <p className="mt-2 text-fg-muted">Strip this metadata before sharing the file outside the investigation, or share a derived copy.</p>
        </div>
      )}
      {r.warnings.some((w) => w.level !== 'sensitive') && (
        <Panel title="Findings and cautions">
          <ul className="space-y-2 text-xs">
            {LEVEL_ORDER.filter((l) => l !== 'sensitive').flatMap((l) => r.warnings.filter((w) => w.level === l).map((w) => (
              <li key={w.code + w.message}><span className="mr-2 font-semibold">{LEVEL_LABEL[l]}</span>{w.message}</li>
            )))}
          </ul>
        </Panel>
      )}

      <Panel title="File hashes" flush>
        <table className="x-table">
          <tbody>
            <tr><th className="w-28">SHA-256</th><td><FullHash value={r.hashes.sha256} /></td></tr>
            <tr><th>SHA-512</th><td><FullHash value={r.hashes.sha512} /></td></tr>
            <tr><th>SHA-1</th><td><FullHash value={r.hashes.sha1} /> <span className="text-[10px] text-fg-muted">legacy</span></td></tr>
            <tr><th>MD5</th><td><FullHash value={r.hashes.md5} /> <span className="text-[10px] text-fg-muted">legacy</span></td></tr>
          </tbody>
        </table>
      </Panel>

      {r.integrity && (
        <div className={`border-2 p-3 text-xs ${r.integrity.match ? 'border-line' : 'border-line-strong'}`} role="status">
          <b>{r.integrity.match ? '✓ MATCH' : '✕ DOES NOT MATCH'}</b> — {r.integrity.algorithm} comparison with the expected value.
          {!r.integrity.match && <div className="mt-1 break-all font-mono text-[11px]">expected {r.integrity.expected}<br />actual&nbsp;&nbsp;&nbsp;{r.integrity.actual}</div>}
        </div>
      )}
      {baseline && baseline !== r && (
        <div className="border border-line bg-raised p-3 text-xs" role="status">
          <b>Compared with baseline “{baseline.name}”:</b>{' '}
          {baseline.hashes.sha256 === r.hashes.sha256 ? '✓ the files are byte-for-byte identical (same SHA-256).' : '✕ the files are different (SHA-256 differs).'}
        </div>
      )}

      <div className="grid gap-5 lg:grid-cols-2">
        <Panel title="Detected properties">
          <dl className="grid grid-cols-[9rem_1fr] gap-x-3 gap-y-1.5 text-xs">
            <dt className="text-fg-muted">Type</dt><dd>{r.type.label}</dd>
            <dt className="text-fg-muted">MIME</dt><dd className="font-mono">{r.type.mime}</dd>
            {r.type.sniffedMime && r.type.sniffedMime !== r.type.mime && <><dt className="text-fg-muted">Generic sniff</dt><dd className="font-mono">{r.type.sniffedMime}</dd></>}
            <dt className="text-fg-muted">Declared extension</dt><dd>{r.type.declaredExtension ? `.${r.type.declaredExtension}` : '(none)'}{r.type.extensionMatches === true ? ' ✓ consistent' : r.type.extensionMatches === false ? ' ✕ does not match content' : ''}</dd>
            <dt className="text-fg-muted">Size</dt><dd>{r.sizeBytes.toLocaleString('en-US')} bytes</dd>
            {r.image && <><dt className="text-fg-muted">Image</dt><dd>{r.image.format.toUpperCase()}, {r.image.width} × {r.image.height} px</dd></>}
          </dl>
        </Panel>
        {r.gps && (
          <Panel title="GPS location (self-declared)">
            <p className="font-mono text-sm">{r.gps.latitude.toFixed(6)}, {r.gps.longitude.toFixed(6)}</p>
            {r.gps.altitudeMeters !== undefined && <p className="font-mono text-xs">Altitude {r.gps.altitudeMeters} m</p>}
            <p className="mt-2 text-[11px] text-fg-muted">Not shown on a map, to avoid sending the coordinates to a map service. GPS data can be missing, wrong, copied from another file or fabricated.</p>
          </Panel>
        )}
      </div>

      {groups(r.metadata).map(([g, fields]) => (
        <Panel key={g} title={`${g} · self-declared`} flush>
          <table className="x-table">
            <tbody>{fields.map((f, i) => <tr key={`${f.name}-${i}`}><th className="w-56 whitespace-nowrap">{f.name}</th><td className="break-words">{f.value}</td></tr>)}</tbody>
          </table>
        </Panel>
      ))}
      {r.metadata.length === 0 && <p className="text-xs text-fg-muted">No embedded metadata was found or this format's metadata is not read. That does not mean none exists, and absence of metadata proves nothing.</p>}
      {r.truncated && <p className="text-[11px] text-fg-muted">Only the first {formatBytes(r.parsedBytes)} were examined for metadata; the hashes cover the whole file.</p>}

      <Panel title="How to read this report">
        <ul className="space-y-1.5 text-[11px]">{r.limitations.map((l) => <li key={l}>▲ {l}</li>)}</ul>
      </Panel>

      {attach && <AttachModal spec={{ kind: 'FILE_INSPECTION', refType: 'file', refId: r.hashes.sha256, defaultTitle: `File inspection: ${r.name}`, snapshot: r }} onClose={() => setAttach(false)} />}
    </div>
  );
}
