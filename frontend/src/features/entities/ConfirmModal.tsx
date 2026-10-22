import { useState } from 'react';
import { Button, ErrorBanner, Modal } from '@/components/ui';

export function ConfirmModal({ title, message, confirmLabel, onConfirm, onClose }: {
  title: string; message: string; confirmLabel: string; onConfirm: () => Promise<void>; onClose: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const run = async () => {
    setBusy(true);
    setError(null);
    try { await onConfirm(); onClose(); } catch (e) { setError(e instanceof Error ? e.message : String(e)); setBusy(false); }
  };
  return (
    <Modal title={title} onClose={onClose} footer={<><Button onClick={onClose}>Cancel</Button><Button variant="danger" loading={busy} onClick={() => void run()}>{confirmLabel}</Button></>}>
      <div className="space-y-3">{error && <ErrorBanner message={error} />}<p className="text-xs">{message}</p></div>
    </Modal>
  );
}
