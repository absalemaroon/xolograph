const usd = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 2 });
const usdSmall = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 6 });

/** USD for display only. Approximate by nature: a reference price, not an executable one. */
export function formatUSD(n: number | undefined | null): string {
  if (n === undefined || n === null || !Number.isFinite(n)) return '—';
  return Math.abs(n) > 0 && Math.abs(n) < 0.01 ? usdSmall.format(n) : usd.format(n);
}

export function signedUSD(n: number): string {
  return `${n > 0 ? '+' : ''}${formatUSD(n)}`;
}

/** Net amounts can be negative integers in smallest units. */
export function signedUnits(raw: string, decimals: number, fmt: (v: string, d: number, f?: number) => string, maxFrac = 6): string {
  if (raw.startsWith('-')) return `-${fmt(raw.slice(1), decimals, maxFrac)}`;
  return raw === '0' ? '0' : `+${fmt(raw, decimals, maxFrac)}`;
}
