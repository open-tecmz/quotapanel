import { Modal, message } from 'ant-design-vue'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { main } from '../api/call'
import { useAppStore } from '../stores/app'

export const isAppStoreBuild = import.meta.env.VITE_APPSTORE_BUILD === 'true'

let cachedAppVersion: string | null = null

function compareVersions(v1: string, v2: string): number {
  const clean1 = v1.replace(/^v/, '')
  const clean2 = v2.replace(/^v/, '')
  const parts1 = clean1.split('.').map(Number)
  const parts2 = clean2.split('.').map(Number)
  const maxLength = Math.max(parts1.length, parts2.length)
  for (let i = 0; i < maxLength; i++) {
    const num1 = parts1[i] || 0
    const num2 = parts2[i] || 0
    if (num1 > num2) return 1
    if (num1 < num2) return -1
  }
  return 0
}

export async function getAppVersion(): Promise<string> {
  if (cachedAppVersion) return cachedAppVersion
  const cfg = await main.Call('setting.getAppConfig')
  cachedAppVersion = (cfg.version as string) || ''
  return cachedAppVersion
}

export interface VersionCheckOptions {
  showLatestMessage?: boolean
  showErrorMessage?: boolean
}

export async function checkVersionAndPrompt(options: VersionCheckOptions = {}): Promise<boolean> {
  const { showLatestMessage = false, showErrorMessage = false } = options
  const appStore = useAppStore()

  try {
    const currentVersion = await getAppVersion()
    let versionInfo = await main.Call('setting.checkVersion')

    if (typeof versionInfo === 'string') {
      versionInfo = JSON.parse(versionInfo)
    }

    const newVersion = versionInfo.version || 'unknown'
    const comparison = compareVersions(newVersion, currentVersion)

    if (comparison <= 0) {
      if (showLatestMessage) {
        message.success(appStore.t('settings.version.latestVersion'))
      }
      return false
    }

    if (versionInfo.url) {
      Modal.confirm({
        title: appStore.t('settings.version.updateAvailable'),
        content: appStore.t('settings.version.updateConfirm', {
          version: newVersion,
        }),
        okText: appStore.t('common.yes'),
        cancelText: appStore.t('common.no'),
        onOk() {
          BrowserOpenURL(versionInfo.url!)
        },
      })
    }

    return true
  } catch (e) {
    console.error('Version check failed:', e)
    if (showErrorMessage) {
      message.error(appStore.t('settings.version.checkFailed'))
    }
    return false
  }
}

export function autoCheckVersion(delayMs: number = 5000): void {
  setTimeout(() => {
    checkVersionAndPrompt({
      showLatestMessage: false,
      showErrorMessage: false,
    })
  }, delayMs)
}
