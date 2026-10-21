export const isEthAddress = (s: string): boolean => /^0x[a-fA-F0-9]{40}$/.test(s.trim());
export const isEthTxHash = (s: string): boolean => /^0x[a-fA-F0-9]{64}$/.test(s.trim());
