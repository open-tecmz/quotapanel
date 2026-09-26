import vue from '@vitejs/plugin-vue'
import { execSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vite'

// 构建 ID：git 短 hash + 时间戳，用于 sourcemap 溯源
function getBuildId(): string {
  try {
    const hash = execSync('git rev-parse --short HEAD', { stdio: ['pipe', 'pipe', 'ignore'] })
      .toString()
      .trim()
    return `${hash}-${Date.now()}`
  } catch {
    return `build-${Date.now()}`
  }
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  define: {
    __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: false,
    'import.meta.env.VITE_BUILD_ID': JSON.stringify(getBuildId()),
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@quotapanel/ui': fileURLToPath(new URL('.', import.meta.url)),
    },
  },
  server: {
    port: 53081,
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
