import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  // Proxy /api to gateway (8000) or API (8080). Set VITE_PROXY_API_TARGET=http://localhost:8080 to hit API directly.
  const apiTarget = env.VITE_PROXY_API_TARGET || 'http://localhost:8000';

  return {
    plugins: [react()],
    server: {
    port: 3000,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api/, ''),
      },
      '/control': {
        target: 'http://localhost:8000',
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/control/, ''),
      },
      '/docs': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8090',
        ws: true,
      },
    },
  },
  };
});
