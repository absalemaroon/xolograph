/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        base: 'var(--bg-primary)',
        raised: 'var(--bg-secondary)',
        elevated: 'var(--bg-elevated)',
        hover: 'var(--bg-hover)',
        selected: 'var(--bg-selected)',
        line: { DEFAULT: 'var(--border-default)', subtle: 'var(--border-subtle)', strong: 'var(--border-strong)' },
        fg: { DEFAULT: 'var(--text-primary)', soft: 'var(--text-secondary)', muted: 'var(--text-muted)', inverse: 'var(--text-inverse)' },
      },
      fontFamily: { mono: ['var(--font-mono)'], sans: ['var(--font-ui)'] },
    },
  },
  plugins: [],
};
