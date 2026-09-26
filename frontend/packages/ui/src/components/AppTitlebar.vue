<script lang="ts" setup>
import { Copy, Minus, Square, X } from 'lucide-vue-next'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { WindowIsMaximised, WindowMinimise, WindowToggleMaximise } from '../../wailsjs/runtime/runtime'
import { main } from '../api/call'
import BrandLogo from './BrandLogo.vue'
import { useModuleConfig } from '../config/module'
import { useAppStore } from '../stores/app'
import type { AppPlatform } from '../utils/platform'
import { testActionSet, testActionUnset } from '../utils/test'
import { getAppVersion } from '../utils/version'

const props = defineProps<{ platform: AppPlatform }>()

const appStore = useAppStore()
const { editionKey } = useModuleConfig()

const appName = 'QuotaPanel'
const appVersion = ref('')
const editionLabel = computed(() => appStore.t(`titlebar.edition.${editionKey}`))

const maximised = ref(false)
let maxTimer: ReturnType<typeof setInterval> | null = null

async function syncMaximised() {
  try {
    maximised.value = await WindowIsMaximised()
  } catch {
    // 窗口状态读取失败（如测试环境）时保持原值
  }
}

onMounted(() => {
  testActionSet('AppTitlebar.getPlatform', () => props.platform)
  testActionSet('AppTitlebar.hasNativeControls', () => props.platform === 'mac')
  testActionSet('AppTitlebar.getControlCount', () => document.querySelectorAll('[data-test="titlebar-control"]').length)
  testActionSet('AppTitlebar.isMaximised', () => maximised.value)
  testActionSet('AppTitlebar.isDraggable', () => {
    const el = document.querySelector('.app-titlebar')
    if (!el) return false
    return getComputedStyle(el).getPropertyValue('--wails-draggable').trim() === 'drag'
  })
  testActionSet('AppTitlebar.getMeta', () => ({
    name: appName,
    version: appVersion.value,
    edition: editionKey,
  }))

  getAppVersion()
    .then((v) => {
      appVersion.value = v
    })
    .catch(() => {
      appVersion.value = ''
    })

  if (props.platform === 'mac') return
  syncMaximised()
  maxTimer = setInterval(syncMaximised, 1000)
})

onUnmounted(() => {
  if (maxTimer) clearInterval(maxTimer)
  testActionUnset([
    'AppTitlebar.getPlatform',
    'AppTitlebar.hasNativeControls',
    'AppTitlebar.getControlCount',
    'AppTitlebar.isMaximised',
    'AppTitlebar.isDraggable',
    'AppTitlebar.getMeta',
  ])
})

function doMinimise() {
  WindowMinimise()
}

function doToggleMaximise() {
  WindowToggleMaximise()
  setTimeout(syncMaximised, 150)
}

async function doClose() {
  // Route through the backend so the configured close behavior (ask / quit / hide) applies.
  try {
    await main.Call('setting.requestClose')
  } catch {
    // Ignore: the window may already be closing.
  }
}
</script>

<template>
  <div class="app-titlebar" :class="`app-titlebar--${platform}`">
    <!-- macOS: 预留系统红绿灯位置，按钮由系统原生绘制 -->
    <div v-if="platform === 'mac'" class="titlebar-traffic-space" aria-hidden="true"></div>

    <!-- 左侧品牌区：Logo + 软件名 + 版本号 + 版本类型（Pro / 社区版） -->
    <div class="titlebar-brand">
      <BrandLogo class="titlebar-logo" />
      <span class="titlebar-app-name">{{ appName }}</span>
      <span v-if="appVersion" class="titlebar-version">{{ appVersion }}</span>
      <span class="titlebar-edition" :class="`titlebar-edition--${editionKey}`">{{ editionLabel }}</span>
    </div>

    <div class="titlebar-drag-region"></div>

    <!-- Windows / Linux: 自绘模拟系统窗口按钮 -->
    <div v-if="platform !== 'mac'" class="titlebar-controls">
      <button
        class="titlebar-control"
        data-test="titlebar-control"
        aria-label="最小化"
        title="最小化"
        @click="doMinimise"
      >
        <Minus class="titlebar-icon" aria-hidden="true" />
      </button>
      <button
        class="titlebar-control"
        data-test="titlebar-control"
        :aria-label="maximised ? '还原' : '最大化'"
        :title="maximised ? '还原' : '最大化'"
        @click="doToggleMaximise"
      >
        <Copy v-if="maximised" class="titlebar-icon" aria-hidden="true" />
        <Square v-else class="titlebar-icon" aria-hidden="true" />
      </button>
      <button
        class="titlebar-control titlebar-control--close"
        data-test="titlebar-control"
        aria-label="关闭"
        title="关闭"
        @click="doClose"
      >
        <X class="titlebar-icon" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.app-titlebar {
  position: relative;
  z-index: 50;
  display: flex;
  align-items: center;
  height: 40px;
  flex-shrink: 0;
  background: var(--token-surface);
  border-bottom: 1px solid var(--token-neutral-200);
  --wails-draggable: drag;
  -webkit-user-select: none;
  user-select: none;
}

