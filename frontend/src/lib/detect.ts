export type Family = 'EVM' | 'SOLANA' | 'BITCOIN';
export interface Detection { kind: 'address' | 'tx'; family: Family; value: string }
export interface Target { det: Detection; network: string }

const EVM_ADDR = /^0x[0-9a-fA-F]{40}$/;
const EVM_TX = /^0x[0-9a-fA-F]{64}$/;
const BTC_TX = /^[0-9a-fA-F]{64}$/;
const BTC_LEGACY = /^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$/;
const BTC_BECH = /^bc1[a-zA-HJ-NP-Z0-9]{11,87}$/i;
const B58 = /^[1-9A-HJ-NP-Za-km-z]+$/;

/**
 * Every plausible reading of a pasted string. Some strings are genuinely ambiguous: a 32-35 character base58 string that
 * starts with 1 or 3 can be a Bitcoin legacy address or a Solana key, so more than one detection may be returned.
 */
export function detect(input: string): Detection[] {
  const s = input.trim();
  const out: Detection[] = [];
  if (EVM_ADDR.test(s)) out.push({ kind: 'address', family: 'EVM', value: s.toLowerCase() });
  else if (EVM_TX.test(s)) out.push({ kind: 'tx', family: 'EVM', value: s.toLowerCase() });
  else if (BTC_TX.test(s)) out.push({ kind: 'tx', family: 'BITCOIN', value: s.toLowerCase() });
  else if (BTC_BECH.test(s)) out.push({ kind: 'address', family: 'BITCOIN', value: s.toLowerCase() });
  else if (B58.test(s)) {
    if (BTC_LEGACY.test(s)) out.push({ kind: 'address', family: 'BITCOIN', value: s });
    // A 32-byte key is 43-44 characters; shorter strings only occur with leading zero bytes, which show as leading "1"s.
    if ((s.length >= 43 && s.length <= 44) || (s.length >= 32 && s.length <= 42 && s.startsWith('11'))) out.push({ kind: 'address', family: 'SOLANA', value: s });
    if (s.length >= 86 && s.length <= 90) out.push({ kind: 'tx', family: 'SOLANA', value: s });
  }
  return out;
}

/** Expands detections into concrete (reading, network) choices using the networks the engine reports. */
export function resolveTargets(input: string, networks: { id: string; family: string }[]): Target[] {
  return detect(input).flatMap((det) => networks.filter((n) => n.family === det.family).map((n) => ({ det, network: n.id })));
}

export function pickTarget(targets: Target[], preferred?: string | null): Target | null {
  return targets.find((t) => t.network === preferred) ?? targets.find((t) => t.network === 'ethereum-mainnet') ?? targets[0] ?? null;
}
