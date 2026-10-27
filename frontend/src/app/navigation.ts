import { Activity, FileSearch, FileText, FolderLock, Gauge, Network, ScrollText, Search, Settings, Users } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import type { ViewId } from '@/state/navStore';

export interface NavItem { id: ViewId; label: string; icon: LucideIcon; planned?: boolean }
export interface NavGroup { title: string; items: NavItem[] }

export const NAV_GROUPS: NavGroup[] = [
  {
    title: 'Investigate',
    items: [
      { id: 'overview', label: 'Overview', icon: Gauge },
      { id: 'cases', label: 'Cases', icon: FolderLock },
      { id: 'evidence', label: 'Evidence Vault', icon: Activity },
      { id: 'audit', label: 'Audit Log', icon: ScrollText },
    ],
  },
  {
    title: 'Intelligence',
    items: [
      { id: 'explorer', label: 'On-chain Explorer', icon: Search },
      { id: 'graph', label: 'Network Graph', icon: Network },
      { id: 'entities', label: 'Entities', icon: Users },
    ],
  },
  { title: 'Tools', items: [{ id: 'tools', label: 'File Inspector', icon: FileSearch }] },
  { title: 'Output', items: [{ id: 'reports', label: 'Reports', icon: FileText }] },
  { title: 'System', items: [{ id: 'settings', label: 'Settings', icon: Settings }] },
];

export const NAV_FLAT: NavItem[] = NAV_GROUPS.flatMap((g) => g.items);
export const viewLabel = (id: ViewId): string => NAV_FLAT.find((i) => i.id === id)?.label ?? id;
