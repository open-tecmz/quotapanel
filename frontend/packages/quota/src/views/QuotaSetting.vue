<template>
  <div class="setting-view">
    <div class="view-header">
      <PageHeader :title="t('settings.title')">
        <template #icon><Settings class="w-5 h-5 text-primary-500" aria-hidden="true" /></template>
      </PageHeader>
    </div>
    <div class="view-body">
      <div class="setting-card">
        <div class="setting-row">
          <Languages class="w-4 h-4 text-primary-500" aria-hidden="true" />
          <div class="row-copy">
            <div class="row-label">{{ t('settings.language') }}</div>
            <div class="row-desc">{{ t('settings.languageDesc') }}</div>
          </div>
          <div class="row-action">
            <a-button :type="appStore.locale === 'zh' ? 'primary' : 'default'" @click="doLocale('zh')">中文</a-button>
            <a-button :type="appStore.locale === 'en' ? 'primary' : 'default'" @click="doLocale('en')"
              >English</a-button
            >
          </div>
        </div>
        <div class="setting-row">
          <Moon class="w-4 h-4 text-primary-500" aria-hidden="true" />
          <div class="row-copy">
            <div class="row-label">{{ t('settings.theme.title') }}</div>
            <div class="row-desc">{{ t('settings.theme.desc') }}</div>
          </div>
          <div class="row-action">
            <a-button :type="!appStore.isDark ? 'primary' : 'default'" @click="doTheme(false)">{{
              t('settings.theme.light')
            }}</a-button>
            <a-button :type="appStore.isDark ? 'primary' : 'default'" @click="doTheme(true)">{{
              t('settings.theme.dark')
            }}</a-button>
          </div>
        </div>
        <div class="setting-row">
          <Power class="w-4 h-4 text-primary-500" aria-hidden="true" />
          <div class="row-copy">
            <div class="row-label">{{ t('settings.autoStart.title') }}</div>
            <div class="row-desc">{{ t('settings.autoStart.desc') }}</div>
          </div>
          <a-switch :checked="autoStart" @change="doAutoStart" />
        </div>
        <div class="setting-row">
          <X class="w-4 h-4 text-primary-500" aria-hidden="true" />
          <div class="row-copy">
            <div class="row-label">{{ t('settings.closeBehavior.title') }}</div>
            <div class="row-desc">{{ t('settings.closeBehavior.desc') }}</div>
          </div>
          <div class="row-action">
            <a-button :type="closeAction === '' ? 'primary' : 'default'" @click="doCloseAction('')">{{
              t('settings.closeBehavior.ask')
            }}</a-button>
            <a-button :type="closeAction === 'quit' ? 'primary' : 'default'" @click="doCloseAction('quit')">{{
              t('settings.closeBehavior.quit')
            }}</a-button>
            <a-button :type="closeAction === 'hide' ? 'primary' : 'default'" @click="doCloseAction('hide')">{{
              t('settings.closeBehavior.hide')
            }}</a-button>
          </div>
        </div>
        <div v-if="!dataRootIsDefault" class="setting-row">
          <FolderOpen class="w-4 h-4 text-primary-500" aria-hidden="true" />
          <div class="row-copy">
            <div class="row-label">{{ t('settings.dataRoot.title') }}</div>
            <div class="row-desc">{{ t('settings.dataRoot.desc') }}</div>
            <div class="data-path" :title="dataRoot">{{ dataRoot }}</div>
          </div>
          <a-button @click="doOpenDataDir">{{ t('settings.dataRoot.open') }}</a-button>
        </div>
        <div class="setting-row">
          <RefreshCw class="w-4 h-4 text-primary-500" aria-hidden="true" />
          <div class="row-copy">
            <div class="row-label">{{ t('settings.version.title') }}</div>
            <div class="row-desc">{{ t('settings.version.currentVersion') }}：{{ appVersion || '-' }}</div>
            <div v-if="versionMessage" class="version-message">{{ versionMessage }}</div>
          </div>
          <a-button :loading="checkingVersion" @click="doCheckVersion">{{
            t('settings.version.checkUpdate')
          }}</a-button>
          <a-button v-if="updateURL" @click="BrowserOpenURL(updateURL)">{{
            t('settings.version.updateAvailable')
          }}</a-button>
        </div>
      </div>
      <div class="setting-card about-card">
        <div class="about-copy">
          <div class="row-label">QuotaPanel</div>
          <div class="row-desc">{{ t('quota.nav.account') }} · {{ appVersion || '-' }}</div>
        </div>
        <a-button v-if="feedbackUrl" type="primary" @click="showFeedback = true">
          <div class="inline-flex items-center gap-1">
            <MessageSquare class="w-4 h-4" aria-hidden="true" />
            {{ t('settings.feedback') }}
          </div>
        </a-button>
      </div>
    </div>

    <FeedbackModal v-model:open="showFeedback" :url="feedbackUrl" />
  </div>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue'
