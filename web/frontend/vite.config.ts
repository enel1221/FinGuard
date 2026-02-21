import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://localhost:8080',
      '/login': 'http://localhost:8080',
      '/callback': 'http://localhost:8080',
      '/logout': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
      '/plugins': 'http://localhost:8080',
      '/swagger': 'http://localhost:8080',
    },
  },
  build: {
    outDir: '../dist',
    emptyOutDir: true,
    sourcemap: false,
  },
});
