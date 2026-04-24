import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  define: {
    'import.meta.env.VITE_PROJECT_ID': '"test-project-id"',
    'import.meta.env.VITE_API_URL': '"http://localhost:8080/v1"',
    'import.meta.env.VITE_API_KEY': '"test-api-key"',
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
  },
});
