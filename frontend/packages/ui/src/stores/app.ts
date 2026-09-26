import { defineStore } from 'pinia'
import { ref } from 'vue'
import { main, type UserInfo } from '../api/call'
import { i18n } from '../plugins/i18n'

export type { UserInfo }

export const useAppStore = defineStore('app', () => {
  const locale = ref<'zh' | 'en'>('zh')
  const isDark = ref(false)
  const userInfo = ref<UserInfo | null>(null)

  const loadUserInfo = async () => {
    try {
      const info = await main.Call('setting.getUserInfo')
      userInfo.value = info.id > 0 ? info : null
    } catch {
      userInfo.value = null
    }
  }

  const t = (key: string, params?: Record<string, unknown>) => i18n.global.t(key, params || {})

  const setLocale = async (nextLocale: 'zh' | 'en') => {
    locale.value = nextLocale
    i18n.global.locale.value = nextLocale
    try {
      await main.Call('setting.setLocale', { locale: nextLocale })
    } catch (error) {
      console.error('Failed to save locale setting:', error)
    }
  }

  const setTheme = async (nextIsDark: boolean) => {
    isDark.value = nextIsDark
    document.documentElement.classList.toggle('dark', nextIsDark)
    localStorage.setItem('theme', nextIsDark ? 'dark' : 'light')
  }

  const initSettings = async () => {
    // 先同步应用本地主题，避免等待后端读取语言时页面短暂使用错误配色。
    const savedTheme = localStorage.getItem('theme')
    if (savedTheme) {
      const dark = savedTheme === 'dark'
      isDark.value = dark
      document.documentElement.classList.toggle('dark', dark)
    }
    try {
      const localeVal = await main.Call('setting.getLocale')
      if (localeVal) {
        const loc = localeVal as 'zh' | 'en'
        locale.value = loc
        i18n.global.locale.value = loc
      }
    } catch (error) {
      console.error('Failed to load settings:', error)
    }
  }

  return {
    locale,
    isDark,
    userInfo,
    setLocale,
    setTheme,
    t,
    initSettings,
    loadUserInfo,
  }
})