import { Languages, FolderOpen, MessageSquare, Moon, Power, RefreshCw, Settings, X } from 'lucide-vue-next'
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { BrowserOpenURL } from '@quotapanel/ui/wailsjs/runtime/runtime'
import FeedbackModal from '@quotapanel/ui/src/components/FeedbackModal.vue'
import PageHeader from '@quotapanel/ui/src/components/PageHeader.vue'
import { useAppStore } from '@quotapanel/ui/src/stores/app'
import { testActionSet, testActionUnset } from '@quotapanel/ui/src/utils/test'
import { main } from '../api/main'

const { t } = useI18n()
const appStore = useAppStore()
const router = useRouter()
const autoStart = ref(false)
const closeAction = ref('')
const appVersion = ref('')
const checkingVersion = ref(false)
const versionMessage = ref('')
const updateURL = ref('')
const feedbackUrl = ref('')
const dataRoot = ref('')
const dataRootIsDefault = ref(true)
const showFeedback = ref(false)

async function loadSettings() {
  try {
    const cfg = await main.Call('setting.getAppConfig')
    appVersion.value = String(cfg?.version || '')
    feedbackUrl.value = String(cfg?.feedbackUrl || '')
    dataRoot.value = String(cfg?.dataRoot || '')
    dataRootIsDefault.value = cfg?.dataRootIsDefault !== false
    autoStart.value = Boolean(await main.Call('setting.getAutoStartEnabled'))
    closeAction.value = String((await main.Call('setting.getCloseAction')) || '')
  } catch (error) {
    message.error(String(error))
  }
}

async function doLocale(locale: 'zh' | 'en') {
  await appStore.setLocale(locale)
}

async function doTheme(dark: boolean) {
  await appStore.setTheme(dark)
}

async function doAutoStart(value: boolean) {
  try {
    await main.Call('setting.setAutoStartEnabled', { enabled: value })
    autoStart.value = value
  } catch (error) {
    message.error(String(error))
  }
}

async function doCloseAction(action: string) {
  try {
    await main.Call('setting.setCloseAction', { action })
    closeAction.value = action
  } catch (error) {
    message.error(String(error))
  }
}

async function doOpenDataDir() {
  try {
    await main.Call('setting.openDataDir')
  } catch (error) {
    message.error(String(error))
  }
}

function compareVersions(left: string, right: string): number {
  const a = left.replace(/^v/i, '').split('.').map(Number)
  const b = right.replace(/^v/i, '').split('.').map(Number)
  for (let index = 0; index < Math.max(a.length, b.length); index++) {
    const diff = (a[index] || 0) - (b[index] || 0)
    if (diff !== 0) return diff
  }
  return 0
}

async function doCheckVersion() {
  checkingVersion.value = true
  updateURL.value = ''
  versionMessage.value = ''
  try {
    const latest = await main.Call('setting.checkVersion')
    if (!latest?.version) throw new Error('版本服务未返回版本号')
    if (compareVersions(latest.version, appVersion.value) > 0) {
      versionMessage.value = t('settings.version.newVersion', { version: latest.version })
      updateURL.value = latest.url || ''
    } else {
      versionMessage.value = t('settings.version.latestVersion')
    }
  } catch (error) {
    versionMessage.value = `${t('settings.version.checkFailed')}：${String(error)}`
  } finally {
    checkingVersion.value = false
  }
}

