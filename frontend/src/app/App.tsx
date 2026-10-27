import { AppShell } from '@/components/layout/AppShell';
import { AuditView } from '@/features/audit/AuditView';
import { CasesView } from '@/features/cases/CasesView';
import { EntitiesView } from '@/features/entities/EntitiesView';
import { EvidenceVaultView } from '@/features/evidence/EvidenceVaultView';
import { ExplorerView } from '@/features/explorer/ExplorerView';
import { GraphView } from '@/features/graph/GraphView';
import { OverviewView } from '@/features/overview/OverviewView';
import { ReportsView } from '@/features/reports/ReportsView';
import { ToolsView } from '@/features/tools/ToolsView';
import { SettingsView } from '@/features/settings/SettingsView';
import { useNav } from '@/state/navStore';

export function App() {
  const view = useNav((s) => s.view);
  return (
    <AppShell>
      {view === 'overview' && <OverviewView />}
      {view === 'cases' && <CasesView />}
      {view === 'evidence' && <EvidenceVaultView />}
      {view === 'audit' && <AuditView />}
      {view === 'explorer' && <ExplorerView />}
      {view === 'graph' && <GraphView />}
      {view === 'entities' && <EntitiesView />}
      {view === 'tools' && <ToolsView />}
      {view === 'reports' && <ReportsView />}
      {view === 'settings' && <SettingsView />}
    </AppShell>
  );
}
