import { Modal, message } from 'ant-design-vue'
import { reactive } from 'vue'
import { BrowserOpenURL, EventsOn } from '../../wailsjs/runtime/runtime'
import { main, type UpdateInstallResult, type UpdateState } from '../api/call'
import { useAppStore } from '../stores/app'

/**
 * 是否为 App Store 构建（VITE_APPSTORE_BUILD=true）。
 * App Store 版本禁止应用内自更新，因此不展示检测入口，也不自动检测。
 */
export const isAppStoreBuild = import.meta.env.VITE_APPSTORE_BUILD === 'true'

/** 手动下载页（后端未返回 release.url 时使用）。 */
const DOWNLOAD_PAGE = 'https://open.tecmz.com/quotapanel'

/** 后端推送的升级状态（update:state 事件或 update.getState 拉取）。 */
export const updateStore = reactive<{ state: UpdateState | null }>({ state: null })

/** 下载弹窗与「最小化」悬浮球共享的界面状态。 */
export const updateUI = reactive<{ modalVisible: boolean; minimized: boolean }>({
  modalVisible: false,
  minimized: false,
})

export function showUpdateModal(): void {
  updateUI.modalVisible = true
  updateUI.minimized = false
}

export function hideUpdateToBackground(): void {
  updateUI.modalVisible = false
  updateUI.minimized = true
}

export function reopenUpdateModal(): void {
  updateUI.modalVisible = true
  updateUI.minimized = false
}

let bindOnce = false

/** 只订阅一次后端进度事件，并拉取初始状态。 */
export function bindUpdateEvents(): void {
  if (bindOnce) return
  bindOnce = true
  EventsOn('update:state', (state: UpdateState) => {
    updateStore.state = state
    const appStore = useAppStore()
    if (state.state === 'ready') {
      if (!updateUI.minimized) {
        updateUI.modalVisible = true
      }
      message.success(appStore.t('settings.version.updateReady'))
    } else if (state.state === 'failed') {
      updateUI.modalVisible = true
      message.error(state.error || appStore.t('settings.version.updateFailed'))
    }
  })
  main
    .Call('update.getState')
    .then((state) => {
      updateStore.state = state
    })
    .catch(() => {
      // 浏览器调试模式没有后端绑定，忽略即可
    })
}

let cachedAppVersion: string | null = null

/** 从后端读取应用版本号（带缓存）。 */
export async function getAppVersion(): Promise<string> {
  if (cachedAppVersion) return cachedAppVersion
  const cfg = await main.Call('setting.getAppConfig')
  cachedAppVersion = (cfg.version as string) || ''
  return cachedAppVersion
}

export interface VersionCheckOptions {
  /** 已是最新版本时提示 */
  showLatestMessage?: boolean
  /** 检测失败时提示 */
  showErrorMessage?: boolean
}

/**
 * 检测新版本并按结果给出下一步：
 * - 存在可自动安装的资产：询问后开始后台下载（进度通过 update:state 推送）
 * - 只有手动安装包（dmg / deb）或后端给的是下载页：询问后打开浏览器
 * - 已是最新版本：按需提示
 */
export async function checkVersionAndPrompt(options: VersionCheckOptions = {}): Promise<boolean> {
  const { showLatestMessage = false, showErrorMessage = false } = options
  const appStore = useAppStore()
  bindUpdateEvents()

  // 已经下载完成时直接回到安装弹窗，避免重复下载。
  if (updateStore.state?.state === 'ready') {
    showUpdateModal()
    return true
  }

  try {
    const state = await main.Call('update.check', { channel: 'latest' })
    updateStore.state = state

    if (state.state === 'up-to-date') {
      if (showLatestMessage) {
        message.success(appStore.t('settings.version.latestVersion'))
      }
      return false
    }
    if (state.state === 'failed') {
      throw new Error(state.error || 'check failed')
    }
    const release = state.release
    if (!release) return false

    if (!state.asset) {
      Modal.confirm({
        title: appStore.t('settings.version.updateAvailable'),
        content: appStore.t('settings.version.updateConfirm', { version: release.version }),
        okText: appStore.t('common.yes'),
        cancelText: appStore.t('common.no'),
        onOk() {
          BrowserOpenURL(release.url || DOWNLOAD_PAGE)
        },
      })
      return true
    }

    Modal.confirm({
      title: appStore.t('settings.version.updateAvailable'),
      content: appStore.t('settings.version.downloadConfirm', {
        version: release.version,
        summary: release.summary || '',
      }),
      okText: appStore.t('settings.version.downloadUpdate'),
      cancelText: appStore.t('common.no'),
      onOk() {
        return downloadUpdate().catch((err) => {
          message.error(String(err))
        })
      },
    })
    return true
  } catch (e) {
    console.error('Version check failed:', e)
    if (showErrorMessage) {
      message.error(appStore.t('settings.version.checkFailed'))
    }
    return false
  }
}

/** 只执行检测，不弹确认框（自动化测试使用）。 */
export async function checkUpdateOnly(channel: 'latest' | 'stable' | 'beta' = 'latest'): Promise<UpdateState> {
  bindUpdateEvents()
  const state = await main.Call('update.check', { channel })
  updateStore.state = state
  return state
}

/** 开始后台下载并打开进度弹窗。 */
export async function downloadUpdate(): Promise<void> {
  bindUpdateEvents()
  showUpdateModal()
  await main.Call('update.download')
}

/** 安装已下载版本；成功后应用会退出（macOS / Windows 由安装助手自动重启）。 */
export async function installUpdate(): Promise<UpdateInstallResult> {
  bindUpdateEvents()
  return main.Call('update.install')
}

/** 切换 latest / stable / beta 更新通道。 */
export async function setUpdateChannel(channel: 'latest' | 'stable' | 'beta'): Promise<UpdateState> {
  bindUpdateEvents()
  const state = await main.Call('update.setChannel', { channel })
  updateStore.state = state
  return state
}

/** 启动后延迟自动检测（不打扰用户：不提示最新 / 失败）。 */
export function autoCheckVersion(delayMs: number = 5000): void {
  bindUpdateEvents()
  setTimeout(async () => {
    try {
      const cfg = await main.Call('setting.getAppConfig')
      if (cfg?.updateCheckDisabled) return
    } catch {
      // 拿不到配置时按默认行为继续检测
    }
    checkVersionAndPrompt({ showLatestMessage: false, showErrorMessage: false })
  }, delayMs)
}
