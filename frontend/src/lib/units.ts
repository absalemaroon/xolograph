/** Format an integer token amount (as a decimal string) with `decimals`, using BigInt (no float rounding). */
export function formatUnits(value: string, decimals: number, maxFrac = 6): string {
  if (!/^\d+$/.test(value) || decimals < 0 || decimals > 77) return value;
  const v = BigInt(value);
  const base = 10n ** BigInt(decimals);
  const whole = v / base;
  const frac = (v % base).toString().padStart(decimals, '0').slice(0, maxFrac).replace(/0+$/, '');
  if (whole === 0n && v > 0n && frac === '') return `<0.${'0'.repeat(maxFrac - 1)}1`;
  const w = whole.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  return frac ? `${w}.${frac}` : w;
}

export const shortAddr = (a: string): string => (a.length > 12 ? `${a.slice(0, 6)}…${a.slice(-4)}` : a);
