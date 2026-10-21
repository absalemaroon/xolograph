import React, { useState, useEffect } from 'react';
import { useThemeStore } from '../state/themeStore';
import { Sidebar } from '../components/navigation/Sidebar';
import { TypographySettings } from '../components/settings/TypographySettings';
import { OverviewView } from '../views/OverviewView';
import { InvestigationsView } from '../views/InvestigationsView';
import { InvestigationDetailView } from '../views/InvestigationDetailView';
import { EvidenceVaultView } from '../views/EvidenceVaultView';
import { BlockchainExplorerView } from '../views/BlockchainExplorerView';
import { TransactionExplorerView } from '../views/TransactionExplorerView';
import { NetworkGraphView } from '../views/NetworkGraphView';
import { EntityIntelligenceView } from '../views/EntityIntelligenceView';
import { CaseExportView } from '../views/CaseExportView';
import '../styles/tokens.css';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<string>('overview');
  const [selectedCaseId, setSelectedCaseId] = useState<string | null>(null);
  const { mode, typography } = useThemeStore();

  useEffect(() => {
    const root = document.documentElement;
    root.setAttribute('data-theme', mode);
    root.style.setProperty('--font-interface', typography.interfaceFont === 'System Sans' ? 'sans-serif' : typography.interfaceFont);
    root.style.setProperty('--font-mono', typography.monoFont === 'System Mono' ? 'monospace' : typography.monoFont);
    root.style.setProperty('--font-size-base', `${typography.baseFontSize}px`);
  }, [mode, typography]);

  const renderContent = () => {
    if (selectedCaseId) {
      return <InvestigationDetailView caseId={selectedCaseId} onBack={() => setSelectedCaseId(null)} />;
    }

    switch (activeTab) {
      case 'overview':
        return <OverviewView />;
      case 'investigations':
        return <InvestigationsView onSelectCase={(id) => setSelectedCaseId(id)} />;
      case 'blockchain':
        return <BlockchainExplorerView />;
      case 'graph':
        return <NetworkGraphView />;
      case 'entity':
        return <EntityIntelligenceView />;
      case 'transactions':
        return <TransactionExplorerView />;
      case 'evidence':
        return <EvidenceVaultView />;
      case 'export':
        return <CaseExportView />;
      case 'settings':
        return <TypographySettings />;
      default:
        return (
          <div className="p-8 border border-dashed border-[var(--border-strong)] text-center">
            <h2 className="text-sm font-mono-val uppercase text-[var(--text-primary)]">
              WORKSPACE [{activeTab.toUpperCase()}] INITIALIZING
            </h2>
            <p className="text-xs text-[var(--text-muted)] font-mono-val mt-1">
              Module pipeline scheduled for subsequent milestone integration.
            </p>
          </div>
        );
    }
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[var(--bg-primary)] text-[var(--text-primary)]">
      <Sidebar activeId={selectedCaseId ? 'investigations' : activeTab} onSelect={(tab) => { setSelectedCaseId(null); setActiveTab(tab); }} />
      
      <main className="flex-1 flex flex-col overflow-hidden bg-grid-pattern">
        <header className="h-12 border-b border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 flex items-center justify-between">
          <span className="text-xs font-mono-val text-[var(--text-muted)]">
            ACTIVE CASE: <span className="text-[var(--text-primary)]">{selectedCaseId || 'NONE SELECTED'}</span>
          </span>
          <span className="text-xs font-mono-val text-[var(--text-muted)]">
            TYPOGRAPHY: {typography.interfaceFont} // {typography.monoFont}
          </span>
        </header>

        <div className="flex-1 overflow-y-auto p-6">
          {renderContent()}
        </div>
      </main>
    </div>
  );
};