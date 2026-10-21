import React from 'react';

interface SidebarProps {
  activeId: string;
  onSelect: (id: string) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ activeId, onSelect }) => {
  const menuItems = [
    { id: 'overview', label: 'OVERVIEW' },
    { id: 'investigations', label: 'CASES & FILES' },
    { id: 'evidence', label: 'EVIDENCE VAULT' },
    { id: 'blockchain', label: 'ON-CHAIN DATA' },
    { id: 'transactions', label: 'TRANSFER LOGS' },
    { id: 'graph', label: 'NETWORK GRAPH' },
    { id: 'entity', label: 'INTELLIGENCE' },
    { id: 'export', label: 'CASE EXPORT' },
    { id: 'settings', label: 'SETTINGS' },
  ];

  return (
    <aside className="w-64 h-full bg-[var(--bg-secondary)] border-r border-[var(--border-subtle)] flex flex-col font-mono-val">
      {/* BRANDING LOGO & HEADER */}
      <div className="h-32 flex flex-col items-center justify-center border-b border-[var(--border-subtle)] bg-[#000000] p-4">
        <img 
          src="/src/assets/xolograph.svg" 
          alt="Xolograph Logo" 
          className="w-16 h-16 mb-2"
        />
        <h1 className="text-sm font-bold uppercase tracking-[0.2em] text-white">
          XOLOGRAPH
        </h1>
      </div>

      {/* NAVIGATION */}
      <nav className="flex-1 overflow-y-auto py-4">
        <ul className="space-y-1">
          {menuItems.map((item) => {
            const isActive = activeId === item.id;
            return (
              <li key={item.id}>
                <button
                  onClick={() => onSelect(item.id)}
                  className={`w-full text-left px-6 py-3 text-xs uppercase tracking-wider transition-colors ${
                    isActive
                      ? 'bg-[var(--accent-mono)] text-[var(--accent-contrast)] font-bold'
                      : 'text-[var(--text-secondary)] hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  {isActive ? `> ${item.label}` : `  ${item.label}`}
                </button>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* STATUS FOOTER */}
      <div className="p-4 border-t border-[var(--border-subtle)] text-[10px] text-[var(--text-muted)] space-y-1">
        <div className="flex justify-between">
          <span>SYSTEM:</span>
          <span className="text-[var(--accent-mono)]">ONLINE</span>
        </div>
        <div className="flex justify-between">
          <span>VERSION:</span>
          <span>1.0.0-RC1</span>
        </div>
        <div className="flex justify-between">
          <span>RPC NODE:</span>
          <span>CONNECTED</span>
        </div>
      </div>
    </aside>
  );
};