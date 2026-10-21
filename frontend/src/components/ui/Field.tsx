import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react';

const control =
  'w-full border border-line-strong bg-base px-3 text-xs text-fg placeholder:text-fg-muted focus-visible:outline-offset-0 disabled:opacity-50';

export function Label({ children, htmlFor, hint }: { children: ReactNode; htmlFor?: string; hint?: string }) {
  return (
    <label htmlFor={htmlFor} className="mb-1.5 block text-[10px] uppercase tracking-[0.14em] text-fg-muted">
      {children}
      {hint && <span className="ml-2 normal-case tracking-normal">{hint}</span>}
    </label>
  );
}
export const Input = (p: InputHTMLAttributes<HTMLInputElement>) => (
  <input {...p} className={`${control} h-9 ${p.className ?? ''}`} />
);
export const Textarea = (p: TextareaHTMLAttributes<HTMLTextAreaElement>) => (
  <textarea {...p} className={`${control} min-h-24 resize-y py-2 ${p.className ?? ''}`} />
);
export const Select = (p: SelectHTMLAttributes<HTMLSelectElement>) => (
  <select {...p} className={`${control} h-9 ${p.className ?? ''}`} />
);

export function Segmented<T extends string>({ value, options, onChange, label }: {
  value: T; options: { value: T; label: string }[]; onChange: (v: T) => void; label: string;
}) {
  return (
    <div role="radiogroup" aria-label={label} className="inline-flex border border-line-strong">
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          role="radio"
          aria-checked={value === o.value}
          onClick={() => onChange(o.value)}
          className={`h-8 px-3 text-[11px] uppercase tracking-wider transition-colors ${
            value === o.value ? 'bg-fg text-fg-inverse' : 'text-fg-soft hover:bg-hover'
          }`}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}
