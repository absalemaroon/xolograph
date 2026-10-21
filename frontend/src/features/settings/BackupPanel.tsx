import { useState } from 'react';
import { api } from '@/api/endpoints';
import type { BackupResult } from '@/api/types';
import { Badge, Button, ErrorBanner, HashText, Input, Label, Panel } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { isDesktop, pickBackupFile, pickSavePath, restartApp } from '@/lib/desktop';
import { formatBytes, formatUTC } from '@/lib/format';

type Busy = 'backup' | 'stage' | 'cancel' | 'restart';

export function BackupPanel() {
  const desktop = isDesktop();
  const status = useAsync(() => api.backup.status(), []);
  const [includeEvidence, setIncludeEvidence] = useState(true);
  const [manualPath, setManualPath] = useState('');
  const [busy, setBusy] = useState<Busy | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<BackupResult | null>(null);

  const run = async (kind: Busy, fn: () => Promise<void>): Promise<void> => {
    setBusy(kind);
    setError(null);
    try {
      await fn();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  };

  const createBackup = (): Promise<void> =>
    run('backup', async () => {
      const name = `xolograph-backup-${new Date().toISOString().slice(0, 10)}.zip`;
      const path = desktop ? await pickSavePath(name) : manualPath.trim();
      if (!path) return;
      setResult(await api.backup.create(path, includeEvidence));
    });

  const chooseBackup = (): Promise<void> =>
    run('stage', async () => {
      const path = desktop ? await pickBackupFile() : manualPath.trim();
      if (!path) return;
      await api.backup.stage(path);
      status.reload();
    });

  const cancel = (): Promise<void> =>
    run('cancel', async () => {
      await api.backup.cancel();
      status.reload();
    });

  const restart = (): Promise<void> => run('restart', () => restartApp());

  const pending = status.data?.pending ?? null;
  const last = status.data?.last ?? null;

  return (
    <Panel title="Backup & restore">
      <div className="space-y-5 text-xs">
        <p className="text-fg-muted">
          A backup is a single archive with a consistent database snapshot, the evidence vault and a SHA-256 manifest.
          Store it somewhere other than this computer. Restoring verifies every file first and keeps your current data in a <code>pre-restore</code> folder.
        </p>
        {error && <ErrorBanner message={error} />}
        {status.error && <ErrorBanner message={status.error} onRetry={status.reload} />}

        {!desktop && (
          <div>
            <Label htmlFor="bk-path" hint="browser/dev mode: absolute file path">Archive path</Label>
            <Input id="bk-path" value={manualPath} onChange={(e) => setManualPath(e.target.value)} placeholder="/path/to/xolograph-backup.zip" spellCheck={false} />
          </div>
        )}

        <div className="flex flex-wrap items-center gap-4">
          <label className="flex items-center gap-2">
            <input type="checkbox" checked={includeEvidence} onChange={(e) => setIncludeEvidence(e.target.checked)} className="accent-[var(--text-primary)]" />
            Include evidence files
          </label>
          <Button variant="primary" loading={busy === 'backup'} disabled={busy !== null || (!desktop && !manualPath.trim())} onClick={() => void createBackup()}>
            Create backup…
          </Button>
          <Button loading={busy === 'stage'} disabled={busy !== null || (!desktop && !manualPath.trim())} onClick={() => void chooseBackup()}>
            Restore from backup…
          </Button>
        </div>

        {result && (
          <div className="space-y-1.5 border border-line p-3">
            <div>● Backup created: <span className="break-all font-mono">{result.path}</span></div>
            <div className="text-fg-muted">{result.files} files · {formatBytes(result.sizeBytes)} · {result.includesEvidence ? 'evidence included' : 'database only'}</div>
            <div className="flex items-center gap-2"><span className="text-fg-muted">Archive SHA-256</span><HashText value={result.sha256} head={16} tail={8} /></div>
            <div className="text-[11px] text-fg-muted">Keep this digest separately to prove the archive is unchanged.</div>
          </div>
        )}

        {pending && (
          <div className="space-y-3 border border-dashed border-fg p-3">
            <div className="font-semibold uppercase tracking-wider">Restore ready — will replace current data</div>
            <dl className="space-y-1.5">
              <div className="flex justify-between gap-4"><dt className="text-fg-muted">Backup created</dt><dd>{formatUTC(pending.archiveCreatedAt)}</dd></div>
              <div className="flex justify-between gap-4"><dt className="text-fg-muted">Contents</dt><dd>{pending.cases} case(s) · {pending.files} files · {formatBytes(pending.bytes)} · {pending.includesEvidence ? 'with evidence' : 'database only'}</dd></div>
              <div className="flex justify-between gap-4"><dt className="text-fg-muted">Audit chain</dt><dd>{pending.auditChainValid ? <Badge tone="outline">● Valid · {pending.auditEntries} entries</Badge> : <Badge tone="alert">▲ Broken at #{pending.auditBrokenAt}</Badge>}</dd></div>
              <div className="flex items-center justify-between gap-4"><dt className="text-fg-muted">Archive SHA-256</dt><dd><HashText value={pending.archiveSha256} head={12} tail={8} /></dd></div>
            </dl>
            {!pending.auditChainValid && <p>▲ The audit chain in this backup does not verify. It may be damaged or altered; treat its contents with caution.</p>}
            <div className="flex flex-wrap items-center gap-3">
              {desktop
                ? <Button variant="primary" loading={busy === 'restart'} disabled={busy !== null} onClick={() => void restart()}>Restart &amp; restore</Button>
                : <span className="text-fg-muted">Restart the engine to apply the restore.</span>}
              <Button variant="ghost" loading={busy === 'cancel'} disabled={busy !== null} onClick={() => void cancel()}>Cancel</Button>
            </div>
          </div>
        )}

        {last && (
          <div className="text-[11px]">
            {last.applied
              ? <p>● Last restore applied {formatUTC(last.appliedAt)}. Previous data kept in <span className="break-all font-mono">{last.previousData}</span>.</p>
              : <p>▲ Last restore attempt failed ({formatUTC(last.appliedAt)}): {last.error}. Your existing data was left unchanged.</p>}
          </div>
        )}
      </div>
    </Panel>
  );
}