onMounted(async () => {
  await loadSettings()
  testActionSet('QuotaSetting.getState', () => ({
    locale: appStore.locale,
    dark: appStore.isDark,
    autoStart: autoStart.value,
    closeAction: closeAction.value,
    version: appVersion.value,
  }))
  testActionSet('QuotaSetting.getThemeStyles', () => {
    const read = (selector: string, property: keyof CSSStyleDeclaration) => {
      const element = document.querySelector(selector)
      return element ? getComputedStyle(element)[property] : ''
    }
    return {
      htmlDark: document.documentElement.classList.contains('dark'),
      body: read('body', 'backgroundColor'),
      setting: read('.setting-view', 'backgroundColor'),
      card: read('.setting-card', 'backgroundColor'),
      sidebar: read('.app-sidebar', 'backgroundColor'),
    }
  })
  testActionSet('QuotaSetting.setLocale', (params: unknown) => doLocale((params as { locale: 'zh' | 'en' }).locale))
  testActionSet('QuotaSetting.setTheme', (params: unknown) => doTheme((params as { dark: boolean }).dark))
  testActionSet('QuotaSetting.setCloseAction', (params: unknown) =>
    doCloseAction((params as { action: string }).action)
  )
  testActionSet('QuotaSetting.getFeedback', () => ({ url: feedbackUrl.value, open: showFeedback.value }))
  testActionSet('QuotaSetting.getDataRoot', () => ({
    path: dataRoot.value,
    isDefault: dataRootIsDefault.value,
  }))
  testActionSet('QuotaSetting.openFeedback', () => {
    showFeedback.value = true
    return true
  })
  testActionSet('QuotaSetting.closeFeedback', () => {
    showFeedback.value = false
    return true
  })

  // ── 截图 prepare / cleanup（截取前准备、截图后清理）────────────────────
  testActionSet('setting.prepare', async () => {
    if (router.currentRoute.value.path !== '/quota/setting') {
      await router.push('/quota/setting')
    }
    await nextTick()
    return true
  })
  testActionSet('setting.cleanup', async () => {
    await nextTick()
    return true
  })
})

onUnmounted(() => {
  testActionUnset([
    'QuotaSetting.getState',
    'QuotaSetting.getThemeStyles',
    'QuotaSetting.setLocale',
    'QuotaSetting.setTheme',
    'QuotaSetting.setCloseAction',
    'QuotaSetting.getFeedback',
    'QuotaSetting.getDataRoot',
    'QuotaSetting.openFeedback',
    'QuotaSetting.closeFeedback',
    'setting.prepare',
    'setting.cleanup',
  ])
})
</script>

<style scoped>
.setting-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f2f2f7;
}
.view-header {
  flex-shrink: 0;
  padding: 10px 16px;
  background: #fff;
  border-bottom: 1px solid #e5e5ea;
}
.view-body {
  min-height: 0;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.setting-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 4px 16px;
}
.setting-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 0;
  border-bottom: 1px solid #f3f4f6;
}
.setting-row:last-child {
  border-bottom: 0;
}
.row-copy {
  min-width: 0;
  flex: 1;
}
.row-label {
  font-size: 13px;
  font-weight: 600;
  color: #1f2937;
}
.row-desc,
.version-message {
  font-size: 11px;
  color: #6b7280;
  margin-top: 2px;
}
.data-path {
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: #374151;
  margin-top: 4px;
  word-break: break-all;
}
.row-action {
  display: flex;
  gap: 6px;
}
.about-card {
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.about-copy {
  min-width: 0;
}
:global(.dark .setting-view) {
  background: var(--token-surface);
}
:global(.dark .view-header),
:global(.dark .setting-card) {
  background: var(--token-panel);
  border-color: #374151;
}
:global(.dark .setting-row) {
  border-color: #374151;
}
:global(.dark .row-label) {
  color: #f9fafb;
}
:global(.dark .row-desc),
:global(.dark .version-message) {
  color: #9ca3af;
}
:global(.dark .data-path) {
  color: #d1d5db;
}
</style>
