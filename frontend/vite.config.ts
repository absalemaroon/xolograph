import { fileURLToPath } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: { port: 3000, strictPort: true },
  build: { target: 'es2020', sourcemap: false },
  test: { environment: 'node', include: ['src/**/*.test.ts'] },
});
