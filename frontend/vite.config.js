import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
export default defineConfig(function (_a) {
    var mode = _a.mode;
    var env = loadEnv(mode, process.cwd(), '');
    // Proxy /api to gateway (8000) or API (8080). Set VITE_PROXY_API_TARGET=http://localhost:8080 to hit API directly.
    var apiTarget = env.VITE_PROXY_API_TARGET || 'http://localhost:8000';
    return {
        plugins: [react()],
        server: {
            port: 3000,
            proxy: {
                '/api': {
                    target: apiTarget,
                    changeOrigin: true,
                    rewrite: function (p) { return p.replace(/^\/api/, ''); },
                },
                '/control': {
                    target: 'http://localhost:8000',
                    changeOrigin: true,
                    rewrite: function (p) { return p.replace(/^\/control/, ''); },
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
