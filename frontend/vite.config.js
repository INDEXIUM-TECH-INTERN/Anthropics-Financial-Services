import { defineConfig } from 'vite';
import { resolve } from 'path';
export default defineConfig(function (_a) {
    var mode = _a.mode;
    var isDev = mode === 'development';
    return {
        root: '.',
        base: '/',
        build: {
            outDir: 'dist',
            emptyOutDir: true,
            sourcemap: isDev,
            minify: 'esbuild',
            target: 'es2022',
            rollupOptions: {
                input: { main: resolve(__dirname, 'index.html') },
            },
        },
        server: {
            port: 5173,
            proxy: {
                '/api': {
                    target: 'http://localhost:8080',
                    changeOrigin: true,
                    ws: true,
                },
            },
        },
        resolve: {
            alias: { '@': resolve(__dirname, 'src') },
        },
        css: { devSourcemap: true },
    };
});
