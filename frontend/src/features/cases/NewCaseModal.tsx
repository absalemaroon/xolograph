import { useState } from 'react';
import type { FormEvent } from 'react';
import { api } from '@/api/endpoints';
import type { Case } from '@/api/types';
import { Button, ErrorBanner, Input, Label, Modal, Textarea } from '@/components/ui';

export function NewCaseModal({ onClose, onCreated }: { onClose: () => void; onCreated: (c: Case) => void }) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      onCreated(await api.cases.create(title, description));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  };

  return (
    <Modal
      title="New case"
      onClose={onClose}
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" loading={busy} disabled={!title.trim()} onClick={(e) => void submit(e as unknown as FormEvent)}>Create case</Button>
        </>
      }
    >
      <form onSubmit={(e) => void submit(e)} className="space-y-4">
        {error && <ErrorBanner message={error} />}
        <div>
          <Label htmlFor="case-title">Title</Label>
          <Input id="case-title" value={title} maxLength={200} onChange={(e) => setTitle(e.target.value)} placeholder="e.g. Bridge exploit — Q4" />
        </div>
        <div>
          <Label htmlFor="case-desc" hint="optional">Description</Label>
          <Textarea id="case-desc" value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <button type="submit" className="hidden" />
      </form>
    </Modal>
  );
}