:global(.dark .app-titlebar) {
  background: var(--token-panel);
  border-bottom-color: var(--token-neutral-700);
}

.titlebar-drag-region {
  flex: 1;
  height: 100%;
}

.titlebar-brand {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  flex-shrink: 0;
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
  user-select: none;
  -webkit-user-select: none;
}

.titlebar-logo {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.titlebar-app-name {
  font-weight: 600;
  color: var(--token-neutral-600);
}

.titlebar-version {
  color: var(--token-neutral-400);
}

.titlebar-edition {
  padding: 1px 6px;
  border-radius: 9999px;
  font-size: 10px;
  font-weight: 600;
}

.titlebar-edition--pro {
  background: rgba(var(--token-primary-rgb), 0.14);
  color: var(--token-primary-500);
}

.titlebar-edition--community {
  background: rgba(var(--token-neutral-500-rgb), 0.14);
  color: var(--token-neutral-500);
}

:global(.dark .titlebar-app-name) {
  color: var(--token-neutral-300);
}

:global(.dark .titlebar-version) {
  color: var(--token-neutral-400);
}

:global(.dark .titlebar-edition--pro) {
  background: rgba(var(--token-primary-rgb), 0.24);
  color: #5eead4;
}

:global(.dark .titlebar-edition--community) {
  background: rgba(var(--token-neutral-400-rgb), 0.18);
  color: var(--token-neutral-300);
}

.app-titlebar--mac .titlebar-traffic-space {
  width: 78px;
  height: 100%;
  flex-shrink: 0;
}

.titlebar-controls {
  display: flex;
  align-items: center;
  height: 100%;
  flex-shrink: 0;
  --wails-draggable: no-drag;
}

.titlebar-control {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--token-neutral-600);
  cursor: pointer;
  padding: 0;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

:global(.dark .titlebar-control) {
  color: var(--token-neutral-300);
}

.titlebar-icon {
  width: 12px;
  height: 12px;
}

/* Windows: 右上角直角按钮，关闭悬停红色 */
.app-titlebar--windows .titlebar-control {
  width: 46px;
  height: 40px;
  border-radius: 0;
}

.app-titlebar--windows .titlebar-control:hover {
  background: rgba(var(--token-neutral-500-rgb), 0.16);
}

.app-titlebar--windows .titlebar-control--close:hover {
  background: #e81123;
  color: #ffffff;
}

/* Linux: 右上角圆形按钮组 */
.app-titlebar--linux .titlebar-controls {
  gap: 8px;
  padding-right: 14px;
}

.app-titlebar--linux .titlebar-control {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: rgba(var(--token-neutral-500-rgb), 0.14);
}

.app-titlebar--linux .titlebar-control:hover {
  background: rgba(var(--token-neutral-500-rgb), 0.28);
}

.app-titlebar--linux .titlebar-control--close {
  background: rgba(var(--token-danger-rgb), 0.85);
  color: #ffffff;
}

.app-titlebar--linux .titlebar-control--close:hover {
  background: #dc2626;
  color: #ffffff;
}
</style>
