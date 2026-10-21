import { Check, Copy } from 'lucide-react';
import { useState } from 'react';
import { shortHash } from '@/lib/format';

export function HashText({ value, head = 10, tail = 6 }: { value: string; head?: number; tail?: number }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch { /* clipboard unavailable */ }
  };
  return (
    <span className="inline-flex items-center gap-1.5">
      <code title={value} className="font-mono text-[11px]">{shortHash(value, head, tail)}</code>
      <button type="button" onClick={copy} aria-label="Copy full value" className="text-fg-muted hover:text-fg">
        {copied ? <Check size={12} /> : <Copy size={12} />}
      </button>
    </span>
  );
}
