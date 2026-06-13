import { defineConfig } from 'vite';
import { resolve } from 'path';
import { fileURLToPath } from 'url';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

export default defineConfig(({ mode }) => {
  const isDev = mode === 'development';
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
