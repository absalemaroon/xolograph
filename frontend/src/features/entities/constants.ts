import type { AttributionStatus } from '@/api/types';

export const ENTITY_TYPES = ['INDIVIDUAL', 'ORGANIZATION', 'COMPANY', 'EXCHANGE', 'PROTOCOL', 'DAPP', 'TOKEN_PROJECT', 'FUND', 'WALLET_OPERATOR', 'OTHER'] as const;

export const STATUSES: { value: AttributionStatus; label: string; hint: string }[] = [
  { value: 'VERIFIED', label: 'Verified', hint: 'Confirmed by the controlling party or a directly verifiable primary source. Source required.' },
  { value: 'PUBLICLY_REPORTED', label: 'Publicly reported', hint: 'Stated by a public source (official docs, explorer label, reliable reporting). Source required.' },
  { value: 'INVESTIGATOR_ATTRIBUTED', label: 'Investigator attributed', hint: 'Your conclusion from stated evidence. Basis or source required.' },
  { value: 'SUSPECTED', label: 'Suspected', hint: 'Plausible but unconfirmed.' },
  { value: 'DISPUTED', label: 'Disputed', hint: 'Conflicting claims exist.' },
  { value: 'REJECTED', label: 'Rejected', hint: 'Considered and ruled out; kept so it is not re-proposed.' },
  { value: 'UNKNOWN', label: 'Unknown', hint: 'No assessment yet.' },
];
export const statusLabel = (s: string): string => STATUSES.find((x) => x.value === s)?.label ?? s;
export const needsSource = (s: string): boolean => s === 'VERIFIED' || s === 'PUBLICLY_REPORTED';

export const NETWORKS = [
  { value: 'ethereum-mainnet', label: 'Ethereum' },
  { value: 'base', label: 'Base' },
  { value: 'arbitrum', label: 'Arbitrum' },
  { value: 'optimism', label: 'Optimism' },
  { value: 'polygon', label: 'Polygon' },
  { value: 'bsc', label: 'BNB Smart Chain' },
  { value: 'solana', label: 'Solana' },
];
export const networkLabel = (n: string): string => NETWORKS.find((x) => x.value === n)?.label ?? n;
/** Only Ethereum mainnet has an explorer/graph adapter today; other chains are recorded, not queried. */
export const explorerSupported = (n: string): boolean => n === 'ethereum-mainnet';

export const PLATFORMS = ['WEBSITE', 'X', 'TELEGRAM', 'DISCORD', 'FACEBOOK', 'LINKEDIN', 'GITHUB', 'EMAIL', 'OTHER'] as const;
export const IDENT_STATUSES = [
  { value: 'UNVERIFIED', label: 'Unverified' },
  { value: 'PUBLICLY_LISTED', label: 'Publicly listed (source required)' },
  { value: 'VERIFIED', label: 'Verified (source required)' },
  { value: 'DISPUTED', label: 'Disputed' },
];

const EVM = /^0x[0-9a-fA-F]{40}$/;
const SOL = /^[1-9A-HJ-NP-Za-km-z]{32,44}$/;
export const isValidAddress = (network: string, a: string): boolean => (network === 'solana' ? SOL.test(a.trim()) : EVM.test(a.trim()));

export const pretty = (v: string): string => v.replace(/_/g, ' ').toLowerCase().replace(/^\w/, (c) => c.toUpperCase());
