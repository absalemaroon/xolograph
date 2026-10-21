import { AppShell } from '@/components/layout/AppShell';
import { AuditView } from '@/features/audit/AuditView';
import { CasesView } from '@/features/cases/CasesView';
import { EvidenceVaultView } from '@/features/evidence/EvidenceVaultView';
import { OverviewView } from '@/features/overview/OverviewView';
import { PlannedModule } from '@/features/planned/PlannedModule';
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
      {(view === 'explorer' || view === 'graph' || view === 'entities' || view === 'reports') && <PlannedModule id={view} />}
      {view === 'settings' && <SettingsView />}
    </AppShell>
  );
}
