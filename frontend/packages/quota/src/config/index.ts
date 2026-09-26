import type { Component } from 'vue'
import { useI18n } from 'vue-i18n'
import QuotaAccountView from '../views/QuotaAccount.vue'
import QuotaSettingView from '../views/QuotaSetting.vue'

export interface NavItem {
  path: string
  iconName: string
  label: string
}

export interface ModuleOption {
  value: string
  label: string
  iconName: string
  defaultRoute: string
  isFullscreen: boolean
}

// 纯静态路由，可在模块加载时使用（无需 setup 上下文）
export const quotaRoutes = [
  { path: '/quota/account', component: QuotaAccountView as Component },
  { path: '/quota/setting', component: QuotaSettingView as Component },
]

// 需在 Vue setup 上下文中调用（使用 useI18n）
export function useQuotaConfig() {
  const { t } = useI18n()

  const navItems: NavItem[] = [
    {
      path: '/quota/account',
      iconName: 'Wallet',
      label: t('quota.nav.account'),
    },
    {
      path: '/quota/setting',
      iconName: 'Settings',
      label: t('quota.nav.setting'),
    },
  ]

  const moduleOption: ModuleOption = {
    value: 'quota',
    label: t('quota.module'),
    iconName: 'Wallet',
    defaultRoute: '/quota/account',
    isFullscreen: false,
  }

  return { routes: quotaRoutes, navItems, moduleOption }
}
