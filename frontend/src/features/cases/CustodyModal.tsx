import { api } from '@/api/endpoints';
import { Badge, Button, ErrorBanner, HashText, Modal, Spinner } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';
import { formatUTC } from '@/lib/format';

export function CustodyModal({ evidenceId, onClose }: { evidenceId: string; onClose: () => void }) {
  const { data, error, loading, reload } = useAsync(() => api.evidence.custody(evidenceId), [evidenceId]);
  return (
    <Modal title="Chain of custody" wide onClose={onClose} footer={<Button onClick={onClose}>Close</Button>}>
      {error && <ErrorBanner message={error} onRetry={reload} />}
      {loading && !data ? <Spinner /> : data && (
        <>
          <div className="mb-4 flex items-center gap-3 text-xs">
            {data.chain.valid
              ? <Badge tone="outline">● Chain valid</Badge>
              : <Badge tone="alert">▲ Chain broken at #{data.chain.brokenAt}</Badge>}
            <span className="text-fg-muted">{data.chain.count} events · hash-linked, tamper-evident</span>
          </div>
          <table className="x-table">
            <thead><tr><th>#</th><th>Time</th><th>Action</th><th>Actor</th><th>Event hash</th></tr></thead>
            <tbody>
              {data.events.map((e) => (
                <tr key={e.id}>
                  <td className="tabular-nums text-fg-muted">{e.id}</td>
                  <td className="whitespace-nowrap">{formatUTC(e.timestamp)}</td>
                  <td><Badge tone={e.action === 'INTEGRITY_FAILURE' ? 'alert' : 'neutral'}>{e.action}</Badge><div className="mt-1 max-w-xs break-words text-[10px] text-fg-muted">{e.detail}</div></td>
                  <td className="text-fg-soft">{e.actor}</td>
                  <td><HashText value={e.eventHash} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </Modal>
  );
}
