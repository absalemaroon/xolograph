import { useEffect } from 'react';
import { api } from '@/api/endpoints';
import { Select } from '@/components/ui';
import { useAsync } from '@/hooks/useAsync';

export function CaseSelect({ id, value, onChange, openOnly = true }: { id?: string; value: string; onChange: (id: string) => void; openOnly?: boolean }) {
  const { data, error, loading } = useAsync(() => api.cases.list(), []);
  const cases = (data ?? []).filter((c) => !openOnly || c.status === 'OPEN');

  useEffect(() => {
    if (!value && cases.length > 0) onChange(cases[0].id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data, value]);

  if (error) return <p className="text-xs">Could not load cases: {error}</p>;
  if (loading && !data) return <Select id={id} disabled><option>Loading…</option></Select>;
  if (cases.length === 0) return <p className="text-xs text-fg-muted">{openOnly ? 'No open cases. Create a case first.' : 'No cases yet.'}</p>;
  return (
    <Select id={id} value={value} onChange={(e) => onChange(e.target.value)}>
      {cases.map((c) => <option key={c.id} value={c.id}>{c.title}</option>)}
    </Select>
  );
}
