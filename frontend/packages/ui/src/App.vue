<script lang="ts" setup>
import { ConfigProvider, theme } from 'ant-design-vue'
import enUS from 'ant-design-vue/es/locale/en_US'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { computed, getCurrentInstance, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { EventsEmit, EventsOff, EventsOn, WindowSetAlwaysOnTop } from '../wailsjs/runtime/runtime'
import AppSidebar from './components/AppSidebar.vue'
import AppTitlebar from './components/AppTitlebar.vue'
import CloseConfirmModal from './components/CloseConfirmModal.vue'
import { useModuleConfig } from './config/module'
import { useAppStore } from './stores/app'
import { designTokens } from './theme/tokens'

import { trackVisit } from './utils/analytics'
import { detectPlatform } from './utils/platform'
import { initTestRegistry, registerNavigate, reportTestError, reportTestWarn, testActionSet } from './utils/test'
import { autoCheckVersion, isAppStoreBuild } from './utils/version'


const appStore = useAppStore()
const router = useRouter()
const route = useRoute()


const pinned = ref(false)
function doTogglePin() {
  pinned.value = !pinned.value
  WindowSetAlwaysOnTop(pinned.value)
}

// ── Cmd/Ctrl+Shift+H 快速三连：打开调试面板 ─────────────────────────────────
// macOS 用 Cmd+Shift+H，其他系统用 Ctrl+Shift+H
const appPlatform = detectPlatform()
const isMac = appPlatform === 'mac'
const _debugKeyTimes: number[] = []
function onKeydownDebug(e: KeyboardEvent) {
  const modifierMatch = isMac ? e.metaKey && !e.ctrlKey : e.ctrlKey && !e.metaKey
  if (modifierMatch && e.shiftKey && (e.key === 'H' || e.key === 'h')) {
    const now = Date.now()
    _debugKeyTimes.push(now)
    // 只保留最近 1.5 秒内的记录
    while (_debugKeyTimes.length > 0 && now - _debugKeyTimes[0] > 1500) {
      _debugKeyTimes.shift()
    }
    if (_debugKeyTimes.length >= 3) {
      _debugKeyTimes.length = 0
      const invoke = (window as unknown as Record<string, unknown>)['WailsInvoke'] as
        | ((msg: string) => void)
        | undefined
      invoke?.('wails:openInspector')
    }
  }
}

const antLocale = computed(() => (appStore.locale === 'zh' ? zhCN : enUS))

const themeConfig = computed(() => ({
  algorithm: appStore.isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: {
    colorPrimary: designTokens.color.primary,
    borderRadius: 8,
    fontFamily: 'Inter, system-ui, -apple-system, BlinkMacSystemFont, sans-serif',
  },
}))

const { navItems, moduleOptions, isFullscreen } = useModuleConfig()

const isActive = (path: string) => route.path === path || route.path.startsWith(path + '/')

const currentModuleValue = computed(() => {
  const found = moduleOptions.find((m) => route.path.startsWith('/' + m.value))
  return found ? found.value : (moduleOptions[0]?.value ?? '')
})

const currentNavItems = computed(() =>
  navItems
    .filter((item) => item.path.startsWith('/' + currentModuleValue.value))
    .map((item) => ({
      ...item,
      label:
        item.path === '/quota/account'
          ? appStore.t('quota.nav.account')
          : item.path === '/quota/setting'
            ? appStore.t('quota.nav.setting')
            : item.label,
    }))
)



// Close confirmation dialog (the backend triggers it when no default close action is set)
const showCloseConfirm = ref(false)

const isStudyMode = computed(() => isFullscreen(route.path))

const isNavLocked = ref(false)

// ── Debug mode (window.__test) ──────────────────────────────────────────────

onMounted(async () => {
  window.addEventListener('keydown', onKeydownDebug)
  await appStore.initSettings()
  appStore.loadUserInfo()
  
  trackVisit('Main')
  if (!isAppStoreBuild) {
    autoCheckVersion(5000)
  }

  // 监听子组件通过 window 事件请求打开用户远端弹窗
  

  // 初始化 window.__test，供各组件注册可测试操作。
  // Vue errorHandler 和 message 拦截仅在测试程序调用 App.startTestMode 后激活。
  const _vueApp = getCurrentInstance()?.appContext.app
  initTestRegistry({
    onStartTestMode: async () => {
      // 接入 Vue 全局错误处理
      if (_vueApp) {
        const _origErrorHandler = _vueApp.config.errorHandler
        _vueApp.config.errorHandler = (err, _instance, info) => {
          const msg = err instanceof Error ? err.message : String(err)
          reportTestError(`[vue:${info}] ${msg}`)
          if (_origErrorHandler) _origErrorHandler(err, _instance, info)
          else console.error(`[vue:${info}]`, err)
        }
        _vueApp.config.warnHandler = (msg, _instance, trace) => {
          reportTestWarn(`[vue:warn] ${msg}${trace ? '\n' + trace : ''}`)
        }
      }
      // 拦截 ant-design-vue message
      const { message: antMessage } = await import('ant-design-vue')
      const _origMsgError = antMessage.error.bind(antMessage)
      const _origMsgWarning = antMessage.warning.bind(antMessage)
      ;(antMessage as unknown as Record<string, unknown>).error = (...args: unknown[]) => {
        const content = typeof args[0] === 'string' ? args[0] : JSON.stringify(args[0])
        reportTestError(`[message.error] ${content}`)
        return (_origMsgError as (...a: unknown[]) => unknown)(...args)
      }
      ;(antMessage as unknown as Record<string, unknown>).warning = (...args: unknown[]) => {
        const content = typeof args[0] === 'string' ? args[0] : JSON.stringify(args[0])
        reportTestWarn(`[message.warning] ${content}`)
        return (_origMsgWarning as (...a: unknown[]) => unknown)(...args)
      }
    },
  })
  registerNavigate(async (path: string) => {
    try {
      await router.push(path)
    } catch {
      // 忽略导航错误
    }
    const deadline = Date.now() + 5000
    while (Date.now() < deadline) {
      if (router.currentRoute.value.path === path) break
      await new Promise((r) => setTimeout(r, 50))
    }
  })

  // 注册 App 级别操作
  testActionSet('App.getTitle', () => document.title)
  testActionSet('App.getPlatform', () => appPlatform)
  testActionSet('App.getRoute', () => router.currentRoute.value.path)
  testActionSet('App.getHash', () => window.location.hash)
  testActionSet('App.navigate', async (params: unknown) => {
    const { route } = params as { route: string }
    try {
      await router.push(route)
    } catch {
      // 忽略导航错误
    }
    const deadline = Date.now() + 5000
    while (Date.now() < deadline) {
      if (router.currentRoute.value.path === route) break
      await new Promise((r) => setTimeout(r, 50))
    }
    return router.currentRoute.value.path === route
  })
  testActionSet('App.exists', (params: unknown) => {
    const { selector } = params as { selector: string }
    return !!document.querySelector(selector)
  })
  testActionSet('App.count', (params: unknown) => {
    const { selector } = params as { selector: string }
    return document.querySelectorAll(selector).length
  })
  testActionSet('App.getText', (params: unknown) => {
    const { selector } = params as { selector: string }
    return document.querySelector(selector)?.textContent?.trim() ?? null
  })
  testActionSet('App.click', (params: unknown) => {
    const { selector } = params as { selector: string }
    const el = document.querySelector(selector) as HTMLElement | null
    if (!el) throw new Error(`元素不存在: "${selector}"`)
    el.click()
    return true
  })
  testActionSet('App.openCloseConfirm', () => {
    showCloseConfirm.value = true
    return true
  })
  testActionSet('App.getCloseConfirm', () => ({ open: showCloseConfirm.value }))
  testActionSet('App.closeCloseConfirm', () => {
    showCloseConfirm.value = false
    return true
  })

  // Backend requests window close (no default close action configured) → show confirmation
  EventsOn('window:closeRequested', () => {
    showCloseConfirm.value = true
  })

  // 监听来自 Go 后端的调用请求（由 HTTP /auto ui-call 转发过来）
  EventsOn('debug:call', async (data: { id: string; name: string; params: unknown }) => {
    try {
      const result = await window.__test?.callAction(data.name, data.params)
      EventsEmit('debug:result:' + data.id, { result })
    } catch (e: unknown) {
      EventsEmit('debug:result:' + data.id, {
        error: (e as Error)?.message || String(e),
      })
    }
  })
})



onUnmounted(() => {
  window.removeEventListener('keydown', onKeydownDebug)
  EventsOff('debug:call')
  EventsOff('window:closeRequested')
})
</script>

<template>
  <ConfigProvider :locale="antLocale" :theme="themeConfig">
    <div class="app-shell" :class="`platform-${appPlatform}`">
      <AppTitlebar :platform="appPlatform" />
      <template v-if="isStudyMode">
        <!-- 学习模式：全屏 RouterView -->
        <div class="study-fullscreen">
          <RouterView />
        </div>
      </template>
      <template v-else>
        <div class="app-layout">
          <!-- Left Sidebar -->
          <AppSidebar
            :nav-items="currentNavItems"
            :is-active="isActive"
            :disabled="isNavLocked"
            :show-pin="true"
            :pinned="pinned"
            @navigate="router.push($event)"
            @toggle-pin="doTogglePin"
          >
            <template #bottom>
              
            </template>
          </AppSidebar>

          <!-- Main Content Area -->
          <div class="main-content">
            <RouterView />
          </div>
        </div>
      </template>
    </div>

    

    <CloseConfirmModal v-model:open="showCloseConfirm" />
  </ConfigProvider>
</template>

<style scoped>
.study-fullscreen {
  flex: 1;
  min-height: 0;
  width: 100%;
  overflow: hidden;
}
</style>
