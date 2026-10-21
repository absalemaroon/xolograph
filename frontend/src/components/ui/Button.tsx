import type { ButtonHTMLAttributes, ReactNode } from 'react';

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger';
type Size = 'sm' | 'md';

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  loading?: boolean;
  icon?: ReactNode;
}

const variants: Record<Variant, string> = {
  primary: 'bg-fg text-fg-inverse border-fg hover:opacity-85',
  secondary: 'bg-transparent text-fg border-line-strong hover:bg-hover',
  ghost: 'bg-transparent text-fg-soft border-transparent hover:bg-hover hover:text-fg',
  danger: 'bg-transparent text-fg border-fg border-dashed hover:bg-fg hover:text-fg-inverse',
};
const sizes: Record<Size, string> = { sm: 'h-7 px-2.5 text-[11px]', md: 'h-9 px-4 text-xs' };

export function Button({ variant = 'secondary', size = 'md', loading, icon, disabled, className = '', children, ...rest }: Props) {
  return (
    <button
      type="button"
      disabled={disabled || loading}
      className={`inline-flex shrink-0 items-center justify-center gap-2 border font-medium uppercase tracking-wider transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${variants[variant]} ${sizes[size]} ${className}`}
      {...rest}
    >
      {loading ? <span aria-hidden className="animate-pulse">…</span> : icon}
      {children}
    </button>
  );
}
