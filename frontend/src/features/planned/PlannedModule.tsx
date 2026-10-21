import { Badge, PageHeader, Panel } from '@/components/ui';

export interface PlannedSpec { title: string; summary: string; capabilities: string[]; honesty: string }

export const PLANNED: Record<'explorer' | 'graph' | 'entities' | 'reports', PlannedSpec> = {
  explorer: {
    title: 'On-chain Explorer',
    summary: 'Address and transaction investigation backed by real chain data.',
    capabilities: [
      'Address profile: first/last activity, inflow/outflow, top counterparties',
      'Transaction detail with decoded token transfers',
      'Source provenance and coverage limits shown on every record',
      'One-click “attach to case” as a finding with a source snapshot',
    ],
    honesty: 'Previously generated placeholder data was removed: nothing is shown unless it comes from a real source.',
  },
  graph: {
    title: 'Network Graph',
    summary: 'Interactive WebGL transaction graph built from retrieved records.',
    capabilities: [
      'Directional address/transaction graph with neighbour expansion',
      'Path highlighting between two addresses',
      'Filters by value, time window, direction and asset',
      'Graph export (JSON / image) attached to a case',
    ],
    honesty: 'Requires the Stage 3 chain-data endpoints in the engine.',
  },
  entities: {
    title: 'Entities',
    summary: 'Documented entity labels with source and confidence.',
    capabilities: [
      'Labels carry source, date and confidence — never bare assertions',
      'Relationship exploration between labelled addresses',
      'Analyst-defined entities with notes',
    ],
    honesty: 'Xolograph will not assert real-world ownership without cited evidence.',
  },
  reports: {
    title: 'Reports',
    summary: 'Case reports and exports that preserve source links.',
    capabilities: [
      'Markdown / JSON / PDF export per case',
      'Evidence digests and full chain-of-custody appendix',
      'Findings linked to their underlying source records',
      'Export checksum recorded in the audit log',
    ],
    honesty: 'Engine-side report builder lands with Stage 3.',
  },
};

export function PlannedModule({ id }: { id: keyof typeof PLANNED }) {
  const spec = PLANNED[id];
  return (
    <>
      <PageHeader title={spec.title} subtitle={spec.summary} actions={<Badge>Planned · Stage 3</Badge>} />
      <Panel title="Scope">
        <ul className="space-y-2 text-xs">
          {spec.capabilities.map((c) => <li key={c} className="flex gap-3"><span aria-hidden className="text-fg-muted">○</span>{c}</li>)}
        </ul>
        <p className="mt-5 border-t border-line pt-4 text-[11px] text-fg-muted">{spec.honesty}</p>
      </Panel>
    </>
  );
}
