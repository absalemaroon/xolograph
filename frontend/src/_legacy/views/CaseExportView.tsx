import React, { useState } from 'react';
import { CaseReport } from '../types/report';
import { reportService } from '../services/reportService';
import { Button } from '../components/common/Button';

export const CaseExportView: React.FC = () => {
  const [report, setReport] = useState<CaseReport | null>(null);

  const handleGenerateReport = async () => {
    const generated = await reportService.generateCaseReport('inv_demo_01', 'Operation Sovereign Trace', 'Analysis tracing flow.');
    setReport(generated);
  };

  return (
    <div className="flex h-full flex-col space-y-4">
      <div className="flex items-center justify-between border-b border-[var(--border-subtle)] pb-3 font-mono-val text-xs">
        <h1 className="text-xl font-bold uppercase tracking-wider text-[var(--text-primary)]">CASE EXPORT & FORENSIC REPORTS</h1>
        <Button variant="primary" size="sm" onClick={handleGenerateReport}>GENERATE REPORT</Button>
      </div>
      <div className="flex-1 border border-[var(--border-default)] bg-[var(--bg-secondary)] flex flex-col overflow-hidden p-6">
        {report ? <pre className="text-xs text-[var(--text-secondary)]">{JSON.stringify(report, null, 2)}</pre> : <span className="text-[var(--text-muted)]">No report generated.</span>}
      </div>
    </div>
  );
};