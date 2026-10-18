import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  
  return {
    plugins: [
      vue(),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
        '@components': fileURLToPath(new URL('./src/components', import.meta.url)),
        '@views': fileURLToPath(new URL('./src/views', import.meta.url)),
        '@stores': fileURLToPath(new URL('./src/stores', import.meta.url)),
        '@api': fileURLToPath(new URL('./src/api', import.meta.url)),
      },
    },
    server: {
      // 本地可用 VITE_PORT 覆盖（Windows/WSL2 会保留 3000 附近的端口段）
      port: Number(env.VITE_PORT) || 3000,
      host: true,
      proxy: {
        '/api': {
          target: env.VITE_API_TARGET || 'http://localhost:8080',
          changeOrigin: true,
          secure: false,
          ws: true,
        },
      },
    },
    build: {
      outDir: 'dist',
      assetsDir: 'static',
      sourcemap: true,
      rollupOptions: {
        output: {
          // Vite 8（Rolldown）只接受函数形式的 manualChunks
          manualChunks(id) {
            if (id.includes('node_modules/video.js')) return 'video-js'
            if (id.includes('node_modules/hls.js')) return 'hls-js'
            if (id.includes('node_modules/vue') ||
                id.includes('node_modules/@vue') ||
                id.includes('node_modules/pinia') ||
                id.includes('node_modules/axios')) return 'vendor'
          },
        },
      },
    },
    test: {
      globals: true,
      environment: 'jsdom',
      include: ['**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}'],
      coverage: {
        reporter: ['text', 'json', 'html'],
        exclude: ['node_modules/', 'src/tests/'],
      },
    },
  }
})
